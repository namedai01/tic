package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	Name      string         `json:"name" gorm:"not null" validate:"required"`
	Role      UserRole       `json:"role" gorm:"not null;default:'user'" validate:"required"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type UserRole string

const (
	AdminRole   UserRole = "admin"
	EditorRole  UserRole = "editor"
	RegularUser UserRole = "user"
	SupportRole UserRole = "support"
)

// Template represents a knowledge entry template
type Template struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string          `json:"name" gorm:"not null" validate:"required"`
	Description string          `json:"description"`
	Category    string          `json:"category" gorm:"not null" validate:"required"`
	Fields      []TemplateField `json:"fields" gorm:"foreignKey:TemplateID;constraint:OnDelete:CASCADE"`
	IsActive    bool            `json:"is_active" gorm:"default:true"`
	CreatedBy   uuid.UUID       `json:"created_by" gorm:"type:uuid;not null"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `json:"-" gorm:"index"`

	// Relations
	Creator User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
}

// TemplateField represents a field in a template
type TemplateField struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TemplateID  uuid.UUID      `json:"template_id" gorm:"type:uuid;not null"`
	Name        string         `json:"name" gorm:"not null" validate:"required"`
	Type        FieldType      `json:"type" gorm:"not null" validate:"required"`
	Label       string         `json:"label" gorm:"not null" validate:"required"`
	Description string         `json:"description"`
	Required    bool           `json:"required" gorm:"default:false"`
	Options     string         `json:"options"` // JSON string for select options
	Placeholder string         `json:"placeholder"`
	Validation  string         `json:"validation"` // JSON string for validation rules
	Order       int            `json:"order" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type FieldType string

const (
	TextFieldType     FieldType = "text"
	TextareaFieldType FieldType = "textarea"
	SelectFieldType   FieldType = "select"
	NumberFieldType   FieldType = "number"
	BooleanFieldType  FieldType = "boolean"
	DateFieldType     FieldType = "date"
	URLFieldType      FieldType = "url"
	EmailFieldType    FieldType = "email"
)

// KnowledgeEntry represents a knowledge base entry
type KnowledgeEntry struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title       string         `json:"title" gorm:"not null" validate:"required"`
	Content     string         `json:"content" gorm:"type:text;not null" validate:"required"`
	Summary     string         `json:"summary" gorm:"type:text"`
	Category    string         `json:"category" gorm:"not null" validate:"required"`
	Tags        string         `json:"tags"` // JSON array of tags
	TemplateID  *uuid.UUID     `json:"template_id" gorm:"type:uuid"`
	FieldData   string         `json:"field_data" gorm:"type:jsonb"` // JSON data for template fields
	IsPublished bool           `json:"is_published" gorm:"default:false"`
	Priority    int            `json:"priority" gorm:"default:0"`
	ViewCount   int            `json:"view_count" gorm:"default:0"`
	CreatedBy   uuid.UUID      `json:"created_by" gorm:"type:uuid;not null"`
	UpdatedBy   *uuid.UUID     `json:"updated_by" gorm:"type:uuid"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Template *Template `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	Creator  User      `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	Updater  *User     `json:"updater,omitempty" gorm:"foreignKey:UpdatedBy"`
}

// ChatSession represents a chat session
type ChatSession struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	Title     string         `json:"title"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	User     User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Messages []ChatMessage `json:"messages,omitempty" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// ChatMessage represents a message in a chat session
type ChatMessage struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SessionID uuid.UUID      `json:"session_id" gorm:"type:uuid;not null"`
	Role      MessageRole    `json:"role" gorm:"not null" validate:"required"`
	Content   string         `json:"content" gorm:"type:text;not null" validate:"required"`
	Metadata  string         `json:"metadata" gorm:"type:jsonb"` // For storing additional data like sources
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type MessageRole string

const (
	UserMessage      MessageRole = "user"
	AssistantMessage MessageRole = "assistant"
	SystemMessage    MessageRole = "system"
)

// Feedback represents user feedback on chat responses
type Feedback struct {
	ID         uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MessageID  uuid.UUID      `json:"message_id" gorm:"type:uuid;not null"`
	UserID     uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	Rating     int            `json:"rating" gorm:"not null" validate:"required,min=1,max=5"`
	Comment    string         `json:"comment" gorm:"type:text"`
	Type       FeedbackType   `json:"type" gorm:"not null"`
	IsResolved bool           `json:"is_resolved" gorm:"default:false"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Message ChatMessage `json:"message,omitempty" gorm:"foreignKey:MessageID"`
	User    User        `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type FeedbackType string

const (
	HelpfulFeedback    FeedbackType = "helpful"
	NotHelpfulFeedback FeedbackType = "not_helpful"
	IncorrectFeedback  FeedbackType = "incorrect"
	IncompleFeedback   FeedbackType = "incomplete"
)

// VectorEmbedding represents vector embeddings for semantic search
type VectorEmbedding struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	KnowledgeEntryID uuid.UUID      `json:"knowledge_entry_id" gorm:"type:uuid;not null"`
	VectorID         string         `json:"vector_id" gorm:"not null"` // ID in vector database
	ChunkIndex       int            `json:"chunk_index" gorm:"default:0"`
	ChunkText        string         `json:"chunk_text" gorm:"type:text"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	KnowledgeEntry KnowledgeEntry `json:"knowledge_entry,omitempty" gorm:"foreignKey:KnowledgeEntryID"`
}
type UploadedFile struct {
	ID         uint      `gorm:"primaryKey"`
	FileName   string    `gorm:"size:255;not null"`
	FilePath   string    `gorm:"size:255;not null"`
	UploadTime time.Time `gorm:"autoCreateTime"`
}

type APICallLog struct {
	ID       uint      `gorm:"primaryKey"`
	APIName  string    `gorm:"size:255;not null;index"`
	CalledAt time.Time `gorm:"autoCreateTime"`
}

type ContextFile struct {
	ID          uint      `gorm:"primaryKey"`
	FileName    string    `gorm:"size:255;not null;uniqueIndex"`
	Labels      string    `gorm:"size:255"` // comma-separated labels
	Description string    `gorm:"size:255"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	Status      string    `gorm:"size:50"`
}

type Topic struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"size:255;not null;uniqueIndex"`
	Description string    `gorm:"size:255"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

type TopicQuestionStat struct {
	ID        uint      `gorm:"primaryKey"`
	TopicID   uint      `gorm:"index"`
	Count     int       `gorm:"default:0"`
	Percent   int       `gorm:"default:0"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type TimeDistributionStat struct {
	ID        uint      `gorm:"primaryKey"`
	TimeRange string    `gorm:"size:50;not null;uniqueIndex"`
	Count     int       `gorm:"default:0"`
	Percent   int       `gorm:"default:0"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
