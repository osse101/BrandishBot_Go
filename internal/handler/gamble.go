package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type GambleHandler struct {
	service        gamble.Service
	progressionSvc progression.Service
}

func NewGambleHandler(service gamble.Service, progressionSvc progression.Service) *GambleHandler {
	return &GambleHandler{
		service:        service,
		progressionSvc: progressionSvc,
	}
}

type StartGambleRequest struct {
	Platform   string              `json:"platform"`
	PlatformID string              `json:"platform_id"`
	Username   string              `json:"username"`
	Bets       []domain.LootboxBet `json:"bets"`
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

	respondJSON(w, http.StatusCreated, gamble)
}

type JoinGambleRequest struct {
	Platform   string              `json:"platform"`
	PlatformID string              `json:"platform_id"`
	Username   string              `json:"username"`
	Bets       []domain.LootboxBet `json:"bets"`
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

	if err := h.service.JoinGamble(r.Context(), gambleID, req.Platform, req.PlatformID, req.Username, req.Bets); err != nil {
		logger.FromContext(r.Context()).Error("Failed to join gamble", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
		return
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
