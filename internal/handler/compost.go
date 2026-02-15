package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/compost"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

// CompostHandler handles compost HTTP endpoints
type CompostHandler struct {
	service        compost.Service
	progressionSvc progression.Service
}

// NewCompostHandler creates a new compost handler
func NewCompostHandler(service compost.Service, progressionSvc progression.Service) *CompostHandler {
	return &CompostHandler{
		service:        service,
		progressionSvc: progressionSvc,
	}
}

// CompostDepositRequest is the request body for depositing items
type CompostDepositRequest struct {
	Platform   string                `json:"platform" validate:"required"`
	PlatformID string                `json:"platform_id" validate:"required"`
	Items      []compost.DepositItem `json:"items" validate:"required,min=1"`
}

// CompostHarvestRequest is the request body for harvesting
type CompostHarvestRequest struct {
	Platform   string `json:"platform" validate:"required"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required"`
}

// CompostDepositResponse is the response for a successful deposit
type CompostDepositResponse struct {
	Message   string `json:"message"`
	Status    string `json:"status"`
	ItemCount int    `json:"item_count"`
	Capacity  int    `json:"capacity"`
	ReadyAt   string `json:"ready_at,omitempty"`
}

// CompostHarvestResponse is the response for harvest (either status or output)
type CompostHarvestResponse struct {
	Message   string         `json:"message"`
	Harvested bool           `json:"harvested"`
	Items     map[string]int `json:"items,omitempty"`
	TimeLeft  string         `json:"time_left,omitempty"`
	Status    string         `json:"status,omitempty"`
}

// HandleDeposit handles compost deposit requests
func (h *CompostHandler) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	handleFeatureAction(w, r, h.progressionSvc, progression.FeatureCompost, "Compost deposit",
		func(ctx context.Context, req CompostDepositRequest) (*domain.CompostBin, error) {
			result, err := h.service.Deposit(ctx, req.Platform, req.PlatformID, req.Items)
			if err == nil {
				recordEngagement(r, h.progressionSvc, req.PlatformID, "compost_deposit", 1)
			}
			return result, err
		},
		func(bin *domain.CompostBin) interface{} {
			resp := CompostDepositResponse{
				Message:   MsgCompostDepositSuccess,
				Status:    string(bin.Status),
				ItemCount: bin.ItemCount,
				Capacity:  bin.Capacity,
			}
			if bin.ReadyAt != nil {
				resp.ReadyAt = bin.ReadyAt.Format(time.RFC3339)
			}
			return resp
		},
	)
}

// HandleHarvest handles compost harvest requests
func (h *CompostHandler) HandleHarvest(w http.ResponseWriter, r *http.Request) {
	handleFeatureAction(w, r, h.progressionSvc, progression.FeatureCompost, "Compost harvest",
		func(ctx context.Context, req CompostHarvestRequest) (*domain.HarvestResult, error) {
			result, err := h.service.Harvest(ctx, req.Platform, req.PlatformID, req.Username)
			if err == nil && result.Harvested {
				recordEngagement(r, h.progressionSvc, req.PlatformID, "compost_harvest", 5)
			}
			return result, err
		},
		func(result *domain.HarvestResult) interface{} {
			if result.Harvested {
				return CompostHarvestResponse{
					Message:   result.Output.Message,
					Harvested: true,
					Items:     result.Output.Items,
				}
			}
			resp := CompostHarvestResponse{
				Harvested: false,
				Status:    string(result.Status.Status),
			}
			if result.Status.TimeLeft != "" {
				resp.Message = result.Status.TimeLeft
				resp.TimeLeft = result.Status.TimeLeft
			} else {
				resp.Message = MsgCompostBinEmpty
			}
			return resp
		},
	)
}

// HandleStatus is a GET convenience endpoint for checking bin status
func (h *CompostHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	platform, ok := GetQueryParam(r, w, "platform")
	if !ok {
		return
	}
	platformID, ok := GetQueryParam(r, w, "platform_id")
	if !ok {
		return
	}
	username := GetOptionalQueryParam(r, "username", "")

	result, err := h.service.Harvest(r.Context(), platform, platformID, username)
	if err != nil {
		respondServiceError(w, r, "Compost status", err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}
