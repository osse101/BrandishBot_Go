package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/database"
)

// HealthResponse represents the response for health endpoints
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthChecker defines the interface for components that can report health
type HealthChecker interface {
	CheckHealth(ctx context.Context) error
}

// HandleHealthz provides a basic liveness check
// @Summary Liveness check
// @Description Returns OK if the service is running
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
func HandleHealthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := HealthResponse{
			Status: "ok",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// HandleReadyz provides a readiness check that validates database connectivity
// @Summary Readiness check
// @Description Returns OK if the service is ready to accept traffic (database connected)
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /readyz [get]
func HandleReadyz(dbPool database.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Check database connectivity
		if err := dbPool.Ping(ctx); err != nil {
			slog.Error("Readiness check failed", "error", err)

			response := HealthResponse{
				Status:  "unavailable",
				Message: "database connection failed",
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := HealthResponse{
			Status: "ok",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
