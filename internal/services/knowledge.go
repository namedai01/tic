package services

import (
	"context"
	"tic-knowledge-system/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KnowledgeService struct {
	db            *gorm.DB
	openAIService *OpenAIService
	vectorService *VectorService
}

func NewKnowledgeService(db *gorm.DB, openAIService *OpenAIService, vectorService *VectorService) *KnowledgeService {
	return &KnowledgeService{
		db:            db,
		openAIService: openAIService,
		vectorService: vectorService,
	}
}

// Template Management
func (s *KnowledgeService) CreateTemplate(template *models.Template) error {
	return s.db.Create(template).Error
}

func (s *KnowledgeService) GetTemplates(category string, isActive *bool) ([]models.Template, error) {
	var templates []models.Template
	query := s.db.Preload("Fields").Preload("Creator")

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	err := query.Find(&templates).Error
	return templates, err
}

func (s *KnowledgeService) GetTemplateByID(id uuid.UUID) (*models.Template, error) {
	var template models.Template
	err := s.db.Preload("Fields").Preload("Creator").First(&template, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (s *KnowledgeService) UpdateTemplate(template *models.Template) error {
	return s.db.Save(template).Error
}

func (s *KnowledgeService) DeleteTemplate(id uuid.UUID) error {
	return s.db.Delete(&models.Template{}, "id = ?", id).Error
}

// Knowledge Entry Management
func (s *KnowledgeService) CreateKnowledgeEntry(ctx context.Context, entry *models.KnowledgeEntry) error {
	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create the knowledge entry
	if err := tx.Create(entry).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create embeddings if the entry is published
	if entry.IsPublished {
		if err := s.createEmbeddings(ctx, tx, entry); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *KnowledgeService) GetKnowledgeEntries(category string, isPublished *bool, limit, offset int) ([]models.KnowledgeEntry, error) {
	var entries []models.KnowledgeEntry
	query := s.db.Preload("Template").Preload("Creator")

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if isPublished != nil {
		query = query.Where("is_published = ?", *isPublished)
	}

	err := query.Limit(limit).Offset(offset).Order("priority DESC, created_at DESC").Find(&entries).Error
	return entries, err
}

func (s *KnowledgeService) GetKnowledgeEntryByID(id uuid.UUID) (*models.KnowledgeEntry, error) {
	var entry models.KnowledgeEntry
	err := s.db.Preload("Template").Preload("Creator").First(&entry, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	// Increment view count
	s.db.Model(&entry).Update("view_count", gorm.Expr("view_count + ?", 1))

	return &entry, nil
}

func (s *KnowledgeService) UpdateKnowledgeEntry(ctx context.Context, entry *models.KnowledgeEntry) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update the entry
	if err := tx.Save(entry).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update embeddings if content changed and entry is published
	if entry.IsPublished {
		// Delete existing embeddings
		if err := tx.Where("knowledge_entry_id = ?", entry.ID).Delete(&models.VectorEmbedding{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Create new embeddings
		if err := s.createEmbeddings(ctx, tx, entry); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (s *KnowledgeService) DeleteKnowledgeEntry(id uuid.UUID) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete embeddings first
	if err := tx.Where("knowledge_entry_id = ?", id).Delete(&models.VectorEmbedding{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete the entry
	if err := tx.Delete(&models.KnowledgeEntry{}, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *KnowledgeService) SearchKnowledgeEntries(ctx context.Context, query string, limit int) ([]models.KnowledgeEntry, error) {
	// First, try vector search if we have a vector service
	if s.vectorService != nil {
		vectorResults, err := s.vectorService.Search(ctx, query, limit)
		if err == nil && len(vectorResults) > 0 {
			// Get the actual entries based on vector search results
			var entryIDs []uuid.UUID
			for _, result := range vectorResults {
				entryIDs = append(entryIDs, result.KnowledgeEntryID)
			}

			var entries []models.KnowledgeEntry
			err := s.db.Preload("Template").Preload("Creator").
				Where("id IN ? AND is_published = true", entryIDs).
				Find(&entries).Error
			if err == nil {
				return entries, nil
			}
		}
	}

	// Fallback to text search
	var entries []models.KnowledgeEntry
	searchTerm := "%" + query + "%"
	err := s.db.Preload("Template").Preload("Creator").
		Where("is_published = true AND (title ILIKE ? OR content ILIKE ? OR summary ILIKE ?)", 
			searchTerm, searchTerm, searchTerm).
		Limit(limit).
		Order("priority DESC, view_count DESC").
		Find(&entries).Error

	return entries, err
}

func (s *KnowledgeService) createEmbeddings(ctx context.Context, tx *gorm.DB, entry *models.KnowledgeEntry) error {
	// Combine title and content for embedding
	fullText := entry.Title + "\n\n" + entry.Content
	if entry.Summary != "" {
		fullText = entry.Summary + "\n\n" + fullText
	}

	// Chunk the text
	chunks := s.openAIService.ChunkText(fullText, 1000)

	for i, chunk := range chunks {
		// Create embedding for this chunk
		embedding, err := s.openAIService.CreateEmbedding(ctx, chunk)
		if err != nil {
			return err
		}

		// Store in vector database and get vector ID
		vectorID, err := s.vectorService.Store(ctx, embedding, chunk, entry.ID)
		if err != nil {
			return err
		}

		// Store embedding record
		vectorEmbedding := &models.VectorEmbedding{
			KnowledgeEntryID: entry.ID,
			VectorID:         vectorID,
			ChunkIndex:       i,
			ChunkText:        chunk,
		}

		if err := tx.Create(vectorEmbedding).Error; err != nil {
			return err
		}
	}

	return nil
}
