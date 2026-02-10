package handler

import (
	"context"
	"errors"
	"net/http"

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
// @Router /api/v1/progression/tree [get]
func (h *ProgressionHandlers) HandleGetTree() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		tree, err := h.service.GetProgressionTree(r.Context())
		if err != nil {
			log.Error("Get progression tree: service error", "error", err)
			respondError(w, http.StatusInternalServerError, ErrMsgGetProgressionTreeFailed)
			return
		}

		response := ProgressionTreeResponse{
			Nodes: tree,
		}

		log.Info("Get progression tree: success")
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
// @Router /api/v1/progression/available [get]
func (h *ProgressionHandlers) HandleGetAvailable() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		available, err := h.service.GetAvailableUnlocks(r.Context())
		if err != nil {
			log.Error("Get available unlocks: service error", "error", err)
			respondError(w, http.StatusInternalServerError, ErrMsgGetAvailableUnlocksFailed)
			return
		}

		response := AvailableUnlocksResponse{
			Available: available,
		}

		log.Info("Get available unlocks: success")
		respondJSON(w, http.StatusOK, response)
	}
}

// HandleVote allows a user to vote for the next unlock
// @Summary Vote for unlock
// @Description Cast a vote for the next unlock by selecting an option index (1-based)
// @Tags progression
// @Accept json
// @Produce json
// @Param request body VoteRequest true "Vote request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/progression/vote [post]
func (h *ProgressionHandlers) HandleVote() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req VoteRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Vote for unlock"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		// Cast vote
		err := h.service.VoteForUnlock(r.Context(), req.Platform, req.PlatformID, req.Username, req.OptionIndex)
		if err != nil {
			if errors.Is(err, domain.ErrUserAlreadyVoted) {
				respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgAlreadyVoted})
				return
			}
			log.Warn("Vote request: service error", "error", err, "platform", req.Platform, "platformID", req.PlatformID, "username", req.Username, "optionIndex", req.OptionIndex)
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		log.Info("Vote cast successfully", "platform", req.Platform, "platformID", req.PlatformID, "username", req.Username, "optionIndex", req.OptionIndex)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgVoteRecordedSuccess})
	}
}

// HandleGetStatus returns current progression status
// @Summary Get progression status
// @Description Returns current community progression status including unlocks and engagement
// @Tags progression
// @Produce json
// @Success 200 {object} domain.ProgressionStatus
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/progression/status [get]
func (h *ProgressionHandlers) HandleGetStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		status, err := h.service.GetProgressionStatus(r.Context())
		if err != nil {
			log.Error("Get progression status: service error", "error", err)
			respondError(w, http.StatusInternalServerError, ErrMsgGetProgressionStatusFailed)
			return
		}

		log.Info("Get progression status: success")
		respondJSON(w, http.StatusOK, status)
	}
}

// HandleGetEngagement returns user's engagement breakdown
// @Summary Get user engagement
// @Description Returns user's engagement contribution breakdown by type
// @Tags progression
// @Produce json
// @Param platform query string true "Platform (twitch, youtube, discord)"
// @Param platform_id query string true "Platform-specific user ID"
// @Success 200 {object} domain.ContributionBreakdown
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/progression/engagement [get]
func (h *ProgressionHandlers) HandleGetEngagement() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleGetEngagementCommon(w, r, "platform_id", h.service.GetUserEngagement, "Get user engagement")
	}
}

// HandleGetEngagementByUsername returns user's engagement breakdown by username
// @Summary Get user engagement by username
// @Description Returns user's engagement contribution breakdown by type using username
// @Tags progression
// @Produce json
// @Param platform query string true "Platform (twitch, youtube, discord)"
// @Param username query string true "Username"
// @Success 200 {object} domain.ContributionBreakdown
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/progression/engagement-by-username [get]
func (h *ProgressionHandlers) HandleGetEngagementByUsername() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleGetEngagementCommon(w, r, "username", h.service.GetUserEngagementByUsername, "Get user engagement by username")
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
// @Router /api/v1/progression/leaderboard [get]
func (h *ProgressionHandlers) HandleGetContributionLeaderboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleGetSimpleMetric(w, r, "limit", 10, func(ctx context.Context, i int) (interface{}, error) {
			return h.service.GetContributionLeaderboard(ctx, i)
		}, "Get contribution leaderboard", ErrMsgGetLeaderboardFailed)
	}
}

