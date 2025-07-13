# Tic Knowledge Management System

A comprehensive knowledge management and chatbot system designed to help organizations manage operational knowledge and provide AI-powered assistance to employees.

## Features

- **Template-based Knowledge Entry**: Pre-defined templates for consistent knowledge capture
- **RAG Content Management**: Manage and organize knowledge base content
- **AI Chatbot**: OpenAI-powered chatbot for answering operational questions
- **Feedback System**: Continuous improvement through user feedback
- **Vector Search**: Semantic search capabilities for better knowledge retrieval
- **User Management**: Role-based access control

## Architecture

- **Backend**: Go with Fiber framework
- **Database**: PostgreSQL for structured data
- **Vector Database**: Qdrant for embeddings and semantic search
- **AI Integration**: OpenAI API for embeddings and chat completions

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

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request
