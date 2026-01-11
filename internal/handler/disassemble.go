package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type DisassembleItemResponse struct {
	Message           string         `json:"message"`
	Outputs           map[string]int `json:"outputs"`
	QuantityProcessed int            `json:"quantity_processed"`
	IsPerfectSalvage  bool           `json:"is_perfect_salvage"`
	Multiplier        float64        `json:"multiplier"`
}

// HandleDisassembleItem handles disassembling items
// @Summary Disassemble item
// @Description Disassemble an item into materials
// @Tags crafting
// @Accept json
// @Produce json
// @Param request body CraftingActionRequest true "Disassemble details"
// @Success 200 {object} DisassembleItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/disassemble [post]
func HandleDisassembleItem(svc crafting.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		// Check if disassemble feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureDisassemble) {
			return
		}

		req, err := decodeCraftingRequest(r, "Disassemble item")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := svc.DisassembleItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.Item, req.Quantity)
		if err != nil {
			log.Error("Failed to disassemble item", "error", err, "username", req.Username, "item", req.Item)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Item disassembled successfully",
			"username", req.Username,
			"item", req.Item,
			"quantity_processed", result.QuantityProcessed,
			"outputs", result.Outputs)

		// Track engagement for disassembling
		trackCraftingEngagement(r.Context(), eventBus, req.Username, "item_disassembled", result.QuantityProcessed)

		// Publish item.disassembled event
		if err := publishCraftingEvent(r.Context(), eventBus, "item.disassembled", map[string]interface{}{
			"user_id":            req.Username,
			"item":               req.Item,
			"quantity_processed": result.QuantityProcessed,
			"materials_gained":   result.Outputs,
			"is_perfect_salvage": result.IsPerfectSalvage,
		}); err != nil {
			_ = err // Error already logged in publishCraftingEvent
		}

		// Construct user message
		// Optimization: Use strings.Builder and avoid fmt.Sprintf in loop
		var sb strings.Builder

		// Sort keys for deterministic output
		keys := make([]string, 0, len(result.Outputs))
		for k := range result.Outputs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for i, k := range keys {
			if i > 0 {
				sb.WriteString(", ")
			}
			v := result.Outputs[k]
			sb.WriteString(strconv.Itoa(v))
			sb.WriteString("x ")
			sb.WriteString(k)
		}
		outputStr := sb.String()

		message := fmt.Sprintf("Disassembled %d items into: %s", result.QuantityProcessed, outputStr)
		if result.IsPerfectSalvage {
			message = fmt.Sprintf("PERFECT SALVAGE! You efficiently recovered more materials! (+50%% Bonus): %s", outputStr)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, DisassembleItemResponse{
			Message:           message,
			Outputs:           result.Outputs,
			QuantityProcessed: result.QuantityProcessed,
			IsPerfectSalvage:  result.IsPerfectSalvage,
			Multiplier:        result.Multiplier,
		})
	}
}
