package user

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// SlowMockRepository adds artificial delays to expose race conditions
type SlowMockRepository struct {
	*MockRepository
	delay time.Duration
}

func (m *SlowMockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	time.Sleep(m.delay)
	return m.MockRepository.GetInventory(ctx, userID)
}

func (m *SlowMockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	time.Sleep(m.delay)
	return m.MockRepository.UpdateInventory(ctx, userID, inventory)
}

func TestConcurrency_AddItem(t *testing.T) {
	// Use a mock repo with a small delay to increase race condition likelihood
	baseRepo := NewMockRepository()
	setupTestData(baseRepo)
	repo := &SlowMockRepository{
		MockRepository: baseRepo,
		delay:          10 * time.Millisecond,
	}
	svc := NewService(repo)
	ctx := context.Background()

	// Initial setup
	username := "alice"
	itemName := domain.ItemLootbox1
	
	// We want to add 1 item, 100 times concurrently
	concurrency := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Suppress INFO logs during bulk operation
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	t.Logf("Starting %d concurrent AddItem operations...", concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			err := svc.AddItem(ctx, username, "twitch", itemName, 1)
			if err != nil {
				t.Errorf("AddItem failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify final count
	inv, _ := repo.GetInventory(ctx, "user-alice")
	
	// We expect exactly 'concurrency' items if thread-safe
	// With the current implementation (read-modify-write without locking), this will likely fail
	// proving the race condition.
	found := false
	for _, slot := range inv.Slots {
		if slot.ItemID == 1 { // Lootbox1 ID
			found = true
			if slot.Quantity != concurrency {
				t.Logf("Race condition detected! Expected %d items, got %d", concurrency, slot.Quantity)
				// We don't fail the test yet because we WANT to demonstrate the failure first, 
				// but for the purpose of this task, I should probably fail it if I want to show "tests written".
				// However, if the user asked me to "write concurrency tests", usually that implies tests that PASS if the code is correct.
				// Since the code is NOT correct yet, this test failing is the correct behavior.
				t.Fail() 
			}
		}
	}
	if !found {
		t.Errorf("Item not found in inventory")
	}
}

func TestConcurrency_BuyItem(t *testing.T) {
	baseRepo := NewMockRepository()
	setupTestData(baseRepo)
	repo := &SlowMockRepository{
		MockRepository: baseRepo,
		delay:          5 * time.Millisecond,
	}
	svc := NewService(repo)
	ctx := context.Background()

	// Give alice 100 money. Lootbox1 costs 50.
	// She should be able to buy exactly 2.
	// If we try to buy 1 concurrently 10 times, only 2 should succeed.
	svc.AddItem(ctx, "alice", "twitch", domain.ItemMoney, 100)

	concurrency := 10
	var wg sync.WaitGroup
	wg.Add(concurrency)
	
	successCount := 0
	var mu sync.Mutex

	// Suppress INFO logs during bulk operation
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	t.Logf("Starting %d concurrent BuyItem operations...", concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			// Try to buy 1
			count, err := svc.BuyItem(ctx, "alice", "twitch", domain.ItemLootbox1, 1)
			if err == nil && count == 1 {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Verify results
	if successCount > 2 {
		t.Logf("Race condition detected! User bought %d items with money for only 2", successCount)
		t.Fail()
	}

	// Verify final money (should be 0 or 50 depending on if 1 or 2 succeeded, but definitely not negative)
	inv, _ := repo.GetInventory(ctx, "user-alice")
	for _, slot := range inv.Slots {
		if slot.ItemID == 3 { // Money
			if slot.Quantity < 0 {
				t.Errorf("Negative money detected: %d", slot.Quantity)
			}
		}
	}
}
