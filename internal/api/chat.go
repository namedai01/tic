package api

import (
	"strconv"
	"tic-knowledge-system/internal/models"
	"tic-knowledge-system/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// @Summary Process chat
// @Description Send a message to the chatbot and get a response
// @Tags chat
// @Accept json
// @Produce json
// @Param request body services.ChatRequest true "Chat request"
// @Success 200 {object} services.ChatResponse
// @Router /chat [post]
func (s *Server) processChat(c *fiber.Ctx) error {
	var req services.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// TODO: Get user ID from JWT token  
	// Using existing user from database for demo purposes
	req.UserID = uuid.MustParse("4566215d-9957-4765-9ac5-a9395879945e")

	response, err := s.chatService.ProcessChat(c.Context(), req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to process chat"})
	}

	return c.JSON(response)
}

// @Summary Get chat sessions
// @Description Get all chat sessions for the current user
// @Tags chat
// @Accept json
// @Produce json
// @Success 200 {array} models.ChatSession
// @Router /chat/sessions [get]
func (s *Server) getChatSessions(c *fiber.Ctx) error {
	// TODO: Get user ID from JWT token
	// Using existing user from database for demo purposes
	userID := uuid.MustParse("4566215d-9957-4765-9ac5-a9395879945e")

	sessions, err := s.chatService.GetChatSessions(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch chat sessions"})
	}

	return c.JSON(sessions)
}

// @Summary Get chat session
// @Description Get a specific chat session with messages
// @Tags chat
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} models.ChatSession
// @Router /chat/sessions/{id} [get]
func (s *Server) getChatSession(c *fiber.Ctx) error {
	idStr := c.Params("id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session ID"})
	}

	// TODO: Get user ID from JWT token
	userID := uuid.New() // Placeholder

	session, err := s.chatService.GetChatSession(sessionID, userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Chat session not found"})
	}

	return c.JSON(session)
}

// @Summary Delete chat session
// @Description Delete a chat session
// @Tags chat
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Success 204
// @Router /chat/sessions/{id} [delete]
func (s *Server) deleteChatSession(c *fiber.Ctx) error {
	idStr := c.Params("id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session ID"})
	}

	// TODO: Get user ID from JWT token
	userID := uuid.New() // Placeholder

	if err := s.chatService.DeleteChatSession(sessionID, userID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to delete chat session"})
	}

	return c.SendStatus(204)
}

// @Summary Submit feedback
// @Description Submit feedback for a chat message
// @Tags feedback
// @Accept json
// @Produce json
// @Param feedback body models.Feedback true "Feedback data"
// @Success 201 {object} models.Feedback
// @Router /feedback [post]
func (s *Server) submitFeedback(c *fiber.Ctx) error {
	var feedback models.Feedback
	if err := c.BodyParser(&feedback); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// TODO: Get user ID from JWT token
	feedback.UserID = uuid.New() // Placeholder

	if err := s.chatService.SubmitFeedback(&feedback); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to submit feedback"})
	}

	return c.Status(201).JSON(feedback)
}

// @Summary Get feedback
// @Description Get feedback with optional filtering
// @Tags feedback
// @Accept json
// @Produce json
// @Param message_id query string false "Filter by message ID"
// @Param user_id query string false "Filter by user ID"
// @Param limit query int false "Limit number of results" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} models.Feedback
// @Router /feedback [get]
func (s *Server) getFeedback(c *fiber.Ctx) error {
	messageIDStr := c.Query("message_id")
	userIDStr := c.Query("user_id")
	limitStr := c.Query("limit", "20")
	offsetStr := c.Query("offset", "0")

	var messageID *uuid.UUID
	var userID *uuid.UUID

	if messageIDStr != "" {
		id, err := uuid.Parse(messageIDStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid message_id parameter"})
		}
		messageID = &id
	}

	if userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid user_id parameter"})
		}
		userID = &id
	}

	limit := 20
	offset := 0

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	feedback, err := s.chatService.GetFeedback(messageID, userID, limit, offset)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch feedback"})
	}

	return c.JSON(feedback)
}

// @Summary Get current user
// @Description Get current user information
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} models.User
// @Router /users/me [get]
func (s *Server) getCurrentUser(c *fiber.Ctx) error {
	// TODO: Implement JWT authentication and get real user
	// For now, return a placeholder user
	user := models.User{
		ID:    uuid.New(),
		Email: "user@example.com",
		Name:  "Test User",
		Role:  models.RegularUser,
		IsActive: true,
	}

	return c.JSON(user)
}
