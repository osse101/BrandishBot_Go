package handler

import (
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
		var req RecordEventRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Record event"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		if err := svc.RecordUserEvent(r.Context(), req.UserID, domain.EventType(req.EventType), req.EventData); err != nil {
			log.Error("Failed to record event", "error", err, "user_id", req.UserID, "event_type", req.EventType)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Event recorded successfully", "user_id", req.UserID, "event_type", req.EventType)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgEventRecordedSuccess})
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

		userID, ok := GetQueryParam(r, w, "user_id")
		if !ok {
			return
		}

		period := GetOptionalQueryParam(r, "period", domain.PeriodDaily)

		log.Debug("Get user stats request", "user_id", userID, "period", period)

		summary, err := svc.GetUserStats(r.Context(), userID, period)
		if err != nil {
			log.Error("Failed to get user stats", "error", err, "user_id", userID)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("User stats retrieved", "user_id", userID, "period", period, "total_events", summary.TotalEvents)

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

		period := GetOptionalQueryParam(r, "period", domain.PeriodDaily)

		log.Debug("Get system stats request", "period", period)

		summary, err := svc.GetSystemStats(r.Context(), period)
		if err != nil {
			log.Error("Failed to get system stats", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("System stats retrieved", "period", period, "total_events", summary.TotalEvents)

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

		eventType, ok := GetQueryParam(r, w, "event_type")
		if !ok {
			return
		}

		period := GetOptionalQueryParam(r, "period", domain.PeriodDaily)

		limitStr := r.URL.Query().Get("limit")
		limit := 10 // Default
		if limitStr != "" {
			var err error
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				log.Warn("Invalid limit parameter", "limit", limitStr)
				http.Error(w, ErrMsgInvalidLimit, http.StatusBadRequest)
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

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"event_type": eventType,
			"period":     period,
			"entries":    entries,
		})
	}
}
