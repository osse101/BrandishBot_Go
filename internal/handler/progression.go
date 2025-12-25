package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

// ProgressionHandlers contains HTTP handlers for progression system
type ProgressionHandlers struct {
	service progression.Service
}

// NewProgressionHandlers creates new progression handlers
func NewProgressionHandlers(service progression.Service) *ProgressionHandlers {
	return &ProgressionHandlers{service: service}
}

// HandleGetTree returns the full progression tree with unlock status
// @Summary Get progression tree
// @Description Returns the complete progression tree with unlock status for each node
// @Tags progression
// @Produce json
// @Success 200 {object} ProgressionTreeResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/tree [get]
func (h *ProgressionHandlers) HandleGetTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		tree, err := h.service.GetProgressionTree(r.Context())
		if err != nil {
			log.Error("Failed to get progression tree", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to retrieve progression tree")
			return
		}

		response := ProgressionTreeResponse{
			Nodes: tree,
		}

		respondJSON(w, http.StatusOK, response)
	}
}

// HandleGetAvailable returns nodes available for voting
// @Summary Get available unlocks
// @Description Returns nodes that are available for voting (prerequisites met, not maxed out)
// @Tags progression
// @Produce json
// @Success 200 {object} AvailableUnlocksResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/available [get]
func (h *ProgressionHandlers) HandleGetAvailable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		available, err := h.service.GetAvailableUnlocks(r.Context())
		if err != nil {
			log.Error("Failed to get available unlocks", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to retrieve available unlocks")
			return
		}

		response := AvailableUnlocksResponse{
			Available: available,
		}

		respondJSON(w, http.StatusOK, response)
	}
}


// HandleVote allows a user to vote for the next unlock
// @Summary Vote for unlock
// @Description Cast a vote for the next unlock (one vote per user per node/level)
// @Tags progression
// @Accept json
// @Produce json
// @Param request body VoteRequest true "Vote request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/vote [post]
func (h *ProgressionHandlers) HandleVote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req VoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
			return
		}

		// Cast vote
		err := h.service.VoteForUnlock(r.Context(), req.UserID, req.NodeKey)
		if err != nil {
			log.Error("Failed to cast vote", "error", err, "userID", req.UserID, "nodeKey", req.NodeKey)
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		log.Info("Vote cast successfully", "userID", req.UserID, "nodeKey", req.NodeKey)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Vote recorded successfully"})
	}
}

// HandleGetStatus returns current progression status
// @Summary Get progression status
// @Description Returns current community progression status including unlocks and engagement
// @Tags progression
// @Produce json
// @Success 200 {object} domain.ProgressionStatus
// @Failure 500 {object} ErrorResponse
// @Router /progression/status [get]
func (h *ProgressionHandlers) HandleGetStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		status, err := h.service.GetProgressionStatus(r.Context())
		if err != nil {
			log.Error("Failed to get progression status", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to retrieve progression status")
			return
		}

		respondJSON(w, http.StatusOK, status)
	}
}

// HandleGetEngagement returns user's engagement breakdown
// @Summary Get user engagement
// @Description Returns user's engagement contribution breakdown by type
// @Tags progression
// @Produce json
// @Param user_id query string true "User ID"
// @Success 200 {object} domain.EngagementBreakdown
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/engagement [get]
func (h *ProgressionHandlers) HandleGetEngagement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			respondError(w, http.StatusBadRequest, "user_id query parameter is required")
			return
		}

		breakdown, err := h.service.GetUserEngagement(r.Context(), userID)
		if err != nil {
			log.Error("Failed to get user engagement", "error", err, "userID", userID)
			respondError(w, http.StatusInternalServerError, "Failed to retrieve engagement data")
			return
		}

		respondJSON(w, http.StatusOK, breakdown)
	}
}

// HandleGetContributionLeaderboard returns top contributors
// @Summary Get contribution leaderboard
// @Description Returns top contributors by total contributions
// @Tags progression
// @Produce json
// @Param limit query int false "Number of entries (default 10, max 100)"
// @Success 200 {array} domain.ContributionLeaderboardEntry
// @Failure 500 {object} ErrorResponse
// @Router /progression/leaderboard [get]
func (h *ProgressionHandlers) HandleGetContributionLeaderboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		limit := 10 // default
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		leaderboard, err := h.service.GetContributionLeaderboard(r.Context(), limit)
		if err != nil {
			log.Error("Failed to get contribution leaderboard", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to retrieve leaderboard")
			return
		}

		respondJSON(w, http.StatusOK, leaderboard)
	}
}

// Admin endpoints

// HandleAdminUnlock admin force-unlocks a node
// @Summary Admin unlock node
// @Description Force unlock a specific node/level (admin only)
// @Tags progression,admin
// @Accept json
// @Produce json
// @Param request body AdminUnlockRequest true "Admin unlock request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/unlock [post]
func (h *ProgressionHandlers) HandleAdminUnlock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req AdminUnlockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
			return
		}

		err := h.service.AdminUnlock(r.Context(), req.NodeKey, req.Level)
		if err != nil {
			log.Error("Failed to admin unlock", "error", err, "nodeKey", req.NodeKey, "level", req.Level)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin unlocked node", "nodeKey", req.NodeKey, "level", req.Level)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Node unlocked successfully"})
	}
}

// HandleAdminRelock admin relocks a node
// @Summary Admin relock node
// @Description Relock a specific node/level (admin only, for testing)
// @Tags progression,admin
// @Accept json
// @Produce json
// @Param request body AdminRelockRequest true "Admin relock request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/relock [post]
func (h *ProgressionHandlers) HandleAdminRelock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req AdminRelockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
			return
		}

		err := h.service.AdminRelock(r.Context(), req.NodeKey, req.Level)
		if err != nil {
			log.Error("Failed to admin relock", "error", err, "nodeKey", req.NodeKey, "level", req.Level)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin relocked node", "nodeKey", req.NodeKey, "level", req.Level)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Node relocked successfully"})
	}
}

