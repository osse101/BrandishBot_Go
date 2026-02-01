package handler

import (
	"errors"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/harvest"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// HarvestRewardsRequest represents the request to harvest accumulated rewards
type HarvestRewardsRequest struct {
	Username   string `json:"username" validate:"required,max=100"`
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
}

// HarvestHandler handles harvest-related HTTP requests
type HarvestHandler struct {
	harvestSvc harvest.Service
}

// NewHarvestHandler creates a new harvest handler
func NewHarvestHandler(harvestSvc harvest.Service) *HarvestHandler {
	return &HarvestHandler{
		harvestSvc: harvestSvc,
	}
}

// Harvest handles the harvest endpoint
// @Summary Harvest accumulated rewards
// @Description Collect rewards that have accumulated since the last harvest
// @Tags harvest
// @Accept json
// @Produce json
// @Param request body HarvestRewardsRequest true "Harvest request"
// @Success 200 {object} domain.HarvestResponse "Harvest successful"
// @Failure 400 {object} ErrorResponse "Invalid request or harvest too soon"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /harvest [post]
func (h *HarvestHandler) Harvest(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	if r.Method != http.MethodPost {
		log.Warn("Method not allowed", "method", r.Method)
		http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	var req HarvestRewardsRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Harvest"); err != nil {
		return
	}

	log.Info("Harvest request received", "username", req.Username, "platform", req.Platform)

	// Call harvest service
	response, err := h.harvestSvc.Harvest(r.Context(), req.Platform, req.PlatformID, req.Username)
	if err != nil {
		log.Error("Harvest failed", "error", err, "username", req.Username, "platform", req.Platform)

		// Map specific errors to appropriate HTTP status codes
		if errors.Is(err, domain.ErrHarvestTooSoon) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "User not found")
			return
		}

		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	log.Info("Harvest successful",
		"username", req.Username,
		"items", len(response.ItemsGained),
		"hours", response.HoursSinceHarvest)

	respondJSON(w, http.StatusOK, response)
}
