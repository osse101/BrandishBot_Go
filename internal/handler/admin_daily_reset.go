package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// AdminDailyResetHandler handles admin endpoints for daily job XP reset management
type AdminDailyResetHandler struct {
	jobService job.Service
}

// NewAdminDailyResetHandler creates a new AdminDailyResetHandler
func NewAdminDailyResetHandler(jobService job.Service) *AdminDailyResetHandler {
	return &AdminDailyResetHandler{
		jobService: jobService,
	}
}

// HandleManualReset manually triggers a daily job XP reset
// POST /api/v1/admin/jobs/reset-daily-xp
// @Summary Manually trigger daily job XP reset
// @Description Triggers an immediate reset of all users' daily XP counters
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /admin/jobs/reset-daily-xp [post]
func (h *AdminDailyResetHandler) HandleManualReset(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Info("Manual daily reset triggered")

	recordsAffected, err := h.jobService.ResetDailyJobXP(r.Context())
	if err != nil {
		log.Error("Manual daily reset failed", "error", err)
		respondError(w, http.StatusInternalServerError, "Failed to reset daily XP")
		return
	}

	log.Info("Manual daily reset completed", "records_affected", recordsAffected)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"message":          "Daily XP reset completed",
		"records_affected": recordsAffected,
	})
}

// HandleGetResetStatus returns the current daily reset status
// GET /api/v1/admin/jobs/reset-status
// @Summary Get daily reset status
// @Description Returns information about the last daily reset and when the next one is scheduled
// @Tags admin
// @Produce json
// @Success 200 {object} domain.DailyResetStatus
// @Failure 500 {object} map[string]interface{}
// @Router /admin/jobs/reset-status [get]
func (h *AdminDailyResetHandler) HandleGetResetStatus(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	status, err := h.jobService.GetDailyResetStatus(r.Context())
	if err != nil {
		log.Error("Failed to get reset status", "error", err)
		respondError(w, http.StatusInternalServerError, "Failed to get reset status")
		return
	}

	respondJSON(w, http.StatusOK, status)
}
