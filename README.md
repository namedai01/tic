# Tic Knowledge Management System

A comprehensive knowledge management and chatbot system designed to help organizations manage operational knowledge and provide AI-powered assistance to employees.

## 🎯 Problem Statement

Products serving operations are often complex, have many features, and require training and guidance from the business for new, inexperienced employees. The cost of this training in businesses with many employees and high turnover/new recruitment rates is very high. Similarly, the cost of senior members guiding/answering new members.

## Features

- **Template-based Knowledge Entry**: Pre-defined templates for consistent knowledge capture
- **RAG Content Management**: Manage and organize knowledge base content
- **AI Chatbot**: OpenAI-powered chatbot for answering operational questions
- **Feedback System**: Continuous improvement through user feedback
- **Vector Search**: Semantic search capabilities for better knowledge retrieval
- **User Management**: Role-based access control

## 🚀 Solution

A comprehensive system that allows employees to:
- Self-manage data/knowledge using pre-defined templates
- Access AI-powered chatbot for operational questions
- Provide feedback to continuously improve the knowledge base
- Search and retrieve relevant information quickly

## 🏗️ System Architecture

- **Backend**: Go 1.23+ with Fiber v2.52.0 framework
- **Primary Database**: PostgreSQL 15 for structured data
- **Vector Database**: Qdrant for semantic search capabilities
- **Cache**: Redis for performance optimization
- **AI Integration**: OpenAI API (GPT-4 + text-embedding-ada-002)
- **Authentication**: JWT-based authentication system

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL
- Qdrant vector database
- OpenAI API key

### Installation

1. Clone the repository
2. Copy `.env.example` to `.env` and configure your environment variables
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Run database migrations:
   ```bash
   make migrate-up
   ```
5. Start the server:
   ```bash
   make run
   ```

## API Documentation

Once the server is running, visit `http://localhost:8080/swagger` for API documentation.

## Project Structure

```
├── cmd/                 # Application entry points
├── internal/           # Internal application code
│   ├── api/           # API handlers and routes
│   ├── config/        # Configuration management
│   ├── db/            # Database connection and migrations
│   ├── models/        # Data models
│   ├── services/      # Business logic
│   └── utils/         # Utility functions
├── migrations/         # Database migrations
├── docs/              # API documentation
└── docker/            # Docker configuration
```

## 📊 Database Schema

### Core Tables Overview

#### 1. **`users` Table**
**Purpose**: User management and role-based access control
```sql
- id (UUID, Primary Key)
- name (VARCHAR) - User full name
- email (VARCHAR, UNIQUE) - User email address
- role (VARCHAR) - User role (admin, support, editor, regular_user)
- is_active (BOOLEAN) - Account status
- created_at, updated_at (TIMESTAMP)
```

#### 2. **`templates` Table**
**Purpose**: Defines structured formats for knowledge entry
```sql
- id (UUID, Primary Key)
- name (VARCHAR) - Template name
- description (TEXT) - Template description
- created_by (UUID, Foreign Key -> users.id)
- is_active (BOOLEAN) - Template availability
- created_at, updated_at (TIMESTAMP)
```

#### 3. **`template_fields` Table**
**Purpose**: Dynamic form fields for each template
```sql
- id (UUID, Primary Key)
- template_id (UUID, Foreign Key -> templates.id)
- field_name (VARCHAR) - Field identifier
- field_type (VARCHAR) - Field type (text, textarea, select, etc.)
- field_label (VARCHAR) - Display label
- is_required (BOOLEAN) - Validation requirement
- field_order (INTEGER) - Display sequence
- field_options (JSONB) - Field configuration
```

#### 4. **`knowledge_entries` Table**
**Purpose**: Core knowledge base content storage
```sql
- id (UUID, Primary Key)
- template_id (UUID, Foreign Key -> templates.id)
- title (VARCHAR) - Knowledge entry title
- content (TEXT) - Main content
- data (JSONB) - Structured data based on template
- tags (TEXT[]) - Categorization tags
- status (VARCHAR) - Content status (draft, published, archived)
- created_by (UUID, Foreign Key -> users.id)
- created_at, updated_at (TIMESTAMP)
```

