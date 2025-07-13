package services

import (
	"context"
	"fmt"
	"log"
)

// AIProvider represents the different AI providers available
type AIProvider string

const (
	OpenAIProvider AIProvider = "openai"
	GeminiProvider AIProvider = "gemini"
)

// UnifiedAIService provides a unified interface for different AI providers
type UnifiedAIService struct {
	openAIService *OpenAIService
	geminiService *GeminiService
	primaryProvider AIProvider
	fallbackProvider AIProvider
}

// UnifiedChatRequest represents a chat request that works with any AI provider
type UnifiedChatRequest struct {
	Messages        []UnifiedChatMessage `json:"messages"`
	Context         []string            `json:"context,omitempty"`
	SessionID       string              `json:"session_id,omitempty"`
	UseKnowledgeBase bool               `json:"use_knowledge_base"`
	SystemPrompt    string              `json:"system_prompt,omitempty"`
	PreferredProvider AIProvider         `json:"preferred_provider,omitempty"`
}

type UnifiedChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type UnifiedChatResponse struct {
	Message   string     `json:"message"`
	Sources   []string   `json:"sources,omitempty"`
	SessionID string     `json:"session_id"`
	Provider  AIProvider `json:"provider"`
	Model     string     `json:"model"`
}

// NewUnifiedAIService creates a new unified AI service with multiple providers
func NewUnifiedAIService(openAIService *OpenAIService, geminiService *GeminiService, primaryProvider AIProvider) *UnifiedAIService {
	log.Printf("[INFO] Initializing unified AI service with primary provider: %s", primaryProvider)
	
	fallbackProvider := GeminiProvider
	if primaryProvider == GeminiProvider {
		fallbackProvider = OpenAIProvider
	}

	return &UnifiedAIService{
		openAIService:    openAIService,
		geminiService:    geminiService,
		primaryProvider:  primaryProvider,
		fallbackProvider: fallbackProvider,
	}
}

// ChatCompletion sends a chat request to the AI provider with fallback support
func (s *UnifiedAIService) ChatCompletion(ctx context.Context, req UnifiedChatRequest) (*UnifiedChatResponse, error) {
	log.Printf("[INFO] Processing unified chat completion request")
	
	// Determine which provider to use
	provider := s.primaryProvider
	if req.PreferredProvider != "" {
		provider = req.PreferredProvider
		log.Printf("[DEBUG] Using preferred provider: %s", provider)
	}

	// Try primary provider first
	response, err := s.callProvider(ctx, req, provider)
	if err != nil {
		log.Printf("[WARNING] Primary provider %s failed: %v", provider, err)
		
		// Try fallback provider
		log.Printf("[INFO] Attempting fallback to provider: %s", s.fallbackProvider)
		response, err = s.callProvider(ctx, req, s.fallbackProvider)
		if err != nil {
			log.Printf("[ERROR] Fallback provider %s also failed: %v", s.fallbackProvider, err)
			return nil, fmt.Errorf("both AI providers failed - primary: %s, fallback: %s", provider, s.fallbackProvider)
		}
		provider = s.fallbackProvider
	}

	response.Provider = provider
	log.Printf("[INFO] Successfully completed chat using provider: %s", provider)
	
	return response, nil
}

