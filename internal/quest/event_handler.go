package quest

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

type EventHandler struct {
	service Service
}

func NewEventHandler(service Service) *EventHandler {
	return &EventHandler{
		service: service,
	}
}

func (h *EventHandler) Register(bus event.Bus) {
	// Crafting events
	bus.Subscribe(event.Type(domain.EventTypeItemUpgraded), h.HandleItemUpgraded)
	bus.Subscribe(event.Type(domain.EventTypeItemDisassembled), h.HandleItemDisassembled)

	// Economy events (moved from economy service)
	bus.Subscribe(event.Type(domain.EventTypeItemSold), h.HandleItemSold)
	bus.Subscribe(event.Type(domain.EventTypeItemBought), h.HandleItemBought)

	// Search events (moved from user service)
	bus.Subscribe(event.Type(domain.EventTypeSearchPerformed), h.HandleSearchPerformed)
}

func (h *EventHandler) HandleItemUpgraded(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[crafting.ItemUpgradedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item upgraded payload: %w", err)
	}

	return h.handleRecipeCrafted(ctx, payload.UserID, payload.RecipeKey, payload.ItemName, payload.Quantity, "upgrade")
}

func (h *EventHandler) HandleItemDisassembled(ctx context.Context, evt event.Event) error {
	payload, err := event.DecodePayload[crafting.ItemDisassembledPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item disassembled payload: %w", err)
	}

	return h.handleRecipeCrafted(ctx, payload.UserID, payload.RecipeKey, payload.ItemName, payload.Quantity, "disassemble")
}

func (h *EventHandler) handleRecipeCrafted(ctx context.Context, userID, recipeKey, itemName string, quantity int, action string) error {
	log := logger.FromContext(ctx)

	key := recipeKey
	if key == "" {
		key = itemName
	}

	if err := h.service.OnRecipeCrafted(ctx, userID, key, quantity); err != nil {
		log.Warn(fmt.Sprintf("Failed to update quest progress for %s", action), "error", err, "user_id", userID)
		return nil
	}

	return nil
}

func (h *EventHandler) HandleItemSold(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.ItemSoldPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item sold payload: %w", err)
	}

	if err := h.service.OnItemSold(ctx, payload.UserID, payload.ItemCategory, payload.Quantity, payload.TotalValue); err != nil {
		log.Warn("Failed to update quest progress for item sold", "error", err, "user_id", payload.UserID)
		return nil
	}

	return nil
}

func (h *EventHandler) HandleItemBought(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.ItemBoughtPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode item bought payload: %w", err)
	}

	if err := h.service.OnItemBought(ctx, payload.UserID, payload.ItemCategory, payload.Quantity); err != nil {
		log.Warn("Failed to update quest progress for item bought", "error", err, "user_id", payload.UserID)
		return nil
	}

	return nil
}

func (h *EventHandler) HandleSearchPerformed(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	payload, err := event.DecodePayload[domain.SearchPerformedPayload](evt.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode search performed payload: %w", err)
	}

	if err := h.service.OnSearch(ctx, payload.UserID); err != nil {
		log.Warn("Failed to update quest progress for search", "error", err, "user_id", payload.UserID)
		return nil
	}

	return nil
}
