package admin

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/sse"
)

// SSEBroadcastRequest represents the request to broadcast an SSE event
type SSEBroadcastRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// SSEHandler handles SSE-related admin tasks
type SSEHandler struct {
	sseHub *sse.Hub
}

// NewSSEHandler creates a new admin SSE handler
func NewSSEHandler(sseHub *sse.Hub) *SSEHandler {
	return &SSEHandler{sseHub: sseHub}
}

// HandleBroadcast broadcasts a manual event to all SSE clients
// POST /api/v1/admin/sse/broadcast
func (h *SSEHandler) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	var req SSEBroadcastRequest
	if err := handler.DecodeAndValidateRequest(r, w, &req, "BroadcastSSE"); err != nil {
		return
	}

	if req.Type == "" {
		handler.RespondError(w, http.StatusBadRequest, "Event type is required")
		return
	}

	// Unmarshal the payload into an interface{} so hub can handle it
	var payload interface{}
	if len(req.Payload) > 0 {
		if err := json.Unmarshal(req.Payload, &payload); err != nil {
			handler.RespondError(w, http.StatusBadRequest, "Invalid payload JSON")
			return
		}
	}

	h.sseHub.Broadcast(req.Type, payload)

	handler.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Event broadcasted successfully",
		"type":    req.Type,
	})
}
