package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// TestConcurrentAddItem_Integration verifies that database transactions properly
// isolate concurrent AddItem operations, preventing lost updates.
func TestConcurrentAddItem_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()

	// Use shared pool and migrations
	ensureMigrations(t)

	// Create repository and service
	repo := NewUserRepository(testPool)
	trapRepo := NewTrapRepository(testPool)
	svc := user.NewService(repo, trapRepo, nil, nil, nil, &mockNamingResolver{}, nil, nil, nil, false)

	// Create a test user
	testUser := &domain.User{
		Username: "concurrency_test_user",
		TwitchID: "twitch_concurrent_123",
	}
	if err := repo.UpsertUser(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Run concurrent AddItem operations
	const concurrentOps = 20 // Reduced to ensure test completes within timeout while still testing concurrency
	const itemName = domain.ItemLootbox1

	var wg sync.WaitGroup
	wg.Add(concurrentOps)
	errChan := make(chan error, concurrentOps)

	t.Logf("Starting %d concurrent AddItem operations...", concurrentOps)
	startTime := time.Now()

	for i := 0; i < concurrentOps; i++ {
		go func() {
			defer wg.Done()
			err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, testUser.Username, itemName, 1)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	duration := time.Since(startTime)
	t.Logf("Completed %d operations in %v", concurrentOps, duration)

	// Check for errors
	errors := make([]error, 0, concurrentOps)
	for err := range errChan {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		t.Fatalf("encountered %d errors during concurrent operations: first error: %v", len(errors), errors[0])
	}

	// Verify final count - this is the critical assertion
	inv, err := repo.GetInventory(ctx, testUser.ID)
	if err != nil {
		t.Fatalf("failed to get inventory: %v", err)
	}

	// Find the item in inventory
	var actualQuantity int
	found := false
	for _, slot := range inv.Slots {
		item, err := repo.GetItemByID(ctx, slot.ItemID)
		if err != nil {
			t.Fatalf("failed to get item: %v", err)
		}
		if item != nil && item.InternalName == itemName {
			actualQuantity = slot.Quantity
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("item %s not found in inventory after %d concurrent additions", itemName, concurrentOps)
	}

	// The critical assertion: if transactions work correctly, we should have exactly
	// concurrentOps items (no lost updates due to race conditions)
	if actualQuantity != concurrentOps {
		t.Errorf("TRANSACTION ISOLATION FAILURE: Expected exactly %d items after %d concurrent additions, but got %d (lost %d updates). "+
			"This indicates database transactions are not properly isolating concurrent operations.",
			concurrentOps, concurrentOps, actualQuantity, concurrentOps-actualQuantity)
	} else {
		t.Logf("SUCCESS: All %d concurrent operations completed correctly with no lost updates", concurrentOps)
	}
}

// mockNamingResolver is a minimal implementation for testing
type mockNamingResolver struct{}

func (m *mockNamingResolver) GetDisplayName(internalName string, qualityLevel domain.QualityLevel) string {
	return internalName
}

func (m *mockNamingResolver) ResolvePublicName(publicName string) (string, bool) {
	return publicName, true
}

func (m *mockNamingResolver) GetActiveTheme() string {
	return ""
}

func (m *mockNamingResolver) Reload() error {
	return nil
}

func (m *mockNamingResolver) RegisterItem(internalName, publicName string) {}