// HandleGetVelocity returns engagement velocity metrics (Admin/Debug)
// @Summary Get engagement velocity
// @Description Returns engagement velocity metrics (points/day) and trend
// @Tags progression,admin
// @Produce json
// @Param days query int false "Number of days (default 7)"
// @Success 200 {object} domain.VelocityMetrics
// @Failure 500 {object} ErrorResponse
// @Router /progression/velocity [get]
func (h *ProgressionHandlers) HandleGetVelocity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleGetSimpleMetric(w, r, "days", 7, func(ctx context.Context, i int) (interface{}, error) {
			return h.service.GetEngagementVelocity(ctx, i)
		}, "Get engagement velocity", ErrMsgGetVelocityMetricsFailed)
	}
}

// Admin endpoints

// HandleAdminUnlock admin force-unlocks a node
// ... (swagger comments)
func (h *ProgressionHandlers) HandleAdminUnlock() http.HandlerFunc {
	return h.handleAdminNodeAction(func(ctx context.Context, nodeKey string, level int) error {
		return h.service.AdminUnlock(ctx, nodeKey, level)
	}, "Admin unlocked node", MsgNodeUnlockedSuccess)
}

// HandleAdminRelock admin relocks a node
// ... (swagger comments)
func (h *ProgressionHandlers) HandleAdminRelock() http.HandlerFunc {
	return h.handleAdminNodeAction(func(ctx context.Context, nodeKey string, level int) error {
		return h.service.AdminRelock(ctx, nodeKey, level)
	}, "Admin relocked node", MsgNodeRelockedSuccess)
}

// HandleAdminUnlockAll admin unlocks all nodes at max level (DEBUG)
// @Summary Admin unlock all nodes
// @Description Unlocks all progression nodes at their maximum level (for debugging)
// @Tags progression,admin
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/unlock-all [post]
func (h *ProgressionHandlers) HandleAdminUnlockAll() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if err := h.service.AdminUnlockAll(r.Context()); err != nil {
			log.Error("Failed to unlock all nodes", "error", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin unlocked all nodes")
		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgAllNodesUnlockedSuccess})
	}
}

func (h *ProgressionHandlers) handleAdminNodeAction(action func(context.Context, string, int) error, logMsg, successMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// We use a shared struct for validation since both use same fields
		var req struct {
			NodeKey string `json:"node_key" validate:"required,max=50"`
			Level   int    `json:"level" validate:"min=0"`
		}
		if err := DecodeAndValidateRequest(r, w, &req, "Admin node action"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		if err := action(r.Context(), req.NodeKey, req.Level); err != nil {
			log.Error("Admin node action: service error", "error", err, "nodeKey", req.NodeKey, "level", req.Level)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info(logMsg, "nodeKey", req.NodeKey, "level", req.Level)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: successMsg})
	}
}

func (h *ProgressionHandlers) handleGetEngagementCommon(w http.ResponseWriter, r *http.Request, paramName string, serviceFn func(context.Context, string, string) (*domain.ContributionBreakdown, error), logMsg string) {
	log := logger.FromContext(r.Context())

	platform, ok := GetQueryParam(r, w, "platform")
	if !ok {
		return
	}
	val, ok := GetQueryParam(r, w, paramName)
	if !ok {
		return
	}

	breakdown, err := serviceFn(r.Context(), platform, val)
	if err != nil {
		log.Error(logMsg+": service error", "error", err, "platform", platform, paramName, val)
		respondError(w, http.StatusInternalServerError, ErrMsgGetEngagementDataFailed)
		return
	}

	log.Info(logMsg+": success", "platform", platform, paramName, val)
	respondJSON(w, http.StatusOK, breakdown)
}

func (h *ProgressionHandlers) handleGetSimpleMetric(w http.ResponseWriter, r *http.Request, paramName string, defaultVal int, serviceFn func(context.Context, int) (interface{}, error), logMsg, errMsg string) {
	log := logger.FromContext(r.Context())

	val := getQueryInt(r, paramName, defaultVal)
	result, err := serviceFn(r.Context(), val)
	if err != nil {
		log.Error(logMsg+": service error", "error", err, paramName, val)
		respondError(w, http.StatusInternalServerError, errMsg)
		return
	}
	log.Info(logMsg+": success", paramName, val)
	respondJSON(w, http.StatusOK, result)
}

