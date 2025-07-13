# OpenAI Assistant API - Usage Examples

The service now includes a complete OpenAI Assistant API implementation that follows your 4-step workflow:

1. **Step 1**: Add message to thread
2. **Step 2**: Create and start a run  
3. **Step 3**: Sleep for 5 seconds
4. **Step 4**: Retrieve messages with run_id

## Available Endpoints

### 1. Health Check
```bash
curl -X GET "http://localhost:8080/api/v1/assistant/health"
```

### 2. Chat with Assistant (4-step workflow)
```bash
curl -X POST "http://localhost:8080/api/v1/assistant/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, can you help me with a question?",
    "assistant_id": "asst_your_assistant_id_here",
    "thread_id": "thread_5GyQSnIxNy8uwMN2liLPuphc"
  }'
```

### 3. Chat with Custom Workflow
```bash
curl -X POST "http://localhost:8080/api/v1/assistant/chat/custom" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, can you help me with a question?",
    "assistant_id": "asst_your_assistant_id_here", 
    "thread_id": "thread_5GyQSnIxNy8uwMN2liLPuphc",
    "wait_time": "10",
    "wait_for_completion": true
  }'
```

### 4. Get Thread Messages
```bash
curl -X GET "http://localhost:8080/api/v1/assistant/threads/thread_5GyQSnIxNy8uwMN2liLPuphc/messages"
```

### 5. Create New Thread
```bash
curl -X POST "http://localhost:8080/api/v1/assistant/threads"
```

## Response Format

### Chat Response
```json
{
  "thread_id": "thread_5GyQSnIxNy8uwMN2liLPuphc",
  "run_id": "run_53oP1Amg0ADv1OVQFDZ9OXqy",
  "messages": [
    {
      "id": "msg_...",
      "role": "assistant",
      "content": [
        {
          "type": "text",
          "text": {
            "value": "Hello! I'd be happy to help you with your question...",
            "annotations": []
          }
        }
      ],
      "created_at": 1705123456,
      "run_id": "run_53oP1Amg0ADv1OVQFDZ9OXqy",
      "metadata": {}
    }
  ],
  "status": "completed",
  "processed_at": "2025-07-13T15:48:00Z",
  "metadata": {
    "assistant_id": "asst_your_assistant_id_here",
    "original_message": "Hello, can you help me with a question?",
    "workflow_completed": true
  }
}
```

## Implementation Details

The OpenAI Assistant API service implements:

- **Exact 4-step workflow** as specified
- **Default thread ID**: `thread_5GyQSnIxNy8uwMN2liLPuphc` (from your example)
- **Custom wait times** for different use cases
- **Completion waiting** for workflows that need to wait for full completion
- **Thread management** for creating and managing conversation threads
- **Error handling** with detailed error messages
- **Logging** for debugging and monitoring

## Service Features

- ✅ **Step 1**: `POST /threads/{thread_id}/messages` - Add message to thread
- ✅ **Step 2**: `POST /threads/{thread_id}/runs` - Create and start run
- ✅ **Step 3**: `sleep(5000ms)` - Wait 5 seconds as specified
- ✅ **Step 4**: `GET /threads/{thread_id}/messages?run_id={run_id}` - Get messages by run ID

## Configuration

The service uses your OpenAI API key from the environment configuration and defaults to the thread ID you specified: `thread_5GyQSnIxNy8uwMN2liLPuphc`.

## Error Handling

All endpoints return proper HTTP status codes and JSON error responses:

```json
{
  "error": "Error description",
  "details": "Detailed error information"
}
```

## Next Steps

To use this with your actual OpenAI Assistant:

1. **Set your OpenAI API key** in the environment variables
2. **Get your Assistant ID** from the OpenAI platform
3. **Create or use existing Thread IDs** for conversations
4. **Make API calls** using the examples above

The service is now ready for production use with your OpenAI Assistant workflow!
