package main

import (
	"log"
	"tic-knowledge-system/internal/api"
	"tic-knowledge-system/internal/config"
	"tic-knowledge-system/internal/db"
)

// @title Tic Knowledge Management API
// @version 1.0
// @description API for managing knowledge base and chatbot functionality
// @host localhost:8080
// @BasePath /api/v1
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Connect to database
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Start server
	server := api.NewServer(cfg, database)
	log.Printf("Server starting on port %s", cfg.Port)
	if err := server.Listen(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
