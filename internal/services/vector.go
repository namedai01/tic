package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// VectorService handles vector database operations (Qdrant)
type VectorService struct {
	baseURL        string
	collectionName string
	httpClient     *http.Client
}

func NewVectorService(baseURL, collectionName string) *VectorService {
	return &VectorService{
		baseURL:        baseURL,
		collectionName: collectionName,
		httpClient:     &http.Client{},
	}
}

type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

type QdrantSearchRequest struct {
	Vector      []float32 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
}

type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
}

type QdrantSearchResult struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

type VectorSearchResult struct {
	KnowledgeEntryID uuid.UUID
	Score            float64
	ChunkText        string
}

func (s *VectorService) InitializeCollection(ctx context.Context, dimension int) error {
	// Create collection if it doesn't exist
	createCollectionReq := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     dimension,
			"distance": "Cosine",
		},
	}

	reqBody, err := json.Marshal(createCollectionReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s", s.baseURL, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 200 OK or 400 Bad Request (if collection already exists) are both acceptable
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("failed to create collection: status %d", resp.StatusCode)
	}

	return nil
}

func (s *VectorService) Store(ctx context.Context, vector []float32, text string, knowledgeEntryID uuid.UUID) (string, error) {
	pointID := uuid.New().String()

	point := QdrantPoint{
		ID:     pointID,
		Vector: vector,
		Payload: map[string]interface{}{
			"text":                text,
			"knowledge_entry_id":  knowledgeEntryID.String(),
		},
	}

	reqBody := map[string]interface{}{
		"points": []QdrantPoint{point},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/collections/%s/points", s.baseURL, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to store vector: status %d", resp.StatusCode)
	}

	return pointID, nil
}

func (s *VectorService) Search(ctx context.Context, query string, limit int) ([]VectorSearchResult, error) {
	// This is a simplified version - in practice, you'd need to convert the query to a vector first
	// using the OpenAI embedding service, then search with that vector
	return nil, fmt.Errorf("search by text not implemented - use SearchByVector instead")
}

func (s *VectorService) SearchByVector(ctx context.Context, vector []float32, limit int) ([]VectorSearchResult, error) {
	searchReq := QdrantSearchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
	}

	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", s.baseURL, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to search vectors: status %d", resp.StatusCode)
	}

	var searchResp QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	var results []VectorSearchResult
	for _, result := range searchResp.Result {
		knowledgeEntryIDStr, ok := result.Payload["knowledge_entry_id"].(string)
		if !ok {
			continue
		}

		knowledgeEntryID, err := uuid.Parse(knowledgeEntryIDStr)
		if err != nil {
			continue
		}

		text, _ := result.Payload["text"].(string)

		results = append(results, VectorSearchResult{
			KnowledgeEntryID: knowledgeEntryID,
			Score:            result.Score,
			ChunkText:        text,
		})
	}

	return results, nil
}

func (s *VectorService) Delete(ctx context.Context, pointID string) error {
	reqBody := map[string]interface{}{
		"points": []string{pointID},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", s.baseURL, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete vector: status %d", resp.StatusCode)
	}

	return nil
}

func (s *VectorService) DeleteByKnowledgeEntry(ctx context.Context, knowledgeEntryID uuid.UUID) error {
	// Delete all points associated with a knowledge entry
	filter := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key":   "knowledge_entry_id",
				"match": map[string]string{"value": knowledgeEntryID.String()},
			},
		},
	}

	reqBody := map[string]interface{}{
		"filter": filter,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", s.baseURL, s.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete vectors: status %d", resp.StatusCode)
	}

	return nil
}
