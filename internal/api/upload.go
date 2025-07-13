package api

import (
	"fmt"
	"path/filepath"
	"time"

	"tic-knowledge-system/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterUploadRoutes(app fiber.Router, db *gorm.DB) {
	app.Post("/upload", func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid multipart form"})
		}
		files := form.File["file"]
		if len(files) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
		}

		uploadedCount := 0
		for _, fileHeader := range files {
			filename := fileHeader.Filename
			destPath := filepath.Join("file", filename)

			if err := c.SaveFile(fileHeader, destPath); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
			}

			record := models.UploadedFile{
				FileName:   filename,
				FilePath:   destPath,
				UploadTime: time.Now(),
			}
			if err := db.Create(&record).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to insert file record"})
			}
			uploadedCount++
		}

		return c.JSON(fiber.Map{
			"message": fmt.Sprintf("%d file(s) uploaded successfully", uploadedCount),
			"count":   uploadedCount,
		})
	})

	app.Post("/context-file", func(c *fiber.Ctx) error {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file uploaded"})
		}
		filename := fileHeader.Filename
		destPath := filepath.Join("file", filename)
		if err := c.SaveFile(fileHeader, destPath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
		}

		labels := c.FormValue("labels", "")
		description := c.FormValue("description", "")
		status := c.FormValue("status", "Active")

		record := models.ContextFile{
			FileName:    filename,
			Labels:      labels,
			Description: description,
			Status:      status,
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&record).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to insert context file record"})
		}

		return c.JSON(fiber.Map{
			"message": "Context file uploaded successfully",
			"file": fiber.Map{
				"name": record.FileName,
				"labels": record.Labels,
				"description": record.Description,
				"status": record.Status,
				"updated": record.UpdatedAt,
			},
		})
	})

	app.Get("/upload/count", func(c *fiber.Ctx) error {
		var count int64
		if err := db.Model(&models.UploadedFile{}).Count(&count).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to count uploaded files"})
		}
		return c.JSON(fiber.Map{"count": count})
	})

	app.Get("/upload/files", func(c *fiber.Ctx) error {
		var files []models.UploadedFile
		if err := db.Find(&files).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch uploaded files"})
		}
		result := make([]fiber.Map, 0, len(files))
		for _, f := range files {
			result = append(result, fiber.Map{
				"name": f.FileName,
				"path": f.FilePath,
				"uploaded_at": f.UploadTime,
			})
		}
		return c.JSON(fiber.Map{"files": result})
	})
}
