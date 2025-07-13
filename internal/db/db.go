package db

import (
	"tic-knowledge-system/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.User{},
		&models.Template{},
		&models.TemplateField{},
		&models.KnowledgeEntry{},
		&models.ChatSession{},
		&models.ChatMessage{},
		&models.Feedback{},
		&models.VectorEmbedding{},
<<<<<<< HEAD
		&models.UploadedFile{},
		&models.APICallLog{},
		&models.ContextFile{},
		&models.Topic{},
		&models.TopicQuestionStat{},
		&models.TimeDistributionStat{},
=======
		&models.UploadedDocument{},
>>>>>>> 7d682b7 (Update code)
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func RunMigrations(databaseURL string) error {
	// For now, we're using GORM's AutoMigrate
	// In production, you might want to use proper migrations
	return nil
}
