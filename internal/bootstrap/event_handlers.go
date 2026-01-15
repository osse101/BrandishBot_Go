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
// - Progression notifier (webhooks to Discord and Streamer.bot)
// - Metrics collector (for event-based metrics)
// - Event logger (persists events to database)
func RegisterEventHandlers(deps EventHandlerDependencies) error {
	// Register progression handler
	progressionHandler := progression.NewEventHandler(deps.ProgressionService)
	progressionHandler.Register(deps.EventBus)

	// Initialize and register Progression Notifier
	discordWebhookURL := fmt.Sprintf(DiscordWebhookURLFormat, deps.Config.DiscordWebhookPort)
	progressionNotifier := progression.NewNotifier(discordWebhookURL, deps.Config.StreamerbotWebhookURL)
	progressionNotifier.Subscribe(deps.EventBus)
	slog.Info(LogMsgProgressionNotifierInit,
		"discord_webhook", discordWebhookURL,
		"streamerbot_webhook", deps.Config.StreamerbotWebhookURL)

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
