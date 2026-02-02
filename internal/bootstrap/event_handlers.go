package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/metrics"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

// EventHandlerDependencies holds the dependencies needed for event handler registration.
type EventHandlerDependencies struct {
	EventBus           event.Bus
	ProgressionService progression.Service
	EventLogService    eventlog.Service
	Config             *config.Config
}

// RegisterEventHandlers sets up all event handlers and subscribers.
// This includes:
// - Progression event handler (for cycle completion events)
// - Metrics collector (for event-based metrics)
// - Event logger (persists events to database)
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

	return nil
}
