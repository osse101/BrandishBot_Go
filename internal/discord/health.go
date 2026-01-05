package discord

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// HealthStatus represents the bot's health status
type HealthStatus struct {
	Status           string    `json:"status"`
	Uptime           string    `json:"uptime"`
	Connected        bool      `json:"connected"`
	CommandsReceived int64     `json:"commands_received"`
	LastCommandTime  time.Time `json:"last_command_time,omitempty"`
	APIReachable     bool      `json:"api_reachable"`
}

var (
	startTime        = time.Now()
	commandCounter   int64
	lastCommandTime  time.Time
)

// RecordCommand increments the command counter
func RecordCommand() {
	atomic.AddInt64(&commandCounter, 1)
	lastCommandTime = time.Now()
}

// HandleHealth returns the bot's health status
func (h *HTTPServer) HandleHealth(w http.ResponseWriter, r *http.Request) {
	connected := h.bot.Session != nil && h.bot.Session.DataReady

	// Quick ping to check API
	apiReachable := false
	if h.bot.Client != nil {
		resp, err := http.Get(h.bot.Client.BaseURL + "/healthz")
		if err == nil {
			apiReachable = resp.StatusCode == http.StatusOK
			resp.Body.Close()
		}
	}

	status := "healthy"
	if !connected || !apiReachable {
		status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	health := HealthStatus{
		Status:           status,
		Uptime:           time.Since(startTime).String(),
		Connected:        connected,
		CommandsReceived: atomic.LoadInt64(&commandCounter),
		LastCommandTime:  lastCommandTime,
		APIReachable:     apiReachable,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		// Can't write error response at this point since headers are sent
		// Silently ignore as this is a health check endpoint
		_ = err
	}
}
