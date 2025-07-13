package services

import (
	"context"
	"encoding/json"
	"log"
	"tic-knowledge-system/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatService struct {
	db            *gorm.DB
	openAIService *OpenAIService
	knowledgeService *KnowledgeService
}

func NewChatService(db *gorm.DB, openAIService *OpenAIService, knowledgeService *KnowledgeService) *ChatService {
	return &ChatService{
		db:            db,
		openAIService: openAIService,
		knowledgeService: knowledgeService,
	}
}

type ChatRequest struct {
	Message   string    `json:"message" validate:"required"`
	SessionID *uuid.UUID `json:"session_id,omitempty"`
	UserID    uuid.UUID `json:"user_id" validate:"required"`
}

type ChatResponse struct {
	Message   string    `json:"message"`
	SessionID uuid.UUID `json:"session_id"`
	Sources   []string  `json:"sources,omitempty"`
}

func (s *ChatService) ProcessChat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	log.Printf("[INFO] ProcessChat started for user_id: %s, message: %.50s...", req.UserID, req.Message)
	
	// Get or create session
	session, err := s.getOrCreateSession(req.UserID, req.SessionID)
	if err != nil {
		log.Printf("[ERROR] Failed to get or create session for user %s: %v", req.UserID, err)
		return nil, err
	}
	log.Printf("[INFO] Using session_id: %s for user_id: %s", session.ID, req.UserID)

	// Save user message
	userMessage := &models.ChatMessage{
		SessionID: session.ID,
		Role:      models.UserMessage,
		Content:   req.Message,
		Metadata:  "{}",
	}
	if err := s.db.Create(userMessage).Error; err != nil {
		log.Printf("[ERROR] Failed to save user message to database: %v", err)
		return nil, err
	}
	log.Printf("[INFO] User message saved with ID: %s", userMessage.ID)

	// Search for relevant knowledge
	log.Printf("[INFO] Searching knowledge base for query: %.50s...", req.Message)
	knowledgeEntries, err := s.knowledgeService.SearchKnowledgeEntries(ctx, req.Message, 5)
	if err != nil {
		log.Printf("[WARNING] Knowledge search failed, continuing without context: %v", err)
		// Log error but continue without knowledge context
		knowledgeEntries = []models.KnowledgeEntry{}
	}
	log.Printf("[INFO] Found %d knowledge entries for context", len(knowledgeEntries))

	// Build context from knowledge entries
	var context []string
	var sources []string
	for _, entry := range knowledgeEntries {
		context = append(context, entry.Title+"\n"+entry.Content)
		sources = append(sources, entry.Title)
		log.Printf("[DEBUG] Added knowledge entry to context: %s", entry.Title)
	}
	
	if len(context) > 0 {
		log.Printf("[INFO] Built context from %d knowledge entries", len(context))
	} else {
		log.Printf("[INFO] No knowledge context available, using general AI knowledge")
	}

	// Get recent conversation history
	log.Printf("[INFO] Retrieving conversation history for session: %s", session.ID)
	var recentMessages []models.ChatMessage
	s.db.Where("session_id = ?", session.ID).
		Order("created_at DESC").
		Limit(10).
		Find(&recentMessages)
	log.Printf("[INFO] Retrieved %d recent messages for context", len(recentMessages))

	// Add the current user message
	var openAIMessages []OpenAIChatMessage
	for i := len(recentMessages) - 1; i >= 0; i-- {
		msg := recentMessages[i]
		if msg.ID != userMessage.ID { // Don't include the message we just created
			openAIMessages = append(openAIMessages, OpenAIChatMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
			log.Printf("[DEBUG] Added historical message to OpenAI context: role=%s, content=%.30s...", msg.Role, msg.Content)
		}
	}

	// Add the current user message
	openAIMessages = append(openAIMessages, OpenAIChatMessage{
		Role:    string(models.UserMessage),
		Content: req.Message,
	})
	log.Printf("[INFO] Prepared %d messages for OpenAI API call", len(openAIMessages))

	// Call OpenAI
	openAIReq := OpenAIChatRequest{
		Messages:        openAIMessages,
		Context:         context,
		SessionID:       session.ID.String(),
		UseKnowledgeBase: len(context) > 0,
	}
	
	log.Printf("[INFO] Calling OpenAI API with %d messages, knowledge_base=%t", len(openAIMessages), len(context) > 0)
	response, err := s.openAIService.ChatCompletion(ctx, openAIReq)
	if err != nil {
		log.Printf("[ERROR] OpenAI API call failed: %v", err)
		return nil, err
	}
	log.Printf("[INFO] OpenAI API call successful, response length: %d characters", len(response.Message))

	// Save assistant message
	assistantMessage := &models.ChatMessage{
		SessionID: session.ID,
		Role:      models.AssistantMessage,
		Content:   response.Message,
		Metadata:  "{}", // Could store sources here as JSON
	}
	if err := s.db.Create(assistantMessage).Error; err != nil {
		log.Printf("[ERROR] Failed to save assistant message to database: %v", err)
		return nil, err
	}
	log.Printf("[INFO] Assistant message saved with ID: %s", assistantMessage.ID)

	chatResponse := &ChatResponse{
		Message:   response.Message,
		SessionID: session.ID,
		Sources:   sources,
	}
	
	log.Printf("[INFO] ProcessChat completed successfully for session: %s, sources: %d", session.ID, len(sources))
	return chatResponse, nil
}

