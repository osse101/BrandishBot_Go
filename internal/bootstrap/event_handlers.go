package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/metrics"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/quest"
	"github.com/osse101/BrandishBot_Go/internal/stats"
)

// EventHandlerDependencies holds the dependencies needed for event handler registration.
type EventHandlerDependencies struct {
	EventBus           event.Bus
	ProgressionService progression.Service
	EventLogService    eventlog.Service
	JobService         job.Service
	QuestService       quest.Service
	StatsService       stats.Service
	Config             *config.Config
}

// RegisterEventHandlers sets up all event handlers and subscribers.
// This includes:
// - Progression event handler (for cycle completion events)
// - Metrics collector (for event-based metrics)
// - Event logger (persists events to database)
// - Job event handler (for XP awards from crafting)
// - Quest event handler (for quest progress from crafting)
// - Stats event handler (for stats recording from crafting)
func RegisterEventHandlers(deps EventHandlerDependencies) error {
	// Register progression handler
	progressionHandler := progression.NewEventHandler(deps.ProgressionService)
	progressionHandler.Register(deps.EventBus)

	// Register Metrics Collector
	metricsCollector := metrics.NewEventMetricsCollector()
	if err := metricsCollector.Register(deps.EventBus); err != nil {
		return fmt.Errorf("%s: %w", ErrMsgFailedRegisterMetrics, err)
	}
	slog.Info(LogMsgMetricsCollectorRegistered)

	// Subscribe Event Logger
	if err := deps.EventLogService.Subscribe(deps.EventBus); err != nil {
		return fmt.Errorf("%s: %w", ErrMsgFailedSubscribeEventLogger, err)
	}
	slog.Info(LogMsgEventLoggerInitialized)

	// Register Job Handler (XP from crafting)
	if deps.JobService != nil {
		jobHandler := job.NewEventHandler(deps.JobService)
		jobHandler.Register(deps.EventBus)
		slog.Info("Job event handler registered")
	}

	// Register Quest Handler (Quest progress from crafting)
	if deps.QuestService != nil {
		questHandler := quest.NewEventHandler(deps.QuestService)
		questHandler.Register(deps.EventBus)
		slog.Info("Quest event handler registered")
	}

	// Register Stats Handler (Stats from crafting)
	if deps.StatsService != nil {
		statsHandler := stats.NewEventHandler(deps.StatsService)
		statsHandler.Register(deps.EventBus)
		slog.Info("Stats event handler registered")
	}

	return nil
}
