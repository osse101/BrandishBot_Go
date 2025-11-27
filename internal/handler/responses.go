package handler

import (
	"encoding/json"
	"net/http"
)

// Standard response types for consistent API responses

// SuccessResponse represents a simple successful operation message
type SuccessResponse struct {
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// DataResponse represents a response with data payload
type DataResponse struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data"`
}

// Helper functions for responding

// respondJSON sends a JSON response with the given status code and payload
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// respondError sends a JSON error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{Error: message})
}
