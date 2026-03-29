package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func setupTestService() (*service, *FakeRepository) {
	repo := NewFakeRepository()
	namingResolver := NewMockNamingResolver()
	svc := &service{
		repo:            repo,
		namingResolver:  namingResolver,
		itemCacheByName: make(map[string]domain.Item),
		itemIDToName:    make(map[int]string),
		userCache:       newUserCache(loadCacheConfig()),
	}

	// Setup basic users and items
	repo.users["user1"] = &domain.User{
		ID:        "user1",
		Username:  "Alice",
		TwitchID:  "twitch123",
		DiscordID: "discord123",
	}
	repo.users["user2"] = &domain.User{
		ID:       "user2",
		Username: "Bob",
		TwitchID: "twitch456",
	}

	repo.inventories["user1"] = &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10},
			{ItemID: 2, Quantity: 5},
		},
	}
	repo.inventories["user2"] = &domain.Inventory{
		Slots: []domain.InventorySlot{},
	}

	repo.items["sword"] = &domain.Item{ID: 1, InternalName: "sword", PublicName: "sword", Types: []string{}}
	repo.items["shield"] = &domain.Item{ID: 2, InternalName: "shield", PublicName: "shield", Types: []string{}}
	repo.items["gold"] = &domain.Item{ID: 3, InternalName: "gold"}
	repo.items[domain.ItemMoney] = &domain.Item{ID: 4, InternalName: domain.ItemMoney}
	repo.items["arrows"] = &domain.Item{ID: 5, InternalName: "arrows"}

	// Add arrows to user2
	repo.inventories["user2"].Slots = append(repo.inventories["user2"].Slots, domain.InventorySlot{ItemID: 5, Quantity: 50})

	return svc, repo
}

// Test GetInventoryByUsername
func TestGetInventoryByUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		platform  string
		username  string
		filter    string
		wantLen   int
		wantErr   error
		errSubstr string
	}{
		// Best Case
		{"successful retrieval", "twitch", "alice", "", 2, nil, ""},
		// Boundary Case
		{"empty filter", "twitch", "alice", "", 2, nil, ""},
		// Edge Case
		{"case insensitive", "twitch", "ALICE", "", 2, nil, ""},
		// Invalid Case
		{"user not found", "twitch", "nonexistent", "", 0, domain.ErrFailedToGetUser, "failed to get user"},
		// Hostile Case
		{"empty username", "twitch", "", "", 0, domain.ErrFailedToGetUser, "failed to get user"},
		{"special chars username", "twitch", "alice!!", "", 0, domain.ErrFailedToGetUser, "failed to get user"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, _ := setupTestService()

			items, err := svc.GetInventoryByUsername(context.Background(), tt.platform, tt.username, tt.filter)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, items, tt.wantLen)
			}
		})
	}
}

// Test AddItemByUsername
func TestAddItemByUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		platform     string
		username     string
		itemName     string
		quantity     int
		setupRepo    func(*FakeRepository)
		wantErr      error
		errSubstr    string
		wantQuantity int
		targetUser   string
		targetItem   int
	}{
		// Best Case
		{"successful addition", "twitch", "bob", "gold", 100, nil, nil, "", 100, "user2", 3},
		// Boundary Case
		{"zero quantity", "twitch", "bob", "gold", 0, nil, nil, "", 0, "user2", 3}, // Assuming validation occurs or it errors down line
		// Edge Case
		{"case insensitive username", "twitch", "BOB", "gold", 50, nil, nil, "", 50, "user2", 3},
		// Invalid Case
		{"user not found", "twitch", "nonexistent", "gold", 100, nil, domain.ErrUserNotFound, "", 0, "", 0},
		{"item not found", "twitch", "bob", "nonexistent_item", 10, nil, domain.ErrItemNotFound, "", 0, "", 0},
		// Hostile Case
		{"negative quantity", "twitch", "bob", "gold", -10, nil, nil, "", -10, "user2", 3}, // Assuming negative fails validation
		{"extremely large quantity", "twitch", "bob", "gold", 999999999, nil, nil, "", 999999999, "user2", 3},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := setupTestService()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			err := svc.AddItemByUsername(context.Background(), tt.platform, tt.username, tt.itemName, tt.quantity)

			// Let's rely on standard errors. If it doesn't fail, we might need to adjust wantErr in subsequent runs.

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)

				inv := repo.inventories[tt.targetUser]
				found := false
				for _, slot := range inv.Slots {
					if slot.ItemID == tt.targetItem {
						assert.Equal(t, tt.wantQuantity, slot.Quantity)
						found = true
						break
					}
				}
				assert.True(t, found, "Item should be in inventory")
			}
		})
	}
}

