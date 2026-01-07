package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/user"
)

// AdminCacheHandler handles admin cache operations
type AdminCacheHandler struct {
	userService user.Service
}

// NewAdminCacheHandler creates a new admin cache handler
func NewAdminCacheHandler(userService user.Service) *AdminCacheHandler {
	return &AdminCacheHandler{
		userService: userService,
	}
}

// HandleGetCacheStats returns current user cache statistics
// GET /api/v1/admin/cache/stats
// @Summary Get user cache stats
// @Description Returns cache hit/miss statistics for monitoring (admin only)
// @Tags admin
// @Produce json
// @Success 200 {object} user.CacheStats
// @Router /api/v1/admin/cache/stats [get]
func (h *AdminCacheHandler) HandleGetCacheStats(w http.ResponseWriter, r *http.Request) {
	stats := h.userService.GetCacheStats()
	respondJSON(w, http.StatusOK, stats)
}
