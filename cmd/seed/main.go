package main

import (
	"log"
	"tic-knowledge-system/internal/config"
	"tic-knowledge-system/internal/db"
	"tic-knowledge-system/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

	log.Println("Starting to populate database with mock data...")

	// Create mock data
	if err := createMockData(database); err != nil {
		log.Fatal("Failed to create mock data:", err)
	}

	log.Println("Successfully populated database with mock data!")
}

func createMockData(db *gorm.DB) error {
	// Create users
	users := []models.User{
		{
			ID:       uuid.New(),
			Email:    "admin@company.com",
			Name:     "System Administrator",
			Role:     models.AdminRole,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Email:    "support@company.com",
			Name:     "Support Manager",
			Role:     models.SupportRole,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Email:    "editor@company.com",
			Name:     "Content Editor",
			Role:     models.EditorRole,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			Email:    "user@company.com",
			Name:     "Regular User",
			Role:     models.RegularUser,
			IsActive: true,
		},
	}

	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}
	log.Printf("Created %d users", len(users))

	// Create templates
	templates := []models.Template{
		{
			ID:          uuid.New(),
			Name:        "Error Resolution Guide",
			Description: "Template for documenting how to resolve common errors",
			Category:    "troubleshooting",
			IsActive:    true,
			CreatedBy:   users[0].ID, // Admin
		},
		{
			ID:          uuid.New(),
			Name:        "Feature Documentation",
			Description: "Template for documenting application features",
			Category:    "documentation",
			IsActive:    true,
			CreatedBy:   users[1].ID, // Support Manager
		},
		{
			ID:          uuid.New(),
			Name:        "Process Guide",
			Description: "Template for documenting business processes",
			Category:    "process",
			IsActive:    true,
			CreatedBy:   users[2].ID, // Content Editor
		},
	}

	for _, template := range templates {
		if err := db.Create(&template).Error; err != nil {
			return err
		}
	}
	log.Printf("Created %d templates", len(templates))

	// Create template fields for Error Resolution Guide
	errorFields := []models.TemplateField{
		{
			TemplateID:  templates[0].ID,
			Name:        "error_code",
			Type:        models.TextFieldType,
			Label:       "Error Code",
			Description: "The specific error code or identifier",
			Required:    true,
			Order:       1,
		},
		{
			TemplateID:  templates[0].ID,
			Name:        "error_message",
			Type:        models.TextareaFieldType,
			Label:       "Error Message",
			Description: "The exact error message shown to users",
			Required:    true,
			Order:       2,
		},
		{
			TemplateID:  templates[0].ID,
			Name:        "solution_steps",
			Type:        models.TextareaFieldType,
			Label:       "Solution Steps",
			Description: "Step-by-step instructions to resolve the error",
			Required:    true,
			Order:       3,
		},
		{
			TemplateID:  templates[0].ID,
			Name:        "priority",
			Type:        models.SelectFieldType,
			Label:       "Priority",
			Description: "The priority level of this error",
			Required:    true,
			Options:     `["low", "medium", "high", "critical"]`,
			Order:       4,
		},
	}

	// Create template fields for Feature Documentation
	featureFields := []models.TemplateField{
		{
			TemplateID:  templates[1].ID,
			Name:        "feature_name",
			Type:        models.TextFieldType,
			Label:       "Feature Name",
			Description: "Name of the feature",
			Required:    true,
			Order:       1,
		},
		{
			TemplateID:  templates[1].ID,
			Name:        "description",
			Type:        models.TextareaFieldType,
			Label:       "Description",
			Description: "Detailed description of the feature",
			Required:    true,
			Order:       2,
		},
		{
			TemplateID:  templates[1].ID,
			Name:        "how_to_access",
			Type:        models.TextareaFieldType,
			Label:       "How to Access",
			Description: "Instructions on how to access this feature",
			Required:    true,
			Order:       3,
		},
		{
			TemplateID:  templates[1].ID,
			Name:        "required_permissions",
			Type:        models.TextFieldType,
			Label:       "Required Permissions",
			Description: "What permissions are needed to use this feature",
			Required:    false,
			Order:       4,
		},
	}

	allFields := append(errorFields, featureFields...)
	for _, field := range allFields {
		if err := db.Create(&field).Error; err != nil {
			return err
		}
	}
	log.Printf("Created %d template fields", len(allFields))

	// Create knowledge entries
	knowledgeEntries := []models.KnowledgeEntry{
		{
			ID:          uuid.New(),
			Title:       "Payment Processing Error PAY_001",
			Content:     "This error occurs when the payment gateway connection fails. Follow these steps to resolve: 1. Check internet connectivity 2. Verify API keys are correct 3. Check payment gateway status page 4. Contact payment provider if issue persists",
			Summary:     "How to resolve payment gateway connection failures",
			Category:    "troubleshooting",
			Tags:        `["payment", "error", "gateway", "PAY_001"]`,
			TemplateID:  &templates[0].ID,
			FieldData:   `{"error_code": "PAY_001", "error_message": "Payment gateway connection failed", "solution_steps": "1. Check internet connectivity\\n2. Verify API keys\\n3. Check gateway status\\n4. Contact provider", "priority": "high"}`,
			IsPublished: true,
			Priority:    5,
			CreatedBy:   users[1].ID,
		},
		{
			ID:          uuid.New(),
			Title:       "How to Process Orders",
			Content:     "To process orders in the system: 1. Navigate to Orders > Pending Orders 2. Click on an order to view details 3. Verify customer information and items 4. Update order status to 'Processing' 5. Generate shipping label 6. Update status to 'Shipped' when dispatched",
			Summary:     "Step-by-step guide for processing customer orders",
			Category:    "process",
			Tags:        `["orders", "processing", "workflow", "shipping"]`,
			IsPublished: true,
			Priority:    4,
			CreatedBy:   users[2].ID,
		},
		{
			ID:          uuid.New(),
			Title:       "User Permission Management",
			Content:     "The User Management feature allows administrators to control user access. To access: Admin Panel > Users > Manage Permissions. You can assign roles (Admin, Editor, User, Support) and specific permissions for each module.",
			Summary:     "How to manage user permissions and roles",
			Category:    "documentation",
			Tags:        `["users", "permissions", "roles", "admin"]`,
			TemplateID:  &templates[1].ID,
			FieldData:   `{"feature_name": "User Permission Management", "description": "Comprehensive user access control system", "how_to_access": "Admin Panel > Users > Manage Permissions", "required_permissions": "Admin role required"}`,
			IsPublished: true,
			Priority:    3,
			CreatedBy:   users[0].ID,
		},
		{
			ID:          uuid.New(),
			Title:       "Database Connection Error DB_502",
			Content:     "This error indicates the application cannot connect to the database. Common causes: 1. Database server is down 2. Connection string is incorrect 3. Network connectivity issues 4. Database credentials expired. Solutions: Check database server status, verify connection parameters, test network connectivity, update credentials if needed.",
			Summary:     "Troubleshooting database connection failures",
			Category:    "troubleshooting",
			Tags:        `["database", "connection", "error", "DB_502"]`,
			TemplateID:  &templates[0].ID,
			FieldData:   `{"error_code": "DB_502", "error_message": "Cannot connect to database server", "solution_steps": "1. Check database server status\\n2. Verify connection string\\n3. Test network connectivity\\n4. Update credentials if expired", "priority": "critical"}`,
			IsPublished: true,
			Priority:    5,
			CreatedBy:   users[1].ID,
		},
		{
			ID:          uuid.New(),
			Title:       "How to Generate Reports",
			Content:     "The reporting feature allows you to generate various business reports. Navigate to Reports > Report Builder. Select report type, date range, and filters. Click 'Generate Report' to create PDF or Excel output. Reports can be scheduled for automatic generation.",
			Summary:     "Guide to using the report generation feature",
			Category:    "documentation",
			Tags:        `["reports", "analytics", "export", "pdf", "excel"]`,
			TemplateID:  &templates[1].ID,
			FieldData:   `{"feature_name": "Report Generator", "description": "Create and schedule business reports", "how_to_access": "Reports > Report Builder", "required_permissions": "Editor role or higher"}`,
			IsPublished: true,
			Priority:    2,
			CreatedBy:   users[2].ID,
		},
		{
			ID:          uuid.New(),
			Title:       "Email Notification Setup",
			Content:     "Configure email notifications for important events: 1. Go to Settings > Notifications 2. Enable email notifications 3. Configure SMTP settings 4. Set up notification rules 5. Test email delivery. Ensure firewall allows SMTP traffic on port 587.",
			Summary:     "How to set up and configure email notifications",
			Category:    "configuration",
			Tags:        `["email", "notifications", "SMTP", "settings"]`,
			IsPublished: true,
			Priority:    3,
			CreatedBy:   users[0].ID,
		},
	}

	for _, entry := range knowledgeEntries {
		if err := db.Create(&entry).Error; err != nil {
			return err
		}
	}
	log.Printf("Created %d knowledge entries", len(knowledgeEntries))

	// Create chat sessions and messages
	chatSessions := []models.ChatSession{
		{
			ID:       uuid.New(),
			UserID:   users[3].ID, // Regular user
			Title:    "Payment Error Help",
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			UserID:   users[3].ID,
			Title:    "Order Processing Question",
			IsActive: true,
		},
	}

	for _, session := range chatSessions {
		if err := db.Create(&session).Error; err != nil {
			return err
		}
	}
	log.Printf("Created %d chat sessions", len(chatSessions))

	// Create chat messages
	chatMessages := []models.ChatMessage{
		{
			SessionID: chatSessions[0].ID,
			Role:      models.UserMessage,
			Content:   "I'm getting a payment error PAY_001 when trying to process customer payments. What should I do?",
		},
		{
			SessionID: chatSessions[0].ID,
			Role:      models.AssistantMessage,
			Content:   "The PAY_001 error indicates a payment gateway connection failure. Here's how to resolve it:\n\n1. Check your internet connectivity\n2. Verify that your API keys are correct\n3. Check the payment gateway status page\n4. Contact your payment provider if the issue persists\n\nThis is a high priority issue that should be addressed immediately to avoid disrupting customer transactions.",
			Metadata:  `{"sources": ["Payment Processing Error PAY_001"], "confidence": 0.95}`,
		},
		{
			SessionID: chatSessions[1].ID,
			Role:      models.UserMessage,
			Content:   "How do I process orders in the system? I'm new to this role.",
		},
		{
			SessionID: chatSessions[1].ID,
			Role:      models.AssistantMessage,
			Content:   "Welcome! Here's the step-by-step process for handling orders:\n\n1. Navigate to Orders > Pending Orders\n2. Click on an order to view its details\n3. Verify customer information and items\n4. Update the order status to 'Processing'\n5. Generate a shipping label\n6. Update status to 'Shipped' when the order is dispatched\n\nThis workflow ensures all orders are properly tracked and customers receive timely updates.",
			Metadata:  `{"sources": ["How to Process Orders"], "confidence": 0.92}`,
		},
	}

	for _, message := range chatMessages {
		if err := db.Create(&message).Error; err != nil {
			return err
		}
	}
	log.Printf("Created %d chat messages", len(chatMessages))

	// Create feedback
	feedbacks := []models.Feedback{
		{
			MessageID:  chatMessages[1].ID, // Feedback on assistant's payment error response
			UserID:     users[3].ID,
			Rating:     5,
			Comment:    "Very helpful! The steps were clear and resolved the issue quickly.",
			Type:       models.HelpfulFeedback,
			IsResolved: true,
		},
		{
			MessageID:  chatMessages[3].ID, // Feedback on order processing response
			UserID:     users[3].ID,
			Rating:     4,
			Comment:    "Good explanation, but could use screenshots for visual learners.",
			Type:       models.HelpfulFeedback,
			IsResolved: false,
		},
	}

	for _, feedback := range feedbacks {
		if err := db.Create(&feedback).Error; err != nil {
			return err
		}
	}
	log.Printf("Created %d feedback entries", len(feedbacks))

	return nil
}
