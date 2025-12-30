package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/linking"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// LinkingHandlers contains handlers for account linking
type LinkingHandlers struct {
	svc linking.Service
}

// NewLinkingHandlers creates new linking handlers
func NewLinkingHandlers(svc linking.Service) *LinkingHandlers {
	return &LinkingHandlers{svc: svc}
}

// InitiateRequest is the request body for initiating a link
type InitiateRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
}

// ClaimRequest is the request body for claiming a link
type ClaimRequest struct {
	Token      string `json:"token"`
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
}

// ConfirmRequest is the request body for confirming a link
type ConfirmRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
}

// UnlinkRequest is the request body for unlinking
type UnlinkRequest struct {
	Platform       string `json:"platform"`
	PlatformID     string `json:"platform_id"`
	TargetPlatform string `json:"target_platform"`
	Confirm        bool   `json:"confirm"`
}

// HandleInitiate handles POST /link/initiate (Step 1)
func (h *LinkingHandlers) HandleInitiate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req InitiateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		token, err := h.svc.InitiateLink(r.Context(), req.Platform, req.PlatformID)
		if err != nil {
			log.Error("Failed to initiate link", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"token":      token.Token,
			"expires_in": int(token.ExpiresAt.Sub(token.CreatedAt).Seconds()),
		}); err != nil {
			log.Error("Failed to encode response", "error", err)
		}
	}
}

// HandleClaim handles POST /link/claim (Step 2)
func (h *LinkingHandlers) HandleClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ClaimRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		token, err := h.svc.ClaimLink(r.Context(), req.Token, req.Platform, req.PlatformID)
		if err != nil {
			log.Warn("Failed to claim link", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"source_platform":       token.SourcePlatform,
			"awaiting_confirmation": true,
		}); err != nil {
			log.Error("Failed to encode response", "error", err)
		}
	}
}

// HandleConfirm handles POST /link/confirm (Step 3)
func (h *LinkingHandlers) HandleConfirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ConfirmRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		result, err := h.svc.ConfirmLink(r.Context(), req.Platform, req.PlatformID)
		if err != nil {
			log.Warn("Failed to confirm link", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Error("Failed to encode response", "error", err)
		}
	}
}

// HandleUnlink handles POST /link/unlink
func (h *LinkingHandlers) HandleUnlink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req UnlinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if !req.Confirm {
			// Step 1: Initiate unlink
			if err := h.svc.InitiateUnlink(r.Context(), req.Platform, req.PlatformID, req.TargetPlatform); err != nil {
				log.Error("Failed to initiate unlink", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"awaiting_confirmation": true,
				"message":               "Confirm within 60 seconds",
			}); err != nil {
				log.Error("Failed to encode response", "error", err)
			}
			return
		}

		// Step 2: Confirm unlink
		if err := h.svc.ConfirmUnlink(r.Context(), req.Platform, req.PlatformID, req.TargetPlatform); err != nil {
			log.Warn("Failed to confirm unlink", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Platform unlinked",
		}); err != nil {
			log.Error("Failed to encode response", "error", err)
		}
	}
}

// HandleStatus handles GET /link/status
func (h *LinkingHandlers) HandleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		platform := r.URL.Query().Get("platform")
		platformID := r.URL.Query().Get("platform_id")

		if platform == "" || platformID == "" {
			http.Error(w, "Missing platform or platform_id", http.StatusBadRequest)
			return
		}

		status, err := h.svc.GetStatus(r.Context(), platform, platformID)
		if err != nil {
			log.Error("Failed to get link status", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(status); err != nil {
			log.Error("Failed to encode response", "error", err)
		}
	}
}
