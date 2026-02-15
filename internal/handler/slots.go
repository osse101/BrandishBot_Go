package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/slots"
)

// SlotsHandler handles slots-related HTTP requests
type SlotsHandler struct {
	service        slots.Service
	progressionSvc progression.Service
}

// NewSlotsHandler creates a new slots handler
func NewSlotsHandler(service slots.Service, progressionSvc progression.Service) *SlotsHandler {
	return &SlotsHandler{
		service:        service,
		progressionSvc: progressionSvc,
	}
}

// SpinSlotsRequest represents a request to spin the slots
type SpinSlotsRequest struct {
	Platform   string `json:"platform" validate:"required"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required"`
	BetAmount  int    `json:"bet_amount" validate:"required,min=10,max=10000"`
}

// HandleSpinSlots processes a slots spin request
func (h *SlotsHandler) HandleSpinSlots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)

	// Check feature lock
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureSlots) {
		return
	}

	// Decode request
	var req SpinSlotsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Platform == "" || req.PlatformID == "" || req.Username == "" {
		respondError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	if req.BetAmount < slots.MinBetAmount || req.BetAmount > slots.MaxBetAmount {
		respondError(w, http.StatusBadRequest, "Bet amount must be between 10 and 10,000")
		return
	}

	// Spin slots
	result, err := h.service.SpinSlots(ctx, req.Platform, req.PlatformID, req.Username, req.BetAmount)
	if err != nil {
		log.Error("Failed to spin slots", "error", err, "username", req.Username)

		// Map errors to user-friendly messages
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "insufficient funds"):
			respondError(w, http.StatusBadRequest, errMsg)
		case strings.Contains(errMsg, "slots feature is not yet unlocked"):
			respondError(w, http.StatusForbidden, errMsg)
		case strings.Contains(errMsg, "minimum bet") || strings.Contains(errMsg, "maximum bet"):
			respondError(w, http.StatusBadRequest, errMsg)
		default:
			respondError(w, http.StatusInternalServerError, "Failed to process slots spin")
		}
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// SlotsResult is the response type (same as domain.SlotsResult but explicitly defined for API)
type SlotsResult = domain.SlotsResult