// HandleAdminInstantUnlock forces immediate unlock of current vote leader
// @Summary Admin instant unlock
// @Description Force immediate unlock of current vote leader (overrides 24hr timer)
// @Tags progression,admin
// @Produce json
// @Success 200 {object} AdminInstantUnlockResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/instant-unlock [post]
func (h *ProgressionHandlers) HandleAdminInstantUnlock() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		unlock, err := h.service.ForceInstantUnlock(r.Context())
		if err != nil {
			log.Error("Failed to instant unlock", "error", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin forced instant unlock", "nodeID", unlock.NodeID, "level", unlock.CurrentLevel)
		respondJSON(w, http.StatusOK, AdminInstantUnlockResponse{
			Unlock:  unlock,
			Message: "Instant unlock successful",
		})
	}
}

// HandleAdminReset performs annual progression tree reset
// @Summary Admin reset tree
// @Description Reset progression tree (annual reset, clears unlocks/voting)
// @Tags progression,admin
// @Accept json
// @Produce json
// @Param request body AdminResetRequest true "Reset request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/reset [post]
func (h *ProgressionHandlers) HandleAdminReset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req AdminResetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
			return
		}

		if req.Reason == "" {
			req.Reason = "Annual reset"
		}

		err := h.service.ResetProgressionTree(r.Context(), req.ResetBy, req.Reason, req.PreserveUserProgression)
		if err != nil {
			log.Error("Failed to reset tree", "error", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin reset progression tree", "resetBy", req.ResetBy, "reason", req.Reason)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Progression tree reset successfully"})
	}
}

// HandleGetVotingSession returns current voting session with options
// @Summary Get voting session
// @Description Returns the current voting session with all available options
// @Tags progression
// @Produce json
// @Success 200 {object} domain.ProgressionVotingSession
// @Failure 500 {object} ErrorResponse
// @Router /progression/session [get]
func (h *ProgressionHandlers) HandleGetVotingSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		session, err := h.service.GetActiveVotingSession(r.Context())
		if err != nil {
			log.Error("Failed to get voting session", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to retrieve voting session")
			return
		}

		if session == nil {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"session": nil,
				"message": "No active voting session",
			})
			return
		}

		respondJSON(w, http.StatusOK, session)
	}
}

// HandleGetUnlockProgress returns current unlock progress
// @Summary Get unlock progress
// @Description Returns the current unlock progress including accumulated contributions
// @Tags progression
// @Produce json
// @Success 200 {object} domain.UnlockProgress
// @Failure 500 {object} ErrorResponse
// @Router /progression/unlock-progress [get]
func (h *ProgressionHandlers) HandleGetUnlockProgress() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		progress, err := h.service.GetUnlockProgress(r.Context())
		if err != nil {
			log.Error("Failed to get unlock progress", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to retrieve unlock progress")
			return
		}

		if progress == nil {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"progress": nil,
				"message":  "No active unlock progress",
			})
			return
		}

		respondJSON(w, http.StatusOK, progress)
	}
}

// HandleAdminEndVoting admin force-ends current voting
// @Summary Admin end voting
// @Description Force end the current voting session and determine winner (admin only)
// @Tags progression,admin
// @Produce json
// @Success 200 {object} AdminEndVotingResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/end-voting [post]
func (h *ProgressionHandlers) HandleAdminEndVoting() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		winner, err := h.service.EndVoting(r.Context())
		if err != nil {
			log.Error("Failed to end voting", "error", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin ended voting", "winningNodeID", winner.NodeID, "votes", winner.VoteCount)
		respondJSON(w, http.StatusOK, AdminEndVotingResponse{
			Winner:  winner,
			Message: "Voting ended successfully",
		})
	}
}

// HandleAdminStartVoting admin starts a new voting session
// @Summary Admin start voting
// @Description Start a new voting session with 4 random options (admin only)
// @Tags progression,admin
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/start-voting [post]
func (h *ProgressionHandlers) HandleAdminStartVoting() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		err := h.service.StartVotingSession(r.Context(), nil)
		if err != nil {
			log.Error("Failed to start voting session", "error", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin started voting session")
		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Voting session started successfully"})
	}
}

// Request/Response types

type ProgressionTreeResponse struct {
	Nodes []*domain.ProgressionTreeNode `json:"nodes"`
}

type AvailableUnlocksResponse struct {
	Available []*domain.ProgressionNode `json:"available"`
}

type VoteRequest struct {
	UserID  string `json:"user_id" validate:"required,max=100,excludesall=\x00\n\r\t"`
	NodeKey string `json:"node_key" validate:"required,max=50"`
}

type AdminUnlockRequest struct {
	NodeKey string `json:"node_key" validate:"required,max=50"`
	Level   int    `json:"level" validate:"min=1"`
}

type AdminRelockRequest struct {
	NodeKey string `json:"node_key" validate:"required,max=50"`
	Level   int    `json:"level" validate:"min=1"`
}

type AdminInstantUnlockResponse struct {
	Unlock  *domain.ProgressionUnlock `json:"unlock"`
	Message string                    `json:"message"`
}

type AdminEndVotingResponse struct {
	Winner  *domain.ProgressionVotingOption `json:"winner"`
	Message string                          `json:"message"`
}

type AdminResetRequest struct {
	ResetBy                 string `json:"reset_by" validate:"required,max=100"`
	Reason                  string `json:"reason" validate:"omitempty,max=255"`
	PreserveUserProgression bool   `json:"preserve_user_progression"`
}
