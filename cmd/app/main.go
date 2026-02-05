package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/osse101/BrandishBot_Go/docs/swagger"
	"github.com/osse101/BrandishBot_Go/internal/bootstrap"
	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/expedition"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/harvest"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/linking"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/prediction"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/scheduler"
	"github.com/osse101/BrandishBot_Go/internal/server"
	"github.com/osse101/BrandishBot_Go/internal/sse"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/streamerbot"
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

//nolint:gocyclo // main function setup is naturally complex
func main() {
	// Load configuration FIRST (single source of truth)
	cfg, err := config.Load()
	if err != nil {
		// Can't use structured logger yet, use basic logging
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	logFile, err := bootstrap.SetupLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
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

	// Initialize Event System
	eventBus, resilientPublisher, err := bootstrap.InitializeEventSystem(cfg)
	if err != nil {
		slog.Error("Failed to initialize event system", "error", err)
		os.Exit(1)
	}

	// Initialize all repositories
	repos := bootstrap.InitializeRepositories(dbPool, eventBus)

	// Initialize core services
	statsService := stats.NewService(repos.Stats)
	progressionService := progression.NewService(repos.Progression, repos.User, eventBus, resilientPublisher, nil)

	// Sync configuration files to database
	if err := bootstrap.SyncProgressionTree(context.Background(), repos.Progression); err != nil {
		slog.Error("Progression tree sync failed", "error", err)
		os.Exit(1)
	}

	itemRepo, err := bootstrap.SyncItems(context.Background(), dbPool)
	if err != nil {
		slog.Error("Items sync failed", "error", err)
		os.Exit(1)
	}

	if err := bootstrap.SyncRecipes(context.Background(), repos.Crafting, itemRepo); err != nil {
		slog.Error("Recipes sync failed", "error", err)
		os.Exit(1)
	}

	// Initialize Job service (needed by user, economy, crafting, gamble)
	jobService := job.NewService(repos.Job, progressionService, statsService, eventBus, resilientPublisher)

	// Initialize Worker Pool
	// Start with 5 workers as per plan
	workerPool := worker.NewPool(5, 100)
	workerPool.Start()
	defer workerPool.Stop()

	// Initialize Event Logger (needed by event handlers)
	eventLogService := eventlog.NewService(repos.EventLog)

	// Register all event handlers
	if err := bootstrap.RegisterEventHandlers(bootstrap.EventHandlerDependencies{
		EventBus:           eventBus,
		ProgressionService: progressionService,
		EventLogService:    eventLogService,
		Config:             cfg,
	}); err != nil {
		slog.Error("Failed to register event handlers", "error", err)
		os.Exit(1)
	}

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

	// Initialize progression state (ensure valid target on startup)
	if err := progressionService.InitializeProgressionState(context.Background()); err != nil {
		slog.Warn("Failed to initialize progression state", "error", err)
		// Don't exit - this is a non-critical error
	}

	// Initialize Lootbox Service
	lootboxSvc, err := lootbox.NewService(repos.User, progressionService, config.ConfigPathLootTables)
	if err != nil {
		slog.Error("Failed to initialize lootbox service", "error", err)
		os.Exit(1)
	}

	// Initialize Naming Resolver for item display names
	namingResolver, err := naming.NewResolver(config.ConfigPathItemAliases, config.ConfigPathItemThemes)

	if err != nil {
		slog.Error("Failed to initialize naming resolver", "error", err)
		os.Exit(1)
	}
	slog.Info("Naming resolver initialized")

	// Register all items with naming resolver for public name resolution
	allItems, err := repos.User.GetAllItems(context.Background())
	if err != nil {
		slog.Error("Failed to load items for naming resolver", "error", err)
		os.Exit(1)
	}
	for _, item := range allItems {
		namingResolver.RegisterItem(item.InternalName, item.PublicName)
	}
	slog.Info("Items registered with naming resolver", "count", len(allItems))

	// Initialize Cooldown Service
	cooldownSvc := cooldown.NewPostgresService(dbPool, cooldown.Config{
		DevMode: cfg.DevMode,
	}, progressionService)
	slog.Info("Cooldown service initialized", "dev_mode", cfg.DevMode)

	// Initialize services that depend on naming resolver
	economyService := economy.NewService(repos.Economy, jobService, namingResolver, progressionService)
	gambleService := gamble.NewService(repos.Gamble, eventBus, resilientPublisher, lootboxSvc, statsService, cfg.GambleJoinDuration, jobService, progressionService, namingResolver, nil)
	craftingService := crafting.NewService(repos.Crafting, jobService, statsService, namingResolver, progressionService)

	// Initialize services that depend on job service and naming resolver
	userService := user.NewService(repos.User, repos.Trap, statsService, jobService, lootboxSvc, namingResolver, cooldownSvc, eventBus, cfg.DevMode)

	// Initialize Harvest Service
	harvestService := harvest.NewService(repos.Harvest, repos.User, progressionService, jobService)
	slog.Info("Harvest service initialized")

	// Initialize Gamble Worker
	gambleWorker := worker.NewGambleWorker(gambleService)
	gambleWorker.Subscribe(eventBus)
	gambleWorker.Start() // Checks for existing active gamble on startup

	// Initialize Expedition Service and Worker
	expeditionConfig, err := expedition.LoadEncounterConfig(config.ConfigPathExpeditionEncounters)
	if err != nil {
		slog.Error("Failed to load expedition encounter config", "error", err)
		os.Exit(1)
	}
	expeditionService := expedition.NewService(
		repos.Expedition,
		eventBus,
		progressionService,
		jobService,
		userService,
		cooldownSvc,
		expeditionConfig,
		3*time.Minute,  // join duration
		15*time.Minute, // cooldown duration
	)
	expeditionWorker := worker.NewExpeditionWorker(expeditionService)
	expeditionWorker.Subscribe(eventBus)
	expeditionWorker.Start()
	slog.Info("Expedition service and worker initialized")

	// Initialize Daily Reset Worker
	dailyResetWorker := worker.NewDailyResetWorker(jobService, resilientPublisher)
	dailyResetWorker.Start()
	slog.Info("Daily reset worker initialized")

	// Initialize Linking service
	linkingService := linking.NewService(repos.Linking, userService)

	// Initialize Prediction service
	predictionService := prediction.NewService(
		progressionService,
		jobService,
		userService,
		statsService,
		eventBus,
		resilientPublisher,
	)
	slog.Info("Prediction service initialized")

	// Initialize SSE Hub for real-time event streaming
	sseHub := sse.NewHub()
	sseHub.Start()
	defer sseHub.Stop()

	// Register SSE subscriber to bridge internal events to SSE clients
	sseSubscriber := sse.NewSubscriber(sseHub, eventBus)
	sseSubscriber.Subscribe()
	slog.Info("SSE hub initialized")

	// Initialize Streamer.bot WebSocket client if enabled
	var sbClient *streamerbot.Client
	if cfg.StreamerbotEnabled && cfg.StreamerbotWebhookURL != "" {
		sbClient = streamerbot.NewClient(cfg.StreamerbotWebhookURL, "")
		sbClient.Start(context.Background())
		defer sbClient.Stop()

		// Register Streamer.bot subscriber to bridge internal events to DoAction commands
		sbSubscriber := streamerbot.NewSubscriber(sbClient, eventBus)
		sbSubscriber.Subscribe()
		slog.Info("Streamer.bot WebSocket client initialized", "url", cfg.StreamerbotWebhookURL)
	}

	srv := server.NewServer(cfg.Port, cfg.APIKey, cfg.TrustedProxies, dbPool, userService, economyService, craftingService, statsService, progressionService, gambleService, jobService, linkingService, harvestService, predictionService, expeditionService, namingResolver, eventBus, sseHub, repos.User)

	// Run server in a goroutine
	go func() {
		slog.Info("Starting server", "port", cfg.Port)
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Create a deadline for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perform graceful shutdown
	bootstrap.GracefulShutdown(shutdownCtx, bootstrap.ShutdownComponents{
		Server:             srv,
		ProgressionService: progressionService,
		UserService:        userService,
		EconomyService:     economyService,
		CraftingService:    craftingService,
		GambleService:      gambleService,
		PredictionService:  predictionService,
		GambleWorker:       gambleWorker,
		ExpeditionWorker:   expeditionWorker,
		DailyResetWorker:   dailyResetWorker,
		ResilientPublisher: resilientPublisher,
	})
}
