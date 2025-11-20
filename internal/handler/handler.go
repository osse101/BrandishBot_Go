package handler

import (
	"encoding/json"
	"net/http"
)

// ExecuteRequest represents the expected body of the execute request
type ExecuteRequest struct {
	Command string `json:"command"`
}

// ExecuteResponse represents the response body
type ExecuteResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ExecuteHandler handles the /execute endpoint
func ExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual command execution logic here
	resp := ExecuteResponse{
		Status:  "success",
		Message: "Command received: " + req.Command,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
