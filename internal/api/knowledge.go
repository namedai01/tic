package api

import (
	"strconv"
	"tic-knowledge-system/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// @Summary Get knowledge entries
// @Description Get knowledge entries with optional filtering and pagination
// @Tags knowledge
// @Accept json
// @Produce json
// @Param category query string false "Filter by category"
// @Param published query boolean false "Filter by published status"
// @Param limit query int false "Limit number of results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} models.KnowledgeEntry
// @Router /knowledge [get]
func (s *Server) getKnowledgeEntries(c *fiber.Ctx) error {
	category := c.Query("category")
	publishedStr := c.Query("published")
	limitStr := c.Query("limit", "20")
	offsetStr := c.Query("offset", "0")

	var isPublished *bool
	if publishedStr != "" {
		published, err := strconv.ParseBool(publishedStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid published parameter"})
		}
		isPublished = &published
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	entries, err := s.knowledgeService.GetKnowledgeEntries(category, isPublished, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch knowledge entries"})
	}

	return c.JSON(entries)
}

// @Summary Create knowledge entry
// @Description Create a new knowledge entry
// @Tags knowledge
// @Accept json
// @Produce json
// @Param entry body models.KnowledgeEntry true "Knowledge entry data"
// @Success 201 {object} models.KnowledgeEntry
// @Router /knowledge [post]
func (s *Server) createKnowledgeEntry(c *fiber.Ctx) error {
	var entry models.KnowledgeEntry
	if err := c.BodyParser(&entry); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// TODO: Get user ID from JWT token
	entry.CreatedBy = uuid.New() // Placeholder

	if err := s.knowledgeService.CreateKnowledgeEntry(c.Context(), &entry); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create knowledge entry"})
	}

	return c.Status(201).JSON(entry)
}

// @Summary Search knowledge entries
// @Description Search knowledge entries by query
// @Tags knowledge
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Limit number of results" default(10)
// @Success 200 {array} models.KnowledgeEntry
// @Router /knowledge/search [get]
func (s *Server) searchKnowledgeEntries(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Query parameter 'q' is required"})
	}

	limitStr := c.Query("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	entries, err := s.knowledgeService.SearchKnowledgeEntries(c.Context(), query, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to search knowledge entries"})
	}

	return c.JSON(entries)
}

// @Summary Get knowledge entry
// @Description Get a knowledge entry by ID
// @Tags knowledge
// @Accept json
// @Produce json
// @Param id path string true "Knowledge entry ID"
// @Success 200 {object} models.KnowledgeEntry
// @Router /knowledge/{id} [get]
func (s *Server) getKnowledgeEntry(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid knowledge entry ID"})
	}

	entry, err := s.knowledgeService.GetKnowledgeEntryByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Knowledge entry not found"})
	}

	return c.JSON(entry)
}

// @Summary Update knowledge entry
// @Description Update an existing knowledge entry
// @Tags knowledge
// @Accept json
// @Produce json
// @Param id path string true "Knowledge entry ID"
// @Param entry body models.KnowledgeEntry true "Knowledge entry data"
// @Success 200 {object} models.KnowledgeEntry
// @Router /knowledge/{id} [put]
func (s *Server) updateKnowledgeEntry(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid knowledge entry ID"})
	}

	var entry models.KnowledgeEntry
	if err := c.BodyParser(&entry); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	entry.ID = id
	// TODO: Get user ID from JWT token
	updatedBy := uuid.New() // Placeholder
	entry.UpdatedBy = &updatedBy

	if err := s.knowledgeService.UpdateKnowledgeEntry(c.Context(), &entry); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update knowledge entry"})
	}

	return c.JSON(entry)
}

// @Summary Delete knowledge entry
// @Description Delete a knowledge entry
// @Tags knowledge
// @Accept json
// @Produce json
// @Param id path string true "Knowledge entry ID"
// @Success 204
// @Router /knowledge/{id} [delete]
func (s *Server) deleteKnowledgeEntry(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid knowledge entry ID"})
	}

	if err := s.knowledgeService.DeleteKnowledgeEntry(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete knowledge entry"})
	}

	return c.SendStatus(204)
}
