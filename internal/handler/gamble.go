package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type GambleHandler struct {
	service        gamble.Service
	userSvc        user.ManagementService
	progressionSvc progression.Service
	eventBus       event.Bus
}

func NewGambleHandler(service gamble.Service, userSvc user.ManagementService, progressionSvc progression.Service, eventBus event.Bus) *GambleHandler {
	return &GambleHandler{
		service:        service,
		userSvc:        userSvc,
		progressionSvc: progressionSvc,
		eventBus:       eventBus,
	}
}

type StartGambleRequest struct {
	Platform   string              `json:"platform" validate:"required,platform"`
	PlatformID string              `json:"platform_id" validate:"required"`
	Username   string              `json:"username" validate:"required"`
	Bets       []domain.LootboxBet `json:"bets" validate:"required,min=1,dive"`
}

type StartGambleResponse struct {
	Message  string `json:"message"`
	GambleID string `json:"gamble_id"`
}

func (h *GambleHandler) HandleStartGamble(w http.ResponseWriter, r *http.Request) {
	// Check if gamble feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureGamble) {
		return
	}

	var req StartGambleRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Start gamble"); err != nil {
		return
	}

	gamble, err := h.service.StartGamble(r.Context(), req.Platform, req.PlatformID, req.Username, req.Bets)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to start gamble", "error", err)
		statusCode, userMsg := MapServiceErrorToUserMessage(err)
		RespondError(w, statusCode, userMsg)
		return
	}

	// Track engagement for gamble start
	if userID, err := h.userSvc.GetUserIDByPlatformID(r.Context(), req.Platform, req.PlatformID); err == nil && userID != "" {
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), userID),
			h.eventBus,
			domain.MetricTypeGambleStarted,
			1,
		)
	}

	response := StartGambleResponse{
		Message:  "Gamble started!",
		GambleID: gamble.ID.String(),
	}
	RespondJSON(w, http.StatusCreated, response)
}

type JoinGambleRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required"`
}

func (h *GambleHandler) HandleJoinGamble(w http.ResponseWriter, r *http.Request) {
	var req JoinGambleRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Join gamble"); err != nil {
		return
	}

	if err := h.service.JoinActiveGamble(r.Context(), req.Platform, req.PlatformID, req.Username); err != nil {
		logger.FromContext(r.Context()).Debug("Failed to join gamble", "error", err)
		statusCode, userMsg := MapServiceErrorToUserMessage(err)
		RespondError(w, statusCode, userMsg)
		return
	}

	// Track engagement for gamble join
	if userID, err := h.userSvc.GetUserIDByPlatformID(r.Context(), req.Platform, req.PlatformID); err == nil && userID != "" {
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), userID),
			h.eventBus,
			domain.MetricTypeGambleJoined,
			1,
		)
	}

	RespondJSON(w, http.StatusOK, map[string]string{"message": MsgJoinedGambleSuccess})
}

func (h *GambleHandler) HandleGetGamble(w http.ResponseWriter, r *http.Request) {
	gambleIDStr, ok := GetQueryParam(r, w, "id")
	if !ok {
		return
	}
	gambleID, err := uuid.Parse(gambleIDStr)
	if err != nil {
		RespondError(w, http.StatusBadRequest, ErrMsgInvalidGambleID)
		return
	}

	gamble, err := h.service.GetGamble(r.Context(), gambleID)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get gamble", "error", err)
		statusCode, userMsg := MapServiceErrorToUserMessage(err)
		RespondError(w, statusCode, userMsg)
		return
	}
	if gamble == nil {
		RespondError(w, http.StatusNotFound, ErrMsgGambleNotFoundHTTP)
		return
	}

	RespondJSON(w, http.StatusOK, gamble)
}

func (h *GambleHandler) HandleGetActiveGamble(w http.ResponseWriter, r *http.Request) {
	gamble, err := h.service.GetActiveGamble(r.Context())
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get active gamble", "error", err)
		statusCode, userMsg := MapServiceErrorToUserMessage(err)
		RespondError(w, statusCode, userMsg)
		return
	}

	if gamble == nil {
		RespondJSON(w, http.StatusOK, map[string]bool{"active": false})
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"active": true,
		"gamble": gamble,
	})
}
