package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// TestConcurrentAddItem_Integration verifies that database transactions properly
// isolate concurrent AddItem operations, preventing lost updates.
func TestConcurrentAddItem_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start Postgres container
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Connect to database
	pool, err := database.NewPool(connStr, 25, 30*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Apply migrations
	if err := applyMigrations(t, ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// Create repository and service
	repo := NewUserRepository(pool)
	svc := user.NewService(repo, nil, nil, nil, &mockNamingResolver{}, nil, false)

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
			err := svc.AddItem(ctx, domain.PlatformTwitch, "twitch_concurrent_123", "concurrency_test_user", itemName, 1)
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

func (m *mockNamingResolver) GetDisplayName(internalName, shineLevel string) string {
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
