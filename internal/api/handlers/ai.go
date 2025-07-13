package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"tic-knowledge-system/internal/models"
	"tic-knowledge-system/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type AIHandler struct {
	enhancedChatService *services.EnhancedChatService
}

func NewAIHandler(enhancedChatService *services.EnhancedChatService) *AIHandler {
	return &AIHandler{
		enhancedChatService: enhancedChatService,
	}
}

// ProcessChatWithAI handles chat requests with AI provider selection
// @Summary Process chat message with AI provider selection
// @Description Send a message to the AI chatbot with the ability to choose provider
// @Tags ai-chat
// @Accept json
// @Produce json
// @Param request body services.EnhancedChatRequest true "Chat request"
// @Success 200 {object} services.EnhancedChatResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ai/chat [post]
func (h *AIHandler) ProcessChatWithAI(c *fiber.Ctx) error {
	log.Printf("[INFO] Received enhanced chat request")

	var req services.EnhancedChatRequest
	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Failed to parse chat request: %v", err)
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	// Validate required fields
	if req.Message == "" {
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Missing required field",
			Message: "message is required",
		})
	}

	if req.UserID == uuid.Nil {
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Missing required field",
			Message: "user_id is required",
		})
	}

	log.Printf("[INFO] Processing enhanced chat for user: %s, provider: %s", req.UserID, req.PreferredProvider)

	// --- TRACKING LOGIC START ---
	db := c.Locals("db").(*gorm.DB)
	t := time.Now()
	hour := t.Hour()
	topicID := 0
	timeRange := ""
	switch {
	case hour >= 6 && hour < 12:
		topicID = 1
		timeRange = "Morning (6AM - 12PM)"
	case hour >= 12 && hour < 18:
		topicID = 2
		timeRange = "Afternoon (12PM - 6PM)"
	case hour >= 18 && hour < 24:
		topicID = 3
		timeRange = "Evening (6PM - 12AM)"
	default:
		topicID = 4
		timeRange = "Night (12AM - 6AM)"
	}
	// Increment TopicQuestionStat
	var topicStat models.TopicQuestionStat
	if err := db.Where("topic_id = ?", topicID).First(&topicStat).Error; err == nil {
		topicStat.Count++
		db.Save(&topicStat)
	} else {
		topicStat = models.TopicQuestionStat{TopicID: uint(topicID), Count: 1}
		db.Create(&topicStat)
	}
	// Increment TimeDistributionStat
	var timeStat models.TimeDistributionStat
	if err := db.Where("time_range = ?", timeRange).First(&timeStat).Error; err == nil {
		timeStat.Count++
		db.Save(&timeStat)
	} else {
		timeStat = models.TimeDistributionStat{TimeRange: timeRange, Count: 1}
		db.Create(&timeStat)
	}
	// --- TRACKING LOGIC END ---

	start := time.Now()
	// Process the chat request
	response, err := h.enhancedChatService.ProcessChat(c.Context(), req)
	if err != nil {
		log.Printf("[ERROR] Chat processing failed: %v", err)
		return c.Status(500).JSON(ErrorResponse{
			Error:   "Chat processing failed",
			Message: err.Error(),
		})
	}

	log.Printf("[INFO] Chat processed successfully using provider: %s", response.Provider)

	resp := fiber.Map{
		"success": true,
		"data":    response,
	}
	responseJSON, _ := json.Marshal(resp)
	responseTime := time.Since(start).Milliseconds()
	if db != nil {
		db.Create(&models.TrackedChatLog{
			APIName:       "ai/chat",
			RequestMsg:    req.Message,
			ResponseValue: string(responseJSON),
			ResponseTime:  responseTime,
		})
	}
	return c.Status(200).JSON(resp)
}

