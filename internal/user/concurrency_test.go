package user

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/concurrency"
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
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager, nil, false)
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
			err := svc.AddItem(ctx, "twitch", "test-platform-id", username, itemName, 1)
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
