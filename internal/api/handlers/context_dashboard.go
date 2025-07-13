package handlers

import (
	"time"
	"tic-knowledge-system/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func LogAPICall(db *gorm.DB, apiName string) {
	db.Create(&models.APICallLog{APIName: apiName, CalledAt: time.Now()})
}

func GetContextDashboard(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		LogAPICall(db, "GetContextDashboard")

		// Total Context Files
		var totalFiles int64
		db.Model(&models.ContextFile{}).Count(&totalFiles)

		// Total Topics
		var totalTopics int64
		db.Model(&models.Topic{}).Count(&totalTopics)

		// Most Attractive Topic (stub: just pick the first for now)
		var mostAttractiveTopic models.Topic
		db.First(&mostAttractiveTopic)

		// Topic Trends (last 30 days)
		var topicStats []models.TopicQuestionStat
		db.Find(&topicStats)
		topicTrends := []fiber.Map{}
		for _, stat := range topicStats {
			var topic models.Topic
			db.First(&topic, stat.TopicID)
			topicTrends = append(topicTrends, fiber.Map{
				"name": topic.Name,
				"count": stat.Count,
				"percent": stat.Percent,
			})
		}

		// Question Distribution by Time
		var timeStats []models.TimeDistributionStat
		db.Find(&timeStats)
		timeDist := []fiber.Map{}
		for _, stat := range timeStats {
			timeDist = append(timeDist, fiber.Map{
				"time_range": stat.TimeRange,
				"count": stat.Count,
				"percent": stat.Percent,
			})
		}

		// Context Files Table
		var files []models.ContextFile
		db.Find(&files)
		fileList := []fiber.Map{}
		for _, f := range files {
			fileList = append(fileList, fiber.Map{
				"name": f.FileName,
				"labels": f.Labels,
				"description": f.Description,
				"updated": f.UpdatedAt,
				"status": f.Status,
			})
		}

		return c.JSON(fiber.Map{
			"total_files": totalFiles,
			"total_topics": totalTopics,
			"most_attractive_topic": mostAttractiveTopic.Name,
			"topic_trends": topicTrends,
			"question_distribution": timeDist,
			"context_files": fileList,
		})
	}
} 