package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenAIAssistantService handles OpenAI Assistant API interactions
type OpenAIAssistantService struct {
	client   *openai.Client
	logger   *log.Logger
	threadID string
}

// NewOpenAIAssistantService creates a new OpenAI Assistant service
func NewOpenAIAssistantService(apiKey, threadID string, logger *log.Logger) *OpenAIAssistantService {
	config := openai.DefaultConfig(apiKey)
	
	// Create custom HTTP client with interceptor to add v2 header
	config.HTTPClient = &http.Client{
		Transport: &headerTransport{
			base: http.DefaultTransport,
		},
	}
	
	client := openai.NewClientWithConfig(config)
	return &OpenAIAssistantService{
		client:   client,
		logger:   logger,
		threadID: threadID,
	}
}

// headerTransport is a custom transport that adds the required OpenAI-Beta header
type headerTransport struct {
	base http.RoundTripper
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add the required header for Assistants API v2
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	return t.base.RoundTrip(req)
}

// ChatAssistantRequest represents the request for chat with assistant
type ChatAssistantRequest struct {
	Message      string `json:"message" validate:"required"`
	AssistantID  string `json:"assistant_id" validate:"required"`
	ThreadID     string `json:"thread_id,omitempty"`   // Optional, will use default if not provided
	TimeoutSeconds int  `json:"timeout_seconds,omitempty"` // Optional timeout in seconds, defaults to 30
}

// ChatAssistantResponse represents the response from assistant chat
type ChatAssistantResponse struct {
	ThreadID     string                    `json:"thread_id"`
	RunID        string                    `json:"run_id"`
	Messages     []AssistantMessage        `json:"messages"`
	Status       string                    `json:"status"`
	ProcessedAt  time.Time                 `json:"processed_at"`
	Metadata     map[string]interface{}    `json:"metadata"`
}

