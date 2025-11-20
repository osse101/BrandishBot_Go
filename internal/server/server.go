package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/server/handler"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new Server instance
func NewServer(port int) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/execute", handler.ExecuteHandler)

	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
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
