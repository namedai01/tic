package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiService struct {
	client              *genai.Client
	model               string
	maxTokens           int32
	temperature         float32
	topP                float32
	topK                int32
}

func NewGeminiService(apiKey, model string, maxTokens int, temperature float32) (*GeminiService, error) {
	log.Printf("[INFO] Initializing Gemini service with model: %s", model)
	
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Printf("[ERROR] Failed to create Gemini client: %v", err)
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Set default model if not provided
	if model == "" {
		model = "gemini-1.5-pro"
	}

	log.Printf("[INFO] Gemini service initialized successfully")
	
	return &GeminiService{
		client:      client,
		model:       model,
		maxTokens:   int32(maxTokens),
		temperature: temperature,
		topP:        0.95,
		topK:        40,
	}, nil
}

type GeminiChatRequest struct {
	Messages        []GeminiChatMessage `json:"messages"`
	Context         []string           `json:"context,omitempty"`
	SessionID       string             `json:"session_id,omitempty"`
	UseKnowledgeBase bool              `json:"use_knowledge_base"`
	SystemPrompt    string             `json:"system_prompt,omitempty"`
}

type GeminiChatMessage struct {
	Role    string `json:"role"`    // "user" or "model"
	Content string `json:"content"`
}

type GeminiChatResponse struct {
	Message   string   `json:"message"`
	Sources   []string `json:"sources,omitempty"`
	SessionID string   `json:"session_id"`
	Model     string   `json:"model"`
}

func (s *GeminiService) ChatCompletion(ctx context.Context, req GeminiChatRequest) (*GeminiChatResponse, error) {
	log.Printf("[INFO] Starting Gemini chat completion")
	log.Printf("[DEBUG] Request contains %d messages, knowledge_base=%t", len(req.Messages), req.UseKnowledgeBase)

	// Get the generative model
	model := s.client.GenerativeModel(s.model)
	
	// Configure generation parameters
	model.SetMaxOutputTokens(s.maxTokens)
	model.SetTemperature(s.temperature)
	model.SetTopP(s.topP)
	model.SetTopK(s.topK)

	// Build system instruction with context
	systemInstruction := s.buildSystemInstruction(req.Context, req.SystemPrompt)
	if systemInstruction != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(systemInstruction)},
		}
		log.Printf("[DEBUG] Set system instruction with %d characters", len(systemInstruction))
	}

	// Start a chat session
	chat := model.StartChat()
	
	// Add conversation history (excluding the last message)
	if len(req.Messages) > 1 {
		for _, msg := range req.Messages[:len(req.Messages)-1] {
			role := s.convertRole(msg.Role)
			chat.History = append(chat.History, &genai.Content{
				Parts: []genai.Part{genai.Text(msg.Content)},
				Role:  role,
			})
		}
		log.Printf("[DEBUG] Added %d messages to chat history", len(req.Messages)-1)
	}

	// Send the current message
	currentMessage := req.Messages[len(req.Messages)-1]
	log.Printf("[DEBUG] Sending message to Gemini: %.100s...", currentMessage.Content)

	resp, err := chat.SendMessage(ctx, genai.Text(currentMessage.Content))
	if err != nil {
		log.Printf("[ERROR] Gemini API call failed: %v", err)
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 {
		log.Printf("[ERROR] No response candidates from Gemini")
		return nil, fmt.Errorf("no response from Gemini")
	}

	// Extract the response text
	var responseText strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		responseText.WriteString(fmt.Sprintf("%v", part))
	}

	response := responseText.String()
	log.Printf("[INFO] Gemini API call successful, response length: %d characters", len(response))

	return &GeminiChatResponse{
		Message:   response,
		Sources:   req.Context, // Return the context sources used
		SessionID: req.SessionID,
		Model:     s.model,
	}, nil
}

func (s *GeminiService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	log.Printf("[INFO] Creating embedding for text with length: %d characters", len(text))
	
	// Use Gemini's embedding model
	model := s.client.EmbeddingModel("text-embedding-004")
	
	resp, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		log.Printf("[ERROR] Gemini embedding error: %v", err)
		return nil, fmt.Errorf("Gemini embedding error: %w", err)
	}

	log.Printf("[INFO] Successfully created embedding with %d dimensions", len(resp.Embedding.Values))
	return resp.Embedding.Values, nil
}

