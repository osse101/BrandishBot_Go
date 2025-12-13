package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
)

// HTTPServer handles internal HTTP requests
type HTTPServer struct {
	server *http.Server
	bot    *Bot
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(port string, bot *Bot) *HTTPServer {
	mux := http.NewServeMux()
	
	srv := &HTTPServer{
		server: &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
		bot: bot,
	}

	mux.HandleFunc("/admin/announce", srv.handleAnnounce)
	return srv
}

// Start starts the HTTP server
func (s *HTTPServer) Start() {
	go func() {
		slog.Info("Starting Discord internal HTTP server", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Discord internal HTTP server failed", "error", err)
		}
	}()
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error("Discord internal HTTP server shutdown failed", "error", err)
	}
}

type AnnounceRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

func (s *HTTPServer) handleAnnounce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AnnounceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use default color if not specified
	if req.Color == 0 {
		req.Color = 0x00FF00 // Green
	}

	embed := &discordgo.MessageEmbed{
		Title:       req.Title,
		Description: req.Description,
		Color:       req.Color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "System Update",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Send to dev channel
	if err := s.bot.SendDevMessage(embed); err != nil {
		slog.Error("Failed to send announcement", "error", err)
		http.Error(w, "Failed to send to Discord", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
