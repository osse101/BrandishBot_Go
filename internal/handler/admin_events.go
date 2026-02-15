package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/eventlog"
)

// AdminEventsHandler handles admin event log queries
type AdminEventsHandler struct {
	eventlogService eventlog.Service
}

// NewAdminEventsHandler creates a new admin events handler
func NewAdminEventsHandler(eventlogService eventlog.Service) *AdminEventsHandler {
	return &AdminEventsHandler{eventlogService: eventlogService}
}

// EventsResponse contains event log query results
type EventsResponse struct {
	Events []EventLogEntry `json:"events"`
}

// EventLogEntry represents a single event log entry
type EventLogEntry struct {
	ID        int64       `json:"id"`
	EventType string      `json:"event_type"`
	UserID    *string     `json:"user_id,omitempty"`
	Payload   interface{} `json:"payload"`
	Metadata  interface{} `json:"metadata,omitempty"`
	CreatedAt string      `json:"created_at"`
}

// HandleGetEvents retrieves events based on query parameters
// GET /api/v1/admin/events?user_id=X&event_type=Y&since=Z&limit=N
func (h *AdminEventsHandler) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := eventlog.EventFilter{
		Limit: 50, // default limit
	}

	if userID := query.Get("user_id"); userID != "" {
		filter.UserID = &userID
	}

	if eventType := query.Get("event_type"); eventType != "" {
		filter.EventType = &eventType
	}

	if sinceStr := query.Get("since"); sinceStr != "" {
		since, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid 'since' timestamp format (use RFC3339)")
			return
		}
		filter.Since = &since
	}

	if untilStr := query.Get("until"); untilStr != "" {
		until, err := time.Parse(time.RFC3339, untilStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid 'until' timestamp format (use RFC3339)")
			return
		}
		filter.Until = &until
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			respondError(w, http.StatusBadRequest, "Invalid 'limit' (must be 1-1000)")
			return
		}
		filter.Limit = limit
	}

	events, err := h.eventlogService.GetEvents(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retrieve events")
		return
	}

	// Convert to response format
	entries := make([]EventLogEntry, len(events))
	for i, evt := range events {
		entries[i] = EventLogEntry{
			ID:        evt.ID,
			EventType: evt.EventType,
			UserID:    evt.UserID,
			Payload:   evt.Payload,
			Metadata:  evt.Metadata,
			CreatedAt: evt.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	respondJSON(w, http.StatusOK, EventsResponse{Events: entries})
}
