package handlers

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"tic-knowledge-system/internal/services"
)

// DocumentHandler handles document-related API endpoints
type DocumentHandler struct {
	documentService *services.DocumentService
	logger          *log.Logger
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(documentService *services.DocumentService, logger *log.Logger) *DocumentHandler {
	return &DocumentHandler{
		documentService: documentService,
		logger:          logger,
	}
}

// ProcessDocumentRequest represents the request for processing a document
type ProcessDocumentRequest struct {
	FilePath     string `json:"file_path" example:"./file/WB.docx"`
	CategoryName string `json:"category_name" example:"Work Procedures"`
	UserID       string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ProcessDocumentResponse represents the response for document processing
type ProcessDocumentResponse struct {
	Success      bool                             `json:"success"`
	Message      string                           `json:"message"`
	Result       *services.DocumentParseResult    `json:"result,omitempty"`
	Error        string                           `json:"error,omitempty"`
}

// ParseDocumentResponse represents the response for document parsing only
type ParseDocumentResponse struct {
	Success bool                             `json:"success"`
	Message string                           `json:"message"`
	Result  *services.DocumentParseResult    `json:"result,omitempty"`
	Error   string                           `json:"error,omitempty"`
}

// ProcessDocument processes a document (parse + save to knowledge base)
// @Summary Process a document file
// @Description Parse a DOCX document and save it to the knowledge base
// @Tags documents
// @Accept json
// @Produce json
// @Param request body ProcessDocumentRequest true "Document processing request"
// @Success 200 {object} ProcessDocumentResponse
// @Failure 400 {object} ProcessDocumentResponse
// @Failure 500 {object} ProcessDocumentResponse
// @Router /api/documents/process [post]
func (dh *DocumentHandler) ProcessDocument(c *fiber.Ctx) error {
	dh.logger.Printf("Received document processing request")
	
	var req ProcessDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		dh.logger.Printf("Error parsing request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(ProcessDocumentResponse{
			Success: false,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
	}
	
	// Validate required fields
	if req.FilePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ProcessDocumentResponse{
			Success: false,
			Message: "File path is required",
			Error:   "file_path cannot be empty",
		})
	}
	
	if req.CategoryName == "" {
		req.CategoryName = "Documents" // Default category
	}
	
	if req.UserID == "" {
		req.UserID = uuid.New().String() // Generate new user ID if not provided
	}
	
	// Validate user ID format
	if _, err := uuid.Parse(req.UserID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ProcessDocumentResponse{
			Success: false,
			Message: "Invalid user ID format",
			Error:   "user_id must be a valid UUID",
		})
	}
	
	dh.logger.Printf("Processing document: %s, Category: %s, User: %s", req.FilePath, req.CategoryName, req.UserID)
	
	// Process the document
	result, err := dh.documentService.ProcessDocument(req.FilePath, req.CategoryName, req.UserID)
	if err != nil {
		dh.logger.Printf("Error processing document: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ProcessDocumentResponse{
			Success: false,
			Message: "Failed to process document",
			Error:   err.Error(),
		})
	}
	
	dh.logger.Printf("Document processed successfully: %s", req.FilePath)
	
	return c.Status(fiber.StatusOK).JSON(ProcessDocumentResponse{
		Success: true,
		Message: fmt.Sprintf("Document processed successfully. Created %d knowledge entries.", len(result.KnowledgeIDs)),
		Result:  result,
	})
}

// ParseDocument parses a document without saving to knowledge base
// @Summary Parse a document file
// @Description Parse a DOCX document and return structured content without saving
// @Tags documents
// @Accept json
// @Produce json
// @Param file_path query string true "Path to the document file"
// @Success 200 {object} ParseDocumentResponse
// @Failure 400 {object} ParseDocumentResponse
// @Failure 500 {object} ParseDocumentResponse
// @Router /api/documents/parse [get]
func (dh *DocumentHandler) ParseDocument(c *fiber.Ctx) error {
	filePath := c.Query("file_path")
	if filePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ParseDocumentResponse{
			Success: false,
			Message: "File path is required",
			Error:   "file_path parameter is missing",
		})
	}
	
	dh.logger.Printf("Parsing document: %s", filePath)
	
	// Parse the document
	result, err := dh.documentService.ParseDOCXFile(filePath)
	if err != nil {
		dh.logger.Printf("Error parsing document: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ParseDocumentResponse{
			Success: false,
			Message: "Failed to parse document",
			Error:   err.Error(),
		})
	}
	
	dh.logger.Printf("Document parsed successfully: %s", filePath)
	
	return c.Status(fiber.StatusOK).JSON(ParseDocumentResponse{
		Success: true,
		Message: fmt.Sprintf("Document parsed successfully. Found %d sections.", len(result.Sections)),
		Result:  result,
	})
}

// ProcessWBDocument is a convenience endpoint specifically for the WB.docx file
// @Summary Process the WB.docx document
// @Description Parse and save the WB.docx document to the knowledge base
// @Tags documents
// @Accept json
// @Produce json
// @Param category_name query string false "Category name for the document" default:"Work Procedures"
// @Param user_id query string false "User ID (UUID format)"
// @Success 200 {object} ProcessDocumentResponse
// @Failure 400 {object} ProcessDocumentResponse
// @Failure 500 {object} ProcessDocumentResponse
// @Router /api/documents/process-wb [post]
func (dh *DocumentHandler) ProcessWBDocument(c *fiber.Ctx) error {
	dh.logger.Printf("Processing WB.docx document")
	
	// Default values for WB.docx
	filePath := "./file/WB.docx" // Relative to the project root
	categoryName := c.Query("category_name", "Work Procedures")
	userID := c.Query("user_id")
	
	if userID == "" {
		userID = uuid.New().String()
	}
	
	// Validate user ID format
	if _, err := uuid.Parse(userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ProcessDocumentResponse{
			Success: false,
			Message: "Invalid user ID format",
			Error:   "user_id must be a valid UUID",
		})
	}
	
	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		dh.logger.Printf("Error getting absolute path for %s: %v", filePath, err)
		absPath = filePath // Fallback to relative path
	}
	
	dh.logger.Printf("Processing WB.docx at: %s", absPath)
	
	// Process the document
	result, err := dh.documentService.ProcessDocument(absPath, categoryName, userID)
	if err != nil {
		dh.logger.Printf("Error processing WB.docx: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ProcessDocumentResponse{
			Success: false,
			Message: "Failed to process WB.docx document",
			Error:   err.Error(),
		})
	}
	
	dh.logger.Printf("WB.docx processed successfully")
	
	return c.Status(fiber.StatusOK).JSON(ProcessDocumentResponse{
		Success: true,
		Message: fmt.Sprintf("WB.docx processed successfully. Created %d knowledge entries in category '%s'.", len(result.KnowledgeIDs), categoryName),
		Result:  result,
	})
}