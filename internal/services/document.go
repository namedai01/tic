package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nguyenthenguyen/docx"
	"gorm.io/gorm"
	"tic-knowledge-system/internal/models"
)

// DocumentService handles document parsing and processing
type DocumentService struct {
	db        *gorm.DB
	aiService *UnifiedAIService
	logger    *log.Logger
}

// NewDocumentService creates a new document service
func NewDocumentService(db *gorm.DB, aiService *UnifiedAIService, logger *log.Logger) *DocumentService {
	return &DocumentService{
		db:        db,
		aiService: aiService,
		logger:    logger,
	}
}

// DocumentParseResult represents the result of parsing a document
type DocumentParseResult struct {
	FilePath     string                    `json:"file_path"`
	Title        string                    `json:"title"`
	Sections     []DocumentSection         `json:"sections"`
	TotalChunks  int                      `json:"total_chunks"`
	ProcessedAt  time.Time                `json:"processed_at"`
	KnowledgeIDs []string                 `json:"knowledge_ids"`
	Metadata     map[string]interface{}   `json:"metadata"`
}

// DocumentSection represents a section of the document
type DocumentSection struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Order    int    `json:"order"`
	WordCount int   `json:"word_count"`
}

// ParseDOCXFile parses a DOCX file and extracts structured content
func (ds *DocumentService) ParseDOCXFile(filePath string) (*DocumentParseResult, error) {
	ds.logger.Printf("Starting DOCX parsing for file: %s", filePath)
	
	// Read the DOCX file
	reader, err := docx.ReadDocxFile(filePath)
	if err != nil {
		ds.logger.Printf("Error reading DOCX file %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to read DOCX file: %w", err)
	}
	defer reader.Close()
	
	// Extract document content
	doc := reader.Editable()
	content := doc.GetContent()
	
	if content == "" {
		ds.logger.Printf("Warning: No content found in DOCX file %s", filePath)
		return nil, errors.New("no content found in document")
	}
	
	ds.logger.Printf("Successfully extracted content from DOCX file, length: %d characters", len(content))
	
	// Get document title from filename
	fileName := filepath.Base(filePath)
	title := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	
	// Generate a better title using AI
	if ds.aiService != nil {
		ctx := context.Background()
		if aiTitle, err := ds.aiService.GenerateTitle(ctx, content); err == nil && aiTitle != "" {
			title = aiTitle
			ds.logger.Printf("Generated AI title: %s", title)
		}
	}
	
	// Split content into manageable sections
	sections := ds.splitIntoSections(content)
	ds.logger.Printf("Split document into %d sections", len(sections))
	
	result := &DocumentParseResult{
		FilePath:    filePath,
		Title:       title,
		Sections:    sections,
		TotalChunks: len(sections),
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"file_type":      "docx",
			"file_size":      len(content),
			"sections_count": len(sections),
			"extracted_at":   time.Now().Format(time.RFC3339),
		},
	}
	
	return result, nil
}

// splitIntoSections splits content into logical sections
func (ds *DocumentService) splitIntoSections(content string) []DocumentSection {
	// Simple section splitting based on double newlines and length
	const maxSectionLength = 2000
	const minSectionLength = 100
	
	var sections []DocumentSection
	
	// First, try to split by double newlines (paragraphs)
	paragraphs := strings.Split(content, "\n\n")
	
	currentSection := ""
	sectionOrder := 0
	
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		
		// If adding this paragraph would make the section too long, save current section
		if len(currentSection)+len(paragraph) > maxSectionLength && len(currentSection) > minSectionLength {
			if currentSection != "" {
				sections = append(sections, DocumentSection{
					Title:     ds.generateSectionTitle(currentSection, sectionOrder),
					Content:   strings.TrimSpace(currentSection),
					Order:     sectionOrder,
					WordCount: len(strings.Fields(currentSection)),
				})
				sectionOrder++
				currentSection = ""
			}
		}
		
		// Add paragraph to current section
		if currentSection != "" {
			currentSection += "\n\n"
		}
		currentSection += paragraph
	}
	
	// Add the last section if it exists
	if currentSection != "" && len(currentSection) > minSectionLength {
		sections = append(sections, DocumentSection{
			Title:     ds.generateSectionTitle(currentSection, sectionOrder),
			Content:   strings.TrimSpace(currentSection),
			Order:     sectionOrder,
			WordCount: len(strings.Fields(currentSection)),
		})
	}
	
	// If no sections were created, create one from the entire content
	if len(sections) == 0 && content != "" {
		sections = append(sections, DocumentSection{
			Title:     "Document Content",
			Content:   content,
			Order:     0,
			WordCount: len(strings.Fields(content)),
		})
	}
	
	return sections
}