// GetAvailableProviders returns the list of available AI providers
// @Summary Get available AI providers
// @Description Get the list of AI providers that are currently available
// @Tags ai-providers
// @Produce json
// @Success 200 {object} object{providers=[]string}
// @Router /ai/providers [get]
func (h *AIHandler) GetAvailableProviders(c *fiber.Ctx) error {
	log.Printf("[INFO] Getting available AI providers")

	providers := h.enhancedChatService.GetAvailableProviders()

	providerStrings := make([]string, len(providers))
	for i, provider := range providers {
		providerStrings[i] = string(provider)
	}

	log.Printf("[INFO] Available providers: %v", providerStrings)

	return c.Status(200).JSON(fiber.Map{
		"success":   true,
		"providers": providerStrings,
		"primary":   string(h.enhancedChatService.GetPrimaryProvider()),
	})
}

// SetPrimaryProvider sets the primary AI provider
// @Summary Set primary AI provider
// @Description Change the primary AI provider for chat requests
// @Tags ai-providers
// @Accept json
// @Produce json
// @Param request body object{provider=string} true "Provider selection"
// @Success 200 {object} object{success=bool,message=string}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ai/providers/primary [post]
func (h *AIHandler) SetPrimaryProvider(c *fiber.Ctx) error {
	log.Printf("[INFO] Setting primary AI provider")

	var req struct {
		Provider string `json:"provider" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Failed to parse provider request: %v", err)
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	if req.Provider == "" {
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Missing required field",
			Message: "provider is required",
		})
	}

	provider := services.AIProvider(req.Provider)

	if err := h.enhancedChatService.SetPrimaryProvider(provider); err != nil {
		log.Printf("[ERROR] Failed to set primary provider: %v", err)
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Invalid provider",
			Message: err.Error(),
		})
	}

	log.Printf("[INFO] Primary provider set to: %s", provider)

	return c.Status(200).JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Primary provider set to %s", provider),
	})
}

// CompareProviders tests the same message with different AI providers
// @Summary Compare AI provider responses
// @Description Send the same message to multiple AI providers for comparison
// @Tags ai-chat
// @Accept json
// @Produce json
// @Param request body object{message=string,user_id=string,providers=[]string} true "Comparison request"
// @Success 200 {object} object{responses=map[string]services.EnhancedChatResponse}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ai/compare [post]
func (h *AIHandler) CompareProviders(c *fiber.Ctx) error {
	log.Printf("[INFO] Received provider comparison request")

	var req struct {
		Message   string   `json:"message" validate:"required"`
		UserID    string   `json:"user_id" validate:"required"`
		Providers []string `json:"providers"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Printf("[ERROR] Failed to parse comparison request: %v", err)
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	if req.Message == "" || req.UserID == "" {
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Missing required fields",
			Message: "message and user_id are required",
		})
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Error:   "Invalid user_id",
			Message: "user_id must be a valid UUID",
		})
	}

	// If no providers specified, use all available
	if len(req.Providers) == 0 {
		availableProviders := h.enhancedChatService.GetAvailableProviders()
		for _, provider := range availableProviders {
			req.Providers = append(req.Providers, string(provider))
		}
	}

	log.Printf("[INFO] Comparing responses from %d providers", len(req.Providers))

	responses := make(map[string]*services.EnhancedChatResponse)
	errors := make(map[string]string)

	// Test each provider
	for _, providerStr := range req.Providers {
		provider := services.AIProvider(providerStr)

		chatReq := services.EnhancedChatRequest{
			Message:           req.Message,
			UserID:            userID,
			PreferredProvider: provider,
		}

		response, err := h.enhancedChatService.ProcessChat(c.Context(), chatReq)
		if err != nil {
			log.Printf("[ERROR] Provider %s failed: %v", provider, err)
			errors[providerStr] = err.Error()
		} else {
			responses[providerStr] = response
			log.Printf("[INFO] Provider %s responded successfully", provider)
		}
	}

	return c.Status(200).JSON(fiber.Map{
		"success":   true,
		"responses": responses,
		"errors":    errors,
		"message":   fmt.Sprintf("Compared %d providers", len(req.Providers)),
	})
}
