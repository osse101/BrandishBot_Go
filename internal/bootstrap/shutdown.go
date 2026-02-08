package bootstrap

import (
	"context"
	"log/slog"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/prediction"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/quest"
	"github.com/osse101/BrandishBot_Go/internal/server"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/osse101/BrandishBot_Go/internal/worker"
)

// ShutdownComponents holds all components that need graceful shutdown.
type ShutdownComponents struct {
	Server             *server.Server
	ProgressionService progression.Service
	UserService        user.Service
	EconomyService     economy.Service
	CraftingService    crafting.Service
	GambleService      gamble.Service
	PredictionService  prediction.Service
	QuestService       quest.Service
	GambleWorker       *worker.GambleWorker
	ExpeditionWorker   *worker.ExpeditionWorker
	DailyResetWorker   *worker.DailyResetWorker
	WeeklyResetWorker  *worker.WeeklyResetWorker
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
	slog.Info(LogMsgShuttingDownServer)

	// Shutdown server first (stop accepting new requests)
	if err := components.Server.Stop(ctx); err != nil {
		slog.Error(LogMsgServerForcedShutdown, "error", err)
	}

	// Shutdown workers first to cancel pending timers
	if components.GambleWorker != nil {
		if err := components.GambleWorker.Shutdown(ctx); err != nil {
			slog.Error("Gamble worker shutdown failed", "error", err)
		}
	}

	if components.ExpeditionWorker != nil {
		if err := components.ExpeditionWorker.Shutdown(ctx); err != nil {
			slog.Error("Expedition worker shutdown failed", "error", err)
		}
	}

	if components.DailyResetWorker != nil {
		if err := components.DailyResetWorker.Shutdown(ctx); err != nil {
			slog.Error("Daily reset worker shutdown failed", "error", err)
		}
	}

	if components.WeeklyResetWorker != nil {
		if err := components.WeeklyResetWorker.Shutdown(ctx); err != nil {
			slog.Error("Weekly reset worker shutdown failed", "error", err)
		}
	}

	// Shutdown services (order doesn't matter, all run independently)
	shutdownService(ctx, ServiceNameProgression, components.ProgressionService)
	shutdownService(ctx, ServiceNameUser, components.UserService)
	shutdownService(ctx, ServiceNameEconomy, components.EconomyService)
	shutdownService(ctx, ServiceNameCrafting, components.CraftingService)
	shutdownService(ctx, ServiceNameGamble, components.GambleService)
	shutdownService(ctx, "prediction", components.PredictionService)
	shutdownService(ctx, "quest", components.QuestService)

	// Shutdown resilient publisher last to flush pending events
	slog.Info(LogMsgShuttingDownEventPublisher)
	if err := components.ResilientPublisher.Shutdown(ctx); err != nil {
		slog.Error(LogMsgResilientPublisherFailed, "error", err)
	}

	slog.Info(LogMsgServerStopped)
}

// shutdownService is a helper that shuts down a service and logs any errors.
// This implements a common pattern for all service shutdowns.
type shutdownableService interface {
	Shutdown(context.Context) error
}

func shutdownService(ctx context.Context, name string, service shutdownableService) {
	if err := service.Shutdown(ctx); err != nil {
		slog.Error(name+LogMsgServiceShutdownFailed, "error", err)
	}
}