func (h *ProgressionHandlers) handleAdminAction(w http.ResponseWriter, r *http.Request, action func(context.Context) (interface{}, error), errLogMsg, infoLogMsg string, responseFactory func(interface{}) interface{}) {
	log := logger.FromContext(r.Context())
	res, err := action(r.Context())
	if err != nil {
		log.Error(errLogMsg, "error", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info(infoLogMsg)
	respondJSON(w, http.StatusOK, responseFactory(res))
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
		h.handleAdminAction(w, r,
			func(ctx context.Context) (interface{}, error) { return h.service.ForceInstantUnlock(ctx) },
			"Failed to instant unlock",
			"Admin forced instant unlock",
			func(res interface{}) interface{} {
				unlock := res.(*domain.ProgressionUnlock)
				return AdminInstantUnlockResponse{Unlock: unlock, Message: MsgInstantUnlockSuccess}
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
		var req AdminResetRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Admin reset"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		if req.Reason == "" {
			req.Reason = "Annual reset"
		}

		err := h.service.ResetProgressionTree(r.Context(), req.ResetBy, req.Reason, req.PreserveUserProgression)
		if err != nil {
			log.Error("Admin reset: service error", "error", err, "resetBy", req.ResetBy)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin reset progression tree", "resetBy", req.ResetBy, "reason", req.Reason)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgProgressionResetSuccess})
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
			log.Error("Get voting session: service error", "error", err)
			respondError(w, http.StatusInternalServerError, ErrMsgGetVotingSessionFailed)
			return
		}

		if session == nil {
			recentSession, err := h.service.GetMostRecentVotingSession(r.Context())
			if err != nil {
				log.Error("Get most recent session: service error", "error", err)
				respondError(w, http.StatusInternalServerError, ErrMsgGetVotingSessionFailed)
				return
			}

			if recentSession != nil && recentSession.Status == "completed" {
				log.Info("Get voting session: returning completed session", "sessionID", recentSession.ID)
				respondJSON(w, http.StatusOK, map[string]interface{}{
					"session":      recentSession,
					"message":      "Voting has ended. Results are shown below.",
					"is_completed": true,
				})
				return
			}

			log.Info("Get voting session: no active session")
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"session": nil,
				"message": MsgNoActiveVotingSession,
			})
			return
		}

		log.Info("Get voting session: success")
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
			log.Error("Get unlock progress: service error", "error", err)
			respondError(w, http.StatusInternalServerError, ErrMsgGetUnlockProgressFailed)
			return
		}

		if progress == nil {
			log.Info("Get unlock progress: no active progress")
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"progress": nil,
				"message":  MsgNoActiveUnlockProgress,
			})
			return
		}

		log.Info("Get unlock progress: success")
		response := h.enrichUnlockProgress(r.Context(), progress)
		respondJSON(w, http.StatusOK, response)
	}
}

func (h *ProgressionHandlers) enrichUnlockProgress(ctx context.Context, progress *domain.UnlockProgress) map[string]interface{} {
	response := map[string]interface{}{
		"id":                        progress.ID,
		"node_id":                   progress.NodeID,
		"target_level":              progress.TargetLevel,
		"contributions_accumulated": progress.ContributionsAccumulated,
		"started_at":                progress.StartedAt,
		"unlocked_at":               progress.UnlockedAt,
		"voting_session_id":         progress.VotingSessionID,
		"completion_percentage":     0.0,
		"target_unlock_cost":        0,
		"target_node_name":          "",
		"is_already_unlocked":       false,
	}

	if progress.NodeID != nil {
		node, err := h.service.GetNode(ctx, *progress.NodeID)
		if err == nil && node != nil {
			response["target_unlock_cost"] = node.UnlockCost
			response["target_node_name"] = node.DisplayName

			if progress.TargetLevel != nil {
				isUnlocked, _ := h.service.IsNodeUnlocked(ctx, node.NodeKey, *progress.TargetLevel)
				if isUnlocked {
					response["is_already_unlocked"] = true
					response["completion_percentage"] = 100.0
					return response
				}
			}

			if node.UnlockCost > 0 {
				percent := (float64(progress.ContributionsAccumulated) / float64(node.UnlockCost)) * 100
				if percent > 100 {
					percent = 100
				}
				response["completion_percentage"] = percent
			} else {
				response["completion_percentage"] = 100.0
			}
		}
	}

	return response
}

