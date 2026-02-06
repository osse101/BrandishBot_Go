package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/quest"
)

type QuestHandler struct {
	questService       quest.Service
	progressionService progression.Service
}

func NewQuestHandler(questService quest.Service, progressionService progression.Service) *QuestHandler {
	return &QuestHandler{
		questService:       questService,
		progressionService: progressionService,
	}
}

// GetActiveQuests returns the current week's active quests
func (h *QuestHandler) GetActiveQuests(w http.ResponseWriter, r *http.Request) {
	if locked := CheckFeatureLocked(w, r, h.progressionService, "feature_weekly_quests"); locked {
		return
	}

	ctx := r.Context()

	quests, err := h.questService.GetActiveQuests(ctx)
	if err != nil {
		http.Error(w, "Failed to retrieve quests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quests)
}

// GetUserQuestProgress returns user's quest progress
func (h *QuestHandler) GetUserQuestProgress(w http.ResponseWriter, r *http.Request) {
	if locked := CheckFeatureLocked(w, r, h.progressionService, "feature_weekly_quests"); locked {
		return
	}

	ctx := r.Context()
	log := logger.FromContext(ctx)

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	progress, err := h.questService.GetUserQuestProgress(ctx, userID)
	if err != nil {
		log.Error("Failed to get quest progress", "error", err)
		http.Error(w, "Failed to retrieve quest progress", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(progress)
}

// ClaimQuestReward claims a completed quest's reward
func (h *QuestHandler) ClaimQuestReward(w http.ResponseWriter, r *http.Request) {
	if locked := CheckFeatureLocked(w, r, h.progressionService, "feature_weekly_quests"); locked {
		return
	}

	ctx := r.Context()
	log := logger.FromContext(ctx)

	var req struct {
		UserID  string `json:"user_id"`
		QuestID int    `json:"quest_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	money, xp, err := h.questService.ClaimQuestReward(ctx, req.UserID, req.QuestID)
	if err != nil {
		log.Error("Failed to claim quest reward", "error", err)
		http.Error(w, "Failed to claim reward", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"money_earned": money,
		"xp_earned":    xp,
		"message":      "Quest reward claimed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
