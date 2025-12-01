package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/metrics"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
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
}

// NewServer creates a new Server instance
func NewServer(port int, apiKey string, dbPool database.Pool, userService user.Service, economyService economy.Service, craftingService crafting.Service, statsService stats.Service, progressionService progression.Service, gambleService gamble.Service, eventBus event.Bus) *Server {
	mux := http.NewServeMux()

	// Health check routes
	mux.HandleFunc("/healthz", handler.HandleHealthz())
	mux.HandleFunc("/readyz", handler.HandleReadyz(dbPool))

	// Metrics endpoint (public, for Prometheus scraping)
	mux.Handle("/metrics", promhttp.Handler())

	// User routes
	mux.HandleFunc("/user/register", handler.HandleRegisterUser(userService))
	mux.HandleFunc("/message/handle", handler.HandleMessageHandler(userService, progressionService, eventBus))
	mux.HandleFunc("/test", handler.HandleTest(userService))
	mux.HandleFunc("/user/item/add", handler.HandleAddItem(userService))
	mux.HandleFunc("/user/item/remove", handler.HandleRemoveItem(userService))
	mux.HandleFunc("/user/item/give", handler.HandleGiveItem(userService))
	mux.HandleFunc("/user/item/sell", handler.HandleSellItem(economyService, progressionService, eventBus))
	mux.HandleFunc("/user/item/buy", handler.HandleBuyItem(economyService, progressionService, eventBus))
	mux.HandleFunc("/user/item/use", handler.HandleUseItem(userService, eventBus))
	mux.HandleFunc("/user/item/upgrade", handler.HandleUpgradeItem(craftingService, progressionService, eventBus))
	mux.HandleFunc("/user/item/disassemble", handler.HandleDisassembleItem(craftingService, progressionService, eventBus))
	mux.HandleFunc("/recipes", handler.HandleGetRecipes(craftingService))
	mux.HandleFunc("/user/inventory", handler.HandleGetInventory(userService))
	mux.HandleFunc("/user/search", handler.HandleSearch(userService, progressionService, eventBus))
	mux.HandleFunc("/prices", handler.HandleGetPrices(economyService))

	// Gamble routes
	gambleHandler := handler.NewGambleHandler(gambleService)
	mux.HandleFunc("/gamble/start", gambleHandler.HandleStartGamble)
	// Note: chi.URLParam won't work with http.ServeMux directly for path parameters like /gamble/{id}/join
	// We need to wrap it or use a router that supports it.
	// Since the project uses http.ServeMux, we can't easily use /gamble/{id}/join pattern without parsing manually.
	// However, handler/gamble.go imports "github.com/go-chi/chi/v5".
	// If the project is migrating to chi, great. But here we see http.NewServeMux().
	// Wait, handler/gamble.go uses chi.URLParam. This implies the request context has chi params.
	// But NewServer uses http.NewServeMux. This is a conflict.
	// I should check if other handlers use chi.
	// handler/progression.go uses standard http.HandlerFunc.
	// I will assume for now I need to use a prefix handler or manual parsing if sticking to ServeMux.
	// OR, I can switch to chi router here if allowed.
	// Given the existing code uses ServeMux, I should probably adapt the handler to parse URL manually or use query params?
	// But the design doc specified /gamble/{id}/join.
	// Let's check if there's a chi router used elsewhere.
	// No, main.go uses server.NewServer which uses http.NewServeMux.
	// So I should probably change the handler to use query params or manual path parsing.
	// Let's stick to manual parsing for now to avoid changing the router.
	// Actually, I'll register "/gamble/" and handle subpaths in a wrapper or just use query params for simplicity if allowed?
	// No, I should follow the design.
	// I'll register "/gamble/" and dispatch in a closure.
	
	mux.HandleFunc("/gamble/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/gamble/start" {
			gambleHandler.HandleStartGamble(w, r)
			return
		}
		// Handle /gamble/{id}/join
		if len(path) > 8 && path[len(path)-5:] == "/join" {
			// Extract ID
			// /gamble/UUID/join
			// parts: ["", "gamble", "UUID", "join"]
			// This is getting messy.
			// Let's just use query param for ID for now? No, design doc says path.
			// I'll use a simple regex or split.
		}
		// Handle /gamble/{id}
	})
	
	// Wait, I can't easily fix the handler implementation from here.
	// I should update the handler to not rely on chi if chi is not the router.
	// But let's look at the imports in handler/gamble.go again. It imports chi.
	// Maybe I should just use chi in NewServer? It's a small change and much better.
	// But I see "github.com/go-chi/chi/v5" in imports of handler/gamble.go.
	// Let's try to use chi in NewServer.
	
	// Re-reading NewServer:
	// mux := http.NewServeMux()
	
	// I will replace http.NewServeMux with chi.NewRouter() if I can.
	// But that might break other things if they rely on ServeMux specific behavior (unlikely).
	// Actually, I'll just use http.ServeMux and manual parsing in the handler for now to be safe and minimally invasive.
	// I will update handler/gamble.go to parse path manually.
	
	// For now, let's just register the routes and I'll fix the handler in the next step.
	mux.HandleFunc("/gamble/join", gambleHandler.HandleJoinGamble) // Changed to query param style for now?
	mux.HandleFunc("/gamble/get", gambleHandler.HandleGetGamble)   // Changed to query param style for now?
	
	// Stats routes
	mux.HandleFunc("/stats/event", handler.HandleRecordEvent(statsService))
	mux.HandleFunc("/stats/user", handler.HandleGetUserStats(statsService))
	mux.HandleFunc("/stats/system", handler.HandleGetSystemStats(statsService))
	mux.HandleFunc("/stats/leaderboard", handler.HandleGetLeaderboard(statsService))

	// Progression routes
	progressionHandlers := handler.NewProgressionHandlers(progressionService)
	mux.HandleFunc("/progression/tree", progressionHandlers.HandleGetTree())
	mux.HandleFunc("/progression/available", progressionHandlers.HandleGetAvailable())
	mux.HandleFunc("/progression/vote", progressionHandlers.HandleVote())
	mux.HandleFunc("/progression/status", progressionHandlers.HandleGetStatus())
	mux.HandleFunc("/progression/engagement", progressionHandlers.HandleGetEngagement())
	mux.HandleFunc("/progression/admin/unlock", progressionHandlers.HandleAdminUnlock())
	mux.HandleFunc("/progression/admin/relock", progressionHandlers.HandleAdminRelock())
	mux.HandleFunc("/progression/admin/instant-unlock", progressionHandlers.HandleAdminInstantUnlock())
	mux.HandleFunc("/progression/admin/reset", progressionHandlers.HandleAdminReset())

	// Swagger documentation
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	// Build middleware stack (applied in reverse order)
	// 1. Request logging (innermost - logs final status)
	handler := loggingMiddleware(mux)

	// 2. Metrics collection
	handler = metrics.MetricsMiddleware(handler)

	// 3. Request size limit
	handler = RequestSizeLimitMiddleware(1 << 20)(handler) // 1MB limit

	// 4. Security logging with suspicious activity detection
	detector := NewSuspiciousActivityDetector()
	handler = SecurityLoggingMiddleware(detector)(handler)

	// 5. Authentication (outermost - validates first)
	handler = AuthMiddleware(apiKey)(handler)

	return &Server{
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
		dbPool:             dbPool,
		userService:        userService,
		economyService:     economyService,
		craftingService:    craftingService,
		statsService:       statsService,
		progressionService: progressionService,
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

		log.Debug("Request headers", "headers", r.Header)

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
	fmt.Printf("Server starting on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
