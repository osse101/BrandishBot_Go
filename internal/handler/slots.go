package handler

import (
	"errors"
	"net/http"

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

	// Decode and validate request
	var req SpinSlotsRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Spin slots"); err != nil {
		return
	}

	// Spin slots
	result, err := h.service.SpinSlots(ctx, req.Platform, req.PlatformID, req.Username, req.BetAmount)
	if err != nil {
		log.Error("Failed to spin slots", "error", err, "username", req.Username)

		// Map errors to user-friendly messages
		switch {
		case errors.Is(err, domain.ErrInsufficientFunds):
			RespondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFeatureLocked):
			RespondError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrInvalidQuantity):
			RespondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrOnCooldown):
			RespondError(w, http.StatusBadRequest, err.Error())
		default:
			//RespondError(w, http.StatusInternalServerError, "Failed to process slots spin")
			RespondError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	RespondJSON(w, http.StatusOK, result)
}

// SlotsResult is the response type (same as domain.SlotsResult but explicitly defined for API)
type SlotsResult = domain.SlotsResult