#### 5. **`chat_sessions` Table**
**Purpose**: Conversation context management
```sql
- id (UUID, Primary Key)
- user_id (UUID, Foreign Key -> users.id)
- title (VARCHAR) - Session title
- is_active (BOOLEAN) - Session status
- created_at, updated_at (TIMESTAMP)
```

#### 6. **`chat_messages` Table**
**Purpose**: Individual message storage in conversations
```sql
- id (UUID, Primary Key)
- session_id (UUID, Foreign Key -> chat_sessions.id)
- role (VARCHAR) - Message type (user/assistant)
- content (TEXT) - Message content
- metadata (JSONB) - Additional context (confidence, sources)
- created_at (TIMESTAMP)
```

#### 7. **`feedbacks` Table**
**Purpose**: Quality improvement through user feedback
```sql
- id (UUID, Primary Key)
- message_id (UUID, Foreign Key -> chat_messages.id)
- user_id (UUID, Foreign Key -> users.id)
- rating (INTEGER) - Quality score (1-5)
- comment (TEXT) - Detailed feedback
- feedback_type (VARCHAR) - Classification
- created_at (TIMESTAMP)
```

#### 8. **`vector_embeddings` Table**
**Purpose**: Semantic search capabilities
```sql
- id (UUID, Primary Key)
- knowledge_entry_id (UUID, Foreign Key -> knowledge_entries.id)
- embedding (VECTOR) - Vector representation
- model_name (VARCHAR) - AI model used
- created_at (TIMESTAMP)
```

## 🔄 RAG Process Flow

1. **User Query** → System receives question
2. **Embedding Generation** → Convert query to vector using OpenAI
3. **Semantic Search** → Find relevant knowledge using vector similarity
4. **Context Building** → Compile relevant knowledge entries
5. **AI Response** → Generate response using OpenAI + context
6. **Storage** → Save conversation and collect feedback
7. **Improvement** → Use feedback to enhance knowledge base

## 📋 Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Qdrant vector database
- Redis (optional, for caching)
- OpenAI API access


### Templates Management
```bash
GET    /api/v1/templates           # List all templates
POST   /api/v1/templates           # Create new template
GET    /api/v1/templates/:id       # Get template by ID
PUT    /api/v1/templates/:id       # Update template
DELETE /api/v1/templates/:id       # Delete template
```

### Knowledge Base Management
```bash
GET    /api/v1/knowledge           # List knowledge entries
POST   /api/v1/knowledge           # Create new knowledge entry
GET    /api/v1/knowledge/search    # Search knowledge entries
GET    /api/v1/knowledge/:id       # Get knowledge entry by ID
PUT    /api/v1/knowledge/:id       # Update knowledge entry
DELETE /api/v1/knowledge/:id       # Delete knowledge entry
```

### Chat & AI Integration
```bash
POST   /api/v1/chat                # Send message to AI chatbot
GET    /api/v1/chat/sessions       # List user's chat sessions
GET    /api/v1/chat/sessions/:id   # Get specific chat session
DELETE /api/v1/chat/sessions/:id   # Delete chat session
```

### Feedback Management
```bash
POST   /api/v1/feedback            # Submit feedback on AI response
GET    /api/v1/feedback            # List feedback (admin only)
```

### User Management
```bash
GET    /api/v1/users/me            # Get current user profile
```

## 🧪 Usage Examples

### 1. Create Knowledge Template
```bash
curl -X POST http://localhost:8080/api/v1/templates \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Error Resolution Guide",
    "description": "Template for documenting error solutions",
    "fields": [
      {
        "name": "error_code",
        "type": "string",
        "label": "Error Code",
        "required": true
      },
      {
        "name": "solution_steps",
        "type": "text",
        "label": "Solution Steps",
        "required": true
      }
    ]
  }'
```

