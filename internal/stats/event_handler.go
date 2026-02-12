package stats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// EventHandler handles events related to stats
type EventHandler struct {
	service Service
}

// NewEventHandler creates a new stats event handler
func NewEventHandler(service Service) *EventHandler {
	return &EventHandler{
		service: service,
	}
}

// Register subscribes the handler to relevant events
func (h *EventHandler) Register(bus event.Bus) {
	bus.Subscribe(event.Type(domain.EventTypeItemUpgraded), h.HandleItemUpgraded)
	bus.Subscribe(event.Type(domain.EventTypeItemDisassembled), h.HandleItemDisassembled)
}

// HandleItemUpgraded handles item upgrade events to record stats
func (h *EventHandler) HandleItemUpgraded(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	var payload crafting.ItemUpgradedPayload
	if err := mapToStruct(evt.Payload, &payload); err != nil {
		return fmt.Errorf("failed to decode item upgraded payload: %w", err)
	}

	// Record critical success if masterwork
	if payload.IsMasterwork {
		err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventCraftingCriticalSuccess, map[string]interface{}{
			"item_name":         payload.ItemName,
			"original_quantity": payload.Quantity,
			"masterwork_count":  1, // Defaulting to 1 as we don't track exact iterations in payload, but IsMasterwork means at least 1.
			"bonus_quantity":    payload.BonusQuantity,
		})
		if err != nil {
			log.Warn("Failed to record crafting critical success stat", "error", err, "user_id", payload.UserID)
		}
	}

	// We could also record generic "Item Crafted" stat here if we wanted to
	return nil
}

// HandleItemDisassembled handles item disassemble events to record stats
func (h *EventHandler) HandleItemDisassembled(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	var payload crafting.ItemDisassembledPayload
	if err := mapToStruct(evt.Payload, &payload); err != nil {
		return fmt.Errorf("failed to decode item disassembled payload: %w", err)
	}

	if payload.IsPerfectSalvage {
		err := h.service.RecordUserEvent(ctx, payload.UserID, domain.EventCraftingPerfectSalvage, map[string]interface{}{
			"item_name":     payload.ItemName,
			"quantity":      payload.Quantity,
			"perfect_count": payload.PerfectSalvageCount,
			"multiplier":    payload.Multiplier,
		})
		if err != nil {
			log.Warn("Failed to record crafting perfect salvage stat", "error", err, "user_id", payload.UserID)
		}
	}

	return nil
}

// mapToStruct converts a map payload to a struct
func mapToStruct(input interface{}, output interface{}) error {
	switch v := input.(type) {
	case crafting.ItemUpgradedPayload:
		if out, ok := output.(*crafting.ItemUpgradedPayload); ok {
			*out = v
			return nil
		}
	case crafting.ItemDisassembledPayload:
		if out, ok := output.(*crafting.ItemDisassembledPayload); ok {
			*out = v
			return nil
		}
	}

	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}
