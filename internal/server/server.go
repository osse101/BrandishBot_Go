package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/osse101/BrandishBot_Go/internal/admin"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/eventlog"
	"github.com/osse101/BrandishBot_Go/internal/expedition"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/harvest"
	"github.com/osse101/BrandishBot_Go/internal/info"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/linking"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/metrics"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/prediction"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/quest"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/scenario"
	"github.com/osse101/BrandishBot_Go/internal/slots"
	"github.com/osse101/BrandishBot_Go/internal/sse"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/subscription"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type Server struct {
	httpServer          *http.Server
	dbPool              database.Pool
	userService         user.Service
	economyService      economy.Service
	craftingService     crafting.Service
	statsService        stats.Service
	progressionService  progression.Service
	gambleService       gamble.Service
	jobService          job.Service
	linkingService      linking.Service
	harvestService      harvest.Service
	predictionService   prediction.Service
	expeditionService   expedition.Service
	questService        quest.Service
	subscriptionService subscription.Service
	slotsService        slots.Service
	namingResolver      naming.Resolver
	sseHub              *sse.Hub
	scenarioEngine      *scenario.Engine
	eventlogService     eventlog.Service
}

// NewServer creates a new Server instance
func NewServer(port int, apiKey string, trustedProxies []string, dbPool database.Pool, userService user.Service, economyService economy.Service, craftingService crafting.Service, statsService stats.Service, progressionService progression.Service, gambleService gamble.Service, jobService job.Service, linkingService linking.Service, harvestService harvest.Service, predictionService prediction.Service, expeditionService expedition.Service, questService quest.Service, subscriptionService subscription.Service, slotsService slots.Service, namingResolver naming.Resolver, eventBus event.Bus, sseHub *sse.Hub, userRepo repository.User, scenarioEngine *scenario.Engine, eventlogService eventlog.Service) *Server {
	r := chi.NewRouter()

	// Middleware stack
	// Chi middleware executes in order defined (outermost to innermost)
	detector := NewSuspiciousActivityDetector()

	r.Use(SecurityHeadersMiddleware())
	r.Use(AuthMiddleware(apiKey, trustedProxies, detector))
	r.Use(SecurityLoggingMiddleware(trustedProxies, detector))
	r.Use(RequestSizeLimitMiddleware(1 << 20)) // 1MB limit
	r.Use(metrics.Middleware)
	r.Use(loggingMiddleware)

	// Health check routes (unversioned)
	r.Get("/healthz", handler.HandleHealthz())
	r.Get("/readyz", handler.HandleReadyz(dbPool))

	// Version endpoint (public, for deployment verification)
	r.Get("/version", handler.HandleVersion())

	// Metrics endpoint (public, for Prometheus scraping)
	r.Handle("/metrics", promhttp.Handler())

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Info endpoint
		infoLoader := info.NewLoader("configs/info")
		r.Get("/info", handler.HandleGetInfo(infoLoader))

		// User routes
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", handler.HandleRegisterUser(userService))
			r.Get("/timeout", handler.HandleGetTimeout(userService))
			r.Put("/timeout", handler.HandleSetTimeout(userService))
			r.Get("/inventory", handler.HandleGetInventory(userService, progressionService))
			r.Get("/inventory-by-username", handler.HandleGetInventoryByUsername(userService))
			r.Post("/search", handler.HandleSearch(userService, progressionService, eventBus))

			r.Route("/item", func(r chi.Router) {
				r.Post("/add", handler.HandleAddItemByUsername(userService))
				r.Post("/remove", handler.HandleRemoveItemByUsername(userService))
				r.Post("/give", handler.HandleGiveItem(userService))
				r.Post("/sell", handler.HandleSellItem(economyService, progressionService, eventBus))
				r.Post("/buy", handler.HandleBuyItem(economyService, progressionService, eventBus))
				r.Post("/use", handler.HandleUseItem(userService, progressionService, eventBus))
				r.Post("/upgrade", handler.HandleUpgradeItem(craftingService, progressionService, eventBus))
				r.Post("/disassemble", handler.HandleDisassembleItem(craftingService, progressionService, eventBus))
			})
		})

		r.Post("/message/handle", handler.HandleMessageHandler(userService, progressionService, eventBus))
		r.Post("/test", handler.HandleTest(userService))

		// Crafting routes
		craftingHandler := handler.NewCraftingHandler(craftingService, userRepo)
		r.Get("/recipes", craftingHandler.HandleGetRecipes())

		r.Route("/prices", func(r chi.Router) {
			r.Get("/", handler.HandleGetPrices(economyService))
			r.Get("/buy", handler.HandleGetBuyPrices(economyService))
		})

		// Gamble routes
		gambleHandler := handler.NewGambleHandler(gambleService, progressionService, eventBus)
		r.Route("/gamble", func(r chi.Router) {
			r.Post("/start", gambleHandler.HandleStartGamble)
			r.Post("/join", gambleHandler.HandleJoinGamble)
			r.Get("/get", gambleHandler.HandleGetGamble)
			r.Get("/active", gambleHandler.HandleGetActiveGamble)
		})

		// Expedition routes
		expeditionHandler := handler.NewExpeditionHandler(expeditionService, progressionService)
		r.Route("/expedition", func(r chi.Router) {
			r.Post("/start", expeditionHandler.HandleStart)
			r.Post("/join", expeditionHandler.HandleJoin)
			r.Get("/get", expeditionHandler.HandleGet)
			r.Get("/active", expeditionHandler.HandleGetActive)
			r.Get("/journal", expeditionHandler.HandleGetJournal)
			r.Get("/status", expeditionHandler.HandleGetStatus)
		})

		// Slots routes
		slotsHandler := handler.NewSlotsHandler(slotsService, progressionService)
		r.Route("/slots", func(r chi.Router) {
			r.Post("/spin", slotsHandler.HandleSpinSlots)
		})

		// Harvest routes
		harvestHandler := handler.NewHarvestHandler(harvestService)
		r.Post("/harvest", harvestHandler.Harvest)

		// Job routes
		jobHandler := handler.NewJobHandler(jobService, userRepo)
		r.Route("/jobs", func(r chi.Router) {
			r.Get("/user", jobHandler.HandleGetUserJobs)
			r.Post("/award-xp", jobHandler.HandleAwardXP)
		})

		// Stats routes
		statsHandler := handler.NewStatsHandler(statsService, userRepo)
		r.Route("/stats", func(r chi.Router) {
			r.Post("/event", handler.HandleRecordEvent(statsService))
			r.Get("/user", statsHandler.HandleGetUserStats())
			r.Get("/system", handler.HandleGetSystemStats(statsService))
			r.Get("/leaderboard", handler.HandleGetLeaderboard(statsService))
		})

		// Quest routes
		questHandler := handler.NewQuestHandler(questService, progressionService)
		r.Route("/quests", func(r chi.Router) {
			r.Get("/active", questHandler.GetActiveQuests)
			r.Get("/progress", questHandler.GetUserQuestProgress)
			r.Post("/claim", questHandler.ClaimQuestReward)
		})

		// Progression routes
		progressionHandlers := handler.NewProgressionHandlers(progressionService)
		r.Route("/progression", func(r chi.Router) {
			r.Get("/tree", progressionHandlers.HandleGetTree())
			r.Get("/available", progressionHandlers.HandleGetAvailable())
			r.Post("/vote", progressionHandlers.HandleVote())
			r.Get("/status", progressionHandlers.HandleGetStatus())
			r.Get("/engagement", progressionHandlers.HandleGetEngagement())
			r.Get("/engagement-by-username", progressionHandlers.HandleGetEngagementByUsername())
			r.Get("/leaderboard", progressionHandlers.HandleGetContributionLeaderboard())
			r.Get("/session", progressionHandlers.HandleGetVotingSession())
			r.Get("/unlock-progress", progressionHandlers.HandleGetUnlockProgress())

			r.Route("/admin", func(r chi.Router) {
				r.Post("/unlock", progressionHandlers.HandleAdminUnlock())
				r.Post("/unlock-all", progressionHandlers.HandleAdminUnlockAll())
				r.Post("/relock", progressionHandlers.HandleAdminRelock())
				r.Post("/instant-unlock", progressionHandlers.HandleAdminInstantUnlock())
				r.Post("/start-voting", progressionHandlers.HandleAdminStartVoting())
				r.Post("/end-voting", progressionHandlers.HandleAdminEndVoting())            // Freezes vote
				r.Post("/force-end-voting", progressionHandlers.HandleAdminForceEndVoting()) // Ends vote immediately
				r.Post("/reset", progressionHandlers.HandleAdminReset())
				r.Post("/contribution", progressionHandlers.HandleAdminAddContribution())
			})
		})

		// Linking routes
		linkingHandlers := handler.NewLinkingHandlers(linkingService)
		r.Route("/link", func(r chi.Router) {
			r.Post("/initiate", linkingHandlers.HandleInitiate())
			r.Post("/claim", linkingHandlers.HandleClaim())
			r.Post("/confirm", linkingHandlers.HandleConfirm())
			r.Post("/unlink", linkingHandlers.HandleUnlink())
			r.Get("/status", linkingHandlers.HandleStatus())
		})

		// Subscription routes
		subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
		r.Route("/subscriptions", func(r chi.Router) {
			r.Post("/event", subscriptionHandler.HandleSubscriptionEvent)
			r.Get("/user", subscriptionHandler.HandleGetUserSubscription)
		})

		// Prediction routes
		predictionHandlers := handler.NewPredictionHandlers(predictionService)
		r.Post("/prediction", predictionHandlers.HandleProcessOutcome())

		// SSE events endpoint
		if sseHub != nil {
			r.Get("/events", sse.Handler(sseHub))
		}

		// Admin routes
		adminJobHandler := handler.NewAdminJobHandler(jobService, userService)
		adminDailyResetHandler := handler.NewAdminDailyResetHandler(jobService)
		adminCacheHandler := handler.NewAdminCacheHandler(userService)
		adminMetricsHandler := handler.NewAdminMetricsHandler(sseHub)
		adminUserHandler := handler.NewAdminUserHandler(userRepo)
		adminEventsHandler := handler.NewAdminEventsHandler(eventlogService)
		adminSSEHandler := handler.NewAdminSSEHandler(sseHub)
		r.Route("/admin", func(r chi.Router) {
			r.Get("/metrics", adminMetricsHandler.HandleGetMetrics)
			r.Post("/sse/broadcast", adminSSEHandler.HandleBroadcast)

			// User management
			r.Route("/users", func(r chi.Router) {
				r.Get("/lookup", adminUserHandler.HandleUserLookup)
				r.Get("/recent", adminUserHandler.HandleGetRecentUsers)
			})

			// Autocomplete lists
			r.Get("/items", adminUserHandler.HandleGetItems)
			r.Get("/jobs", adminUserHandler.HandleGetJobs)

			// Event log
			r.Get("/events", adminEventsHandler.HandleGetEvents)
			r.Post("/reload-aliases", handler.HandleReloadAliases(namingResolver))

			// Admin timeout routes
			r.Route("/timeout", func(r chi.Router) {
				r.Post("/clear", handler.HandleAdminClearTimeout(userService))
			})

			// Admin job routes
			r.Route("/jobs", func(r chi.Router) {
				r.Post("/award-xp", adminJobHandler.HandleAdminAwardXP)
				r.Post("/reset-daily-xp", adminDailyResetHandler.HandleManualReset)
				r.Get("/reset-status", adminDailyResetHandler.HandleGetResetStatus)
			})

			// Admin progression routes
			r.Route("/progression", func(r chi.Router) {
				r.Post("/reload-weights", progressionHandlers.HandleAdminReloadWeights())
			})

			// Admin cache routes
			r.Route("/cache", func(r chi.Router) {
				r.Get("/stats", adminCacheHandler.HandleGetCacheStats)
			})

			// Admin scenario simulation routes
			if scenarioEngine != nil {
				scenarioHandler := handler.NewScenarioHandler(scenarioEngine)
				r.Route("/simulate", func(r chi.Router) {
					r.Get("/capabilities", scenarioHandler.HandleGetCapabilities())
					r.Get("/scenarios", scenarioHandler.HandleGetScenarios())
					r.Get("/scenario", scenarioHandler.HandleGetScenario())
					r.Post("/run", scenarioHandler.HandleRunScenario())
					r.Post("/run-custom", scenarioHandler.HandleRunCustomScenario())
				})
			}
		})
	})

	// Admin dashboard (embedded SPA)
	r.Handle("/admin", http.RedirectHandler("/admin/", http.StatusMovedPermanently))
	r.Handle("/admin/*", http.StripPrefix("/admin", admin.Handler()))

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           r,
			ReadHeaderTimeout: 5 * time.Second,
		},
		dbPool:              dbPool,
		userService:         userService,
		economyService:      economyService,
		craftingService:     craftingService,
		statsService:        statsService,
		progressionService:  progressionService,
		gambleService:       gambleService,
		jobService:          jobService,
		linkingService:      linkingService,
		harvestService:      harvestService,
		predictionService:   predictionService,
		expeditionService:   expeditionService,
		questService:        questService,
		subscriptionService: subscriptionService,
		slotsService:        slotsService,
		namingResolver:      namingResolver,
		sseHub:              sseHub,
		scenarioEngine:      scenarioEngine,
		eventlogService:     eventlogService,
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code and error message
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	written      bool
	errorMessage string
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // default status
	}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}

	// Capture error message from JSON error responses (status >= 400)
	if rw.statusCode >= 400 && rw.errorMessage == "" && len(b) > 0 {
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(b, &errorResp); err == nil && errorResp.Error != "" {
			rw.errorMessage = errorResp.Error
		}
	}

	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip logging for health check endpoints, metrics, and quiet paths
		for _, path := range PublicPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}
		for _, path := range QuietPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Generate unique request ID
		requestID := logger.GenerateRequestID()

		// Add request ID to context
		ctx := logger.WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		// Get scoped logger
		log := logger.FromContext(ctx)

		// Log request start with details
		log.Info(LogMsgRequestStarted,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"content_length", r.ContentLength,
			"user_agent", r.UserAgent())

		// Sanitize headers for logging
		sanitizedHeaders := make(http.Header)
		for k, v := range r.Header {
			if strings.EqualFold(k, HeaderAPIKey) || strings.EqualFold(k, HeaderAuthorization) {
				sanitizedHeaders[k] = []string{RedactedValue}
			} else {
				sanitizedHeaders[k] = v
			}
		}
		log.Debug(LogMsgRequestHeaders, "headers", sanitizedHeaders)

		// Wrap response writer to capture status code
		rw := newResponseWriter(w)

		// Process request
		next.ServeHTTP(rw, r)

		// Log request completion with metrics
		duration := time.Since(start)
		logFields := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"duration", duration,
		}
		if rw.errorMessage != "" {
			logFields = append(logFields, "error", rw.errorMessage)
		}
		log.Info(LogMsgRequestCompleted, logFields...)
	})
}

// Start starts the server
func (s *Server) Start() error {
	slog.Default().Info(LogMsgServerStarting, "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
