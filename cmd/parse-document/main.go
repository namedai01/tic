package main

import (
	"log"
	"os"
	"tic-knowledge-system/internal/config"
	"tic-knowledge-system/internal/db"
	"tic-knowledge-system/internal/services"
)

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

	// Initialize services
	knowledgeService := services.NewKnowledgeService(database, nil, nil)
	documentParser := services.NewDocumentParserService(database, knowledgeService)

	// Parse the WB.docx file
	filePath := "/Applications/Me/git-prjs/daindq-prjs/tic/file/WB.docx"
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Fatalf("File does not exist: %s", filePath)
	}

	log.Printf("Parsing document: %s", filePath)

	// Use a default user ID (you can change this to an actual user ID from your users table)
	createdBy := "4566215d-9957-4765-9ac5-a9395879945e" // This is the user ID we've been using in tests

	result, err := documentParser.ParseDocumentFromPath(filePath, createdBy)
	if err != nil {
		log.Fatalf("Failed to parse document: %v", err)
	}

	log.Printf("Successfully parsed document!")
	log.Printf("- Original file: %s", result.OriginalFile)
	log.Printf("- Total chunks: %d", result.TotalChunks)
	log.Printf("- Knowledge entries created: %d", len(result.KnowledgeEntries))
	log.Printf("- Parsed at: %s", result.ParsedAt.Format("2006-01-02 15:04:05"))

	for i, entry := range result.KnowledgeEntries {
		log.Printf("  Entry %d:", i+1)
		log.Printf("    ID: %s", entry.ID)
		log.Printf("    Title: %s", entry.Title)
		log.Printf("    Content length: %d characters", len(entry.Content))
		log.Printf("    Tags: %s", entry.Tags)
		log.Printf("    Category: %s", entry.Category)
		log.Printf("    Is Published: %t", entry.IsPublished)
	}

	log.Println("Document parsing completed successfully!")
}
