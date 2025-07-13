# File Upload API Documentation

## Overview

This API provides a complete file upload workflow that:
1. Accepts document files via HTTP upload
2. Stores files locally in the database
3. Uploads files to OpenAI Files API 
4. Adds files to a specific OpenAI Vector Store

## API Endpoints

### 1. Upload Document

**Endpoint:** `POST /api/v1/documents/upload`

**Description:** Upload a document file and automatically process it through OpenAI

**Request:**
- Method: POST
- Content-Type: multipart/form-data
- Parameters:
  - `file_name` (string, required): The name to save the file as
  - `file` (file, required): The document file to upload

**Example:**
```bash
curl -X POST "http://localhost:8080/api/v1/documents/upload" \
  -F "file_name=my_document.docx" \
  -F "file=@/path/to/document.docx"
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "file_name": "my_document.docx", 
  "status": "uploaded",
  "message": "Document uploaded successfully"
}
```

### 2. Get Document Status

**Endpoint:** `GET /api/v1/documents/{id}/status`

**Description:** Check the processing status of an uploaded document

**Parameters:**
- `id` (path parameter): Document ID from upload response

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/documents/550e8400-e29b-41d4-a716-446655440000/status"
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "file_name": "my_document.docx",
  "original_file_name": "document.docx",
  "file_path": "./uploads/my_document.docx",
  "file_size": 1024000,
  "mime_type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "openai_file_id": "file-abc123",
  "vector_store_id": "vs_6873699daedc8191bb505a14254eeab3",
  "vector_file_id": "file-vector-xyz789",
  "status": "added_to_vector",
  "uploaded_by": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-07-14T10:30:00Z",
  "updated_at": "2025-07-14T10:31:00Z"
}
```

### 3. List Documents

**Endpoint:** `GET /api/v1/documents`

**Description:** List uploaded documents with pagination

**Query Parameters:**
- `limit` (integer, optional): Number of documents to return (default: 10)
- `offset` (integer, optional): Number of documents to skip (default: 0)
- `uploaded_by` (UUID, optional): Filter by uploader user ID

**Example:**
```bash
curl -X GET "http://localhost:8080/api/v1/documents?limit=20&offset=0"
```

**Response:**
```json
{
  "documents": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "file_name": "my_document.docx",
      "status": "added_to_vector",
      "created_at": "2025-07-14T10:30:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

## Document Processing Workflow

### Status Flow
1. **`uploaded`** - File uploaded to local storage and database record created
2. **`sent_to_openai`** - File uploaded to OpenAI Files API (Step 1 complete)
3. **`added_to_vector`** - File added to vector store (Step 2 complete)
4. **`processing_failed`** - Error occurred during processing

### Automatic Processing Steps

#### Step 1: Upload to OpenAI Files API
```bash
curl --location 'https://api.openai.com/v1/files' \
--header 'Authorization: Bearer YOUR_OPENAI_API_KEY' \
--form 'purpose="assistants"' \
--form 'file=@"document.docx"'
```

#### Step 2: Add to Vector Store
```bash
curl https://api.openai.com/v1/vector_stores/vs_6873699daedc8191bb505a14254eeab3/files \
-H "Authorization: Bearer YOUR_OPENAI_API_KEY" \
-H "Content-Type: application/json" \
-H "OpenAI-Beta: assistants=v2" \
-d '{
  "file_id": "file-abc123"
}'
```

## Configuration

### Environment Variables
- `OPENAI_API_KEY`: Your OpenAI API key
- Vector Store ID is fixed: `vs_6873699daedc8191bb505a14254eeab3`
- Upload directory: `./uploads` (configurable)

### Database Schema

The `uploaded_documents` table stores:
- File metadata (name, size, mime type)
- Local file path
- OpenAI file ID and vector file ID
- Processing status and error messages
- Timestamps and uploader information

## Error Handling

### Common Errors
- **400 Bad Request**: Missing required parameters (`file_name` or `file`)
- **404 Not Found**: Document ID not found
- **500 Internal Server Error**: File processing or OpenAI API errors

### Error Response Format
```json
{
  "error": "Error description",
  "details": "Detailed error message"
}
```

## Testing

Use the provided test script:
```bash
./test_file_upload.sh
```

This script will:
1. Create a test document
2. Upload it via the API
3. Monitor processing status
4. List all documents
5. Clean up test files

## Supported File Types

The API accepts any file type, but OpenAI Files API supports:
- Text files (.txt, .md, .rtf)
- Documents (.docx, .pdf)
- Spreadsheets (.xlsx, .csv)
- Code files (.py, .js, .html, etc.)

## Rate Limits

Inherits OpenAI API rate limits:
- Files API: Varies by plan
- Vector Stores API: Varies by plan

## Security Considerations

- Files are stored locally in the uploads directory
- Implement proper authentication/authorization
- Validate file types and sizes
- Monitor OpenAI API usage and costs
