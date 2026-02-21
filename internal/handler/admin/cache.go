package admin

import (
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/user"
)

// CacheHandler handles admin cache operations
type CacheHandler struct {
	userService user.Service
}

// NewCacheHandler creates a new admin cache handler
func NewCacheHandler(userService user.Service) *CacheHandler {
	return &CacheHandler{
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
func (h *CacheHandler) HandleGetCacheStats(w http.ResponseWriter, r *http.Request) {
	stats := h.userService.GetCacheStats()
	handler.RespondJSON(w, http.StatusOK, stats)
}
