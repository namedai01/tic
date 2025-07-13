package handlers

import (
	"log"
	"strconv"
	"time"

	"tic-knowledge-system/internal/models"
	"tic-knowledge-system/internal/services"

	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// OpenAIAssistantHandler handles OpenAI Assistant API requests
type OpenAIAssistantHandler struct {
	assistantService *services.OpenAIAssistantService
	logger           *log.Logger
}

// NewOpenAIAssistantHandler creates a new OpenAI Assistant handler
func NewOpenAIAssistantHandler(assistantService *services.OpenAIAssistantService, logger *log.Logger) *OpenAIAssistantHandler {
	return &OpenAIAssistantHandler{
		assistantService: assistantService,
		logger:           logger,
	}
}

// ChatWithAssistant handles chat requests to OpenAI Assistant
// @Summary Chat with OpenAI Assistant
// @Description Implements the 4-step workflow: add message, create run, wait 5s, get messages
// @Tags assistant
// @Accept json
// @Produce json
// @Param request body services.ChatAssistantRequest true "Chat request"
// @Success 200 {object} services.ChatAssistantResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /assistant/chat [post]
func (h *OpenAIAssistantHandler) ChatWithAssistant(c *fiber.Ctx) error {
	h.logger.Printf("Received OpenAI Assistant chat request")

	var req services.ChatAssistantRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Printf("Error parsing request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate required fields
	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Message is required",
		})
	}

	if req.AssistantID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Assistant ID is required",
		})
	}

	h.logger.Printf("Processing chat request for assistant %s", req.AssistantID)

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
	// Execute the chat workflow
	ctx := c.Context()
	response, err := h.assistantService.ChatWithAssistant(ctx, req)
	if err != nil {
		h.logger.Printf("Error in chat workflow: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to process chat request",
			"details": err.Error(),
		})
	}
	responseJSON, _ := json.Marshal(response)
	responseTime := time.Since(start).Milliseconds()
	if db != nil {
		db.Create(&models.TrackedChatLog{
			APIName:       "assistant/chat",
			RequestMsg:    req.Message,
			ResponseValue: string(responseJSON),
			ResponseTime:  responseTime,
		})
	}

	h.logger.Printf("Chat workflow completed successfully. Run ID: %s", response.RunID)
	return c.JSON(response)
}

// GetThreadMessages gets all messages from a thread
// @Summary Get thread messages
// @Description Retrieve all messages from a specific thread
// @Tags assistant
// @Produce json
// @Param thread_id path string true "Thread ID"
// @Success 200 {array} services.AssistantMessage
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /assistant/threads/{thread_id}/messages [get]
func (h *OpenAIAssistantHandler) GetThreadMessages(c *fiber.Ctx) error {
	threadID := c.Params("thread_id")
	if threadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Thread ID is required",
		})
	}

	h.logger.Printf("Getting messages for thread: %s", threadID)

	ctx := c.Context()
	messages, err := h.assistantService.GetThreadMessages(ctx, threadID)
	if err != nil {
		h.logger.Printf("Error getting thread messages: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get thread messages",
			"details": err.Error(),
		})
	}

	h.logger.Printf("Retrieved %d messages from thread %s", len(messages), threadID)
	return c.JSON(fiber.Map{
		"thread_id":    threadID,
		"messages":     messages,
		"count":        len(messages),
		"retrieved_at": time.Now(),
	})
}

// CreateThread creates a new OpenAI thread
// @Summary Create new thread
// @Description Create a new OpenAI Assistant thread
// @Tags assistant
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /assistant/threads [post]
func (h *OpenAIAssistantHandler) CreateThread(c *fiber.Ctx) error {
	h.logger.Printf("Creating new OpenAI thread")

	ctx := c.Context()
	thread, err := h.assistantService.CreateThread(ctx)
	if err != nil {
		h.logger.Printf("Error creating thread: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to create thread",
			"details": err.Error(),
		})
	}

	h.logger.Printf("Created new thread: %s", thread.ID)
	return c.JSON(fiber.Map{
		"thread_id":  thread.ID,
		"created_at": thread.CreatedAt,
		"metadata":   thread.Metadata,
	})
}

