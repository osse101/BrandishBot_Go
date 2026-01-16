package handler

import (
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
			http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}

		var req InitiateRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Initiate link"); err != nil {
			return
		}

		token, err := h.svc.InitiateLink(r.Context(), req.Platform, req.PlatformID)
		if err != nil {
			log.Error("Failed to initiate link", "error", err)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"token":      token.Token,
			"expires_in": int(token.ExpiresAt.Sub(token.CreatedAt).Seconds()),
		})
	}
}

// HandleClaim handles POST /link/claim (Step 2)
func (h *LinkingHandlers) HandleClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}

		var req ClaimRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Claim link"); err != nil {
			return
		}

		token, err := h.svc.ClaimLink(r.Context(), req.Token, req.Platform, req.PlatformID)
		if err != nil {
			log.Warn("Failed to claim link", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"source_platform":       token.SourcePlatform,
			"awaiting_confirmation": true,
		})
	}
}

// HandleConfirm handles POST /link/confirm (Step 3)
func (h *LinkingHandlers) HandleConfirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}

		var req ConfirmRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Confirm link"); err != nil {
			return
		}

		result, err := h.svc.ConfirmLink(r.Context(), req.Platform, req.PlatformID)
		if err != nil {
			log.Warn("Failed to confirm link", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		respondJSON(w, http.StatusOK, result)
	}
}

// HandleUnlink handles POST /link/unlink
func (h *LinkingHandlers) HandleUnlink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}

		var req UnlinkRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Unlink"); err != nil {
			return
		}

		if !req.Confirm {
			// Step 1: Initiate unlink
			if err := h.svc.InitiateUnlink(r.Context(), req.Platform, req.PlatformID, req.TargetPlatform); err != nil {
				log.Error("Failed to initiate unlink", "error", err)
				statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
				return
			}

			respondJSON(w, http.StatusOK, map[string]interface{}{
				"awaiting_confirmation": true,
				"message":               MsgConfirmWithinSeconds,
			})
			return
		}

		// Step 2: Confirm unlink
		if err := h.svc.ConfirmUnlink(r.Context(), req.Platform, req.PlatformID, req.TargetPlatform); err != nil {
			log.Warn("Failed to confirm unlink", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": MsgPlatformUnlinked,
		})
	}
}

// HandleStatus handles GET /link/status
func (h *LinkingHandlers) HandleStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		platform, ok := GetQueryParam(r, w, "platform")
		if !ok {
			return
		}
		platformID, ok := GetQueryParam(r, w, "platform_id")
		if !ok {
			return
		}

		status, err := h.svc.GetStatus(r.Context(), platform, platformID)
		if err != nil {
			log.Error("Failed to get link status", "error", err)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		respondJSON(w, http.StatusOK, status)
	}
}
