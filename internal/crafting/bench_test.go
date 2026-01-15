package crafting

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func BenchmarkUpgradeItem(b *testing.B) {
	// Setup logic outside the loop
	repo := NewMockRepository()
	setupTestData(repo)

	// Ensure fast path for RNG
	svc := NewService(repo, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 0.5 } // Deterministic RNG

	ctx := context.Background()

	// Ensure we have enough items for the benchmark
	// In a real DB we'd transactionally decrement, but here in the mock
	// we can just start with a huge amount to avoid running out during the loop
	// or we can reset in the loop (but that measures reset time).
	// Actually, the service *does* decrement.
	// Best approach: Mock UpdateInventory to do nothing or use a "Infinite" inventory mode for benchmarking.
	// But `MockRepository` is a bit complex to modify for "Infinite".
	// Let's just give a massive amount.

	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: b.N + 100}, // Enough for all iterations
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err != nil {
			b.Fatalf("UpgradeItem failed: %v", err)
		}
	}
}

func BenchmarkDisassembleItem(b *testing.B) {
	repo := NewMockRepository()
	setupTestData(repo)

	svc := NewService(repo, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 0.5 }

	ctx := context.Background()

	repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 2, Quantity: b.N + 100},
	}})
	repo.UnlockRecipe(ctx, "user-alice", 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err != nil {
			b.Fatalf("DisassembleItem failed: %v", err)
		}
	}
}