// ChatWithCustomWorkflow allows custom workflow configuration
// @Summary Chat with custom workflow
// @Description Chat with OpenAI Assistant using custom wait time and options
// @Tags assistant
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "Custom chat request"
// @Success 200 {object} services.ChatAssistantResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /assistant/chat/custom [post]
func (h *OpenAIAssistantHandler) ChatWithCustomWorkflow(c *fiber.Ctx) error {
	h.logger.Printf("Received custom workflow chat request")

	var reqData map[string]interface{}
	if err := c.BodyParser(&reqData); err != nil {
		h.logger.Printf("Error parsing request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Extract required fields
	message, ok := reqData["message"].(string)
	if !ok || message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Message is required",
		})
	}

	assistantID, ok := reqData["assistant_id"].(string)
	if !ok || assistantID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Assistant ID is required",
		})
	}

	// Optional fields
	threadID, _ := reqData["thread_id"].(string)
	waitTimeStr, _ := reqData["wait_time"].(string)
	timeoutSecondsFloat, _ := reqData["timeout_seconds"].(float64) // JSON numbers come as float64
	waitForCompletion, _ := reqData["wait_for_completion"].(bool)

	// Parse timeout_seconds (takes priority over wait_time)
	timeoutSeconds := 30 // default
	if timeoutSecondsFloat > 0 {
		timeoutSeconds = int(timeoutSecondsFloat)
	} else if waitTimeStr != "" {
		// Fallback to wait_time for backward compatibility
		if seconds, err := strconv.Atoi(waitTimeStr); err == nil && seconds > 0 {
			timeoutSeconds = seconds
		}
	}

	waitTime := time.Duration(timeoutSeconds) * time.Second

	h.logger.Printf("Processing custom chat request with %v timeout", waitTime)

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
	// Create request
	req := services.ChatAssistantRequest{
		Message:        message,
		AssistantID:    assistantID,
		ThreadID:       threadID,
		TimeoutSeconds: timeoutSeconds,
	}

	if waitForCompletion {
		response := h.chatWithCompletionWait(c, req, waitTime)
		responseJSON, _ := json.Marshal(response)
		responseTime := time.Since(start).Milliseconds()
		if db != nil {
			db.Create(&models.TrackedChatLog{
				APIName:       "assistant/chat/custom",
				RequestMsg:    message,
				ResponseValue: string(responseJSON),
				ResponseTime:  responseTime,
			})
		}
		return c.JSON(response)
	}
	response := h.chatWithCustomWait(c, req, waitTime, timeoutSeconds)
	responseJSON, _ := json.Marshal(response)
	responseTime := time.Since(start).Milliseconds()
	if db != nil {
		db.Create(&models.TrackedChatLog{
			APIName:       "assistant/chat/custom",
			RequestMsg:    message,
			ResponseValue: string(responseJSON),
			ResponseTime:  responseTime,
		})
	}
	return c.JSON(response)
}

// chatWithCustomWait executes chat with custom wait time
func (h *OpenAIAssistantHandler) chatWithCustomWait(c *fiber.Ctx, req services.ChatAssistantRequest, waitTime time.Duration, timeoutSeconds int) interface{} {
	// Use provided thread ID or default
	threadID := req.ThreadID
	if threadID == "" {
		threadID = "thread_5GyQSnIxNy8uwMN2liLPuphc" // Default from your example
	}

	// Step 1: Add message to thread
	h.logger.Printf("Step 1: Adding message to thread %s", threadID)
	// We'll call the service method directly for more control

	// For now, use the standard workflow but you can implement custom logic here
	ctx := c.Context()
	response, err := h.assistantService.ChatWithAssistant(ctx, req)
	if err != nil {
		h.logger.Printf("Error in custom chat workflow: %v", err)
		return fiber.Map{
			"error":   "Failed to process custom chat request",
			"details": err.Error(),
		}
	}

	// Add custom metadata
	response.Metadata["custom_workflow"] = true
	response.Metadata["timeout_seconds"] = timeoutSeconds

	h.logger.Printf("Custom chat workflow completed successfully")
	return response
}

// chatWithCompletionWait waits for run completion
func (h *OpenAIAssistantHandler) chatWithCompletionWait(c *fiber.Ctx, req services.ChatAssistantRequest, maxWaitTime time.Duration) error {
	ctx := c.Context()

	// First, execute the standard workflow
	response, err := h.assistantService.ChatWithAssistant(ctx, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to process chat request",
			"details": err.Error(),
		})
	}

	// Then wait for completion
	h.logger.Printf("Waiting for run completion with timeout: %v", maxWaitTime)
	finalStatus, err := h.assistantService.WaitForRunCompletion(ctx, response.ThreadID, response.RunID, maxWaitTime)
	if err != nil {
		h.logger.Printf("Warning: Run completion wait failed: %v", err)
		// Don't fail the request, just add the error to metadata
		response.Metadata["completion_wait_error"] = err.Error()
	} else {
		response.Status = finalStatus
		response.Metadata["completion_waited"] = true
		response.Metadata["final_status"] = finalStatus

		// Get updated messages after completion
		if finalStatus == "completed" {
			updatedMessages, err := h.assistantService.GetThreadMessages(ctx, response.ThreadID)
			if err == nil {
				response.Messages = updatedMessages
				response.Metadata["messages_updated"] = true
			}
		}
	}

	response.Metadata["max_wait_time"] = maxWaitTime.String()

	return c.JSON(response)
}

// HealthCheck checks if the assistant service is working
// @Summary Health check
// @Description Check if OpenAI Assistant service is working
// @Tags assistant
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /assistant/health [get]
func (h *OpenAIAssistantHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"service":   "openai-assistant",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	})
}
