package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/compost"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type CompostHandler struct {
	service        compost.Service
	progressionSvc progression.Service
}

func NewCompostHandler(service compost.Service, progressionSvc progression.Service) *CompostHandler {
	return &CompostHandler{
		service:        service,
		progressionSvc: progressionSvc,
	}
}

// DepositRequest represents a compost deposit request
type DepositRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
	ItemKey    string `json:"item_key"`
	Quantity   int    `json:"quantity"`
}

// DepositResponse represents a compost deposit response
type DepositResponse struct {
	Message   string `json:"message"`
	DepositID string `json:"deposit_id"`
	ReadyAt   string `json:"ready_at"`
}

// HandleDeposit handles compost deposit requests
func (h *CompostHandler) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	// Check if compost feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureCompost) {
		return
	}

	var req DepositRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Deposit compost"); err != nil {
		return
	}

	deposit, err := h.service.Deposit(r.Context(), req.Platform, req.PlatformID, req.ItemKey, req.Quantity)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to create compost deposit", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	// Record engagement
	if err := h.progressionSvc.RecordEngagement(r.Context(), req.PlatformID, "compost_deposit", 1); err != nil {
		logger.FromContext(r.Context()).Error("Failed to record compost engagement", "error", err)
	}

	response := DepositResponse{
		Message:   "Items composting!",
		DepositID: deposit.ID.String(),
		ReadyAt:   deposit.ReadyAt.Format("2006-01-02 15:04:05"),
	}
	respondJSON(w, http.StatusCreated, response)
}

// HandleGetStatus handles compost status requests
func (h *CompostHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	username, ok := GetQueryParam(r, w, "username")
	if !ok {
		return
	}

	status, err := h.service.GetStatus(r.Context(), "twitch", username)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get compost status", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// HarvestRequest represents a compost harvest request
type HarvestRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
}

// HarvestResponse represents a compost harvest response
type HarvestResponse struct {
	Message     string `json:"message"`
	GemsAwarded int    `json:"gems_awarded"`
}

// HandleHarvest handles compost harvest requests
func (h *CompostHandler) HandleHarvest(w http.ResponseWriter, r *http.Request) {
	var req HarvestRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Harvest compost"); err != nil {
		return
	}

	gemsAwarded, err := h.service.Harvest(r.Context(), req.Platform, req.PlatformID)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to harvest compost", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	// Record engagement
	if err := h.progressionSvc.RecordEngagement(r.Context(), req.PlatformID, "compost_harvest", 1); err != nil {
		logger.FromContext(r.Context()).Error("Failed to record harvest engagement", "error", err)
	}

	response := HarvestResponse{
		Message:     "Compost harvested!",
		GemsAwarded: gemsAwarded,
	}
	respondJSON(w, http.StatusOK, response)
}
