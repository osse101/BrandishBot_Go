package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// EventHandler handles events related to jobs
type EventHandler struct {
	service Service
}

// NewEventHandler creates a new job event handler
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

// HandleItemUpgraded handles item upgrade events to award Blacksmith XP
func (h *EventHandler) HandleItemUpgraded(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	// Decode payload
	var payload crafting.ItemUpgradedPayload
	if err := mapToStruct(evt.Payload, &payload); err != nil {
		return fmt.Errorf("failed to decode item upgraded payload: %w", err)
	}

	// Award XP
	totalXP := BlacksmithXPPerItem * payload.Quantity
	metadata := map[string]interface{}{
		"source":        "upgrade",
		"item_name":     payload.ItemName,
		"quantity":      payload.Quantity,
		"is_masterwork": payload.IsMasterwork,
	}

	result, err := h.service.AwardXP(ctx, payload.UserID, JobKeyBlacksmith, totalXP, "upgrade", metadata)
	if err != nil {
		log.Warn("Failed to award Blacksmith XP for upgrade", "error", err, "user_id", payload.UserID)
		return nil // Don't return error to event bus to avoid retries for logic errors
	}

	if result.LeveledUp {
		log.Info("Blacksmith leveled up!", "user_id", payload.UserID, "new_level", result.NewLevel)
	}

	return nil
}

// HandleItemDisassembled handles item disassemble events to award Blacksmith XP
func (h *EventHandler) HandleItemDisassembled(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	// Decode payload
	var payload crafting.ItemDisassembledPayload
	if err := mapToStruct(evt.Payload, &payload); err != nil {
		return fmt.Errorf("failed to decode item disassembled payload: %w", err)
	}

	// Award XP
	totalXP := BlacksmithXPPerItem * payload.Quantity
	metadata := map[string]interface{}{
		"source":             "disassemble",
		"item_name":          payload.ItemName,
		"quantity":           payload.Quantity,
		"is_perfect_salvage": payload.IsPerfectSalvage,
	}

	result, err := h.service.AwardXP(ctx, payload.UserID, JobKeyBlacksmith, totalXP, "disassemble", metadata)
	if err != nil {
		log.Warn("Failed to award Blacksmith XP for disassemble", "error", err, "user_id", payload.UserID)
		return nil
	}

	if result.LeveledUp {
		log.Info("Blacksmith leveled up!", "user_id", payload.UserID, "new_level", result.NewLevel)
	}

	return nil
}

// mapToStruct converts a map payload to a struct
func mapToStruct(input interface{}, output interface{}) error {
	// If input is already the correct type, simple assignment won't work easily here due to interface{}
	// But usually the EventBus (MemoryBus) passes the payload as is.
	// However, if coming from JSON (e.g. over network), it might be a map.
	// For MemoryBus, we put the struct in the payload directly in crafting/service.go.
	// So we can try type assertion first.

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

	// Fallback to JSON round-trip for maps
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}
