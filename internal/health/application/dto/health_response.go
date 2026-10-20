package dto

import "time"

type HealthResponse struct {
	Timestamp time.Time `json:"timestamp" example:"2024-01-15T10.30:00Z"`
	Service   string    `json:"service" example:"microservice-seed-go"`
	Database  string    `json:"database"`
}

// Constructor
func NewHealthResponse(dbStatus string) *HealthResponse {
	return &HealthResponse{
		Timestamp: time.Now(),
		Service:   "microservice",
		Database:  dbStatus,
	}
}
