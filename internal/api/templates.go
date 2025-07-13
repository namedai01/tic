package api

import (
	"strconv"
	"tic-knowledge-system/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// @Summary Get templates
// @Description Get all templates with optional filtering
// @Tags templates
// @Accept json
// @Produce json
// @Param category query string false "Filter by category"
// @Param active query boolean false "Filter by active status"
// @Success 200 {array} models.Template
// @Router /templates [get]
func (s *Server) getTemplates(c *fiber.Ctx) error {
	category := c.Query("category")
	activeStr := c.Query("active")

	var isActive *bool
	if activeStr != "" {
		active, err := strconv.ParseBool(activeStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid active parameter"})
		}
		isActive = &active
	}

	templates, err := s.knowledgeService.GetTemplates(category, isActive)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch templates"})
	}

	return c.JSON(templates)
}

// @Summary Create template
// @Description Create a new template
// @Tags templates
// @Accept json
// @Produce json
// @Param template body models.Template true "Template data"
// @Success 201 {object} models.Template
// @Router /templates [post]
func (s *Server) createTemplate(c *fiber.Ctx) error {
	var template models.Template
	if err := c.BodyParser(&template); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// TODO: Get user ID from JWT token
	template.CreatedBy = uuid.New() // Placeholder

	if err := s.knowledgeService.CreateTemplate(&template); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create template"})
	}

	return c.Status(201).JSON(template)
}

// @Summary Get template
// @Description Get a template by ID
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 200 {object} models.Template
// @Router /templates/{id} [get]
func (s *Server) getTemplate(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid template ID"})
	}

	template, err := s.knowledgeService.GetTemplateByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Template not found"})
	}

	return c.JSON(template)
}

// @Summary Update template
// @Description Update an existing template
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Param template body models.Template true "Template data"
// @Success 200 {object} models.Template
// @Router /templates/{id} [put]
func (s *Server) updateTemplate(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid template ID"})
	}

	var template models.Template
	if err := c.BodyParser(&template); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	template.ID = id
	if err := s.knowledgeService.UpdateTemplate(&template); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update template"})
	}

	return c.JSON(template)
}

// @Summary Delete template
// @Description Delete a template
// @Tags templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID"
// @Success 204
// @Router /templates/{id} [delete]
func (s *Server) deleteTemplate(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid template ID"})
	}

	if err := s.knowledgeService.DeleteTemplate(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete template"})
	}

	return c.SendStatus(204)
}
