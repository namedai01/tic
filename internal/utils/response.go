package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta represents metadata for paginated responses
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// SuccessResponse creates a successful API response
func SuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
	}
}

// ErrorResponse creates an error API response
func ErrorResponse(code int, message string, details ...string) APIResponse {
	apiError := &APIError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		apiError.Details = details[0]
	}
	return APIResponse{
		Success: false,
		Error:   apiError,
	}
}

// PaginatedResponse creates a paginated API response
func PaginatedResponse(data interface{}, page, limit, total int) APIResponse {
	totalPages := (total + limit - 1) / limit
	return APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// SendJSON sends a JSON response
func SendJSON(c *fiber.Ctx, status int, response APIResponse) error {
	return c.Status(status).JSON(response)
}

// SendSuccess sends a successful JSON response
func SendSuccess(c *fiber.Ctx, data interface{}) error {
	return SendJSON(c, http.StatusOK, SuccessResponse(data))
}

// SendError sends an error JSON response
func SendError(c *fiber.Ctx, status int, message string, details ...string) error {
	return SendJSON(c, status, ErrorResponse(status, message, details...))
}

// SendPaginated sends a paginated JSON response
func SendPaginated(c *fiber.Ctx, data interface{}, page, limit, total int) error {
	return SendJSON(c, http.StatusOK, PaginatedResponse(data, page, limit, total))
}

// ParsePagination parses pagination parameters from query string
func ParsePagination(c *fiber.Ctx) (page, limit int) {
	page = 1
	limit = 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	return page, limit
}

// BindAndValidate binds request body to struct and validates it
func BindAndValidate(c *fiber.Ctx, obj interface{}) error {
	if err := c.BodyParser(obj); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	// Here you would typically add validation using a validator library
	// For now, we'll skip validation
	return nil
}

// ToJSON converts a struct to JSON string
func ToJSON(obj interface{}) string {
	data, err := json.Marshal(obj)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// FromJSON parses JSON string to struct
func FromJSON(jsonStr string, obj interface{}) error {
	return json.Unmarshal([]byte(jsonStr), obj)
}
