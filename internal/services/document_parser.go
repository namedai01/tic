package services

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	// "github.com/unidoc/unioffice/document" // Removed unused import
	"tic-knowledge-system/internal/models"
	"gorm.io/gorm"
)

type DocumentParserService struct {
	db              *gorm.DB
	knowledgeService *KnowledgeService
}

func NewDocumentParserService(db *gorm.DB, knowledgeService *KnowledgeService) *DocumentParserService {
	return &DocumentParserService{
		db:              db,
		knowledgeService: knowledgeService,
	}
}

type DocumentParseRequest struct {
	FilePath     string `json:"file_path"`
	TemplateID   string `json:"template_id,omitempty"`
	CreatedBy    string `json:"created_by"`
	Title        string `json:"title,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	ChunkSize    int    `json:"chunk_size,omitempty"` // For splitting large documents
}

type LegacyDocumentParseResult struct {
	KnowledgeEntries []models.KnowledgeEntry `json:"knowledge_entries"`
	TotalChunks      int                     `json:"total_chunks"`
	ParsedAt         time.Time               `json:"parsed_at"`
	OriginalFile     string                  `json:"original_file"`
}

func (s *DocumentParserService) ParseWordDocument(req DocumentParseRequest) (*LegacyDocumentParseResult, error) {
	log.Printf("[INFO] Starting document parsing for file: %s", req.FilePath)
	
	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(req.FilePath), ".docx") {
		return nil, fmt.Errorf("unsupported file format, only .docx files are supported")
	}

	// Extract content from Word document
	content, err := s.extractWordContent(req.FilePath)
	if err != nil {
		log.Printf("[ERROR] Failed to extract content from Word document: %v", err)
		return nil, fmt.Errorf("failed to extract content: %w", err)
	}

	log.Printf("[INFO] Successfully extracted %d characters from document", len(content))

	// Set default values
	if req.ChunkSize <= 0 {
		req.ChunkSize = 3000 // Default chunk size for large documents
	}

	if req.Title == "" {
		req.Title = s.generateTitleFromFilename(req.FilePath)
	}

	// Split content into chunks if necessary
	chunks := s.splitContent(content, req.ChunkSize)
	log.Printf("[INFO] Split document into %d chunks", len(chunks))

	// Create knowledge entries for each chunk
	var knowledgeEntries []models.KnowledgeEntry
	
	for i, chunk := range chunks {
		entry, err := s.createKnowledgeEntry(req, chunk, i+1, len(chunks))
		if err != nil {
			log.Printf("[ERROR] Failed to create knowledge entry for chunk %d: %v", i+1, err)
			continue
		}
		
		knowledgeEntries = append(knowledgeEntries, *entry)
		log.Printf("[INFO] Created knowledge entry for chunk %d/%d, ID: %s", i+1, len(chunks), entry.ID)
	}

	result := &LegacyDocumentParseResult{
		KnowledgeEntries: knowledgeEntries,
		TotalChunks:      len(chunks),
		ParsedAt:         time.Now(),
		OriginalFile:     req.FilePath,
	}

	log.Printf("[INFO] Successfully parsed document into %d knowledge entries", len(knowledgeEntries))
	return result, nil
}

func (s *DocumentParserService) extractWordContent(filePath string) (string, error) {
	log.Printf("[DEBUG] Opening Word document: %s", filePath)
	
	// TODO: This function uses unioffice library which has licensing issues
	// Use the DocumentService.ParseDOCXFile instead
	return "", fmt.Errorf("this function is deprecated, use DocumentService.ParseDOCXFile instead")
	
	/*
	// Extract text from tables
	for _, table := range doc.Tables() {
		for _, row := range table.Rows() {
			for _, cell := range row.Cells() {
				for _, para := range cell.Paragraphs() {
					for _, run := range para.Runs() {
						content.WriteString(run.Text())
					}
					content.WriteString(" | ")
				}
				content.WriteString("\n")
			}
		}
	}

	// Extract text from headers and footers
	for _, header := range doc.Headers() {
		for _, para := range header.Paragraphs() {
			for _, run := range para.Runs() {
				content.WriteString(run.Text())
			}
			content.WriteString("\n")
		}
	}

	for _, footer := range doc.Footers() {
		for _, para := range footer.Paragraphs() {
			for _, run := range para.Runs() {
				content.WriteString(run.Text())
			}
			content.WriteString("\n")
		}
	}

	log.Printf("[DEBUG] Extracted %d characters from Word document", content.Len())
	return content.String(), nil
	*/
}

func (s *DocumentParserService) splitContent(content string, chunkSize int) []string {
	if len(content) <= chunkSize {
		return []string{content}
	}

	var chunks []string
	paragraphs := strings.Split(content, "\n")
	
	var currentChunk strings.Builder
	
	for _, paragraph := range paragraphs {
		// If adding this paragraph would exceed chunk size, start a new chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+len(paragraph) > chunkSize {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}
		
		// If a single paragraph is longer than chunk size, split it
		if len(paragraph) > chunkSize {
			// Split long paragraph into sentences or words
			words := strings.Fields(paragraph)
			var sentenceBuilder strings.Builder
			
			for _, word := range words {
				if sentenceBuilder.Len() > 0 && sentenceBuilder.Len()+len(word) > chunkSize {
					if currentChunk.Len() > 0 {
						chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
						currentChunk.Reset()
					}
					chunks = append(chunks, strings.TrimSpace(sentenceBuilder.String()))
					sentenceBuilder.Reset()
				}
				
				if sentenceBuilder.Len() > 0 {
					sentenceBuilder.WriteString(" ")
				}
				sentenceBuilder.WriteString(word)
			}
			
			if sentenceBuilder.Len() > 0 {
				if currentChunk.Len() > 0 {
					currentChunk.WriteString("\n")
				}
				currentChunk.WriteString(sentenceBuilder.String())
			}
		} else {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n")
			}
			currentChunk.WriteString(paragraph)
		}
	}
	
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}
	
	return chunks
}

func (s *DocumentParserService) createKnowledgeEntry(req DocumentParseRequest, content string, chunkIndex, totalChunks int) (*models.KnowledgeEntry, error) {
	// Generate title for chunk
	title := req.Title
	if totalChunks > 1 {
		title = fmt.Sprintf("%s (Part %d of %d)", req.Title, chunkIndex, totalChunks)
	}

	// Generate summary from first 200 characters
	summary := content
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}

	// Convert tags to JSON string
	tagsJSON := "[]"
	if len(req.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Tags)
		tagsJSON = string(tagsBytes)
	}

	// Create field data with document metadata
	fieldDataMap := map[string]interface{}{
		"source_file":    req.FilePath,
		"chunk_index":    chunkIndex,
		"total_chunks":   totalChunks,
		"document_type":  "docx",
		"parsed_at":      time.Now().Format(time.RFC3339),
	}
	fieldDataBytes, _ := json.Marshal(fieldDataMap)
	fieldData := string(fieldDataBytes)

	// Parse UUIDs
	entryID := uuid.New()
	createdByUUID, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("invalid created_by UUID: %w", err)
	}

	var templateID *uuid.UUID
	if req.TemplateID != "" {
		parsedTemplateID, err := uuid.Parse(req.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("invalid template_id UUID: %w", err)
		}
		templateID = &parsedTemplateID
	}

	// Create knowledge entry
	entry := &models.KnowledgeEntry{
		ID:          entryID,
		TemplateID:  templateID,
		Title:       title,
		Content:     content,
		Summary:     summary,
		Category:    "imported_document",
		Tags:        tagsJSON,
		FieldData:   fieldData,
		IsPublished: true,
		Priority:    0,
		ViewCount:   0,
		CreatedBy:   createdByUUID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to database
	if err := s.db.Create(entry).Error; err != nil {
		return nil, fmt.Errorf("failed to save knowledge entry: %w", err)
	}

	log.Printf("[DEBUG] Created knowledge entry: %s", entry.ID)
	return entry, nil
}

func (s *DocumentParserService) generateTitleFromFilename(filePath string) string {
	filename := filepath.Base(filePath)
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Replace underscores and hyphens with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	// Capitalize first letter of each word
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// ParseDocumentFromPath is a helper function to parse a document with minimal configuration
func (s *DocumentParserService) ParseDocumentFromPath(filePath, createdBy string) (*LegacyDocumentParseResult, error) {
	req := DocumentParseRequest{
		FilePath:  filePath,
		CreatedBy: createdBy,
		Tags:      []string{"imported", "document"},
	}
	
	return s.ParseWordDocument(req)
}
