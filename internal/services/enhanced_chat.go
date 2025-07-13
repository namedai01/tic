package services

import (
	"context"
	"encoding/json"
	"log"
	"tic-knowledge-system/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EnhancedChatService struct {
	db               *gorm.DB
	unifiedAIService *UnifiedAIService
	knowledgeService *KnowledgeService
}

func NewEnhancedChatService(db *gorm.DB, unifiedAIService *UnifiedAIService, knowledgeService *KnowledgeService) *EnhancedChatService {
	return &EnhancedChatService{
		db:               db,
		unifiedAIService: unifiedAIService,
		knowledgeService: knowledgeService,
	}
}

type EnhancedChatRequest struct {
	Message           string     `json:"message" validate:"required"`
	SessionID         *uuid.UUID `json:"session_id,omitempty"`
	UserID            uuid.UUID  `json:"user_id" validate:"required"`
	PreferredProvider AIProvider `json:"preferred_provider,omitempty"`
	SystemPrompt      string     `json:"system_prompt,omitempty"`
}

type EnhancedChatResponse struct {
	Response      string     `json:"response"`
	SessionID     uuid.UUID  `json:"session_id"`
	Sources       []string   `json:"sources,omitempty"`
	Provider      AIProvider `json:"provider"`
	Model         string     `json:"model"`
	CreatedAt     string     `json:"created_at"`
}

func (s *EnhancedChatService) ProcessChat(ctx context.Context, req EnhancedChatRequest) (*EnhancedChatResponse, error) {
	log.Printf("[INFO] ProcessChat started for user_id: %s, message: %.50s...", req.UserID, req.Message)

	// Get or create session
	session, err := s.getOrCreateSession(req.UserID, req.SessionID)
	if err != nil {
		log.Printf("[ERROR] Failed to get or create session for user %s: %v", req.UserID, err)
		return nil, err
	}
	log.Printf("[INFO] Using session_id: %s for user_id: %s", session.ID, req.UserID)

	// Save user message to database
	userMessage := &models.ChatMessage{
		SessionID: session.ID,
		Role:      "user",
		Content:   req.Message,
		Metadata:  "{}",
	}

	if err := s.db.Create(userMessage).Error; err != nil {
		log.Printf("[ERROR] Failed to save user message to database: %v", err)
		return nil, err
	}
	log.Printf("[INFO] User message saved with ID: %s", userMessage.ID)

	// Search knowledge base for relevant information
	log.Printf("[INFO] Searching knowledge base for query: %.50s...", req.Message)
	knowledgeEntries, err := s.knowledgeService.SearchKnowledgeEntries(context.Background(), req.Message, 3)
	if err != nil {
		log.Printf("[WARNING] Knowledge search failed, continuing without context: %v", err)
	}

	log.Printf("[INFO] Found %d knowledge entries for context", len(knowledgeEntries))

	// Build context from knowledge entries
	var context []string
	if len(knowledgeEntries) > 0 {
		for _, entry := range knowledgeEntries {
			contextEntry := entry.Title + ": " + entry.Content
			context = append(context, contextEntry)
			log.Printf("[DEBUG] Added knowledge entry to context: %s", entry.Title)
		}
		log.Printf("[INFO] Built context from %d knowledge entries", len(context))
	} else {
		log.Printf("[INFO] No knowledge context available, using general AI knowledge")
	}

	// Get conversation history
	log.Printf("[INFO] Retrieving conversation history for session: %s", session.ID)
	recentMessages, err := s.getRecentMessages(session.ID, 10)
	if err != nil {
		log.Printf("[WARNING] Failed to get recent messages: %v", err)
		recentMessages = []models.ChatMessage{}
	}
	log.Printf("[INFO] Retrieved %d recent messages for context", len(recentMessages))

	// Build messages for AI
	var messages []UnifiedChatMessage

	// Add conversation history (excluding the current message)
	for _, msg := range recentMessages {
		if msg.ID != userMessage.ID {
			role := string(msg.Role)
			if role == "assistant" {
				role = "model" // Gemini uses "model" instead of "assistant"
			}
			messages = append(messages, UnifiedChatMessage{
				Role:    role,
				Content: msg.Content,
			})
			log.Printf("[DEBUG] Added historical message to AI context: role=%s, content=%.30s...", msg.Role, msg.Content)
		}
	}

	// Add current user message
	messages = append(messages, UnifiedChatMessage{
		Role:    "user",
		Content: req.Message,
	})

	log.Printf("[INFO] Prepared %d messages for AI API call", len(messages))

	// Create AI request
	aiRequest := UnifiedChatRequest{
		Messages:         messages,
		Context:          context,
		SessionID:        session.ID.String(),
		UseKnowledgeBase: len(context) > 0,
		SystemPrompt:     req.SystemPrompt,
		PreferredProvider: req.PreferredProvider,
	}

	log.Printf("[INFO] Calling AI service with %d messages, knowledge_base=%t", len(messages), len(context) > 0)
	if req.PreferredProvider != "" {
		log.Printf("[INFO] Using preferred provider: %s", req.PreferredProvider)
	}

	// Call AI service
	aiResponse, err := s.unifiedAIService.ChatCompletion(ctx, aiRequest)
	if err != nil {
		log.Printf("[ERROR] AI API call failed: %v", err)
		return nil, err
	}
	log.Printf("[INFO] AI API call successful, provider: %s, response length: %d characters", aiResponse.Provider, len(aiResponse.Message))

	// Save assistant response to database
	assistantMessage := &models.ChatMessage{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   aiResponse.Message,
		Metadata:  s.buildMetadata(aiResponse.Provider, aiResponse.Model, aiResponse.Sources),
	}

	if err := s.db.Create(assistantMessage).Error; err != nil {
		log.Printf("[ERROR] Failed to save assistant message to database: %v", err)
		return nil, err
	}
	log.Printf("[INFO] Assistant message saved with ID: %s", assistantMessage.ID)

	// Prepare sources
	var sources []string
	for _, entry := range knowledgeEntries {
		sources = append(sources, entry.ID.String())
	}

	response := &EnhancedChatResponse{
		Response:  aiResponse.Message,
		SessionID: session.ID,
		Sources:   sources,
		Provider:  aiResponse.Provider,
		Model:     aiResponse.Model,
		CreatedAt: assistantMessage.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	log.Printf("[INFO] ProcessChat completed successfully for session: %s, provider: %s, sources: %d", session.ID, aiResponse.Provider, len(sources))
	return response, nil
}

func (s *EnhancedChatService) getOrCreateSession(userID uuid.UUID, sessionID *uuid.UUID) (*models.ChatSession, error) {
	var session models.ChatSession

	if sessionID != nil {
		// Try to find existing session
		if err := s.db.Where("id = ? AND user_id = ? AND is_active = ?", sessionID, userID, true).First(&session).Error; err == nil {
			return &session, nil
		}
	}

	// Create new session
	log.Printf("[INFO] Creating new chat session for user %s", userID)
	session = models.ChatSession{
		UserID:   userID,
		Title:    "New Chat",
		IsActive: true,
	}

	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}

	log.Printf("[INFO] Created new session %s for user %s", session.ID, userID)
	return &session, nil
}

