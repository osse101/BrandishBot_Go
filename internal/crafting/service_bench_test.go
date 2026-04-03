package crafting

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func setupBenchService() (*service, *MockRepository) {
	repo := NewMockRepository()
	setupTestData(repo) // Sets up some default recipes

	// Add plenty of test items to inventory for the benchmarks
	repo.inventories["user-alice"] = &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 100000}, // Required for recipe 1
			{ItemID: TestItemID2, Quantity: 100000}, // Required for recipe 2 and disassemble recipe 1
			{ItemID: TestItemID3, Quantity: 100000},
		},
	}
	repo.UnlockRecipe(context.Background(), "user-alice", 1)

	svc := NewService(repo, &MockEventPublisher{}, nil, nil, NewMockJobService()).(*service)
	svc.rnd = func() float64 { return 1.0 } // Prevent masterwork/perfect salvage branches to isolate core performance
	return svc, repo
}

func BenchmarkService_UpgradeItem(b *testing.B) {
	svc, repo := setupBenchService()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Replenish inventory
		repo.Lock()
		repo.inventories["user-alice"].Slots = []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 100000},
			{ItemID: TestItemID2, Quantity: 100000},
			{ItemID: TestItemID3, Quantity: 100000},
		}
		repo.Unlock()
		b.StartTimer()

		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkService_DisassembleItem(b *testing.B) {
	svc, repo := setupBenchService()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Replenish inventory
		repo.Lock()
		repo.inventories["user-alice"].Slots = []domain.InventorySlot{
			{ItemID: TestItemID1, Quantity: 100000},
			{ItemID: TestItemID2, Quantity: 100000},
			{ItemID: TestItemID3, Quantity: 100000},
		}
		repo.Unlock()
		b.StartTimer()

		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}