// generateSectionTitle generates a title for a section based on its content
func (ds *DocumentService) generateSectionTitle(content string, order int) string {
	// Extract first meaningful line or first few words
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 && len(line) < 100 {
			return line
		}
	}
	
	// Fallback: use first 50 characters
	if len(content) > 50 {
		return strings.TrimSpace(content[:50]) + "..."
	}
	
	return fmt.Sprintf("Section %d", order+1)
}

// SaveToKnowledgeBase saves parsed document sections to the knowledge base
func (ds *DocumentService) SaveToKnowledgeBase(result *DocumentParseResult, categoryName string, userID string) error {
	ds.logger.Printf("Saving document to knowledge base: %s", result.Title)
	
	// Get or create user
	var user models.User
	err := ds.db.Where("id = ?", userID).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		user = models.User{
			ID:        uuid.MustParse(userID),
			Name:      "system",
			Email:     fmt.Sprintf("user-%s@example.com", userID[:8]),
			Role:      models.RegularUser,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := ds.db.Create(&user).Error; err != nil {
			ds.logger.Printf("Error creating user %s: %v", userID, err)
			return fmt.Errorf("failed to create user: %w", err)
		}
		ds.logger.Printf("Created new user: %s", userID)
	} else if err != nil {
		ds.logger.Printf("Error finding user %s: %v", userID, err)
		return fmt.Errorf("failed to find user: %w", err)
	}
	
	var knowledgeIDs []string
	
	// Process each section
	for i, section := range result.Sections {
		ds.logger.Printf("Processing section %d/%d: %s", i+1, len(result.Sections), section.Title)
		
		// Create knowledge entry
		knowledge := models.KnowledgeEntry{
			ID:          uuid.New(),
			Title:       section.Title,
			Content:     section.Content,
			Category:    categoryName,
			Tags:        fmt.Sprintf("document,section-%d,word-count-%d", section.Order, section.WordCount),
			FieldData:   "{}",  // Empty JSON object
			IsPublished: true,
			Priority:    0,
			ViewCount:   0,
			CreatedBy:   user.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		
		// Save knowledge entry
		if err := ds.db.Create(&knowledge).Error; err != nil {
			ds.logger.Printf("Error saving knowledge entry for section %d: %v", i+1, err)
			return fmt.Errorf("failed to save knowledge entry: %w", err)
		}
		
		knowledgeIDs = append(knowledgeIDs, knowledge.ID.String())
		
		// Generate and save embeddings
		if ds.aiService != nil {
			ds.logger.Printf("Generating embeddings for section %d", i+1)
			
			// Create combined text for embedding
			embeddingText := fmt.Sprintf("Title: %s\n\nContent: %s", section.Title, section.Content)
			
			ctx := context.Background()
			_, err := ds.aiService.CreateEmbedding(ctx, embeddingText, OpenAIProvider)
			if err != nil {
				ds.logger.Printf("Warning: Failed to create embedding for section %d: %v", i+1, err)
				continue // Don't fail the entire process for embedding errors
			}
			
			// Save vector embedding (without actual embedding vector for now)
			vectorEmbedding := models.VectorEmbedding{
				ID:               uuid.New(),
				KnowledgeEntryID: knowledge.ID,
				VectorID:         knowledge.ID.String(), // Use knowledge ID as vector ID
				ChunkIndex:       0,
				ChunkText:        embeddingText,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			
			if err := ds.db.Create(&vectorEmbedding).Error; err != nil {
				ds.logger.Printf("Warning: Failed to save vector embedding for section %d: %v", i+1, err)
				continue // Don't fail the entire process for vector errors
			}
			
			ds.logger.Printf("Successfully created embedding for section %d", i+1)
		}
	}
	
	// Update result with knowledge IDs
	result.KnowledgeIDs = knowledgeIDs
	
	ds.logger.Printf("Successfully saved document to knowledge base. Created %d knowledge entries", len(knowledgeIDs))
	return nil
}

// ProcessDocument is a convenience method that parses and saves a document in one call
func (ds *DocumentService) ProcessDocument(filePath, categoryName, userID string) (*DocumentParseResult, error) {
	ds.logger.Printf("Processing document: %s", filePath)
	
	// Parse the document
	result, err := ds.ParseDOCXFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}
	
	// Save to knowledge base
	if err := ds.SaveToKnowledgeBase(result, categoryName, userID); err != nil {
		return nil, fmt.Errorf("failed to save to knowledge base: %w", err)
	}
	
	ds.logger.Printf("Document processing completed successfully: %s", filePath)
	return result, nil
}