func (s *EnhancedChatService) getRecentMessages(sessionID uuid.UUID, limit int) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	err := s.db.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

func (s *EnhancedChatService) buildMetadata(provider AIProvider, model string, sources []string) string {
	metadata := map[string]interface{}{
		"provider": string(provider),
		"model":    model,
		"sources":  sources,
	}

	metadataJSON, _ := json.Marshal(metadata)
	return string(metadataJSON)
}

func (s *EnhancedChatService) GetChatSessions(userID uuid.UUID) ([]models.ChatSession, error) {
	log.Printf("[INFO] Getting chat sessions for user: %s", userID)

	var sessions []models.ChatSession
	err := s.db.Where("user_id = ? AND is_active = ?", userID, true).
		Order("updated_at DESC").
		Find(&sessions).Error

	if err != nil {
		log.Printf("[ERROR] Failed to get chat sessions: %v", err)
		return nil, err
	}

	log.Printf("[INFO] Retrieved %d chat sessions for user: %s", len(sessions), userID)
	return sessions, nil
}

func (s *EnhancedChatService) GetChatSession(userID, sessionID uuid.UUID) (*models.ChatSession, error) {
	log.Printf("[INFO] Getting chat session %s for user: %s", sessionID, userID)

	var session models.ChatSession
	err := s.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error

	if err != nil {
		log.Printf("[ERROR] Failed to get chat session: %v", err)
		return nil, err
	}

	log.Printf("[INFO] Retrieved chat session: %s", sessionID)
	return &session, nil
}

func (s *EnhancedChatService) DeleteChatSession(userID, sessionID uuid.UUID) error {
	log.Printf("[INFO] Deleting chat session %s for user: %s", sessionID, userID)

	result := s.db.Where("id = ? AND user_id = ?", sessionID, userID).
		Update("is_active", false)

	if result.Error != nil {
		log.Printf("[ERROR] Failed to delete chat session: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Printf("[WARNING] No chat session found to delete: %s", sessionID)
		return gorm.ErrRecordNotFound
	}

	log.Printf("[INFO] Successfully deleted chat session: %s", sessionID)
	return nil
}

// GetAvailableProviders returns the list of available AI providers
func (s *EnhancedChatService) GetAvailableProviders() []AIProvider {
	return s.unifiedAIService.GetAvailableProviders()
}

// SetPrimaryProvider changes the primary AI provider
func (s *EnhancedChatService) SetPrimaryProvider(provider AIProvider) error {
	return s.unifiedAIService.SetPrimaryProvider(provider)
}

// GetPrimaryProvider returns the current primary provider
func (s *EnhancedChatService) GetPrimaryProvider() AIProvider {
	return s.unifiedAIService.GetPrimaryProvider()
}
