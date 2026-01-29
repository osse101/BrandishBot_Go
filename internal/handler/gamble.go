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
)

type GambleHandler struct {
	service        gamble.Service
	progressionSvc progression.Service
	eventBus       event.Bus
}

func NewGambleHandler(service gamble.Service, progressionSvc progression.Service, eventBus event.Bus) *GambleHandler {
	return &GambleHandler{
		service:        service,
		progressionSvc: progressionSvc,
		eventBus:       eventBus,
	}
}

type StartGambleRequest struct {
	Platform   string              `json:"platform"`
	PlatformID string              `json:"platform_id"`
	Username   string              `json:"username"`
	Bets       []domain.LootboxBet `json:"bets"`
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
		statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
		return
	}

	log := logger.FromContext(r.Context())

	// Track engagement for gamble start
	middleware.TrackEngagementFromContext(
		middleware.WithUserID(r.Context(), req.Username),
		h.eventBus,
		"gamble_started",
		1,
	)

	// Record contribution for gamble start (higher value)
	if err := h.progressionSvc.RecordEngagement(r.Context(), req.Username, "gamble_started", 3); err != nil {
		log.Error("Failed to record gamble start engagement", "error", err)
		// Don't fail the request
	}

	response := StartGambleResponse{
		Message:  "Gamble started! Others can join using the gamble ID.",
		GambleID: gamble.ID.String(),
	}
	respondJSON(w, http.StatusCreated, response)
}

type JoinGambleRequest struct {
	Platform   string              `json:"platform"`
	PlatformID string              `json:"platform_id"`
	Username   string              `json:"username"`
}

func (h *GambleHandler) HandleJoinGamble(w http.ResponseWriter, r *http.Request) {
	gambleIDStr, ok := GetQueryParam(r, w, "id")
	if !ok {
		return
	}
	gambleID, err := uuid.Parse(gambleIDStr)
	if err != nil {
		http.Error(w, ErrMsgInvalidGambleID, http.StatusBadRequest)
		return
	}

	var req JoinGambleRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Join gamble"); err != nil {
		return
	}

	if err := h.service.JoinGamble(r.Context(), gambleID, req.Platform, req.PlatformID, req.Username); err != nil {
		logger.FromContext(r.Context()).Error("Failed to join gamble", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
		return
	}

	log := logger.FromContext(r.Context())

	// Track engagement for gamble join
	middleware.TrackEngagementFromContext(
		middleware.WithUserID(r.Context(), req.Username),
		h.eventBus,
		"gamble_joined",
		1,
	)

	// Record contribution for gamble join
	if err := h.progressionSvc.RecordEngagement(r.Context(), req.Username, "gamble_joined", 2); err != nil {
		log.Error("Failed to record gamble join engagement", "error", err)
		// Don't fail the request
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": MsgJoinedGambleSuccess})
}

func (h *GambleHandler) HandleGetGamble(w http.ResponseWriter, r *http.Request) {
	gambleIDStr, ok := GetQueryParam(r, w, "id")
	if !ok {
		return
	}
	gambleID, err := uuid.Parse(gambleIDStr)
	if err != nil {
		http.Error(w, ErrMsgInvalidGambleID, http.StatusBadRequest)
		return
	}

	gamble, err := h.service.GetGamble(r.Context(), gambleID)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get gamble", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
		return
	}
	if gamble == nil {
		http.Error(w, ErrMsgGambleNotFoundHTTP, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, gamble)
}

func (h *GambleHandler) HandleGetActiveGamble(w http.ResponseWriter, r *http.Request) {
	gamble, err := h.service.GetActiveGamble(r.Context())
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get active gamble", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
		return
	}

	if gamble == nil {
		respondJSON(w, http.StatusOK, map[string]bool{"active": false})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"active": true,
		"gamble": gamble,
	})
}
