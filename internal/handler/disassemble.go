package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type DisassembleItemRequest struct {
	Platform string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	Item     string `json:"item" validate:"required,max=100"`
	Quantity int    `json:"quantity" validate:"min=1,max=10000"`
}

type DisassembleItemResponse struct {
	Outputs           map[string]int `json:"outputs"`
	QuantityProcessed int            `json:"quantity_processed"`
}

// HandleDisassembleItem handles disassembling items
// @Summary Disassemble item
// @Description Disassemble an item into materials
// @Tags crafting
// @Accept json
// @Produce json
// @Param request body DisassembleItemRequest true "Disassemble details"
// @Success 200 {object} DisassembleItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/disassemble [post]
// HandleDisassembleItem handles disassembling items
// @Summary Disassemble item
// @Description Disassemble an item into materials
// @Tags crafting
// @Accept json
// @Produce json
// @Param request body DisassembleItemRequest true "Disassemble details"
// @Success 200 {object} DisassembleItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/disassemble [post]
func HandleDisassembleItem(svc crafting.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		// Check if disassemble feature is unlocked
		unlocked, err := progressionSvc.IsFeatureUnlocked(r.Context(), progression.FeatureDisassemble)
		if err != nil {
			log.Error("Failed to check feature unlock status", "error", err)
			http.Error(w, "Failed to check feature availability", http.StatusInternalServerError)
			return
		}
		if !unlocked {
			log.Warn("Disassemble feature is locked")
			http.Error(w, "Disassemble feature is not yet unlocked", http.StatusForbidden)
			return
		}

		var req DisassembleItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode disassemble item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Disassemble item request",
			"username", req.Username,
			"item", req.Item,
			"quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		outputs, quantityProcessed, err := svc.DisassembleItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.Item, req.Quantity)
		if err != nil {
			log.Error("Failed to disassemble item", "error", err, "username", req.Username, "item", req.Item)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Item disassembled successfully",
			"username", req.Username,
			"item", req.Item,
			"quantity_processed", quantityProcessed,
			"outputs", outputs)

		// Publish item.disassembled event
		if err := eventBus.Publish(r.Context(), event.Event{
			Type: "item.disassembled",
			Payload: map[string]interface{}{
				"user_id":             req.Username,
				"item":                req.Item,
				"quantity_processed":  quantityProcessed,
				"materials_gained":    outputs,
			},
		}); err != nil {
			log.Error("Failed to publish item.disassembled event", "error", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, DisassembleItemResponse{
			Outputs:           outputs,
			QuantityProcessed: quantityProcessed,
		})
	}
}