// AssistantMessage represents a message in the thread
type AssistantMessage struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	Content   []MessageContent       `json:"content"`
	CreatedAt int64                  `json:"created_at"`
	RunID     string                 `json:"run_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MessageContent represents the content of a message
type MessageContent struct {
	Type string          `json:"type"`
	Text MessageTextData `json:"text,omitempty"`
}

// MessageTextData represents text content
type MessageTextData struct {
	Value       string `json:"value"`
	Annotations []any  `json:"annotations,omitempty"`
}

// ChatWithAssistant implements the 4-step workflow you specified
func (s *OpenAIAssistantService) ChatWithAssistant(ctx context.Context, req ChatAssistantRequest) (*ChatAssistantResponse, error) {
	s.logger.Printf("Starting OpenAI Assistant chat workflow")
	
	// Use provided thread ID or default
	threadID := req.ThreadID
	if threadID == "" {
		threadID = s.threadID
	}
	
	// Step 1: Add message to thread
	s.logger.Printf("Step 1: Adding message to thread %s", threadID)
	message, err := s.addMessageToThread(ctx, threadID, req.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to add message to thread: %w", err)
	}
	s.logger.Printf("Message added successfully: %s", message.ID)
	
	// Step 2: Create and start a run
	s.logger.Printf("Step 2: Creating run for thread %s with assistant %s", threadID, req.AssistantID)
	run, err := s.createRun(ctx, threadID, req.AssistantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}
	s.logger.Printf("Run created successfully: %s", run.ID)
	
	// Step 3: Wait for run completion instead of fixed delay
	timeoutSeconds := req.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30 // Default to 30 seconds timeout
	}
	s.logger.Printf("Step 3: Waiting for run %s to complete (timeout: %d seconds)...", run.ID, timeoutSeconds)
	
	startTime := time.Now()
	finalStatus, err := s.WaitForRunCompletion(ctx, threadID, run.ID, time.Duration(timeoutSeconds)*time.Second)
	waitDuration := time.Since(startTime)
	
	if err != nil {
		s.logger.Printf("Warning: Run completion wait failed after %v: %v", waitDuration, err)
		// Continue to get messages even if wait failed
	} else {
		s.logger.Printf("Run %s completed with status: %s after %v", run.ID, finalStatus, waitDuration)
	}
	
	// Step 4: Get messages with run_id
	s.logger.Printf("Step 4: Retrieving messages ONLY for run %s", run.ID)
	messages, err := s.getMessagesWithRunID(ctx, threadID, run.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	s.logger.Printf("Retrieved %d messages specifically for run %s", len(messages), run.ID)
	
	// Check run status
	// runStatus, err := s.getRunStatus(ctx, threadID, run.ID)
	// if err != nil {
	// 	s.logger.Printf("Warning: Could not get run status: %v", err)
	// 	runStatus = "unknown"
	// }
	
	response := &ChatAssistantResponse{
		ThreadID:    threadID,
		RunID:       run.ID,
		Messages:    messages,
		Status:      finalStatus,
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"assistant_id":       req.AssistantID,
			"original_message":   req.Message,
			"timeout_seconds":    timeoutSeconds,
			"workflow_completed": true,
		},
	}
	
	s.logger.Printf("OpenAI Assistant workflow completed successfully")
	return response, nil
}

// addMessageToThread adds a message to the specified thread
func (s *OpenAIAssistantService) addMessageToThread(ctx context.Context, threadID, content string) (*openai.Message, error) {
	messageRequest := openai.MessageRequest{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	}
	
	message, err := s.client.CreateMessage(ctx, threadID, messageRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}
	
	return &message, nil
}

// createRun creates and starts a run on the thread
func (s *OpenAIAssistantService) createRun(ctx context.Context, threadID, assistantID string) (*openai.Run, error) {
	runRequest := openai.RunRequest{
		AssistantID: assistantID,
	}
	
	run, err := s.client.CreateRun(ctx, threadID, runRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}
	
	return &run, nil
}

// getMessagesWithRunID retrieves messages for a specific run
func (s *OpenAIAssistantService) getMessagesWithRunID(ctx context.Context, threadID, runID string) ([]AssistantMessage, error) {
	s.logger.Printf("Getting messages for specific run_id: %s in thread: %s", runID, threadID)
	
	// Get messages using the updated API with higher limit to ensure we get all messages
	limit := 100
	order := "desc"
	
	// Note: ListMessage API doesn't support filtering by runID directly
	// Parameters: ctx, threadID, limit, order, after, before
	messagesList, err := s.client.ListMessage(ctx, threadID, &limit, &order, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	
	var assistantMessages []AssistantMessage
	
	// Filter messages by run_id and convert to our format
	// Only return messages that specifically belong to this run
	for _, msg := range messagesList.Messages {
		// Check if message belongs to the specified run
		if msg.RunID != nil && *msg.RunID == runID {
			s.logger.Printf("Found message %s for run %s", msg.ID, runID)
			assistantMsg := s.convertToAssistantMessage(msg)
			assistantMessages = append(assistantMessages, assistantMsg)
		}
	}
	
	s.logger.Printf("Total messages found for run %s: %d", runID, len(assistantMessages))
	
	// Return only messages from this specific run (don't fallback to all messages)
	return assistantMessages, nil
}

// getRunStatus gets the current status of a run
func (s *OpenAIAssistantService) getRunStatus(ctx context.Context, threadID, runID string) (string, error) {
	run, err := s.client.RetrieveRun(ctx, threadID, runID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve run: %w", err)
	}
	
	return string(run.Status), nil
}

// convertToAssistantMessage converts OpenAI message to our format
func (s *OpenAIAssistantService) convertToAssistantMessage(msg openai.Message) AssistantMessage {
	var content []MessageContent
	
	// Convert message content
	for _, msgContent := range msg.Content {
		switch msgContent.Type {
		case "text":
			content = append(content, MessageContent{
				Type: "text",
				Text: MessageTextData{
					Value:       msgContent.Text.Value,
					Annotations: msgContent.Text.Annotations,
				},
			})
		default:
			// Handle other content types if needed
			content = append(content, MessageContent{
				Type: msgContent.Type,
			})
		}
	}
	
	// Convert metadata
	metadata := make(map[string]interface{})
	if msg.Metadata != nil {
		// Convert metadata map
		for k, v := range msg.Metadata {
			metadata[k] = v
		}
	}
	
	runID := ""
	if msg.RunID != nil {
		runID = *msg.RunID
	}
	
	return AssistantMessage{
		ID:        msg.ID,
		Role:      msg.Role,
		Content:   content,
		CreatedAt: int64(msg.CreatedAt),
		RunID:     runID,
		Metadata:  metadata,
	}
}

// GetThreadMessages gets all messages from a thread (utility method)
func (s *OpenAIAssistantService) GetThreadMessages(ctx context.Context, threadID string) ([]AssistantMessage, error) {
	if threadID == "" {
		threadID = s.threadID
	}
	
	limit := 50
	order := "desc"
	
	messagesList, err := s.client.ListMessage(ctx, threadID, &limit, &order, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	
	var assistantMessages []AssistantMessage
	for _, msg := range messagesList.Messages {
		assistantMsg := s.convertToAssistantMessage(msg)
		assistantMessages = append(assistantMessages, assistantMsg)
	}
	
	return assistantMessages, nil
}

// CreateThread creates a new thread (utility method)
func (s *OpenAIAssistantService) CreateThread(ctx context.Context) (*openai.Thread, error) {
	threadRequest := openai.ThreadRequest{}
	
	thread, err := s.client.CreateThread(ctx, threadRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}
	
	return &thread, nil
}

// WaitForRunCompletion waits for a run to complete (utility method)
func (s *OpenAIAssistantService) WaitForRunCompletion(ctx context.Context, threadID, runID string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		startTime := time.Now()
		status, err := s.getRunStatus(ctx, threadID, runID)
		waitDuration := time.Since(startTime)
		s.logger.Printf("getRunStatus after %v", waitDuration)
		if err != nil {
			return "", err
		}
		
		// Check if run is completed
		switch status {
		case "completed":
			return status, nil
		case "failed", "cancelled", "expired":
			return status, fmt.Errorf("run finished with status: %s", status)
		case "requires_action":
			return status, fmt.Errorf("run requires action, please handle manually")
		}
		
		// Wait before checking again
		time.Sleep(11 * time.Second)
	}
	
	return "", fmt.Errorf("timeout waiting for run completion")
}
