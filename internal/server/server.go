package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/features"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/linking"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/metrics"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type Server struct {
	httpServer         *http.Server
	dbPool             database.Pool
	userService        user.Service
	economyService     economy.Service
	craftingService    crafting.Service
	statsService       stats.Service
	progressionService progression.Service
	gambleService      gamble.Service
	jobService         job.Service
	linkingService     linking.Service
	namingResolver     naming.Resolver
}

// NewServer creates a new Server instance
func NewServer(port int, apiKey string, trustedProxies []string, dbPool database.Pool, userService user.Service, economyService economy.Service, craftingService crafting.Service, statsService stats.Service, progressionService progression.Service, gambleService gamble.Service, jobService job.Service, linkingService linking.Service, namingResolver naming.Resolver, eventBus event.Bus) *Server {
	r := chi.NewRouter()

	// Middleware stack
	// Chi middleware executes in order defined (outermost to innermost)
	detector := NewSuspiciousActivityDetector()

	r.Use(SecurityHeadersMiddleware())
	r.Use(AuthMiddleware(apiKey, trustedProxies, detector))
	r.Use(SecurityLoggingMiddleware(trustedProxies, detector))
	r.Use(RequestSizeLimitMiddleware(1 << 20)) // 1MB limit
	r.Use(metrics.MetricsMiddleware)
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
		featureLoader := features.NewLoader("configs/info")
		r.Get("/info", handler.HandleGetInfo(featureLoader))

		// User routes
		r.Route("/user", func(r chi.Router) {
			r.Post("/register", handler.HandleRegisterUser(userService))
			r.Get("/timeout", handler.HandleGetTimeout(userService))
			r.Get("/inventory", handler.HandleGetInventory(userService, progressionService))
			r.Get("/inventory-by-username", handler.HandleGetInventoryByUsername(userService))
			r.Post("/search", handler.HandleSearch(userService, progressionService, eventBus))

			r.Route("/item", func(r chi.Router) {
				r.Post("/add", handler.HandleAddItem(userService))
				r.Post("/add-by-username", handler.HandleAddItemByUsername(userService))
				r.Post("/remove", handler.HandleRemoveItem(userService))
				r.Post("/remove-by-username", handler.HandleRemoveItemByUsername(userService))
				r.Post("/give", handler.HandleGiveItem(userService))
				r.Post("/give-by-username", handler.HandleGiveItemByUsername(userService))
				r.Post("/sell", handler.HandleSellItem(economyService, progressionService, eventBus))
				r.Post("/buy", handler.HandleBuyItem(economyService, progressionService, eventBus))
				r.Post("/use", handler.HandleUseItem(userService, eventBus))
				r.Post("/use-by-username", handler.HandleUseItemByUsername(userService, eventBus))
				r.Post("/upgrade", handler.HandleUpgradeItem(craftingService, progressionService, eventBus))
				r.Post("/disassemble", handler.HandleDisassembleItem(craftingService, progressionService, eventBus))
			})
		})

		r.Post("/message/handle", handler.HandleMessageHandler(userService, progressionService, eventBus))
		r.Post("/test", handler.HandleTest(userService))
		r.Get("/recipes", handler.HandleGetRecipes(craftingService))

		r.Route("/prices", func(r chi.Router) {
			r.Get("/", handler.HandleGetPrices(economyService))
			r.Get("/buy", handler.HandleGetBuyPrices(economyService))
		})

		// Gamble routes
		gambleHandler := handler.NewGambleHandler(gambleService, progressionService)
		r.Route("/gamble", func(r chi.Router) {
			r.Post("/start", gambleHandler.HandleStartGamble)
			r.Post("/join", gambleHandler.HandleJoinGamble)
			r.Get("/get", gambleHandler.HandleGetGamble)
		})

		// Job routes
		jobHandler := handler.NewJobHandler(jobService)
		r.Get("/jobs", jobHandler.HandleGetAllJobs) // Handle /jobs exactly
		r.Route("/jobs", func(r chi.Router) {
			r.Get("/", jobHandler.HandleGetAllJobs) // Handle /jobs/ if needed
			r.Get("/user", jobHandler.HandleGetUserJobs)
			r.Post("/award-xp", jobHandler.HandleAwardXP)
			r.Get("/bonus", jobHandler.HandleGetJobBonus)
		})

		// Stats routes
		r.Route("/stats", func(r chi.Router) {
			r.Post("/event", handler.HandleRecordEvent(statsService))
			r.Get("/user", handler.HandleGetUserStats(statsService))
			r.Get("/system", handler.HandleGetSystemStats(statsService))
			r.Get("/leaderboard", handler.HandleGetLeaderboard(statsService))
		})

		// Progression routes
		progressionHandlers := handler.NewProgressionHandlers(progressionService)
		r.Route("/progression", func(r chi.Router) {
			r.Get("/tree", progressionHandlers.HandleGetTree())
			r.Get("/available", progressionHandlers.HandleGetAvailable())
			r.Post("/vote", progressionHandlers.HandleVote())
			r.Get("/status", progressionHandlers.HandleGetStatus())
			r.Get("/engagement", progressionHandlers.HandleGetEngagement())
			r.Get("/leaderboard", progressionHandlers.HandleGetContributionLeaderboard())
			r.Get("/session", progressionHandlers.HandleGetVotingSession())
			r.Get("/unlock-progress", progressionHandlers.HandleGetUnlockProgress())

			r.Route("/admin", func(r chi.Router) {
				r.Post("/unlock", progressionHandlers.HandleAdminUnlock())
				r.Post("/unlock-all", progressionHandlers.HandleAdminUnlockAll())
				r.Post("/relock", progressionHandlers.HandleAdminRelock())
				r.Post("/instant-unlock", progressionHandlers.HandleAdminInstantUnlock())
				r.Post("/start-voting", progressionHandlers.HandleAdminStartVoting())
				r.Post("/end-voting", progressionHandlers.HandleAdminEndVoting())
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

		// Admin routes
		adminJobHandler := handler.NewAdminJobHandler(jobService, userService)
		adminCacheHandler := handler.NewAdminCacheHandler(userService)
		r.Route("/admin", func(r chi.Router) {
			r.Post("/reload-aliases", handler.HandleReloadAliases(namingResolver))

			// Admin job routes
			r.Route("/job", func(r chi.Router) {
				r.Post("/award-xp", adminJobHandler.HandleAdminAwardXP)
			})

			// Admin progression routes
			r.Route("/progression", func(r chi.Router) {
				r.Post("/reload-weights", progressionHandlers.HandleAdminReloadWeights())
			})

			// Admin cache routes
			r.Route("/cache", func(r chi.Router) {
				r.Get("/stats", adminCacheHandler.HandleGetCacheStats)
			})
		})
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           r,
			ReadHeaderTimeout: 5 * time.Second,
		},
		dbPool:             dbPool,
		userService:        userService,
		economyService:     economyService,
		craftingService:    craftingService,
		statsService:       statsService,
		progressionService: progressionService,
		gambleService:      gambleService,
		jobService:         jobService,
		linkingService:     linkingService,
		namingResolver:     namingResolver,
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
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
	return rw.ResponseWriter.Write(b)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip logging for health check endpoints and metrics
		// Use HasPrefix to catch potential variations (e.g. /healthz/)
		if strings.HasPrefix(r.URL.Path, "/healthz") ||
			strings.HasPrefix(r.URL.Path, "/readyz") ||
			strings.HasPrefix(r.URL.Path, "/metrics") {
			next.ServeHTTP(w, r)
			return
		}

		// Generate unique request ID
		requestID := logger.GenerateRequestID()

		// Add request ID to context
		ctx := logger.WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		// Get scoped logger
		log := logger.FromContext(ctx)

		// Log request start with details
		log.Info("Request started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"content_length", r.ContentLength,
			"user_agent", r.UserAgent())

		// Sanitize headers for logging
		sanitizedHeaders := make(http.Header)
		for k, v := range r.Header {
			if strings.EqualFold(k, "X-API-Key") || strings.EqualFold(k, "Authorization") {
				sanitizedHeaders[k] = []string{"[REDACTED]"}
			} else {
				sanitizedHeaders[k] = v
			}
		}
		log.Debug("Request headers", "headers", sanitizedHeaders)

		// Wrap response writer to capture status code
		rw := newResponseWriter(w)

		// Process request
		next.ServeHTTP(rw, r)

		// Log request completion with metrics
		duration := time.Since(start)
		log.Info("Request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", duration.Milliseconds(),
			"duration", duration)
	})
}

// Start starts the server
func (s *Server) Start() error {
	slog.Default().Info("Server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