// callProvider calls the specific AI provider
func (s *UnifiedAIService) callProvider(ctx context.Context, req UnifiedChatRequest, provider AIProvider) (*UnifiedChatResponse, error) {
	switch provider {
	case OpenAIProvider:
		return s.callOpenAI(ctx, req)
	case GeminiProvider:
		return s.callGemini(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", provider)
	}
}

// callOpenAI converts the request and calls OpenAI
func (s *UnifiedAIService) callOpenAI(ctx context.Context, req UnifiedChatRequest) (*UnifiedChatResponse, error) {
	if s.openAIService == nil {
		return nil, fmt.Errorf("OpenAI service not available")
	}

	log.Printf("[DEBUG] Converting request for OpenAI")
	
	// Convert to OpenAI format
	openAIReq := OpenAIChatRequest{
		Context:         req.Context,
		SessionID:       req.SessionID,
		UseKnowledgeBase: req.UseKnowledgeBase,
	}

	// Convert messages
	for _, msg := range req.Messages {
		openAIReq.Messages = append(openAIReq.Messages, OpenAIChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	response, err := s.openAIService.ChatCompletion(ctx, openAIReq)
	if err != nil {
		return nil, err
	}

	return &UnifiedChatResponse{
		Message:   response.Message,
		Sources:   response.Sources,
		SessionID: response.SessionID,
		Model:     "openai", // Will be filled by the caller
	}, nil
}

// callGemini converts the request and calls Gemini
func (s *UnifiedAIService) callGemini(ctx context.Context, req UnifiedChatRequest) (*UnifiedChatResponse, error) {
	if s.geminiService == nil {
		return nil, fmt.Errorf("Gemini service not available")
	}

	log.Printf("[DEBUG] Converting request for Gemini")
	
	// Convert to Gemini format
	geminiReq := GeminiChatRequest{
		Context:         req.Context,
		SessionID:       req.SessionID,
		UseKnowledgeBase: req.UseKnowledgeBase,
		SystemPrompt:    req.SystemPrompt,
	}

	// Convert messages
	for _, msg := range req.Messages {
		geminiReq.Messages = append(geminiReq.Messages, GeminiChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	response, err := s.geminiService.ChatCompletion(ctx, geminiReq)
	if err != nil {
		return nil, err
	}

	return &UnifiedChatResponse{
		Message:   response.Message,
		Sources:   response.Sources,
		SessionID: response.SessionID,
		Model:     response.Model,
	}, nil
}

// CreateEmbedding creates an embedding using the preferred provider
func (s *UnifiedAIService) CreateEmbedding(ctx context.Context, text string, provider AIProvider) ([]float32, error) {
	log.Printf("[INFO] Creating embedding using provider: %s", provider)
	
	switch provider {
	case OpenAIProvider:
		if s.openAIService == nil {
			return nil, fmt.Errorf("OpenAI service not available")
		}
		return s.openAIService.CreateEmbedding(ctx, text)
	case GeminiProvider:
		if s.geminiService == nil {
			return nil, fmt.Errorf("Gemini service not available")
		}
		return s.geminiService.CreateEmbedding(ctx, text)
	default:
		return nil, fmt.Errorf("unsupported provider for embeddings: %s", provider)
	}
}

// GenerateTitle generates a title using Gemini (if available)
func (s *UnifiedAIService) GenerateTitle(ctx context.Context, content string) (string, error) {
	if s.geminiService != nil {
		return s.geminiService.GenerateTitle(ctx, content)
	}
	
	// Fallback: use the first few words as title
	words := fmt.Sprintf("%.50s", content)
	if len(content) > 50 {
		words += "..."
	}
	return words, nil
}

// SummarizeContent summarizes content using Gemini (if available)
func (s *UnifiedAIService) SummarizeContent(ctx context.Context, content string) (string, error) {
	if s.geminiService != nil {
		return s.geminiService.SummarizeContent(ctx, content)
	}
	
	// Fallback: use the first few sentences as summary
	summary := fmt.Sprintf("%.200s", content)
	if len(content) > 200 {
		summary += "..."
	}
	return summary, nil
}

// ExtractKeywords extracts keywords using Gemini (if available)
func (s *UnifiedAIService) ExtractKeywords(ctx context.Context, content string) ([]string, error) {
	if s.geminiService != nil {
		return s.geminiService.ExtractKeywords(ctx, content)
	}
	
	// Fallback: return basic keywords
	return []string{"content", "document"}, nil
}

// GetAvailableProviders returns the list of available AI providers
func (s *UnifiedAIService) GetAvailableProviders() []AIProvider {
	providers := make([]AIProvider, 0)
	
	if s.openAIService != nil {
		providers = append(providers, OpenAIProvider)
	}
	
	if s.geminiService != nil {
		providers = append(providers, GeminiProvider)
	}
	
	return providers
}

// SetPrimaryProvider changes the primary AI provider
func (s *UnifiedAIService) SetPrimaryProvider(provider AIProvider) error {
	availableProviders := s.GetAvailableProviders()
	
	for _, available := range availableProviders {
		if available == provider {
			s.primaryProvider = provider
			log.Printf("[INFO] Primary AI provider changed to: %s", provider)
			return nil
		}
	}
	
	return fmt.Errorf("provider %s is not available", provider)
}

// GetPrimaryProvider returns the current primary provider
func (s *UnifiedAIService) GetPrimaryProvider() AIProvider {
	return s.primaryProvider
}
