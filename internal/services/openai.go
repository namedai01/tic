package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type OpenAIService struct {
	client              *openai.Client
	model               string
	embeddingModel      string
	maxTokens           int
	temperature         float32
}

func NewOpenAIService(apiKey, model, embeddingModel string, maxTokens int, temperature float32) *OpenAIService {
	client := openai.NewClient(apiKey)
	return &OpenAIService{
		client:              client,
		model:               model,
		embeddingModel:      embeddingModel,
		maxTokens:           maxTokens,
		temperature:         temperature,
	}
}

type OpenAIChatRequest struct {
	Messages        []OpenAIChatMessage `json:"messages"`
	Context         []string      `json:"context,omitempty"`
	SessionID       string        `json:"session_id,omitempty"`
	UseKnowledgeBase bool         `json:"use_knowledge_base"`
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatResponse struct {
	Message   string   `json:"message"`
	Sources   []string `json:"sources,omitempty"`
	SessionID string   `json:"session_id"`
}

func (s *OpenAIService) ChatCompletion(ctx context.Context, req OpenAIChatRequest) (*OpenAIChatResponse, error) {
	// Build system message with context
	systemMessage := s.buildSystemMessage(req.Context)
	
	// Convert messages to OpenAI format
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemMessage,
		},
	}
	
	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Create chat completion request
	chatReq := openai.ChatCompletionRequest{
		Model:       s.model,
		Messages:    messages,
		MaxTokens:   s.maxTokens,
		Temperature: s.temperature,
	}

	resp, err := s.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	return &OpenAIChatResponse{
		Message:   resp.Choices[0].Message.Content,
		Sources:   req.Context, // Return the context sources used
		SessionID: req.SessionID,
	}, nil
}

func (s *OpenAIService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	}

	resp, err := s.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI embedding error: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return resp.Data[0].Embedding, nil
}

func (s *OpenAIService) CreateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.AdaEmbeddingV2,
	}

	resp, err := s.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI embedding error: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

func (s *OpenAIService) buildSystemMessage(context []string) string {
	baseMessage := `You are a helpful AI assistant for operational support. Your primary role is to help employees with questions about:
- How to operate the application/webapp
- Understanding error messages and their solutions
- Role and permission requirements
- Finding specific screens and features for processing orders and tasks
- Operational procedures and best practices

Guidelines:
1. Always be helpful, accurate, and concise
2. If you're not sure about something, say so
3. Provide step-by-step instructions when helpful
4. Reference specific screens, buttons, or features when relevant
5. If the question is outside your knowledge base, suggest who the user should contact

`

	if len(context) > 0 {
		baseMessage += "Based on the following knowledge base information:\n\n"
		for i, ctx := range context {
			baseMessage += fmt.Sprintf("Knowledge %d:\n%s\n\n", i+1, ctx)
		}
		baseMessage += "Please answer the user's question using this information as context."
	}

	return baseMessage
}

// ChunkText splits text into smaller chunks for better embedding
func (s *OpenAIService) ChunkText(text string, maxChunkSize int) []string {
	if maxChunkSize <= 0 {
		maxChunkSize = 1000 // Default chunk size
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var chunks []string
	var currentChunk strings.Builder
	
	for _, word := range words {
		// Check if adding this word would exceed the limit
		if currentChunk.Len() > 0 && currentChunk.Len()+len(word)+1 > maxChunkSize {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}
		
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(word)
	}
	
	// Add the last chunk if it's not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}
	
	return chunks
}
