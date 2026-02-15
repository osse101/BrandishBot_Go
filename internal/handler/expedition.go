package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/expedition"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type ExpeditionHandler struct {
	service        expedition.Service
	progressionSvc progression.Service
}

func NewExpeditionHandler(service expedition.Service, progressionSvc progression.Service) *ExpeditionHandler {
	return &ExpeditionHandler{
		service:        service,
		progressionSvc: progressionSvc,
	}
}

// StartExpeditionRequest represents an expedition start request
type StartExpeditionRequest struct {
	Platform       string `json:"platform"`
	PlatformID     string `json:"platform_id"`
	Username       string `json:"username"`
	ExpeditionType string `json:"expedition_type"`
}

// StartExpeditionResponse represents an expedition start response
type StartExpeditionResponse struct {
	Message      string `json:"message"`
	ExpeditionID string `json:"expedition_id"`
	JoinDeadline string `json:"join_deadline"`
}

// HandleStart handles expedition start requests
func (h *ExpeditionHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	handleFeatureAction(w, r, h.progressionSvc, progression.FeatureExpedition, "Start expedition",
		func(ctx context.Context, req StartExpeditionRequest) (*domain.Expedition, error) {
			exp, err := h.service.StartExpedition(ctx, req.Platform, req.PlatformID, req.Username, req.ExpeditionType)
			if err == nil {
				recordEngagement(r, h.progressionSvc, req.Username, "expedition_started", 2)
			}
			return exp, err
		},
		h.formatStartResponse,
	)
}

func (h *ExpeditionHandler) formatStartResponse(e *domain.Expedition) interface{} {
	return StartExpeditionResponse{
		Message:      "Expedition started! Others can join.",
		ExpeditionID: e.ID.String(),
		JoinDeadline: e.JoinDeadline.Format("2006-01-02 15:04:05"),
	}
}

// JoinExpeditionRequest represents an expedition join request
type JoinExpeditionRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
	Username   string `json:"username"`
}

// HandleJoin handles expedition join requests
func (h *ExpeditionHandler) HandleJoin(w http.ResponseWriter, r *http.Request) {
	// Check if expedition feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureExpedition) {
		return
	}

	expeditionID, ok := h.parseExpeditionID(w, r)
	if !ok {
		return
	}

	var req JoinExpeditionRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Join expedition"); err != nil {
		return
	}

	if err := h.service.JoinExpedition(r.Context(), req.Platform, req.PlatformID, req.Username, expeditionID); err != nil {
		respondServiceError(w, r, "Failed to join expedition", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Joined expedition!"})
}

// HandleGet handles expedition get requests
func (h *ExpeditionHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	h.handleByID(w, r, "Failed to get expedition", func(ctx context.Context, id uuid.UUID) (interface{}, error) {
		return h.service.GetExpedition(ctx, id)
	})
}

// HandleGetActive handles active expedition requests
func (h *ExpeditionHandler) HandleGetActive(w http.ResponseWriter, r *http.Request) {
	// Check if expedition feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureExpedition) {
		return
	}

	expedition, err := h.service.GetActiveExpedition(r.Context())
	if err != nil {
		respondServiceError(w, r, "Failed to get active expedition", err)
		return
	}

	respondJSON(w, http.StatusOK, expedition)
}

// HandleGetJournal handles expedition journal requests
func (h *ExpeditionHandler) HandleGetJournal(w http.ResponseWriter, r *http.Request) {
	h.handleByID(w, r, "Failed to get expedition journal", func(ctx context.Context, id uuid.UUID) (interface{}, error) {
		return h.service.GetJournal(ctx, id)
	})
}

// HandleGetStatus handles expedition status requests
func (h *ExpeditionHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	// Check if expedition feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureExpedition) {
		return
	}

	status, err := h.service.GetStatus(r.Context())
	if err != nil {
		respondServiceError(w, r, "Failed to get expedition status", err)
		return
	}

	respondJSON(w, http.StatusOK, status)
}

func (h *ExpeditionHandler) parseExpeditionID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	idStr, ok := GetQueryParam(r, w, "id")
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid expedition ID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *ExpeditionHandler) handleByID(w http.ResponseWriter, r *http.Request, opName string, fn func(context.Context, uuid.UUID) (interface{}, error)) {
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureExpedition) {
		return
	}

	expeditionID, ok := h.parseExpeditionID(w, r)
	if !ok {
		return
	}

	result, err := fn(r.Context(), expeditionID)
	if err != nil {
		respondServiceError(w, r, opName, err)
		return
	}

	respondJSON(w, http.StatusOK, result)
}
