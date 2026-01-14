package crafting

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func BenchmarkUpgradeItem(b *testing.B) {
	// Setup base state
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 } // Deterministic: no masterwork

	ctx := context.Background()
	userID := "user-alice"
	platform := domain.PlatformTwitch
	platformID := "twitch-alice"
	username := "alice"

	// Ensure user has plenty of materials for the benchmark duration
	// We'll reset inventory in the loop if needed, or just give a massive amount
	initialQuantity := 1000000
	repo.UpdateInventory(ctx, userID, domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 1, Quantity: initialQuantity}, // lootbox0
	}})
	repo.UnlockRecipe(ctx, userID, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We upgrade 1 item at a time
		_, err := svc.UpgradeItem(ctx, platform, platformID, username, domain.ItemLootbox1, 1)
		if err != nil {
			b.Fatalf("UpgradeItem failed: %v", err)
		}

		// Stop timer to replenish inventory if getting low (optional, but good for correctness if logic checks bounds)
		// With 1M items and default bench time, we likely won't run out, but let's be safe
		// Actually, checking atomic/mutex protected map in a tight loop might affect bench.
		// Given 1M items, if N > 1M, we'd fail.
		// Let's just handle the error or re-provision.
		if i%initialQuantity == initialQuantity-1 {
			b.StopTimer()
			repo.UpdateInventory(ctx, userID, domain.Inventory{Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: initialQuantity},
			}})
			b.StartTimer()
		}
	}
}

func BenchmarkDisassembleItem(b *testing.B) {
	// Setup base state
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 } // Deterministic: no perfect salvage

	ctx := context.Background()
	userID := "user-alice"
	platform := domain.PlatformTwitch
	platformID := "twitch-alice"
	username := "alice"

	initialQuantity := 1000000
	repo.UpdateInventory(ctx, userID, domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 2, Quantity: initialQuantity}, // lootbox1
	}})
	repo.UnlockRecipe(ctx, userID, 1) // Unlock upgrade recipe (required for disassemble)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.DisassembleItem(ctx, platform, platformID, username, domain.ItemLootbox1, 1)
		if err != nil {
			b.Fatalf("DisassembleItem failed: %v", err)
		}

		if i%initialQuantity == initialQuantity-1 {
			b.StopTimer()
			repo.UpdateInventory(ctx, userID, domain.Inventory{Slots: []domain.InventorySlot{
				{ItemID: 2, Quantity: initialQuantity},
			}})
			b.StartTimer()
		}
	}
}

// Benchmark with Perfect Salvage calculation enabled (randomness involved)
func BenchmarkDisassembleItem_WithRNG(b *testing.B) {
	repo := NewMockRepository()
	setupTestData(repo)
	// Mock stats service to avoid nil pointer if events are recorded
	mockStats := &MockStatsService{}
	svc := NewService(repo, nil, mockStats, nil) // Use real RNG from NewService? No, let's keep it controlled but non-trivial
	// Actually, let's use the real RNG or a fast deterministic one.
	// The service uses utils.RandomFloat by default.
	// We want to measure the impact of the logic loop in calculatePerfectSalvage.

	ctx := context.Background()
	userID := "user-alice"
	platform := domain.PlatformTwitch
	platformID := "twitch-alice"
	username := "alice"

	initialQuantity := 1000000
	repo.UpdateInventory(ctx, userID, domain.Inventory{Slots: []domain.InventorySlot{
		{ItemID: 2, Quantity: initialQuantity},
	}})
	repo.UnlockRecipe(ctx, userID, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Process batch of 10 to exercise the loop in calculatePerfectSalvage
		_, err := svc.DisassembleItem(ctx, platform, platformID, username, domain.ItemLootbox1, 10)
		if err != nil {
			b.Fatalf("DisassembleItem failed: %v", err)
		}

		// Replenish occasionally
		if i%10000 == 0 {
             b.StopTimer()
             // Just force reset to max
             repo.UpdateInventory(ctx, userID, domain.Inventory{Slots: []domain.InventorySlot{
                 {ItemID: 2, Quantity: initialQuantity},
             }})
             b.StartTimer()
        }
	}
}
