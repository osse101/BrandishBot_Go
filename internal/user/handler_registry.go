package user

import (
	"context"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// ItemHandler defines the interface for handling item effects
type ItemHandler interface {
	// CanHandle returns true if this handler can process the given item
	CanHandle(itemName string) bool

	// Handle processes the item effect and returns a result message
	Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error)
}

// HandlerRegistry manages item effect handlers
type HandlerRegistry struct {
	handlers []ItemHandler
}

// NewHandlerRegistry creates a new handler registry with default handlers
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: []ItemHandler{
			&LootboxHandler{},
			&BlasterHandler{},
		},
	}
}

// GetHandler finds the appropriate handler for the given item name
func (r *HandlerRegistry) GetHandler(itemName string) ItemHandler {
	for _, handler := range r.handlers {
		if handler.CanHandle(itemName) {
			return handler
		}
	}
	return nil
}

// LootboxHandler handles all lootbox tiers
type LootboxHandler struct{}

// CanHandle returns true for any lootbox item
func (h *LootboxHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemLootbox0 ||
		itemName == domain.ItemLootbox1 ||
		itemName == domain.ItemLootbox2 ||
		strings.HasPrefix(itemName, "lootbox_tier")
}

// Handle processes lootbox opening
func (h *LootboxHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, user, inventory, item, quantity)
}

// BlasterHandler handles the blaster item
type BlasterHandler struct{}

// CanHandle returns true for blaster items
func (h *BlasterHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemBlaster
}

// Handle processes blaster usage
func (h *BlasterHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleBlaster(ctx, s, user, inventory, item, quantity, args)
}