func (s *ChatService) GetChatSessions(userID uuid.UUID) ([]models.ChatSession, error) {
	log.Printf("[INFO] Getting chat sessions for user: %s", userID)
	var sessions []models.ChatSession
	err := s.db.Where("user_id = ? AND is_active = true", userID).
		Order("updated_at DESC").
		Find(&sessions).Error
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve chat sessions for user %s: %v", userID, err)
		return nil, err
	}
	log.Printf("[INFO] Retrieved %d chat sessions for user: %s", len(sessions), userID)
	return sessions, err
}

func (s *ChatService) GetChatSession(sessionID uuid.UUID, userID uuid.UUID) (*models.ChatSession, error) {
	log.Printf("[INFO] Getting chat session %s for user %s", sessionID, userID)
	var session models.ChatSession
	err := s.db.Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve chat session %s for user %s: %v", sessionID, userID, err)
		return nil, err
	}
	log.Printf("[INFO] Retrieved chat session %s with %d messages", sessionID, len(session.Messages))
	return &session, nil
}

func (s *ChatService) DeleteChatSession(sessionID uuid.UUID, userID uuid.UUID) error {
	log.Printf("[INFO] Deleting chat session %s for user %s", sessionID, userID)
	err := s.db.Where("id = ? AND user_id = ?", sessionID, userID).
		Update("is_active", false).Error
	if err != nil {
		log.Printf("[ERROR] Failed to delete chat session %s for user %s: %v", sessionID, userID, err)
		return err
	}
	log.Printf("[INFO] Successfully deleted chat session %s for user %s", sessionID, userID)
	return nil
}

func (s *ChatService) getOrCreateSession(userID uuid.UUID, sessionID *uuid.UUID) (*models.ChatSession, error) {
	if sessionID != nil {
		log.Printf("[INFO] Attempting to find existing session %s for user %s", *sessionID, userID)
		// Try to find existing session
		var session models.ChatSession
		err := s.db.Where("id = ? AND user_id = ? AND is_active = true", *sessionID, userID).First(&session).Error
		if err == nil {
			log.Printf("[INFO] Found existing session %s for user %s", *sessionID, userID)
			return &session, nil
		}
		log.Printf("[WARNING] Existing session %s not found for user %s, creating new session: %v", *sessionID, userID, err)
	}

	// Create new session
	log.Printf("[INFO] Creating new chat session for user %s", userID)
	session := &models.ChatSession{
		UserID:   userID,
		Title:    "New Chat",
		IsActive: true,
	}

	err := s.db.Create(session).Error
	if err != nil {
		log.Printf("[ERROR] Failed to create new session for user %s: %v", userID, err)
		return nil, err
	}

	log.Printf("[INFO] Created new session %s for user %s", session.ID, userID)
	return session, nil
}

// Feedback management
func (s *ChatService) SubmitFeedback(feedback *models.Feedback) error {
	log.Printf("[INFO] Submitting feedback for message %s by user %s, rating: %d", feedback.MessageID, feedback.UserID, feedback.Rating)
	
	// Log feedback content for debugging (be careful with sensitive data)
	feedbackJson, _ := json.Marshal(feedback)
	log.Printf("[DEBUG] Feedback details: %s", string(feedbackJson))
	
	err := s.db.Create(feedback).Error
	if err != nil {
		log.Printf("[ERROR] Failed to submit feedback for message %s: %v", feedback.MessageID, err)
		return err
	}
	log.Printf("[INFO] Feedback submitted successfully with ID: %s", feedback.ID)
	return nil
}

func (s *ChatService) GetFeedback(messageID *uuid.UUID, userID *uuid.UUID, limit, offset int) ([]models.Feedback, error) {
	log.Printf("[INFO] Getting feedback with messageID=%v, userID=%v, limit=%d, offset=%d", messageID, userID, limit, offset)
	
	var feedbacks []models.Feedback
	query := s.db.Preload("Message").Preload("User")

	if messageID != nil {
		query = query.Where("message_id = ?", *messageID)
		log.Printf("[DEBUG] Filtering feedback by message_id: %s", *messageID)
	}
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
		log.Printf("[DEBUG] Filtering feedback by user_id: %s", *userID)
	}

	err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&feedbacks).Error
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve feedback: %v", err)
		return nil, err
	}
	
	log.Printf("[INFO] Retrieved %d feedback records", len(feedbacks))
	return feedbacks, err
}