### 2. Add Knowledge Entry
```bash
curl -X POST http://localhost:8080/api/v1/knowledge \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "template-uuid-here",
    "title": "Payment Processing Error PAY_001",
    "content": "How to resolve payment processing errors",
    "data": {
      "error_code": "PAY_001",
      "solution_steps": "1. Check payment gateway\n2. Verify API keys\n3. Review logs"
    },
    "tags": ["payment", "error", "troubleshooting"]
  }'
```

### 3. Chat with AI Assistant
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "How do I resolve payment processing errors?",
    "user_id": "user-uuid-here"
  }'
```

**Example Response:**
```json
{
  "response": "Based on the knowledge base, here are the steps to resolve payment processing errors:\n\n1. **Check Payment Gateway Connection**\n   - Verify network connectivity\n   - Check gateway status page\n\n2. **Verify API Keys**\n   - Ensure API keys are valid\n   - Check expiration dates\n\n3. **Review Transaction Logs**\n   - Look for specific error codes\n   - Check for patterns in failures\n\nFor error code PAY_001 specifically, this usually indicates a gateway timeout issue.",
  "session_id": "session-uuid-here",
  "relevant_knowledge": ["knowledge-entry-uuid"],
  "confidence": 0.89,
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 4. Search Knowledge Base
```bash
curl "http://localhost:8080/api/v1/knowledge/search?q=payment%20error&limit=5"
```

### 5. Submit Feedback
```bash
curl -X POST http://localhost:8080/api/v1/feedback \
  -H "Content-Type: application/json" \
  -d '{
    "message_id": "message-uuid-here",
    "rating": 5,
    "comment": "Very helpful response!",
    "feedback_type": "helpful"
  }'
```

## 🛠️ Development Commands

```bash
# Development
make run          # Run the application
make test         # Run tests
make build        # Build binary

# Database
make docker-up    # Start infrastructure services
make docker-down  # Stop infrastructure services
make seed         # Populate database with sample data

# Documentation
make swagger      # Generate API documentation

# Production
make prod-build   # Build production binary
```

## 🔧 System Features

### ✅ Implemented Features
- **Template System**: Pre-defined structures for knowledge entry
- **Knowledge Management**: Full CRUD operations with structured data
- **AI-Powered Chat**: OpenAI integration with context-aware responses
- **Semantic Search**: Vector embeddings for finding relevant knowledge
- **Feedback System**: Continuous improvement through user feedback
- **Session Management**: Conversation history and context preservation
- **Role-Based Access**: Different permission levels for users
- **Comprehensive Logging**: Full request/response tracking

### 🔄 RAG Capabilities
- **Context Retrieval**: Finds relevant knowledge for user queries
- **Response Generation**: Uses OpenAI with retrieved context
- **Conversation Memory**: Maintains chat history for better context
- **Confidence Scoring**: Measures response quality
- **Source Attribution**: Links responses to source knowledge

## 📈 Performance & Scalability

- **Vector Search**: Qdrant provides fast semantic search
- **Database Indexing**: Optimized queries with proper indexes
- **Connection Pooling**: Efficient database connection management
- **Stateless Design**: Horizontal scaling capability
- **Caching**: Redis integration for performance optimization

## 🔐 Security Features

- **JWT Authentication**: Secure user sessions
- **Role-Based Access**: Permission-based feature access
- **Input Validation**: Prevents injection attacks
- **Rate Limiting**: (Ready for implementation)
- **CORS Configuration**: Configurable cross-origin policies

## 📊 Monitoring & Observability

- **Comprehensive Logging**: Request/response tracking
- **Health Checks**: System status monitoring
- **Error Tracking**: Detailed error logging
- **Performance Metrics**: (Ready for implementation)

## 🚀 Production Deployment

### Docker Deployment
```bash
# Build production image
make docker-build

# Run in production mode
make docker-run
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request
