package api

import (
	"log"
	"strconv"
	"tic-knowledge-system/internal/api/handlers"
	"tic-knowledge-system/internal/config"
	"tic-knowledge-system/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"gorm.io/gorm"
)

type Server struct {
	app                 *fiber.App
	cfg                 *config.Config
	db                  *gorm.DB
	knowledgeService    *services.KnowledgeService
	chatService         *services.ChatService
	openAIService       *services.OpenAIService
	geminiService       *services.GeminiService
	unifiedAIService    *services.UnifiedAIService
	enhancedChatService *services.EnhancedChatService
	vectorService       *services.VectorService
	documentService     *services.DocumentService
	fileUploadService   *services.FileUploadService
	assistantService    *services.OpenAIAssistantService
	aiHandler           *handlers.AIHandler
	documentHandler     *handlers.DocumentHandler
	fileUploadHandler   *handlers.FileUploadHandler
	assistantHandler    *handlers.OpenAIAssistantHandler
}

func NewServer(cfg *config.Config, db *gorm.DB) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	// Initialize services
	maxTokens, _ := strconv.Atoi(cfg.MaxTokens)
	temperature64, _ := strconv.ParseFloat(cfg.Temperature, 32)
	temperature := float32(temperature64)

	openAIService := services.NewOpenAIService(cfg.OpenAIKey, cfg.OpenAIModel, cfg.OpenAIEmbeddingModel, maxTokens, temperature)
	geminiService, err := services.NewGeminiService(cfg.GeminiAPIKey, cfg.GeminiModel, maxTokens, temperature)
	if err != nil {
		log.Printf("[WARNING] Failed to initialize Gemini service: %v", err)
		// Continue without Gemini service
	}
	unifiedAIService := services.NewUnifiedAIService(openAIService, geminiService, services.AIProvider(cfg.PrimaryAIProvider))
	vectorService := services.NewVectorService(cfg.VectorDBURL, cfg.QdrantCollectionName)
	knowledgeService := services.NewKnowledgeService(db, openAIService, vectorService)
	chatService := services.NewChatService(db, openAIService, knowledgeService)
	enhancedChatService := services.NewEnhancedChatService(db, unifiedAIService, knowledgeService)
	documentService := services.NewDocumentService(db, unifiedAIService, log.Default())

	// Initialize file upload service
	uploadDir := "./uploads"                               // You can configure this
	vectorStoreID := "vs_6873699daedc8191bb505a14254eeab3" // Fixed vector store ID
	fileUploadService := services.NewFileUploadService(db, cfg.OpenAIKey, vectorStoreID, uploadDir)

	// Initialize OpenAI Assistant service with default thread ID
	defaultThreadID := "thread_5GyQSnIxNy8uwMN2liLPuphc" // Your example thread ID
	assistantService := services.NewOpenAIAssistantService(cfg.OpenAIKey, defaultThreadID, log.Default())

	// Initialize handlers
	aiHandler := handlers.NewAIHandler(enhancedChatService)
	documentHandler := handlers.NewDocumentHandler(documentService, log.Default())
	fileUploadHandler := handlers.NewFileUploadHandler(fileUploadService, db, log.Default())
	assistantHandler := handlers.NewOpenAIAssistantHandler(assistantService, log.Default())

	server := &Server{
		app:                 app,
		cfg:                 cfg,
		db:                  db,
		knowledgeService:    knowledgeService,
		chatService:         chatService,
		openAIService:       openAIService,
		geminiService:       geminiService,
		unifiedAIService:    unifiedAIService,
		enhancedChatService: enhancedChatService,
		vectorService:       vectorService,
		documentService:     documentService,
		fileUploadService:   fileUploadService,
		assistantService:    assistantService,
		aiHandler:           aiHandler,
		documentHandler:     documentHandler,
		fileUploadHandler:   fileUploadHandler,
		assistantHandler:    assistantHandler,
	}

	// Middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("db", db)
		return c.Next()
	})
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Swagger documentation
	app.Get("/swagger/*", swagger.HandlerDefault)

	// API routes
	api := app.Group("/api/v1")
	server.setupRoutes(api)

	// Register upload routes
	RegisterUploadRoutes(api, db)

	// Register context dashboard route
	api.Get("/context-dashboard", handlers.GetContextDashboard(db))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	return app
}

func (s *Server) setupRoutes(api fiber.Router) {
	// Template routes
	templates := api.Group("/templates")
	templates.Get("/", s.getTemplates)
	templates.Post("/", s.createTemplate)
	templates.Get("/:id", s.getTemplate)
	templates.Put("/:id", s.updateTemplate)
	templates.Delete("/:id", s.deleteTemplate)

	// Knowledge entry routes
	knowledge := api.Group("/knowledge")
	knowledge.Get("/", s.getKnowledgeEntries)
	knowledge.Post("/", s.createKnowledgeEntry)
	knowledge.Get("/search", s.searchKnowledgeEntries)
	knowledge.Get("/:id", s.getKnowledgeEntry)
	knowledge.Put("/:id", s.updateKnowledgeEntry)
	knowledge.Delete("/:id", s.deleteKnowledgeEntry)

	// Chat routes
	chat := api.Group("/chat")
	chat.Post("/", s.processChat)
	chat.Get("/sessions", s.getChatSessions)
	chat.Get("/sessions/:id", s.getChatSession)
	chat.Delete("/sessions/:id", s.deleteChatSession)

	// Feedback routes
	feedback := api.Group("/feedback")
	feedback.Post("/", s.submitFeedback)
	feedback.Get("/", s.getFeedback)

	// User routes (basic implementation)
	users := api.Group("/users")
	users.Get("/me", s.getCurrentUser)

	// AI routes (new Gemini integration)
	ai := api.Group("/ai")
	ai.Post("/chat", s.aiHandler.ProcessChatWithAI)
	ai.Get("/providers", s.aiHandler.GetAvailableProviders)
	ai.Post("/providers/primary", s.aiHandler.SetPrimaryProvider)
	ai.Post("/compare", s.aiHandler.CompareProviders)

	// Document processing routes
	documents := api.Group("/documents")
	documents.Post("/process", s.documentHandler.ProcessDocument)
	documents.Get("/parse", s.documentHandler.ParseDocument)
	documents.Post("/process-wb", s.documentHandler.ProcessWBDocument)

	// File upload routes
	documents.Post("/upload", s.fileUploadHandler.UploadDocument)
	documents.Get("/:id/status", s.fileUploadHandler.GetDocumentStatus)
	documents.Post("/", s.fileUploadHandler.ListDocuments)

	// OpenAI Assistant routes
	assistant := api.Group("/assistant")
	assistant.Get("/health", s.assistantHandler.HealthCheck)
	assistant.Post("/chat", s.assistantHandler.ChatWithAssistant)
	assistant.Post("/chat/custom", s.assistantHandler.ChatWithCustomWorkflow)
	assistant.Post("/threads", s.assistantHandler.CreateThread)
	assistant.Get("/threads/:thread_id/messages", s.assistantHandler.GetThreadMessages)
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
		"code":  code,
	})
}
