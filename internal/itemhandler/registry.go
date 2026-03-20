package itemhandler

import (
	"context"
	"strings"

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

// LootboxHandler handles all lootbox tiers.
type LootboxHandler struct{}

// CanHandle returns true for any lootbox item.
func (h *LootboxHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemLootbox0 ||
		itemName == domain.ItemLootbox1 ||
		itemName == domain.ItemLootbox2 ||
		itemName == domain.ItemLootbox3 ||
		strings.HasPrefix(itemName, "lootbox_tier")
}

// Handle processes lootbox opening.
func (h *LootboxHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return ProcessLootbox(ctx, ec, user, inventory, item, quantity)
}

// WeaponHandler handles all weapon items.
type WeaponHandler struct{}

// CanHandle returns true for weapon items.
func (h *WeaponHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemMissile ||
		itemName == domain.ItemHugeMissile ||
		itemName == domain.ItemThis ||
		itemName == domain.ItemDeez ||
		itemName == domain.ItemGrenade ||
		itemName == domain.ItemTNT ||
		strings.HasPrefix(itemName, "weapon_") ||
		strings.HasPrefix(itemName, "explosive_")
}

// Handle processes weapon usage.
func (h *WeaponHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleWeapon(ctx, ec, user, inventory, item, quantity, args)
}

// ReviveHandler handles all revive items.
type ReviveHandler struct{}

// CanHandle returns true for revive items.
func (h *ReviveHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemReviveSmall ||
		itemName == domain.ItemReviveMedium ||
		itemName == domain.ItemReviveLarge ||
		strings.HasPrefix(itemName, "revive_")
}

// Handle processes revive usage.
func (h *ReviveHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleRevive(ctx, ec, inventory, item, quantity, args)
}

// ShieldHandler handles shield items.
type ShieldHandler struct{}

// CanHandle returns true for shield items.
func (h *ShieldHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemShield ||
		itemName == domain.ItemMirrorShield
}

// Handle processes shield activation.
func (h *ShieldHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleShield(ctx, ec, user, inventory, item, quantity)
}

// RareCandyHandler handles rare candy items.
type RareCandyHandler struct{}

// CanHandle returns true for rare candy items.
func (h *RareCandyHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemRareCandy ||
		strings.HasPrefix(itemName, "xp_")
}

// Handle processes rare candy usage.
func (h *RareCandyHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleRareCandy(ctx, ec, user, inventory, item, quantity, args)
}

// ResourceGeneratorHandler handles items that generate other items.
type ResourceGeneratorHandler struct{}

// CanHandle returns true for resource generator items.
func (h *ResourceGeneratorHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemShovel
}

// Handle processes resource generation.
func (h *ResourceGeneratorHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleResourceGenerator(ctx, ec, inventory, item, quantity, args)
}

// UtilityHandler handles miscellaneous items with simple effects.
type UtilityHandler struct{}

// CanHandle returns true for utility items.
func (h *UtilityHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemStick
}

// Handle processes utility item usage.
func (h *UtilityHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleUtility(ctx, ec, inventory, item, quantity, args)
}

// TrapHandler handles trap items.
type TrapHandler struct{}

// CanHandle returns true for trap items.
func (h *TrapHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemTrap ||
		itemName == domain.ItemMine
}

// Handle processes trap placement.
func (h *TrapHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleTrap(ctx, ec, user, inventory, item, quantity, args)
}

// VideoFilterHandler handles video filter items.
type VideoFilterHandler struct{}

// CanHandle returns true for video filter items.
func (h *VideoFilterHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemVideoFilter
}

// Handle processes video filter application.
func (h *VideoFilterHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleVideoFilter(ctx, ec, user, inventory, item, quantity, args)
}

// BombHandler handles bomb items.
type BombHandler struct{}

// CanHandle returns true for bomb items.
func (h *BombHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemBomb
}

// Handle processes bomb usage.
func (h *BombHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleBomb(ctx, ec, user, inventory, item, quantity, args)
}
