package progression

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/event"
)

func TestUnlockCacheInvalidatesOnEvent(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)

	// Create event bus
	bus := event.NewMemoryBus()

	// Create service with event bus (so handlers work)
	service := NewService(repo, NewMockUser(), bus)
	ctx := context.Background()

	// First check - populates cache with "false"
	unlocked, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if unlocked {
		t.Error("Money should not be unlocked yet")
	}

	// Unlock money via repository
	repo.UnlockNode(ctx, 2, 1, "test", 0)

	// Publish unlock event (this should trigger cache invalidation)
	bus.Publish(ctx, event.Event{
		Type:    "progression.node_unlocked",
		Version: "1.0",
		Payload: map[string]interface{}{
			"node_id":  2,
			"node_key": "item_money",
			"level":    1,
		},
	})

	// Cache should be invalidated, so this should hit DB and see the unlock
	unlockedNow, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if !unlockedNow {
		t.Error("Money should be unlocked after event invalidation")
	}
}

func TestUnlockCacheInvalidatesOnRelockEvent(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)

	// Create event bus
	bus := event.NewMemoryBus()

	// Create service with event bus
	service := NewService(repo, NewMockUser(), bus)
	ctx := context.Background()

	// Unlock money first
	repo.UnlockNode(ctx, 2, 1, "test", 0)

	// Check - populates cache with "true"
	unlocked, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if !unlocked {
		t.Error("Money should be unlocked")
	}

	// Relock money via repository
	repo.RelockNode(ctx, 2, 1)

	// Publish relock event (this should trigger cache invalidation)
	bus.Publish(ctx, event.Event{
		Type:    "progression.node_relocked",
		Version: "1.0",
		Payload: map[string]interface{}{
			"node_id":  2,
			"node_key": "item_money",
			"level":    1,
		},
	})

	// Cache should be invalidated, so this should hit DB and see it's relocked
	unlockedNow, err := service.IsItemUnlocked(ctx, "money")
	if err != nil {
		t.Fatalf("IsItemUnlocked failed: %v", err)
	}
	if unlockedNow {
		t.Error("Money should be locked after relock event invalidation")
	}
}
