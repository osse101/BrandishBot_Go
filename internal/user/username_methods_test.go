package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Test GetInventoryByUsername
func TestGetInventoryByUsername(t *testing.T) {
	repo := NewFakeRepository()
	namingResolver := NewMockNamingResolver()
	svc := &service{
		repo:            repo,
		namingResolver:  namingResolver,
		itemCacheByName: make(map[string]domain.Item),
		itemIDToName:    make(map[int]string),
	}

	// Setup test user
	user := &domain.User{
		ID:        "user1",
		Username:  "Alice",
		TwitchID:  "twitch123",
		DiscordID: "discord123",
	}
	repo.users["user1"] = user
	repo.inventories["user1"] = &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 10},
			{ItemID: 2, Quantity: 5},
		},
	}
	repo.items["sword"] = &domain.Item{ID: 1, InternalName: "sword", PublicName: "sword", Types: []string{}}
	repo.items["shield"] = &domain.Item{ID: 2, InternalName: "shield", PublicName: "shield", Types: []string{}}

	t.Run("successful retrieval", func(t *testing.T) {
		items, err := svc.GetInventoryByUsername(context.Background(), "twitch", "alice", "")
		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("case insensitive", func(t *testing.T) {
		items, err := svc.GetInventoryByUsername(context.Background(), "twitch", "ALICE", "")
		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := svc.GetInventoryByUsername(context.Background(), "twitch", "nonexistent", "")
		assert.ErrorIs(t, err, domain.ErrFailedToGetUser)
	})
}

// Test AddItemByUsername
func TestAddItemByUsername(t *testing.T) {
	repo := NewFakeRepository()
	svc := &service{
		repo:            repo,
		namingResolver:  NewMockNamingResolver(),
		itemCacheByName: make(map[string]domain.Item),
		itemIDToName:    make(map[int]string),
	}

	user := &domain.User{
		ID:       "user1",
		Username: "Bob",
		TwitchID: "twitch456",
	}
	repo.users["user1"] = user
	repo.inventories["user1"] = &domain.Inventory{Slots: []domain.InventorySlot{}}
	repo.items["gold"] = &domain.Item{ID: 1, InternalName: "gold"}

	t.Run("successful addition", func(t *testing.T) {
		err := svc.AddItemByUsername(context.Background(), "twitch", "bob", "gold", 100)
		require.NoError(t, err)

		inv := repo.inventories["user1"]
		require.Len(t, inv.Slots, 1)
		assert.Equal(t, 1, inv.Slots[0].ItemID)
		assert.Equal(t, 100, inv.Slots[0].Quantity)
	})

	t.Run("user not found", func(t *testing.T) {
		err := svc.AddItemByUsername(context.Background(), "twitch", "nonexistent", "gold", 100)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

// Test RemoveItemByUsername
func TestRemoveItemByUsername(t *testing.T) {
	repo := NewFakeRepository()
	svc := &service{
		repo:            repo,
		namingResolver:  NewMockNamingResolver(),
		itemCacheByName: make(map[string]domain.Item),
		itemIDToName:    make(map[int]string),
	}

	user := &domain.User{
		ID:       "user1",
		Username: "Charlie",
		TwitchID: "twitch789",
	}
	repo.users["user1"] = user
	repo.inventories["user1"] = &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 50},
		},
	}
	repo.items["arrows"] = &domain.Item{ID: 1, InternalName: "arrows"}

	t.Run("successful removal", func(t *testing.T) {
		removed, err := svc.RemoveItemByUsername(context.Background(), "twitch", "charlie", "arrows", 20)
		require.NoError(t, err)
		assert.Equal(t, 20, removed)

		inv := repo.inventories["user1"]
		require.Len(t, inv.Slots, 1)
		assert.Equal(t, 30, inv.Slots[0].Quantity)
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := svc.RemoveItemByUsername(context.Background(), "twitch", "nonexistent", "arrows", 10)
		assert.ErrorIs(t, err, domain.ErrFailedToGetUser)
	})
}

// Test GiveItemByUsername
func TestGiveItemByUsername(t *testing.T) {
	repo := NewFakeRepository()
	svc := &service{
		repo:            repo,
		namingResolver:  NewMockNamingResolver(),
		itemCacheByName: make(map[string]domain.Item),
		itemIDToName:    make(map[int]string),
		userCache:       newUserCache(loadCacheConfig()),
	}

	sender := &domain.User{
		ID:       "user1",
		Username: "Dave",
		TwitchID: "twitch111",
	}
	receiver := &domain.User{
		ID:        "user2",
		Username:  "Eve",
		DiscordID: "discord222",
	}
	repo.users["user1"] = sender
	repo.users["user2"] = receiver
	repo.inventories["user1"] = &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 100},
		},
	}
	repo.inventories["user2"] = &domain.Inventory{Slots: []domain.InventorySlot{}}
	repo.items[domain.ItemMoney] = &domain.Item{ID: 1, InternalName: domain.ItemMoney}

	t.Run("successful transfer", func(t *testing.T) {
		err := svc.GiveItem(context.Background(), domain.PlatformTwitch, sender.TwitchID, sender.Username, domain.PlatformDiscord, receiver.Username, domain.ItemMoney, 50)
		require.NoError(t, err)

		senderInv := repo.inventories["user1"]
		receiverInv := repo.inventories["user2"]

		assert.Equal(t, 50, senderInv.Slots[0].Quantity)
		require.Len(t, receiverInv.Slots, 1)
		assert.Equal(t, 50, receiverInv.Slots[0].Quantity)
	})

	t.Run("receiver not found", func(t *testing.T) {
		err := svc.GiveItem(context.Background(), domain.PlatformTwitch, sender.TwitchID, sender.Username, domain.PlatformTwitch, "NonexistentUser", domain.ItemMoney, 10)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("insufficient quantity", func(t *testing.T) {
		err := svc.GiveItem(context.Background(), domain.PlatformTwitch, sender.TwitchID, sender.Username, domain.PlatformDiscord, receiver.Username, domain.ItemMoney, 200)
		assert.ErrorIs(t, err, domain.ErrInsufficientQuantity)
	})

	t.Run("invalid quantity", func(t *testing.T) {
		err := svc.GiveItem(context.Background(), domain.PlatformTwitch, sender.TwitchID, sender.Username, domain.PlatformDiscord, receiver.Username, domain.ItemMoney, -1)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
	})
}
