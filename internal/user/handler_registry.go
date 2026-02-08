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
			&TrapHandler{}, // Must come before WeaponHandler to avoid matching "explosive_" prefix
			&WeaponHandler{},
			&ReviveHandler{},
			&ShieldHandler{},
			&RareCandyHandler{},
			&ResourceGeneratorHandler{},
			&UtilityHandler{},
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
		itemName == domain.ItemLootbox3 ||
		strings.HasPrefix(itemName, "lootbox_tier")
}

// Handle processes lootbox opening
func (h *LootboxHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, user, inventory, item, quantity)
}

// WeaponHandler handles all weapon items (blaster, bigblaster, hugeblaster, this, deez)
type WeaponHandler struct{}

// CanHandle returns true for weapon items
func (h *WeaponHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemBlaster ||
		itemName == domain.ItemBigBlaster ||
		itemName == domain.ItemHugeBlaster ||
		itemName == domain.ItemThis ||
		itemName == domain.ItemDeez ||
		itemName == domain.ItemMissile ||
		itemName == domain.ItemGrenade ||
		itemName == domain.ItemTNT ||
		strings.HasPrefix(itemName, "weapon_") ||
		strings.HasPrefix(itemName, "explosive_")
}

// Handle processes weapon usage
func (h *WeaponHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleWeapon(ctx, s, user, inventory, item, quantity, args)
}

// ReviveHandler handles all revive items
type ReviveHandler struct{}

// CanHandle returns true for revive items
func (h *ReviveHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemReviveSmall ||
		itemName == domain.ItemReviveMedium ||
		itemName == domain.ItemReviveLarge ||
		strings.HasPrefix(itemName, "revive_")
}

// Handle processes revive usage
func (h *ReviveHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleRevive(ctx, s, user, inventory, item, quantity, args)
}

// ShieldHandler handles shield items
type ShieldHandler struct{}

// CanHandle returns true for shield items
func (h *ShieldHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemShield ||
		itemName == domain.ItemMirrorShield
}

// Handle processes shield activation
func (h *ShieldHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleShield(ctx, s, user, inventory, item, quantity, args)
}

// RareCandyHandler handles rare candy items
type RareCandyHandler struct{}

// CanHandle returns true for rare candy items
func (h *RareCandyHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemRareCandy ||
		strings.HasPrefix(itemName, "xp_")
}

// Handle processes rare candy usage
func (h *RareCandyHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleRareCandy(ctx, s, user, inventory, item, quantity, args)
}

// ResourceGeneratorHandler handles items that generate other items
type ResourceGeneratorHandler struct{}

// CanHandle returns true for resource generator items
func (h *ResourceGeneratorHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemShovel
}

// Handle processes resource generation
func (h *ResourceGeneratorHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleResourceGenerator(ctx, s, user, inventory, item, quantity, args)
}

// UtilityHandler handles miscellaneous items with simple effects
type UtilityHandler struct{}

// CanHandle returns true for utility items
func (h *UtilityHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemStick
}

// Handle processes utility item usage
func (h *UtilityHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleUtility(ctx, s, user, inventory, item, quantity, args)
}

// TrapHandler handles trap items (must come before WeaponHandler to prevent matching "explosive_" prefix)
type TrapHandler struct{}

// CanHandle returns true for trap items
func (h *TrapHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemTrap ||
		itemName == domain.ItemMine
}

// Handle processes trap placement
func (h *TrapHandler) Handle(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleTrap(ctx, s, user, inventory, item, quantity, args)
}
