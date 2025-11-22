package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type Server struct {
	httpServer  *http.Server
	userService user.Service
}

// NewServer creates a new Server instance
func NewServer(port int, userService user.Service) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/register", handler.HandleRegisterUser(userService))
	mux.HandleFunc("/message/handle", handler.HandleMessageHandler(userService))
	mux.HandleFunc("/test", handler.HandleTest(userService))
	mux.HandleFunc("/user/item/add", handler.HandleAddItem(userService))
	mux.HandleFunc("/user/item/remove", handler.HandleRemoveItem(userService))
	mux.HandleFunc("/user/item/give", handler.HandleGiveItem(userService))
	mux.HandleFunc("/user/item/sell", handler.HandleSellItem(userService))
	mux.HandleFunc("/user/item/buy", handler.HandleBuyItem(userService))
	mux.HandleFunc("/user/item/use", handler.HandleUseItem(userService))
	mux.HandleFunc("/user/inventory", handler.HandleGetInventory(userService))
	mux.HandleFunc("/prices", handler.HandleGetPrices(userService))

	// Wrap mux with logging middleware
	loggedMux := loggingMiddleware(mux)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: loggedMux,
		},
		userService: userService,
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(w, r)

		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
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
