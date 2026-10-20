package dto

import "time"

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error     string    `json:"error" example:"Service unhealthy"`
	Timestamp time.Time `json:"timestamp" example:"2024-01-15T10:30:00Z"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string) *ErrorResponse {
	return &ErrorResponse{
		Error:     message,
		Timestamp: time.Now(),
	}
}
