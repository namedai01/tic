package handlers

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"tic-knowledge-system/internal/models"
	"tic-knowledge-system/internal/services"
)

type FileUploadHandler struct {
	uploadService *services.FileUploadService
	db            *gorm.DB
	logger        *log.Logger
}

func NewFileUploadHandler(uploadService *services.FileUploadService, db *gorm.DB, logger *log.Logger) *FileUploadHandler {
	return &FileUploadHandler{
		uploadService: uploadService,
		db:            db,
		logger:        logger,
	}
}

// UploadDocument handles file upload to OpenAI and vector store
// @Summary Upload document file
// @Description Upload a document file, store it locally, then upload to OpenAI and add to vector store
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param file_name formData string true "File name"
// @Param file formData file true "Document file"
// @Success 200 {object} services.DocumentUploadResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /documents/upload [post]
func (h *FileUploadHandler) UploadDocument(c *fiber.Ctx) error {
	// Get file name from form
	fileName := c.FormValue("file_name")
	if fileName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file_name is required",
		})
	}

	// Get file from form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		h.logger.Printf("Error getting file from form: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file is required",
		})
	}

	// Open and read file content
	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Printf("Error opening file: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process file",
		})
	}
	defer file.Close()

	// Read file content
	fileContent := make([]byte, fileHeader.Size)
	_, err = file.Read(fileContent)
	if err != nil {
		h.logger.Printf("Error reading file content: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read file content",
		})
	}

	// Create or get default user for uploads
	uploadedBy, err := h.getOrCreateDefaultUser()
	if err != nil {
		h.logger.Printf("Error getting/creating default user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set up user for upload",
		})
	}

	// Create upload request
	req := services.DocumentUploadRequest{
		FileName: fileName,
	}

	// Upload document
	response, err := h.uploadService.UploadDocument(
		c.Context(),
		req,
		fileContent,
		fileHeader.Filename,
		fileHeader.Header.Get("Content-Type"),
		uploadedBy,
	)

	if err != nil {
		h.logger.Printf("Error uploading document: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to upload document",
			"details": err.Error(),
		})
	}

	return c.JSON(response)
}

// GetDocumentStatus gets the status of an uploaded document
// @Summary Get document upload status
// @Description Get the status of a document upload including OpenAI and vector store processing
// @Tags documents
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} models.UploadedDocument
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /documents/{id}/status [get]
func (h *FileUploadHandler) GetDocumentStatus(c *fiber.Ctx) error {
	// Parse document ID
	idStr := c.Params("id")
	documentID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid document ID",
		})
	}

	// Get document status
	document, err := h.uploadService.GetDocumentStatus(c.Context(), documentID)
	if err != nil {
		h.logger.Printf("Error getting document status: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	return c.JSON(document)
}

// ListDocuments lists uploaded documents
// @Summary List uploaded documents
// @Description List uploaded documents with pagination
// @Tags documents
// @Accept json
// @Produce json
// @Param request body map[string]interface{} false "Request body with limit, offset, uploaded_by"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /documents [post]
func (h *FileUploadHandler) ListDocuments(c *fiber.Ctx) error {
	// Default values
	limit := 10
	offset := 0
	var uploadedBy *uuid.UUID

	// Parse JSON body if provided
	var requestBody map[string]interface{}
	if err := c.BodyParser(&requestBody); err == nil {
		// Parse limit from request body
		if limitVal, ok := requestBody["limit"]; ok {
			if limitFloat, ok := limitVal.(float64); ok {
				limit = int(limitFloat)
			} else if limitStr, ok := limitVal.(string); ok {
				if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
					limit = parsedLimit
				}
			}
		}

		// Parse offset from request body
		if offsetVal, ok := requestBody["offset"]; ok {
			if offsetFloat, ok := offsetVal.(float64); ok {
				offset = int(offsetFloat)
			} else if offsetStr, ok := offsetVal.(string); ok {
				if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
					offset = parsedOffset
				}
			}
		}

		// Parse uploaded_by from request body
		if uploadedByVal, ok := requestBody["uploaded_by"]; ok {
			if uploadedByStr, ok := uploadedByVal.(string); ok && uploadedByStr != "" {
				if parsedUUID, err := uuid.Parse(uploadedByStr); err == nil {
					uploadedBy = &parsedUUID
				}
			}
		}
	}

	// List documents
	documents, total, err := h.uploadService.ListDocuments(c.Context(), uploadedBy, limit, offset)
	if err != nil {
		h.logger.Printf("Error listing documents: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to list documents",
		})
	}

	return c.JSON(fiber.Map{
		"documents": documents,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// getOrCreateDefaultUser creates or returns the default user for file uploads
func (h *FileUploadHandler) getOrCreateDefaultUser() (uuid.UUID, error) {
	defaultUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	
	// Check if user exists
	var user models.User
	err := h.db.First(&user, defaultUserID).Error
	if err == gorm.ErrRecordNotFound {
		// Create default user
		user = models.User{
			ID:       defaultUserID,
			Email:    "default@system.local",
			Name:     "System Default User",
			Role:     models.RegularUser,
			IsActive: true,
		}
		if err := h.db.Create(&user).Error; err != nil {
			return uuid.Nil, err
		}
		h.logger.Printf("Created default user: %s", user.ID)
	} else if err != nil {
		return uuid.Nil, err
	}
	
	return user.ID, nil
}
