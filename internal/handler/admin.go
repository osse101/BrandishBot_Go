package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
)

// HandleReloadAliases reloads the naming resolver configuration (admin only)
// @Summary Reload alias configuration
// @Description Reloads the item name aliases and themes from JSON configuration files
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /admin/reload-aliases [post]
// @Security ApiKeyAuth
func HandleReloadAliases(resolver naming.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := logger.FromContext(ctx)

		log.Info("Reloading naming resolver configuration")

		if err := resolver.Reload(); err != nil {
			log.Error("Failed to reload naming resolver", "error", err)
			respondError(w, http.StatusInternalServerError, "Failed to reload configuration")
			return
		}

		// Get current active theme for confirmation
		activeTheme := resolver.GetActiveTheme()

		log.Info("Naming resolver configuration reloaded successfully", "active_theme", activeTheme)

		response := map[string]interface{}{
			"message":      "Alias configuration reloaded successfully",
			"active_theme": activeTheme,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
