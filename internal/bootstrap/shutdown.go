package bootstrap

import (
	"context"
	"log/slog"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/server"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// ShutdownComponents holds all components that need graceful shutdown.
type ShutdownComponents struct {
	Server             *server.Server
	ProgressionService progression.Service
	UserService        user.Service
	EconomyService     economy.Service
	CraftingService    crafting.Service
	GambleService      gamble.Service
	ResilientPublisher *event.ResilientPublisher
}

// GracefulShutdown performs graceful shutdown of all application components.
// It shuts down services in the correct order:
// 1. HTTP server (stop accepting new requests)
// 2. Application services (complete in-flight operations)
// 3. Event publisher (flush pending events to ensure consistency)
//
// Errors during shutdown are logged but do not stop the shutdown sequence.
func GracefulShutdown(ctx context.Context, components ShutdownComponents) {
	slog.Info("Shutting down server...")

	// Shutdown server first (stop accepting new requests)
	if err := components.Server.Stop(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	// Shutdown services (order doesn't matter, all run independently)
	shutdownService(ctx, "progression", components.ProgressionService)
	shutdownService(ctx, "user", components.UserService)
	shutdownService(ctx, "economy", components.EconomyService)
	shutdownService(ctx, "crafting", components.CraftingService)
	shutdownService(ctx, "gamble", components.GambleService)

	// Shutdown resilient publisher last to flush pending events
	slog.Info("Shutting down event publisher...")
	if err := components.ResilientPublisher.Shutdown(ctx); err != nil {
		slog.Error("Resilient publisher shutdown failed", "error", err)
	}

	slog.Info("Server stopped")
}

// shutdownService is a helper that shuts down a service and logs any errors.
// This implements a common pattern for all service shutdowns.
type shutdownableService interface {
	Shutdown(context.Context) error
}

func shutdownService(ctx context.Context, name string, service shutdownableService) {
	if err := service.Shutdown(ctx); err != nil {
		slog.Error(name+" service shutdown failed", "error", err)
	}
}
