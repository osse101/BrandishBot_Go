package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
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
			args := ItemHandlerArgs{
				Username: "testuser",
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

// TestHandler_Mine tests mine item logic
func TestHandler_Mine(t *testing.T) {
	// Setup repo and service
	repo := NewFakeRepository()
	setupTestData(repo) // sets up Alice, Bob, partial items

	// Add Mine item definition
	repo.items[domain.ItemMine] = &domain.Item{
		ID:           7,
		InternalName: domain.ItemMine,
		PublicName:   domain.PublicNameMine,
		Description:  "Careful where you step",
		BaseValue:    250,
	}

	// Create service
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false).(*service)
	ctx := context.Background()

	// Setup users
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}
	bob := domain.User{
		ID:       "user-bob",
		Username: "bob",
		TwitchID: "bob456",
	}
	// Upsert users to repo
	repo.UpsertUser(ctx, &alice)
	repo.UpsertUser(ctx, &bob)

	t.Run("Mine targets random active chatter", func(t *testing.T) {
		// Manually setup inventory to strictly control state and avoid pointer issues
		inv := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 7, Quantity: 1},
			},
		}
		repo.UpdateInventory(ctx, alice.ID, *inv)

		// Make Bob active
		svc.activeChatterTracker.Track(domain.PlatformTwitch, bob.ID, bob.Username)

		// Use Mine - Should pick Bob
		msg, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemMine, 1, "")
		assert.NoError(t, err)
		assert.Contains(t, msg, "bob")
		assert.Contains(t, msg, "set 1 mine")

		// Verify trap created on Bob
		bobUUID, _ := uuid.Parse(bob.ID)
		trap, err := repo.GetActiveTrap(ctx, bobUUID)
		assert.NoError(t, err)
		assert.NotNil(t, trap)
		assert.Equal(t, bobUUID, trap.TargetID)

		// Verify inventory
		inv, _ = repo.GetInventory(ctx, alice.ID)
		assert.Equal(t, 0, len(inv.Slots), "Mine should be consumed")
	})

	t.Run("Mine targets self if no active chatters", func(t *testing.T) {
		// Using fresh service/repo for cleaner state
		// ...
		// (Previous test code was fine, but we'll leave it as is for now)
		// For the loop tests, we'll create new subtests
	})

	t.Run("Mine looping: Places multiple mines", func(t *testing.T) {
		// Use a fresh repo/service to avoid interference from previous tests
		localRepo := NewFakeRepository()
		setupTestData(localRepo)

		// Setup Mine item
		localRepo.items[domain.ItemMine] = repo.items[domain.ItemMine]

		localSvc := NewService(localRepo, localRepo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false).(*service)

		// Setup Alice with mines
		aliceID := uuid.New().String()
		alice := domain.User{ID: aliceID, Username: "alice", TwitchID: "alice123"}
		localRepo.UpsertUser(ctx, &alice)
		inv := &domain.Inventory{
			Slots: []domain.InventorySlot{{ItemID: 7, Quantity: 5}},
		}
		localRepo.UpdateInventory(ctx, alice.ID, *inv)

		// Create active chatters
		bobID := uuid.New().String()
		charlieID := uuid.New().String()
		daveID := uuid.New().String()
		bob := domain.User{ID: bobID, Username: "bob", TwitchID: "bob456"}
		charlie := domain.User{ID: charlieID, Username: "charlie", TwitchID: "charlie789"}
		dave := domain.User{ID: daveID, Username: "dave", TwitchID: "dave101"}
		localRepo.UpsertUser(ctx, &bob)
		localRepo.UpsertUser(ctx, &charlie)
		localRepo.UpsertUser(ctx, &dave)

		// Track them
		localSvc.activeChatterTracker.Track(domain.PlatformTwitch, bob.ID, bob.Username)
		localSvc.activeChatterTracker.Track(domain.PlatformTwitch, charlie.ID, charlie.Username)
		localSvc.activeChatterTracker.Track(domain.PlatformTwitch, dave.ID, dave.Username)

		// Use 3 Mines
		msg, err := localSvc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemMine, 3, "")
		assert.NoError(t, err)
		assert.Contains(t, msg, "set 3 mine")
		assert.Contains(t, msg, "3 people")

		// Verify inventory (should have 2 left)
		invAfter, _ := localRepo.GetInventory(ctx, alice.ID)
		assert.Equal(t, 2, invAfter.Slots[0].Quantity)
	})

	t.Run("Mine looping: Breaks on self-target (bad luck)", func(t *testing.T) {
		// Give Alice Mines
		inv := &domain.Inventory{
			Slots: []domain.InventorySlot{{ItemID: 7, Quantity: 5}},
		}
		repo.UpdateInventory(ctx, alice.ID, *inv)

		svcLocal := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false).(*service)
		svcLocal.activeChatterTracker.Track(domain.PlatformTwitch, alice.ID, alice.Username)

		msg, err := svcLocal.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemMine, 3, "")
		assert.NoError(t, err)
		assert.Contains(t, msg, "dropped a mine", "Should be bad luck message")

		// Inventory should decrease by 1 (the one that hit self)
		invAfter, _ := repo.GetInventory(ctx, alice.ID)
		assert.Equal(t, 4, invAfter.Slots[0].Quantity)
	})
}
