package handler

import (
	"encoding/json"
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
	// Check if gamble feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureGamble) {
		return
	}

	var req StartGambleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	gamble, err := h.service.StartGamble(r.Context(), req.Platform, req.PlatformID, req.Username, req.Bets)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to start gamble", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(gamble); err != nil {
		logger.FromContext(r.Context()).Error("Failed to encode response", "error", err)
	}
}

type JoinGambleRequest struct {
	Platform   string              `json:"platform"`
	PlatformID string              `json:"platform_id"`
	Username   string              `json:"username"`
	Bets       []domain.LootboxBet `json:"bets"`
}

func (h *GambleHandler) HandleJoinGamble(w http.ResponseWriter, r *http.Request) {
	gambleIDStr := r.URL.Query().Get("id")
	if gambleIDStr == "" {
		http.Error(w, "Missing gamble ID", http.StatusBadRequest)
		return
	}
	gambleID, err := uuid.Parse(gambleIDStr)
	if err != nil {
		http.Error(w, "Invalid gamble ID", http.StatusBadRequest)
		return
	}

	var req JoinGambleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.JoinGamble(r.Context(), gambleID, req.Platform, req.PlatformID, req.Username, req.Bets); err != nil {
		logger.FromContext(r.Context()).Error("Failed to join gamble", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Successfully joined gamble"}); err != nil {
		logger.FromContext(r.Context()).Error("Failed to encode response", "error", err)
	}
}

func (h *GambleHandler) HandleGetGamble(w http.ResponseWriter, r *http.Request) {
	gambleIDStr := r.URL.Query().Get("id")
	if gambleIDStr == "" {
		http.Error(w, "Missing gamble ID", http.StatusBadRequest)
		return
	}
	gambleID, err := uuid.Parse(gambleIDStr)
	if err != nil {
		http.Error(w, "Invalid gamble ID", http.StatusBadRequest)
		return
	}

	gamble, err := h.service.GetGamble(r.Context(), gambleID)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get gamble", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if gamble == nil {
		http.Error(w, "Gamble not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(gamble); err != nil {
		logger.FromContext(r.Context()).Error("Failed to encode response", "error", err)
	}
}