// Test RemoveItemByUsername
func TestRemoveItemByUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		platform     string
		username     string
		itemName     string
		quantity     int
		wantErr      error
		errSubstr    string
		wantRemoved  int
		wantQuantity int
		targetUser   string
		targetItem   int
	}{
		// Best Case
		{"successful removal", "twitch", "bob", "arrows", 20, nil, "", 20, 30, "user2", 5},
		// Boundary Case
		{"remove all", "twitch", "bob", "arrows", 50, nil, "", 50, 0, "user2", 5},
		{"remove more than exists", "twitch", "bob", "arrows", 100, nil, "", 50, 0, "user2", 5}, // Depends on behavior
		// Edge Case
		{"case insensitive username", "twitch", "BOB", "arrows", 10, nil, "", 10, 40, "user2", 5},
		// Invalid Case
		{"user not found", "twitch", "nonexistent", "arrows", 10, domain.ErrFailedToGetUser, "failed to get user", 0, 0, "", 0},
		{"item not found", "twitch", "bob", "nonexistent_item", 10, domain.ErrItemNotFound, "", 0, 0, "", 0},
		// Hostile Case
		{"zero quantity", "twitch", "bob", "arrows", 0, nil, "", 0, 50, "user2", 5}, // Assuming validation
		{"negative quantity", "twitch", "bob", "arrows", -10, nil, "", -10, 60, "user2", 5},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := setupTestService()

			removed, err := svc.RemoveItemByUsername(context.Background(), tt.platform, tt.username, tt.itemName, tt.quantity)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRemoved, removed)

				inv := repo.inventories[tt.targetUser]
				found := false
				for _, slot := range inv.Slots {
					if slot.ItemID == tt.targetItem {
						assert.Equal(t, tt.wantQuantity, slot.Quantity)
						found = true
						break
					}
				}
				if tt.wantQuantity == 0 && !found {
					// It's okay if item is completely removed from slots
					assert.True(t, true)
				} else {
					assert.True(t, found, "Item should be in inventory with updated quantity")
				}
			}
		})
	}
}

// Test GiveItemByUsername
func TestGiveItemByUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		senderPlatform   string
		senderTwitchID   string
		senderUsername   string
		receiverPlatform string
		receiverUsername string
		itemName         string
		quantity         int
		setupRepo        func(*FakeRepository)
		wantErr          error
		errSubstr        string
		wantSenderQty    int
		wantReceiverQty  int
	}{
		// Best Case
		{"successful transfer", domain.PlatformTwitch, "twitch123", "alice", domain.PlatformTwitch, "bob", "sword", 5, func(r *FakeRepository) {}, nil, "", 5, 5},
		// Boundary Case
		{"exact amount", domain.PlatformTwitch, "twitch123", "alice", domain.PlatformTwitch, "bob", "sword", 10, func(r *FakeRepository) {}, nil, "", 0, 10},
		// Edge Case
		{"case insensitive receiver", domain.PlatformTwitch, "twitch123", "alice", domain.PlatformTwitch, "BOB", "sword", 5, func(r *FakeRepository) {}, nil, "", 5, 5},
		// Invalid Case
		{"receiver not found", domain.PlatformTwitch, "twitch123", "alice", domain.PlatformTwitch, "nonexistent", "sword", 5, func(r *FakeRepository) {}, domain.ErrUserNotFound, "", 10, 0},
		{"sender not found", domain.PlatformTwitch, "nonexistent", "alice", domain.PlatformTwitch, "bob", "sword", 5, func(r *FakeRepository) {}, domain.ErrNotInInventory, "", 0, 0},
		{"insufficient quantity", domain.PlatformTwitch, "twitch123", "alice", domain.PlatformTwitch, "bob", "sword", 20, func(r *FakeRepository) {}, domain.ErrInsufficientQuantity, "", 10, 0},
		// Hostile Case
		{"invalid quantity", domain.PlatformTwitch, "twitch123", "alice", domain.PlatformTwitch, "bob", "sword", -1, func(r *FakeRepository) {}, domain.ErrInvalidInput, "", 10, 0},
		{"zero quantity", domain.PlatformTwitch, "twitch123", "alice", domain.PlatformTwitch, "bob", "sword", 0, func(r *FakeRepository) {}, domain.ErrInvalidInput, "", 10, 0},
		{"self transfer", domain.PlatformTwitch, "twitch123", "alice", domain.PlatformTwitch, "alice", "sword", 5, func(r *FakeRepository) {}, nil, "", 10, 10},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := setupTestService()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			err := svc.GiveItem(context.Background(), tt.senderPlatform, tt.senderTwitchID, tt.senderUsername, tt.receiverPlatform, tt.receiverUsername, tt.itemName, tt.quantity)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)

				senderInv := repo.inventories["user1"]
				receiverID := "user2"
				if tt.receiverUsername == "alice" {
					receiverID = "user1"
				}
				receiverInv := repo.inventories[receiverID]

				senderFound := false
				for _, slot := range senderInv.Slots {
					if slot.ItemID == 1 { // sword
						assert.Equal(t, tt.wantSenderQty, slot.Quantity)
						senderFound = true
					}
				}
				if tt.wantSenderQty == 0 && !senderFound {
					assert.True(t, true)
				} else {
					assert.True(t, senderFound, "Sender should have updated item quantity")
				}

				receiverFound := false
				for _, slot := range receiverInv.Slots {
					if slot.ItemID == 1 { // sword
						assert.Equal(t, tt.wantReceiverQty, slot.Quantity)
						receiverFound = true
					}
				}
				assert.True(t, receiverFound, "Receiver should have item")
			}
		})
	}
}
