package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/stats"
)

// RecordEventRequest represents a request to record a custom event
type RecordEventRequest struct {
	UserID    string                 `json:"user_id" validate:"required,max=100,excludesall=\x00\n\r\t"`
	EventType string                 `json:"event_type" validate:"required,max=50"`
	EventData map[string]interface{} `json:"event_data,omitempty"`
}

// HandleRecordEvent handles POST requests to record custom events
// @Summary Record event
// @Description Record a custom user event
// @Tags stats
// @Accept json
// @Produce json
// @Param request body RecordEventRequest true "Event details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /stats/event [post]
func HandleRecordEvent(svc stats.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req RecordEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode record event request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Record event request", "user_id", req.UserID, "event_type", req.EventType)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if err := svc.RecordUserEvent(r.Context(), req.UserID, domain.EventType(req.EventType), req.EventData); err != nil {
			log.Error("Failed to record event", "error", err, "user_id", req.UserID, "event_type", req.EventType)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Event recorded successfully", "user_id", req.UserID, "event_type", req.EventType)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Event recorded successfully"})
	}
}

// HandleGetUserStats handles GET requests for user statistics
// @Summary Get user stats
// @Description Get statistics for a specific user
// @Tags stats
// @Produce json
// @Param user_id query string true "User ID"
// @Param period query string false "Period (daily, weekly, all_time)"
// @Success 200 {object} domain.StatsSummary
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /stats/user [get]
func HandleGetUserStats(svc stats.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			log.Warn("Missing user_id query parameter")
			http.Error(w, "Missing user_id query parameter", http.StatusBadRequest)
			return
		}

		period := r.URL.Query().Get("period")
		if period == "" {
			period = domain.PeriodDaily // Default to daily
		}

		log.Debug("Get user stats request", "user_id", userID, "period", period)

		summary, err := svc.GetUserStats(r.Context(), userID, period)
		if err != nil {
			log.Error("Failed to get user stats", "error", err, "user_id", userID)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("User stats retrieved", "user_id", userID, "period", period, "total_events", summary.TotalEvents)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, summary)
	}
}

// HandleGetSystemStats handles GET requests for system-wide statistics
// @Summary Get system stats
// @Description Get system-wide statistics
// @Tags stats
// @Produce json
// @Param period query string false "Period (daily, weekly, all_time)"
// @Success 200 {object} domain.StatsSummary
// @Failure 500 {object} ErrorResponse
// @Router /stats/system [get]
func HandleGetSystemStats(svc stats.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		period := r.URL.Query().Get("period")
		if period == "" {
			period = domain.PeriodDaily // Default to daily
		}

		log.Debug("Get system stats request", "period", period)

		summary, err := svc.GetSystemStats(r.Context(), period)
		if err != nil {
			log.Error("Failed to get system stats", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("System stats retrieved", "period", period, "total_events", summary.TotalEvents)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, summary)
	}
}

// HandleGetLeaderboard handles GET requests for leaderboards
// @Summary Get leaderboard
// @Description Get leaderboard for a specific event type
// @Tags stats
// @Produce json
// @Param event_type query string true "Event Type"
// @Param period query string false "Period (daily, weekly, all_time)"
// @Param limit query int false "Limit (default 10)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /stats/leaderboard [get]
func HandleGetLeaderboard(svc stats.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		eventType := r.URL.Query().Get("event_type")
		if eventType == "" {
			log.Warn("Missing event_type query parameter")
			http.Error(w, "Missing event_type query parameter", http.StatusBadRequest)
			return
		}

		period := r.URL.Query().Get("period")
		if period == "" {
			period = domain.PeriodDaily // Default to daily
		}

		limitStr := r.URL.Query().Get("limit")
		limit := 10 // Default
		if limitStr != "" {
			var err error
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				log.Warn("Invalid limit parameter", "limit", limitStr)
				http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
				return
			}
		}

		log.Debug("Get leaderboard request", "event_type", eventType, "period", period, "limit", limit)

		entries, err := svc.GetLeaderboard(r.Context(), domain.EventType(eventType), period, limit)
		if err != nil {
			log.Error("Failed to get leaderboard", "error", err, "event_type", eventType)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Leaderboard retrieved", "event_type", eventType, "period", period, "entries", len(entries))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"event_type": eventType,
			"period":     period,
			"entries":    entries,
		})
	}
}
