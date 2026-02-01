package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// TestUtilityHandler tests the stick handler
func TestUtilityHandler(t *testing.T) {
	tests := []struct {
		name          string
		itemInSlot    int
		quantity      int
		wantError     bool
		errorContains string
	}{
		{
			name:       "Use 1 stick",
			itemInSlot: 1,
			quantity:   1,
			wantError:  false,
		},
		{
			name:          "Insufficient sticks",
			itemInSlot:    2,
			quantity:      3,
			wantError:     true,
			errorContains: ErrMsgNotEnoughItemsInInventory,
		},
		{
			name:          "No sticks in inventory",
			itemInSlot:    0,
			quantity:      1,
			wantError:     true,
			errorContains: ErrMsgItemNotFoundInInventory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup - minimal service with no dependencies needed for stick handler
			ctx := logger.WithRequestID(context.Background(), "test")
			svc := &service{}

			stickItem := &domain.Item{ID: 2, InternalName: domain.ItemStick}
			inventory := &domain.Inventory{
				Slots: []domain.InventorySlot{},
			}
			if tt.itemInSlot > 0 {
				inventory.Slots = append(inventory.Slots, domain.InventorySlot{
					ItemID:   stickItem.ID,
					Quantity: tt.itemInSlot,
				})
			}

			// Execute
			args := map[string]interface{}{
				ArgsUsername: "testuser",
			}
			handler := &UtilityHandler{}
			result, err := handler.Handle(ctx, svc, &domain.User{ID: "user1"}, inventory, stickItem, tt.quantity, args)

			// Assert
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, "testuser planted a stick")

				// Verify stick was removed
				stickSlot, _ := utils.FindSlot(inventory, stickItem.ID)
				if tt.itemInSlot == tt.quantity {
					assert.Equal(t, -1, stickSlot, "Stick should be removed from inventory")
				}
			}
		})
	}
}

// TestWeaponHandler_NewWeapons tests the new weapon types
func TestWeaponHandler_NewWeapons(t *testing.T) {
	tests := []struct {
		name       string
		weaponName string
		canHandle  bool
	}{
		{
			name:       "Missile weapon",
			weaponName: domain.ItemMissile,
			canHandle:  true,
		},
		{
			name:       "Grenade weapon",
			weaponName: domain.ItemGrenade,
			canHandle:  true,
		},
		{
			name:       "TNT weapon",
			weaponName: domain.ItemTNT,
			canHandle:  true,
		},
		{
			name:       "Unknown weapon",
			weaponName: "weapon_unknown",
			canHandle:  true, // Should handle due to prefix matching
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &WeaponHandler{}
			result := handler.CanHandle(tt.weaponName)
			assert.Equal(t, tt.canHandle, result)
		})
	}
}

// TestShieldHandler_MirrorShield tests mirror shield support
func TestShieldHandler_MirrorShield(t *testing.T) {
	tests := []struct {
		name       string
		shieldName string
		canHandle  bool
	}{
		{
			name:       "Standard shield",
			shieldName: domain.ItemShield,
			canHandle:  true,
		},
		{
			name:       "Mirror shield",
			shieldName: domain.ItemMirrorShield,
			canHandle:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &ShieldHandler{}
			result := handler.CanHandle(tt.shieldName)
			assert.Equal(t, tt.canHandle, result)
		})
	}
}

// TestHandlerRegistry_NewHandlers verifies new handlers are registered
func TestHandlerRegistry_NewHandlers(t *testing.T) {
	registry := NewHandlerRegistry()

	tests := []struct {
		name     string
		itemName string
		wantNil  bool
	}{
		{
			name:     "Shovel has handler",
			itemName: domain.ItemShovel,
			wantNil:  false,
		},
		{
			name:     "Stick has handler",
			itemName: domain.ItemStick,
			wantNil:  false,
		},
		{
			name:     "Mirror shield has handler",
			itemName: domain.ItemMirrorShield,
			wantNil:  false,
		},
		{
			name:     "Missile has handler",
			itemName: domain.ItemMissile,
			wantNil:  false,
		},
		{
			name:     "Grenade has handler",
			itemName: domain.ItemGrenade,
			wantNil:  false,
		},
		{
			name:     "TNT has handler",
			itemName: domain.ItemTNT,
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := registry.GetHandler(tt.itemName)
			if tt.wantNil {
				assert.Nil(t, handler)
			} else {
				assert.NotNil(t, handler, "Expected handler for %s", tt.itemName)
			}
		})
	}
}
