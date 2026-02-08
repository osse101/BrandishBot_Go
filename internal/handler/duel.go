package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/duel"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type DuelHandler struct {
	service        duel.Service
	progressionSvc progression.Service
}

func NewDuelHandler(service duel.Service, progressionSvc progression.Service) *DuelHandler {
	return &DuelHandler{
		service:        service,
		progressionSvc: progressionSvc,
	}
}

// ChallengeRequest represents a duel challenge request
type ChallengeRequest struct {
	Platform         string            `json:"platform"`
	PlatformID       string            `json:"platform_id"`
	OpponentUsername string            `json:"opponent_username"`
	Stakes           domain.DuelStakes `json:"stakes"`
}

// ChallengeResponse represents a duel challenge response
type ChallengeResponse struct {
	Message   string `json:"message"`
	DuelID    string `json:"duel_id"`
	ExpiresAt string `json:"expires_at"`
}

// HandleChallenge handles duel challenge requests
func (h *DuelHandler) HandleChallenge(w http.ResponseWriter, r *http.Request) {
	handleFeatureAction(w, r, h.progressionSvc, progression.FeatureDuel, "Challenge duel",
		func(ctx context.Context, req ChallengeRequest) (*domain.Duel, error) {
			duel, err := h.service.Challenge(ctx, req.Platform, req.PlatformID, req.OpponentUsername, req.Stakes)
			if err == nil {
				recordEngagement(r, h.progressionSvc, req.OpponentUsername, "duel_challenged", 1)
			}
			return duel, err
		},
		h.formatChallengeResponse,
	)
}

func (h *DuelHandler) formatChallengeResponse(d *domain.Duel) interface{} {
	return ChallengeResponse{
		Message:   "Duel challenge sent!",
		DuelID:    d.ID.String(),
		ExpiresAt: d.ExpiresAt.Format("2006-01-02 15:04:05"),
	}
}

// AcceptDuelRequest represents a duel accept request
type AcceptDuelRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
}

// AcceptDuelResponse represents a duel accept response
type AcceptDuelResponse struct {
	Message string             `json:"message"`
	Result  *domain.DuelResult `json:"result"`
}

// HandleAccept handles duel accept requests
func (h *DuelHandler) HandleAccept(w http.ResponseWriter, r *http.Request) {
	duelIDStr := chi.URLParam(r, "id")
	if duelIDStr == "" {
		respondError(w, http.StatusBadRequest, "Missing duel ID")
		return
	}
	duelID, err := uuid.Parse(duelIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid duel ID")
		return
	}

	var req AcceptDuelRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Accept duel"); err != nil {
		return
	}

	result, err := h.service.Accept(r.Context(), req.Platform, req.PlatformID, duelID)
	if err != nil {
		respondServiceError(w, r, "Failed to accept duel", err)
		return
	}

	response := AcceptDuelResponse{
		Message: "Duel completed!",
		Result:  result,
	}
	respondJSON(w, http.StatusOK, response)
}

// DeclineDuelRequest represents a duel decline request
type DeclineDuelRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
}

// HandleDecline handles duel decline requests
func (h *DuelHandler) HandleDecline(w http.ResponseWriter, r *http.Request) {
	duelIDStr := chi.URLParam(r, "id")
	if duelIDStr == "" {
		respondError(w, http.StatusBadRequest, "Missing duel ID")
		return
	}
	duelID, err := uuid.Parse(duelIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid duel ID")
		return
	}

	var req DeclineDuelRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Decline duel"); err != nil {
		return
	}

	if err := h.service.Decline(r.Context(), req.Platform, req.PlatformID, duelID); err != nil {
		respondServiceError(w, r, "Failed to decline duel", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Duel declined"})
}

// HandleGetPending handles requests to get pending duels
func (h *DuelHandler) HandleGetPending(w http.ResponseWriter, r *http.Request) {
	username, ok := GetQueryParam(r, w, "username")
	if !ok {
		return
	}

	duels, err := h.service.GetPendingDuels(r.Context(), "twitch", username)
	if err != nil {
		respondServiceError(w, r, "Failed to get pending duels", err)
		return
	}

	respondJSON(w, http.StatusOK, duels)
}

// HandleGetDuel handles requests to get a specific duel
func (h *DuelHandler) HandleGetDuel(w http.ResponseWriter, r *http.Request) {
	duelIDStr := chi.URLParam(r, "id")
	if duelIDStr == "" {
		respondError(w, http.StatusBadRequest, "Missing duel ID")
		return
	}
	duelID, err := uuid.Parse(duelIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid duel ID")
		return
	}

	duel, err := h.service.GetDuel(r.Context(), duelID)
	if err != nil {
		respondServiceError(w, r, "Failed to get duel", err)
		return
	}

	respondJSON(w, http.StatusOK, duel)
}
