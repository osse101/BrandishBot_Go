package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/logger"
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
}

// NewServer creates a new Server instance
func NewServer(port int, apiKey string, dbPool database.Pool, userService user.Service, economyService economy.Service, craftingService crafting.Service, statsService stats.Service, progressionService progression.Service) *Server {
	mux := http.NewServeMux()
	
	// Health check routes
	mux.HandleFunc("/healthz", handler.HandleHealthz())
	mux.HandleFunc("/readyz", handler.HandleReadyz(dbPool))
	
	// User routes
	mux.HandleFunc("/user/register", handler.HandleRegisterUser(userService))
	mux.HandleFunc("/message/handle", handler.HandleMessageHandler(userService, progressionService))
	mux.HandleFunc("/test", handler.HandleTest(userService))
	mux.HandleFunc("/user/item/add", handler.HandleAddItem(userService))
	mux.HandleFunc("/user/item/remove", handler.HandleRemoveItem(userService))
	mux.HandleFunc("/user/item/give", handler.HandleGiveItem(userService))
	mux.HandleFunc("/user/item/sell", handler.HandleSellItem(economyService, progressionService))
	mux.HandleFunc("/user/item/buy", handler.HandleBuyItem(economyService, progressionService))
	mux.HandleFunc("/user/item/use", handler.HandleUseItem(userService, progressionService))
	mux.HandleFunc("/user/item/upgrade", handler.HandleUpgradeItem(craftingService, progressionService))
	mux.HandleFunc("/user/item/disassemble", handler.HandleDisassembleItem(craftingService, progressionService))
	mux.HandleFunc("/recipes", handler.HandleGetRecipes(craftingService))
	mux.HandleFunc("/user/inventory", handler.HandleGetInventory(userService))
	mux.HandleFunc("/user/search", handler.HandleSearch(userService, progressionService))
	mux.HandleFunc("/prices", handler.HandleGetPrices(economyService))
	
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
	
	// 2. Request size limit
	handler = RequestSizeLimitMiddleware(1 << 20)(handler) // 1MB limit
	
	// 3. Security logging with suspicious activity detection
	detector := NewSuspiciousActivityDetector()
	handler = SecurityLoggingMiddleware(detector)(handler)
	
	// 4. Authentication (outermost - validates first)
	handler = AuthMiddleware(apiKey)(handler)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: handler,
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
