#!/bin/bash

# File Upload API Test Script
# This script demonstrates the complete file upload workflow

# Configuration
BASE_URL="http://localhost:8080/api/v1"
TEST_FILE="test_document.txt"
FILE_NAME="test_upload_$(date +%s).txt"

# Create a test file
echo "Creating test file..."
cat > "$TEST_FILE" << EOF
# Test Document for OpenAI Assistant

This is a test document that will be uploaded to the system.
It contains sample content for testing the file upload workflow.

## Features Tested:
1. Local file storage
2. OpenAI Files API upload
3. Vector Store integration

## Workflow:
- Step 1: Upload file to local storage and database
- Step 2: Send file to OpenAI Files API
- Step 3: Add file to vector store (vs_6873699daedc8191bb505a14254eeab3)

This document should be processed automatically by the system.
EOF

echo "Test file created: $TEST_FILE"

# Function to upload document
upload_document() {
    echo "Uploading document..."
    
    RESPONSE=$(curl -s -X POST "$BASE_URL/documents/upload" \
        -F "file_name=$FILE_NAME" \
        -F "file=@$TEST_FILE")
    
    echo "Upload Response:"
    echo "$RESPONSE" | jq '.'
    
    # Extract document ID
    DOCUMENT_ID=$(echo "$RESPONSE" | jq -r '.id')
    echo "Document ID: $DOCUMENT_ID"
    
    return 0
}

# Function to check document status
check_status() {
    local doc_id=$1
    echo "Checking document status..."
    
    curl -s -X GET "$BASE_URL/documents/$doc_id/status" | jq '.'
}

# Function to list documents
list_documents() {
    echo "Listing documents..."
    
    curl -s -X GET "$BASE_URL/documents?limit=10&offset=0" | jq '.'
}

# Function to test health endpoint
test_health() {
    echo "Testing health endpoint..."
    
    curl -s -X GET "$BASE_URL/../health" | jq '.'
}

# Main execution
echo "=== File Upload API Test ==="
echo "Base URL: $BASE_URL"
echo "Test File: $TEST_FILE"
echo "Upload Name: $FILE_NAME"
echo ""

# Test health first
echo "1. Testing server health..."
test_health
echo ""

# Upload document
echo "2. Uploading document..."
upload_document
echo ""

if [ ! -z "$DOCUMENT_ID" ] && [ "$DOCUMENT_ID" != "null" ]; then
    # Wait a bit for processing
    echo "3. Waiting 5 seconds for initial processing..."
    sleep 5
    
    # Check status
    echo "4. Checking document status..."
    check_status "$DOCUMENT_ID"
    echo ""
    
    # Check status again after some time for OpenAI processing
    echo "5. Waiting 10 seconds for OpenAI processing..."
    sleep 10
    
    echo "6. Checking document status again..."
    check_status "$DOCUMENT_ID"
    echo ""
fi

# List all documents
echo "7. Listing all documents..."
list_documents
echo ""

# Cleanup
echo "Cleaning up test file..."
rm -f "$TEST_FILE"

echo "=== Test Complete ==="
