package user

import (
	"context"
	"errors"
	"testing"

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
	repo.items["sword"] = &domain.Item{ID: 1, InternalName: "sword", Types: []string{}}
	repo.items["shield"] = &domain.Item{ID: 2, InternalName: "shield", Types: []string{}}

	t.Run("successful retrieval", func(t *testing.T) {
		items, err := svc.GetInventoryByUsername(context.Background(), "twitch", "alice", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		items, err := svc.GetInventoryByUsername(context.Background(), "twitch", "ALICE", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("Expected 2 items with uppercase username, got %d", len(items))
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := svc.GetInventoryByUsername(context.Background(), "twitch", "nonexistent", "")
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("Expected ErrUserNotFound, got %v", err)
		}
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
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		inv := repo.inventories["user1"]
		if len(inv.Slots) != 1 || inv.Slots[0].ItemID != 1 || inv.Slots[0].Quantity != 100 {
			t.Fatalf("Item not added correctly")
		}
	})

	t.Run("user not found", func(t *testing.T) {
		err := svc.AddItemByUsername(context.Background(), "twitch", "nonexistent", "gold", 100)
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("Expected ErrUserNotFound, got %v", err)
		}
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
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if removed != 20 {
			t.Fatalf("Expected 20 removed, got %d", removed)
		}

		inv := repo.inventories["user1"]
		if len(inv.Slots) != 1 || inv.Slots[0].Quantity != 30 {
			t.Fatalf("Item quantity not updated: got %d", inv.Slots[0].Quantity)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := svc.RemoveItemByUsername(context.Background(), "twitch", "nonexistent", "arrows", 10)
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("Expected ErrUserNotFound, got %v", err)
		}
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
	repo.items["coins"] = &domain.Item{ID: 1, InternalName: "coins"}

	t.Run("successful transfer", func(t *testing.T) {
		msg, err := svc.GiveItemByUsername(context.Background(), "twitch", "dave", "discord", "eve", "coins", 50)
		if err != nil {
			t.Fatal("Expected no error, got", err)
		}
		if msg == "" {
			t.Fatal("Expected success message")
		}

		senderInv := repo.inventories["user1"]
		receiverInv := repo.inventories["user2"]

		if senderInv.Slots[0].Quantity != 50 {
			t.Fatalf("Sender should have 50, has %d", senderInv.Slots[0].Quantity)
		}
		if len(receiverInv.Slots) != 1 || receiverInv.Slots[0].Quantity != 50 {
			t.Fatal("Receiver should have 50 coins")
		}
	})

	t.Run("sender not found", func(t *testing.T) {
		_, err := svc.GiveItemByUsername(context.Background(), "twitch", "nonexistent", "discord", "eve", "coins", 10)
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("Expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("receiver not found", func(t *testing.T) {
		_, err := svc.GiveItemByUsername(context.Background(), "twitch", "dave", "discord", "nonexistent", "coins", 10)
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("Expected ErrUserNotFound, got %v", err)
		}
	})
}