// HandleAdminEndVoting admin freezes current voting (pauses until unlock)
// @Summary Admin freeze voting
// @Description Freeze the current voting session until the next unlock completes (admin only)
// @Tags progression,admin
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/end-voting [post]
func (h *ProgressionHandlers) HandleAdminEndVoting() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		err := h.service.AdminFreezeVoting(r.Context())
		if err != nil {
			if errors.Is(err, domain.ErrNoActiveSession) {
				log.Warn("No active voting session to freeze")
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
			if errors.Is(err, domain.ErrSessionAlreadyFrozen) {
				log.Info("Voting session already frozen")
				respondJSON(w, http.StatusOK, SuccessResponse{Message: "Voting session already frozen"})
				return
			}
			log.Error("Failed to freeze voting", "error", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin froze voting session")
		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Voting session frozen"})
	}
}

// HandleAdminForceEndVoting admin force-ends current voting and selects winner
// @Summary Admin force end voting
// @Description Force end the current voting session and determine winner immediately (admin only)
// @Tags progression,admin
// @Produce json
// @Success 200 {object} AdminEndVotingResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/force-end-voting [post]
func (h *ProgressionHandlers) HandleAdminForceEndVoting() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handleAdminAction(w, r,
			func(ctx context.Context) (interface{}, error) { return h.service.EndVoting(ctx) },
			"Failed to end voting",
			"Admin ended voting",
			func(res interface{}) interface{} {
				winner := res.(*domain.ProgressionVotingOption)
				return AdminEndVotingResponse{Winner: winner, Message: MsgVotingEndedSuccess}
			})
	}
}

// HandleAdminStartVoting admin starts or resumes a voting session
// @Summary Admin start/resume voting
// @Description Resume a frozen voting session OR start a new one if nodes are available (admin only)
// @Tags progression,admin
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/start-voting [post]
func (h *ProgressionHandlers) HandleAdminStartVoting() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		err := h.service.AdminStartVoting(r.Context())
		if err != nil {
			if errors.Is(err, domain.ErrSessionAlreadyActive) {
				log.Warn("Voting session already active")
				respondError(w, http.StatusConflict, err.Error())
				return
			}
			if errors.Is(err, domain.ErrNoNodesAvailable) {
				log.Info("No nodes available for voting")
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
			if errors.Is(err, domain.ErrAccumulationInProgress) {
				log.Warn("Cannot start voting while accumulation in progress")
				respondError(w, http.StatusConflict, err.Error())
				return
			}
			log.Error("Failed to start/resume voting session", "error", err)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin started/resumed voting session")
		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgVotingSessionStartSuccess})
	}
}

// HandleAdminAddContribution admin adds contribution points
// @Summary Admin add contribution
// @Description Manually add contribution points to the current unlock progress (admin only)
// @Tags progression,admin
// @Accept json
// @Produce json
// @Param request body AdminAddContributionRequest true "Contribution request"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /progression/admin/contribution [post]
func (h *ProgressionHandlers) HandleAdminAddContribution() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AdminAddContributionRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Admin add contribution"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		if req.Amount <= 0 {
			log.Warn("Admin add contribution: invalid amount", "amount", req.Amount)
			respondError(w, http.StatusBadRequest, ErrMsgAmountMustBePositive)
			return
		}

		if err := h.service.AddContribution(r.Context(), req.Amount); err != nil {
			log.Error("Admin add contribution: service error", "error", err, "amount", req.Amount)
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("Admin added contribution", "amount", req.Amount)
		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgContributionAddedSuccess})
	}
}

// HandleAdminReloadWeights invalidates the engagement weight cache
// @Summary Admin reload weights
// @Description Invalidate engagement weight cache to force reload from database (admin only)
// @Tags progression,admin
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /admin/progression/reload-weights [post]
func (h *ProgressionHandlers) HandleAdminReloadWeights() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		h.service.InvalidateWeightCache()

		log.Info("Admin invalidated engagement weight cache")
		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgWeightCacheInvalidated})
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
	Platform    string `json:"platform" validate:"required,max=20"`
	PlatformID  string `json:"platform_id" validate:"required,max=100,excludesall=\x00\n\r\t"`
	Username    string `json:"username" validate:"required,max=100"`
	OptionIndex int    `json:"option_index" validate:"required,min=1"`
}

type AdminUnlockRequest struct {
	NodeKey string `json:"node_key" validate:"required,max=50"`
	Level   int    `json:"level" validate:"min=0"`
}

type AdminRelockRequest struct {
	NodeKey string `json:"node_key" validate:"required,max=50"`
	Level   int    `json:"level" validate:"min=0"`
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

type AdminAddContributionRequest struct {
	Amount int `json:"amount"`
}
