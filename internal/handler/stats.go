package handler

import (
	"net/http"
	"strconv"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/stats"
)

// StatsHandler handles stats-related requests
type StatsHandler struct {
	service  stats.Service
	userRepo repository.User
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(service stats.Service, userRepo repository.User) *StatsHandler {
	return &StatsHandler{
		service:  service,
		userRepo: userRepo,
	}
}

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
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Event recorded successfully", "user_id", req.UserID, "event_type", req.EventType)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgEventRecordedSuccess})
	}
}

// HandleGetUserStats handles GET requests for user statistics
// Supports dual-mode: platform+platform_id (self-mode) or platform+username (target-mode)
// @Summary Get user stats
// @Description Get statistics for a specific user
// @Tags stats
// @Produce json
// @Param platform query string true "Platform (twitch, youtube, discord)"
// @Param platform_id query string false "Platform-specific user ID (self-mode)"
// @Param username query string false "Username (target-mode)"
// @Param period query string false "Period (daily, weekly, all_time)"
// @Success 200 {object} domain.StatsSummary
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /stats/user [get]
func (h *StatsHandler) HandleGetUserStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		platform, ok := GetQueryParam(r, w, "platform")
		if !ok {
			return
		}

		platformID := r.URL.Query().Get("platform_id")
		username := r.URL.Query().Get("username")

		// Require either platform_id or username
		if platformID == "" && username == "" {
			log.Warn("Missing required parameter: either platform_id or username required")
			respondError(w, http.StatusBadRequest, "Either platform_id or username is required")
			return
		}

		// Target-mode: resolve user by username
		var userID string
		if platformID == "" && username != "" {
			user, err := h.userRepo.GetUserByPlatformUsername(r.Context(), platform, username)
			if err != nil {
				log.Error("Failed to find user by username", "error", err, "platform", platform, "username", username)
				respondError(w, http.StatusNotFound, "User not found")
				return
			}
			userID = user.ID
			log.Debug("Resolved username to user_id", "username", username, "user_id", userID)
		} else {
			// Self-mode: get user by platform_id
			user, err := h.userRepo.GetUserByPlatformID(r.Context(), platform, platformID)
			if err != nil {
				log.Error("Failed to find user by platform_id", "error", err, "platform", platform, "platform_id", platformID)
				respondError(w, http.StatusNotFound, "User not found")
				return
			}
			userID = user.ID
		}

		period := GetOptionalQueryParam(r, "period", domain.PeriodDaily)

		log.Debug("Get user stats request", "user_id", userID, "period", period)

		summary, err := h.service.GetUserStats(r.Context(), userID, period)
		if err != nil {
			log.Error("Failed to get user stats", "error", err, "user_id", userID)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
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
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
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
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
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
