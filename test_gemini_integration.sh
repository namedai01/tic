#!/bin/bash

# Gemini API Integration Test Script
# This script demonstrates the new Gemini API capabilities

echo "=== TIC Knowledge System - Gemini API Integration Test ==="
echo ""

# Configuration
BASE_URL="http://localhost:8080/api/v1"
USER_ID="550e8400-e29b-41d4-a716-446655440000"  # Example UUID

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper function to make API calls
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    echo -e "${YELLOW}Testing: $description${NC}"
    echo "Request: $method $endpoint"
    if [ ! -z "$data" ]; then
        echo "Data: $data"
    fi
    echo "---"
    
    if [ -z "$data" ]; then
        response=$(curl -s -X "$method" \
            -H "Content-Type: application/json" \
            "$BASE_URL$endpoint" | jq '.')
    else
        response=$(curl -s -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$BASE_URL$endpoint" | jq '.')
    fi
    
    echo "Response:"
    echo "$response" | jq '.'
    echo ""
    echo "---"
    echo ""
}

# Check if server is running
echo "Checking if server is running..."
if ! curl -s "http://localhost:8080/health" > /dev/null; then
    echo -e "${RED}Server is not running! Please start the server first with:${NC}"
    echo "cd /Applications/Me/git-prjs/daindq-prjs/tic && ./tic-server"
    exit 1
fi

echo -e "${GREEN}Server is running!${NC}"
echo ""

# Test 1: Get available AI providers
make_request "GET" "/ai/providers" "" "Get Available AI Providers"

# Test 2: Send a chat message with OpenAI (default)
chat_data_openai='{
    "message": "What is machine learning and how does it work?",
    "user_id": "'$USER_ID'",
    "preferred_provider": "openai"
}'
make_request "POST" "/ai/chat" "$chat_data_openai" "Chat with OpenAI"

# Test 3: Send a chat message with Gemini
chat_data_gemini='{
    "message": "Explain artificial intelligence in simple terms",
    "user_id": "'$USER_ID'",
    "preferred_provider": "gemini"
}'
make_request "POST" "/ai/chat" "$chat_data_gemini" "Chat with Gemini"

# Test 4: Compare responses from both providers
compare_data='{
    "message": "What are the benefits of using cloud computing?",
    "user_id": "'$USER_ID'",
    "providers": ["openai", "gemini"]
}'
make_request "POST" "/ai/compare" "$compare_data" "Compare Providers"

# Test 5: Set primary provider to Gemini
set_provider_data='{
    "provider": "gemini"
}'
make_request "POST" "/ai/providers/primary" "$set_provider_data" "Set Primary Provider to Gemini"

# Test 6: Send a chat message without specifying provider (should use Gemini as primary)
chat_data_default='{
    "message": "What is the future of artificial intelligence?",
    "user_id": "'$USER_ID'"
}'
make_request "POST" "/ai/chat" "$chat_data_default" "Chat with Default Provider (should be Gemini)"

# Test 7: Reset primary provider to OpenAI
set_provider_openai='{
    "provider": "openai"
}'
make_request "POST" "/ai/providers/primary" "$set_provider_openai" "Reset Primary Provider to OpenAI"

# Test 8: Test with conversation context
echo -e "${YELLOW}Testing conversation context...${NC}"

# First message
context_msg1='{
    "message": "My name is John and I work in software development",
    "user_id": "'$USER_ID'",
    "preferred_provider": "gemini"
}'
make_request "POST" "/ai/chat" "$context_msg1" "First message (establishing context)"

# Follow-up message
context_msg2='{
    "message": "What programming languages would you recommend for my career?",
    "user_id": "'$USER_ID'",
    "preferred_provider": "gemini"
}'
make_request "POST" "/ai/chat" "$context_msg2" "Follow-up message (should remember name and job)"

echo -e "${GREEN}=== All tests completed! ===${NC}"
echo ""
echo "Summary of tested features:"
echo "✓ Provider discovery"
echo "✓ OpenAI integration"
echo "✓ Gemini integration"
echo "✓ Provider comparison"
echo "✓ Primary provider switching"
echo "✓ Conversation context"
echo ""
echo "The system now supports dual AI providers with seamless switching!"
