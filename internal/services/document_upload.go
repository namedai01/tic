package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"tic-knowledge-system/internal/models"
)

type FileUploadService struct {
	db            *gorm.DB
	openaiAPIKey  string
	vectorStoreID string
	uploadDir     string
}

type DocumentUploadRequest struct {
	FileName string `json:"file_name" validate:"required"`
}

type DocumentUploadResponse struct {
	ID               uuid.UUID `json:"id"`
	FileName         string    `json:"file_name"`
	Status           string    `json:"status"`
	OpenAIFileID     string    `json:"openai_file_id,omitempty"`
	VectorFileID     string    `json:"vector_file_id,omitempty"`
	Message          string    `json:"message"`
}

type OpenAIFileUploadResponse struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Bytes    int    `json:"bytes"`
	Filename string `json:"filename"`
	Purpose  string `json:"purpose"`
}

type VectorStoreFileResponse struct {
	ID            string `json:"id"`
	Object        string `json:"object"`
	VectorStoreID string `json:"vector_store_id"`
	Status        string `json:"status"`
}

func NewFileUploadService(db *gorm.DB, openaiAPIKey, vectorStoreID, uploadDir string) *FileUploadService {
	return &FileUploadService{
		db:            db,
		openaiAPIKey:  openaiAPIKey,
		vectorStoreID: vectorStoreID,
		uploadDir:     uploadDir,
	}
}

func (s *FileUploadService) UploadDocument(ctx context.Context, req DocumentUploadRequest, fileContent []byte, originalFileName string, mimeType string, uploadedBy uuid.UUID) (*DocumentUploadResponse, error) {
	// Step 1: Save file locally
	filePath := filepath.Join(s.uploadDir, req.FileName)
	if err := os.WriteFile(filePath, fileContent, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file locally: %w", err)
	}

	// Create database record
	document := &models.UploadedDocument{
		FileName:         req.FileName,
		OriginalFileName: originalFileName,
		FilePath:         filePath,
		FileSize:         int64(len(fileContent)),
		MimeType:         mimeType,
		VectorStoreID:    s.vectorStoreID,
		Status:           models.DocumentUploaded,
		UploadedBy:       uploadedBy,
	}

	if err := s.db.Create(document).Error; err != nil {
		// Clean up file if database insert fails
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create document record: %w", err)
	}

	response := &DocumentUploadResponse{
		ID:       document.ID,
		FileName: document.FileName,
		Status:   string(document.Status),
		Message:  "Document uploaded successfully",
	}

	// Step 2: Upload to OpenAI (async)
	go s.processOpenAIUpload(document.ID, filePath, req.FileName)

	return response, nil
}

func (s *FileUploadService) processOpenAIUpload(documentID uuid.UUID, filePath, fileName string) {
	// Step 1: Upload to OpenAI Files API
	openaiFileID, err := s.uploadToOpenAI(filePath, fileName)
	if err != nil {
		s.updateDocumentStatus(documentID, models.DocumentProcessingFailed, "", "", err.Error())
		return
	}

	// Update document with OpenAI file ID
	s.updateDocumentStatus(documentID, models.DocumentSentToOpenAI, openaiFileID, "", "")

	// Step 2: Add to Vector Store
	vectorFileID, err := s.addToVectorStore(openaiFileID)
	if err != nil {
		s.updateDocumentStatus(documentID, models.DocumentProcessingFailed, openaiFileID, "", err.Error())
		return
	}

	// Final update
	s.updateDocumentStatus(documentID, models.DocumentAddedToVector, openaiFileID, vectorFileID, "")
}

func (s *FileUploadService) uploadToOpenAI(filePath, fileName string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add purpose field
	if err := writer.WriteField("purpose", "assistants"); err != nil {
		return "", fmt.Errorf("failed to write purpose field: %w", err)
	}

	// Add file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/files", &b)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.openaiAPIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var uploadResp OpenAIFileUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return uploadResp.ID, nil
}

func (s *FileUploadService) addToVectorStore(fileID string) (string, error) {
	requestBody := map[string]string{
		"file_id": fileID,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://api.openai.com/v1/vector_stores/%s/files", s.vectorStoreID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.openaiAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Vector Store API error: %d - %s", resp.StatusCode, string(body))
	}

	var vectorResp VectorStoreFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&vectorResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return vectorResp.ID, nil
}

func (s *FileUploadService) updateDocumentStatus(documentID uuid.UUID, status models.DocumentStatus, openaiFileID, vectorFileID, errorMessage string) {
	updates := map[string]interface{}{
		"status":      status,
		"updated_at":  time.Now(),
	}

	if openaiFileID != "" {
		updates["openai_file_id"] = openaiFileID
	}
	if vectorFileID != "" {
		updates["vector_file_id"] = vectorFileID
	}
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	s.db.Model(&models.UploadedDocument{}).Where("id = ?", documentID).Updates(updates)
}

func (s *FileUploadService) GetDocumentStatus(ctx context.Context, documentID uuid.UUID) (*models.UploadedDocument, error) {
	var document models.UploadedDocument
	if err := s.db.Preload("Uploader").First(&document, documentID).Error; err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}
	return &document, nil
}

func (s *FileUploadService) ListDocuments(ctx context.Context, uploadedBy *uuid.UUID, limit, offset int) ([]models.UploadedDocument, int64, error) {
	var documents []models.UploadedDocument
	var total int64

	query := s.db.Model(&models.UploadedDocument{}).Preload("Uploader")
	
	if uploadedBy != nil {
		query = query.Where("uploaded_by = ?", *uploadedBy)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %w", err)
	}

	if err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&documents).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list documents: %w", err)
	}

	return documents, total, nil
}
