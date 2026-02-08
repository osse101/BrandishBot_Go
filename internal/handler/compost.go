package handler

import (
	"context"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/compost"
	"github.com/osse101/BrandishBot_Go/internal/domain"
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
	handleFeatureAction(w, r, h.progressionSvc, progression.FeatureCompost, "Deposit compost",
		func(ctx context.Context, req DepositRequest) (*domain.CompostDeposit, error) {
			result, err := h.service.Deposit(ctx, req.Platform, req.PlatformID, req.ItemKey, req.Quantity)
			if err == nil {
				recordEngagement(r, h.progressionSvc, req.PlatformID, "compost_deposit", 1)
			}
			return result, err
		},
		h.formatDepositResponse,
	)
}

func (h *CompostHandler) formatDepositResponse(res *domain.CompostDeposit) interface{} {
	return DepositResponse{
		Message:   "Items composting!",
		DepositID: res.ID.String(),
		ReadyAt:   res.ReadyAt.Format("2006-01-02 15:04:05"),
	}
}

// HandleGetStatus handles compost status requests
func (h *CompostHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	username, ok := GetQueryParam(r, w, "username")
	if !ok {
		return
	}

	status, err := h.service.GetStatus(r.Context(), "twitch", username)
	if err != nil {
		respondServiceError(w, r, "Failed to get compost status", err)
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
	handleFeatureAction(w, r, h.progressionSvc, progression.FeatureCompost, "Harvest compost",
		func(ctx context.Context, req HarvestRequest) (int, error) {
			gemsAwarded, err := h.service.Harvest(ctx, req.Platform, req.PlatformID)
			if err == nil {
				recordEngagement(r, h.progressionSvc, req.PlatformID, "compost_harvest", 1)
			}
			return gemsAwarded, err
		},
		func(gemsAwarded int) interface{} {
			return HarvestResponse{
				Message:     "Compost harvested!",
				GemsAwarded: gemsAwarded,
			}
		},
	)
}