func (s *GeminiService) GenerateTitle(ctx context.Context, content string) (string, error) {
	log.Printf("[INFO] Generating title for content with length: %d characters", len(content))
	
	model := s.client.GenerativeModel(s.model)
	model.SetMaxOutputTokens(50)
	model.SetTemperature(0.3)

	prompt := fmt.Sprintf(`Generate a concise, descriptive title (maximum 10 words) for the following content:

%s

Title:`, content[:min(len(content), 500)]) // Limit content to first 500 chars

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("[ERROR] Failed to generate title: %v", err)
		return "", fmt.Errorf("failed to generate title: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no title generated")
	}

	title := strings.TrimSpace(fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]))
	log.Printf("[INFO] Generated title: %s", title)
	
	return title, nil
}

func (s *GeminiService) SummarizeContent(ctx context.Context, content string) (string, error) {
	log.Printf("[INFO] Summarizing content with length: %d characters", len(content))
	
	model := s.client.GenerativeModel(s.model)
	model.SetMaxOutputTokens(200)
	model.SetTemperature(0.3)

	prompt := fmt.Sprintf(`Provide a concise summary (2-3 sentences) of the following content:

%s

Summary:`, content)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("[ERROR] Failed to generate summary: %v", err)
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no summary generated")
	}

	summary := strings.TrimSpace(fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]))
	log.Printf("[INFO] Generated summary with length: %d characters", len(summary))
	
	return summary, nil
}

func (s *GeminiService) ExtractKeywords(ctx context.Context, content string) ([]string, error) {
	log.Printf("[INFO] Extracting keywords from content with length: %d characters", len(content))
	
	model := s.client.GenerativeModel(s.model)
	model.SetMaxOutputTokens(100)
	model.SetTemperature(0.3)

	prompt := fmt.Sprintf(`Extract 5-10 relevant keywords or phrases from the following content. Return them as a comma-separated list:

%s

Keywords:`, content[:min(len(content), 1000)]) // Limit content

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("[ERROR] Failed to extract keywords: %v", err)
		return nil, fmt.Errorf("failed to extract keywords: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return []string{}, fmt.Errorf("no keywords generated")
	}

	keywordsText := strings.TrimSpace(fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]))
	keywords := make([]string, 0)
	
	// Split by commas and clean up
	for _, keyword := range strings.Split(keywordsText, ",") {
		cleaned := strings.TrimSpace(keyword)
		if cleaned != "" {
			keywords = append(keywords, cleaned)
		}
	}

	log.Printf("[INFO] Extracted %d keywords: %v", len(keywords), keywords)
	return keywords, nil
}

func (s *GeminiService) buildSystemInstruction(context []string, customPrompt string) string {
	var instruction strings.Builder
	
	// Base system instruction
	baseInstruction := `You are an AI assistant for a knowledge management system. You help employees find information and answer questions about operational procedures, troubleshooting, and company processes.

Instructions:
- Provide accurate, helpful, and concise responses
- If you use information from the knowledge base, acknowledge the source
- If you're unsure about something, say so rather than guessing
- Format your responses clearly with bullet points or numbered lists when appropriate
- Focus on practical, actionable advice`

	instruction.WriteString(baseInstruction)

	// Add custom system prompt if provided
	if customPrompt != "" {
		instruction.WriteString("\n\nAdditional Instructions:\n")
		instruction.WriteString(customPrompt)
	}

	// Add knowledge base context if available
	if len(context) > 0 {
		instruction.WriteString("\n\nRelevant Knowledge Base Information:\n")
		for i, ctx := range context {
			instruction.WriteString(fmt.Sprintf("%d. %s\n", i+1, ctx))
		}
		instruction.WriteString("\nUse this information to help answer the user's question when relevant.")
	}

	return instruction.String()
}

func (s *GeminiService) convertRole(role string) string {
	switch strings.ToLower(role) {
	case "user":
		return "user"
	case "assistant", "model":
		return "model"
	default:
		return "user"
	}
}

func (s *GeminiService) Close() error {
	log.Printf("[INFO] Closing Gemini client")
	return s.client.Close()
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
