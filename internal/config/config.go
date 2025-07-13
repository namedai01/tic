package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	OpenAIKey   string
	JWTSecret   string
	VectorDBURL string
	CORSOrigins string

	// Database config
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	// OpenAI config
	OpenAIModel          string
	OpenAIEmbeddingModel string
	MaxTokens            string
	Temperature          string

	// Gemini config
	GeminiAPIKey string
	GeminiModel  string

	// AI Provider config
	PrimaryAIProvider string
	EmbeddingProvider string

	// Vector DB config
	QdrantHost           string
	QdrantPort           string
	QdrantCollectionName string
	VectorDimension      string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		OpenAIKey:   getEnv("OPENAI_API_KEY", ""),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		VectorDBURL: getEnv("VECTOR_DB_URL", "http://localhost:6333"),
		CORSOrigins: getEnv("CORS_ORIGINS", "*"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "tic_knowledge_db"),
		DBUser:     getEnv("DB_USER", "username"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		OpenAIModel:          getEnv("OPENAI_MODEL", "gpt-4"),
		OpenAIEmbeddingModel: getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-ada-002"),
		MaxTokens:            getEnv("MAX_TOKENS", "1000"),
		Temperature:          getEnv("TEMPERATURE", "0.7"),

		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		GeminiModel:  getEnv("GEMINI_MODEL", "gemini-1.5-pro"),

		PrimaryAIProvider: getEnv("PRIMARY_AI_PROVIDER", "openai"),
		EmbeddingProvider: getEnv("EMBEDDING_PROVIDER", "openai"),

		QdrantHost:           getEnv("QDRANT_HOST", "localhost"),
		QdrantPort:           getEnv("QDRANT_PORT", "6333"),
		QdrantCollectionName: getEnv("QDRANT_COLLECTION_NAME", "knowledge_base"),
		VectorDimension:      getEnv("VECTOR_DIMENSION", "1536"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
