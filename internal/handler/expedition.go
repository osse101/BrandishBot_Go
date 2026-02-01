package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/expedition"
	"github.com/osse101/BrandishBot_Go/internal/logger"
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
	// Check if expedition feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureExpedition) {
		return
	}

	var req StartExpeditionRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Start expedition"); err != nil {
		return
	}

	expedition, err := h.service.StartExpedition(r.Context(), req.Platform, req.PlatformID, req.Username, req.ExpeditionType)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to start expedition", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	// Record engagement
	if err := h.progressionSvc.RecordEngagement(r.Context(), req.Username, "expedition_started", 2); err != nil {
		logger.FromContext(r.Context()).Error("Failed to record expedition engagement", "error", err)
	}

	response := StartExpeditionResponse{
		Message:      "Expedition started! Others can join.",
		ExpeditionID: expedition.ID.String(),
		JoinDeadline: expedition.JoinDeadline.Format("2006-01-02 15:04:05"),
	}
	respondJSON(w, http.StatusCreated, response)
}

// JoinExpeditionRequest represents an expedition join request
type JoinExpeditionRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
	Username   string `json:"username"`
}

// HandleJoin handles expedition join requests
func (h *ExpeditionHandler) HandleJoin(w http.ResponseWriter, r *http.Request) {
	expeditionIDStr, ok := GetQueryParam(r, w, "id")
	if !ok {
		return
	}
	expeditionID, err := uuid.Parse(expeditionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid expedition ID")
		return
	}

	var req JoinExpeditionRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Join expedition"); err != nil {
		return
	}

	if err := h.service.JoinExpedition(r.Context(), req.Platform, req.PlatformID, req.Username, expeditionID); err != nil {
		logger.FromContext(r.Context()).Error("Failed to join expedition", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Joined expedition!"})
}

// HandleGet handles expedition get requests
func (h *ExpeditionHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	expeditionIDStr, ok := GetQueryParam(r, w, "id")
	if !ok {
		return
	}
	expeditionID, err := uuid.Parse(expeditionIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid expedition ID")
		return
	}

	expedition, err := h.service.GetExpedition(r.Context(), expeditionID)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get expedition", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	respondJSON(w, http.StatusOK, expedition)
}

// HandleGetActive handles active expedition requests
func (h *ExpeditionHandler) HandleGetActive(w http.ResponseWriter, r *http.Request) {
	expedition, err := h.service.GetActiveExpedition(r.Context())
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get active expedition", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	respondJSON(w, http.StatusOK, expedition)
}
