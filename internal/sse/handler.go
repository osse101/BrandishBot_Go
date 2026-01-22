package sse

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Handler returns an HTTP handler for SSE connections
func Handler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Check for flusher support
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		// Parse event type filters from query param
		var eventTypes []string
		filterParam := r.URL.Query().Get("types")
		if filterParam != "" {
			eventTypes = strings.Split(filterParam, ",")
		}

		// Register client
		client := hub.Register(eventTypes)
		slog.Info(LogMsgClientConnected,
			"client_id", client.ID,
			"filters", eventTypes,
			"total_clients", hub.ClientCount())

		// Ensure cleanup on disconnect
		defer func() {
			hub.Unregister(client.ID)
			slog.Info(LogMsgClientDisconnected,
				"client_id", client.ID,
				"total_clients", hub.ClientCount())
		}()

		// Send initial connection event
		connectEvent := SSEEvent{
			ID:        client.ID,
			Type:      "connected",
			Timestamp: time.Now().Unix(),
			Payload: map[string]interface{}{
				"client_id": client.ID,
				"filters":   eventTypes,
			},
		}
		if msg, err := FormatSSEMessage(connectEvent); err == nil {
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		}

		// Keepalive ticker
		ticker := time.NewTicker(KeepaliveInterval)
		defer ticker.Stop()

		// Event loop
		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				// Client disconnected
				return

			case event, ok := <-client.EventChannel:
				if !ok {
					// Channel closed, hub is shutting down
					return
				}

				msg, err := FormatSSEMessage(event)
				if err != nil {
					slog.Error(LogMsgWriteError, "error", err)
					continue
				}

				if _, err := w.Write(msg); err != nil {
					slog.Warn(LogMsgWriteError, "error", err)
					return
				}
				flusher.Flush()

			case <-ticker.C:
				// Send keepalive ping
				keepalive := SSEEvent{
					ID:        "",
					Type:      EventTypeKeepalive,
					Timestamp: time.Now().Unix(),
					Payload:   nil,
				}
				msg, _ := FormatSSEMessage(keepalive)
				if _, err := w.Write(msg); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}
