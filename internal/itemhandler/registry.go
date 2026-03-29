package itemhandler

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Handler defines the interface for handling item effects.
type Handler interface {
	// CanHandle returns true if this handler can process the given item.
	CanHandle(itemName string) bool

	// Handle processes the item effect and returns a result message.
	Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error)
}

// Registry manages item effect handlers.
type Registry struct {
	handlers []Handler
}

// NewRegistry creates a new handler registry with default handlers.
func NewRegistry() *Registry {
	return &Registry{
		handlers: []Handler{
			&LootboxHandler{},
			&TrapHandler{}, // Must come before WeaponHandler to avoid matching "explosive_" prefix
			&WeaponHandler{},
			&ReviveHandler{},
			&ShieldHandler{},
			&RareCandyHandler{},
			&ResourceGeneratorHandler{},
			&UtilityHandler{},
			&VideoFilterHandler{},
			&BombHandler{},
		},
	}
}

// GetHandler finds the appropriate handler for the given item name.
func (r *Registry) GetHandler(itemName string) Handler {
	for _, handler := range r.handlers {
		if handler.CanHandle(itemName) {
			return handler
		}
	}
	return nil
}
