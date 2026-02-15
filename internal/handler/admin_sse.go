package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/sse"
)

// AdminSSEBroadcastRequest represents the request to broadcast an SSE event
type AdminSSEBroadcastRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// AdminSSEHandler handles SSE-related admin tasks
type AdminSSEHandler struct {
	sseHub *sse.Hub
}

// NewAdminSSEHandler creates a new admin SSE handler
func NewAdminSSEHandler(sseHub *sse.Hub) *AdminSSEHandler {
	return &AdminSSEHandler{sseHub: sseHub}
}

// HandleBroadcast broadcasts a manual event to all SSE clients
// POST /api/v1/admin/sse/broadcast
func (h *AdminSSEHandler) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	var req AdminSSEBroadcastRequest
	if err := DecodeAndValidateRequest(r, w, &req, "BroadcastSSE"); err != nil {
		return
	}

	if req.Type == "" {
		respondError(w, http.StatusBadRequest, "Event type is required")
		return
	}

	// Unmarshal the payload into an interface{} so hub can handle it
	var payload interface{}
	if len(req.Payload) > 0 {
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid payload JSON")
			return
		}
	}

	h.sseHub.Broadcast(req.Type, payload)

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "Event broadcasted successfully",
		"type":    req.Type,
	})
}
