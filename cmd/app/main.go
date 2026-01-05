package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/osse101/BrandishBot_Go/docs/swagger"
	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/linking"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/metrics"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/scheduler"
	"github.com/osse101/BrandishBot_Go/internal/server"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/osse101/BrandishBot_Go/internal/worker"
)

// @title BrandishBot API
// @version 1.0
// @description API for BrandishBot game engine - inventory, crafting, economy, and stats management
// @contact.name API Support
// @contact.url https://github.com/osse101/BrandishBot_Go
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

func main() {
	// Load configuration FIRST (single source of truth)
	cfg, err := config.Load()
	if err != nil {
		// Can't use structured logger yet, use basic logging
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging directory and file
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create logs directory: %v", err))
	}

	// Cleanup old logs
	cleanupLogs(cfg.LogDir)

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logFileName := filepath.Join(cfg.LogDir, fmt.Sprintf("session_%s.log", timestamp))

	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}
	defer logFile.Close()

	// Initialize logger with MultiWriter (stdout + file)
	mw := io.MultiWriter(os.Stdout, logFile)

	var level slog.Level
	switch strings.ToUpper(cfg.LogLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	slog.Info("Logging initialized", "level", level)
	slog.Info("Starting BrandishBot",
		"environment", cfg.Environment,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
		"version", cfg.Version)

	slog.Debug("Configuration loaded",
		"db_host", cfg.DBHost,
		"db_port", cfg.DBPort,
		"db_name", cfg.DBName,
		"port", cfg.Port)
	// Connect to database with retry logic
	dbPool, err := database.NewPool(cfg.GetDBConnString(), cfg.DBMaxConns, cfg.DBMaxConnIdleTime, cfg.DBMaxConnLifetime)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		slog.Error("Database connection failed",
			"host", cfg.DBHost,
			"port", cfg.DBPort,
			"database", cfg.DBName,
			"user", cfg.DBUser)
		slog.Info("💡 Hint: If using Docker, ensure the database is running:")
		slog.Info("   Run: ./scripts/check_db.sh")
		slog.Info("   Or: docker-compose up -d db")
		os.Exit(1)
	}
	defer dbPool.Close()

	userRepo := postgres.NewUserRepository(dbPool)

	statsRepo := postgres.NewStatsRepository(dbPool)
	statsService := stats.NewService(statsRepo)

	// Initialize Event Bus
	eventBus := event.NewMemoryBus()

	// Initialize Resilient Publisher for event retry logic
	maxRetries := cfg.EventMaxRetries
	if maxRetries == 0 {
		maxRetries = 5 // Default to 5 retries
	}
	retryDelay := cfg.EventRetryDelay
	if retryDelay == 0 {
		retryDelay = 2 * time.Second // Default to 2s base delay
	}
	deadLetterPath := cfg.EventDeadLetterPath
	if deadLetterPath == "" {
		deadLetterPath = "logs/event_deadletter.jsonl" // Default path
	}

	// Ensure dead-letter directory exists
	if err := os.MkdirAll(filepath.Dir(deadLetterPath), 0755); err != nil {
		slog.Error("Failed to create dead-letter directory", "error", err)
		os.Exit(1)
	}

	resilientPublisher, err := event.NewResilientPublisher(eventBus, maxRetries, retryDelay, deadLetterPath)
	if err != nil {
		slog.Error("Failed to create resilient publisher", "error", err)
		os.Exit(1)
	}
	slog.Info("Resilient event publisher initialized",
		"max_retries", maxRetries,
		"retry_delay", retryDelay,
		"deadletter_path", deadLetterPath)

	progressionRepo := postgres.NewProgressionRepository(dbPool)
	progressionService := progression.NewService(progressionRepo, eventBus)

	// Optional: Sync progression tree from JSON configuration
	if cfg.SyncProgressionTree {
		slog.Info("Syncing progression tree from JSON config...")
		treeLoader := progression.NewTreeLoader()

		treeConfig, err := treeLoader.Load("configs/progression_tree.json")
		if err != nil {
			slog.Error("Failed to load progression tree config", "error", err)
			os.Exit(1)
		}

		if err := treeLoader.Validate(treeConfig); err != nil {
			slog.Error("Invalid progression tree config", "error", err)
			os.Exit(1)
		}

		// Sync to database (progressionRepo implements NodeInserter/NodeUpdater)
		syncResult, err := treeLoader.SyncToDatabase(context.Background(), treeConfig, progressionRepo)
		if err != nil {
			slog.Error("Failed to sync progression tree to database", "error", err)
			os.Exit(1)
		}

		slog.Info("Progression tree synced successfully",
			"inserted", syncResult.NodesInserted,
			"updated", syncResult.NodesUpdated,
			"skipped", syncResult.NodesSkipped,
			"auto_unlocked", syncResult.AutoUnlocked)
	}

	// Initialize Job service (needed by user, economy, crafting, gamble)
	jobRepo := postgres.NewJobRepository(dbPool)
	jobService := job.NewService(jobRepo, progressionService, statsService, eventBus, resilientPublisher)

	// Initialize services that depend on job service
	economyService := economy.NewService(userRepo, jobService)
	craftingService := crafting.NewService(userRepo, jobService, statsService)

	// Initialize Worker Pool
	// Start with 5 workers as per plan
	workerPool := worker.NewPool(5, 100)
	workerPool.Start()
	defer workerPool.Stop()

	// Register Event Handlers
	progressionHandler := progression.NewEventHandler(progressionService)
	progressionHandler.Register(eventBus)

	// Initialize and register Progression Notifier
	discordWebhookURL := fmt.Sprintf("http://discord:%s/admin/announce", cfg.DiscordWebhookPort)
	progressionNotifier := progression.NewNotifier(discordWebhookURL, cfg.StreamerbotWebhookURL)
	progressionNotifier.Subscribe(eventBus)
	slog.Info("Progression notifier initialized",
		"discord_webhook", discordWebhookURL,
		"streamerbot_webhook", cfg.StreamerbotWebhookURL)

	// Register Metrics Collector
	metricsCollector := metrics.NewEventMetricsCollector()
	if err := metricsCollector.Register(eventBus); err != nil {
		slog.Error("Failed to register metrics collector", "error", err)
		os.Exit(1)
	}
	slog.Info("Metrics collector registered")

	// Initialize Event Logger
	eventLogRepo := postgres.NewEventLogRepository(dbPool)
	eventLogService := eventlog.NewService(eventLogRepo)
	if err := eventLogService.Subscribe(eventBus); err != nil {
		slog.Error("Failed to subscribe event logger", "error", err)
		os.Exit(1)
	}
	slog.Info("Event logger initialized")

	// Initialize Job Scheduler
	jobScheduler := scheduler.New(workerPool)
	// Schedule event log cleanup every 24 hours
	cleanupJob := eventlog.NewCleanupJob(eventLogService, 10)
	jobScheduler.Schedule(24*time.Hour, cleanupJob)
	// Schedule progression unlock checker every 30 minutes
	unlockCheckerJob := progression.NewUnlockCheckerJob(progressionService)
	jobScheduler.Schedule(30*time.Minute, unlockCheckerJob)
	jobScheduler.Start()
	defer jobScheduler.Stop()
	slog.Info("Job scheduler initialized")

	// Initialize Gamble components
	gambleRepo := postgres.NewGambleRepository(dbPool)

	// Initialize Lootbox Service (reusing userRepo for item data)
	lootboxSvc, err := lootbox.NewService(userRepo, "configs/loot_tables.json")
	if err != nil {
		slog.Error("Failed to initialize lootbox service", "error", err)
		os.Exit(1)
	}

	// Initialize Naming Resolver for item display names
	namingResolver, err := naming.NewResolver("configs/items/aliases.json", "configs/items/themes.json")
	if err != nil {
		slog.Error("Failed to initialize naming resolver", "error", err)
		os.Exit(1)
	}
	slog.Info("Naming resolver initialized")

	// Initialize Cooldown Service
	cooldownSvc := cooldown.NewPostgresService(dbPool, cooldown.Config{
		DevMode: cfg.DevMode,
	})
	slog.Info("Cooldown service initialized", "dev_mode", cfg.DevMode)

	gambleService := gamble.NewService(gambleRepo, eventBus, lootboxSvc, statsService, cfg.GambleJoinDuration, jobService)

	// Initialize services that depend on job service
	userService := user.NewService(userRepo, statsService, jobService, lootboxSvc, namingResolver, cooldownSvc, cfg.DevMode)

	// Initialize Gamble Worker
	gambleWorker := worker.NewGambleWorker(gambleService)
	gambleWorker.Subscribe(eventBus)
	gambleWorker.Start() // Checks for existing active gamble on startup

	// Initialize Linking service
	linkingRepo := postgres.NewLinkingRepository(dbPool)
	linkingService := linking.NewService(linkingRepo, userService)

	srv := server.NewServer(cfg.Port, cfg.APIKey, cfg.TrustedProxies, dbPool, userService, economyService, craftingService, statsService, progressionService, gambleService, jobService, linkingService, namingResolver, eventBus)

	// Run server in a goroutine
	go func() {
		slog.Info("Starting server", "port", cfg.Port)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")

	// Create a deadline for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Gracefully shutdown the server
	if err := srv.Stop(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	// Gracefully shutdown services
	if err := progressionService.Shutdown(shutdownCtx); err != nil {
		slog.Error("Progression service shutdown failed", "error", err)
	}
	if err := userService.Shutdown(shutdownCtx); err != nil {
		slog.Error("User service shutdown failed", "error", err)
	}
	if err := economyService.Shutdown(shutdownCtx); err != nil {
		slog.Error("Economy service shutdown failed", "error", err)
	}
	if err := craftingService.Shutdown(shutdownCtx); err != nil {
		slog.Error("Crafting service shutdown failed", "error", err)
	}
	if err := gambleService.Shutdown(shutdownCtx); err != nil {
		slog.Error("Gamble service shutdown failed", "error", err)
	}

	// Shutdown resilient publisher last to flush pending events
	slog.Info("Shutting down event publisher...")
	if err := resilientPublisher.Shutdown(shutdownCtx); err != nil {
		slog.Error("Resilient publisher shutdown failed", "error", err)
	}

	slog.Info("Server stopped")
}

func cleanupLogs(logDir string) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	var logFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			logFiles = append(logFiles, entry)
		}
	}

	if len(logFiles) >= 10 {
		// Delete oldest files until we have 9 left
		toDelete := len(logFiles) - 9
		for i := 0; i < toDelete; i++ {
			err := os.Remove(filepath.Join(logDir, logFiles[i].Name()))
			if err != nil {
				fmt.Printf("Failed to delete old log file %s: %v\n", logFiles[i].Name(), err)
			}
		}
	}
}
