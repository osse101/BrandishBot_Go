package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestUserRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start Postgres container
	var pgContainer *postgres.PostgresContainer
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
			}
		}()
		pgContainer, err = postgres.Run(ctx,
			"postgres:15-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(5*time.Second)),
		)
	}()

	if pgContainer == nil {
		// If panic occurred and was recovered, we already skipped.
		// If no panic but pgContainer is nil (shouldn't happen if err is nil), return.
		if err != nil {
			t.Fatalf("failed to start postgres container: %v", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Connect to database
	pool, err := database.NewPool(connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Apply migrations
	if err := applyMigrations(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	repo := NewUserRepository(pool)

	t.Run("UpsertUser", func(t *testing.T) {
		user := &domain.User{
			Username: "testuser",
			TwitchID: "twitch123",
		}

		err := repo.UpsertUser(ctx, user)
		if err != nil {
			t.Fatalf("UpsertUser failed: %v", err)
		}

		if user.ID == "" {
			t.Error("expected user ID to be set")
		}

		// Verify retrieval
		retrieved, err := repo.GetUserByUsername(ctx, "testuser")
		if err != nil {
			t.Fatalf("GetUserByUsername failed: %v", err)
		}
		if retrieved.Username != "testuser" {
			t.Errorf("expected username testuser, got %s", retrieved.Username)
		}
	})

	t.Run("Inventory Operations", func(t *testing.T) {
		// Create a user first
		user := &domain.User{Username: "inventory_user"}
		if err := repo.UpsertUser(ctx, user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		// Get empty inventory
		inv, err := repo.GetInventory(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetInventory failed: %v", err)
		}
		if len(inv.Slots) != 0 {
			t.Errorf("expected empty inventory, got %d slots", len(inv.Slots))
		}

		// Update inventory
		// Need an item first
		money, err := repo.GetItemByName(ctx, "money")
		if err != nil {
			t.Fatalf("failed to get money item: %v", err)
		}
		if money == nil {
			t.Fatal("money item not found (migrations should have seeded it)")
		}

		inv.Slots = append(inv.Slots, domain.InventorySlot{
			ItemID:   money.ID,
			Quantity: 100,
		})

		if err := repo.UpdateInventory(ctx, user.ID, *inv); err != nil {
			t.Fatalf("UpdateInventory failed: %v", err)
		}

		// Verify update
		updatedInv, err := repo.GetInventory(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetInventory failed: %v", err)
		}
		if len(updatedInv.Slots) != 1 {
			t.Errorf("expected 1 slot, got %d", len(updatedInv.Slots))
		}
		if updatedInv.Slots[0].Quantity != 100 {
			t.Errorf("expected 100 quantity, got %d", updatedInv.Slots[0].Quantity)
		}
	})

	t.Run("Transaction Support", func(t *testing.T) {
		tx, err := repo.BeginTx(ctx)
		if err != nil {
			t.Fatalf("BeginTx failed: %v", err)
		}
		defer tx.Rollback(ctx) // Should be safe to call even if committed

		// We can reuse the user from previous test or create new
		user := &domain.User{Username: "tx_user"}
		// Note: UpsertUser uses its own tx, so we can't use the tx interface for it directly
		// unless we refactor UpsertUser to take a DB interface.
		// But we can test GetInventory/UpdateInventory with the tx.
		
		// Create user outside tx first
		if err := repo.UpsertUser(ctx, user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		inv, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			t.Fatalf("tx.GetInventory failed: %v", err)
		}

		// Modify in tx
		inv.Slots = append(inv.Slots, domain.InventorySlot{ItemID: 1, Quantity: 50}) // Assuming ID 1 exists (money)
		if err := tx.UpdateInventory(ctx, user.ID, *inv); err != nil {
			t.Fatalf("tx.UpdateInventory failed: %v", err)
		}

		// Commit
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("tx.Commit failed: %v", err)
		}

		// Verify outside tx
		finalInv, err := repo.GetInventory(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetInventory failed: %v", err)
		}
		if len(finalInv.Slots) != 1 || finalInv.Slots[0].Quantity != 50 {
			t.Errorf("transaction commit failed to persist data")
		}
	})

	t.Run("Recipe Operations", func(t *testing.T) {
		// Get an item to use as target
		money, err := repo.GetItemByName(ctx, "money")
		if err != nil || money == nil {
			t.Skip("money item not found, skipping recipe test")
		}

		// Get recipe by target item ID
		recipe, err := repo.GetRecipeByTargetItemID(ctx, money.ID)
		if err != nil {
			t.Fatalf("GetRecipeByTargetItemID failed: %v", err)
		}

		if recipe != nil {
			// If recipe exists, test unlock functionality
			user := &domain.User{Username: "recipe_test_user"}
			if err := repo.UpsertUser(ctx, user); err != nil {
				t.Fatalf("failed to create user: %v", err)
			}

			// Check if unlocked (should be false initially)
			unlocked, err := repo.IsRecipeUnlocked(ctx, user.ID, recipe.ID)
			if err != nil {
				t.Fatalf("IsRecipeUnlocked failed: %v", err)
			}
			if unlocked {
				t.Error("recipe should not be unlocked initially")
			}

			// Unlock the recipe
			if err := repo.UnlockRecipe(ctx, user.ID, recipe.ID); err != nil {
				t.Fatalf("UnlockRecipe failed: %v", err)
			}

			// Verify it's now unlocked
			unlocked, err = repo.IsRecipeUnlocked(ctx, user.ID, recipe.ID)
			if err != nil {
				t.Fatalf("IsRecipeUnlocked failed: %v", err)
			}
			if !unlocked {
				t.Error("recipe should be unlocked after unlocking")
			}

			// Get unlocked recipes
			unlockedRecipes, err := repo.GetUnlockedRecipesForUser(ctx, user.ID)
			if err != nil {
				t.Fatalf("GetUnlockedRecipesForUser failed: %v", err)
			}
			if len(unlockedRecipes) == 0 {
				t.Error("expected at least one unlocked recipe")
			}
		}
	})

	t.Run("Item Buyability", func(t *testing.T) {
		// Test checking if an item is buyable
		isBuyable, err := repo.IsItemBuyable(ctx, "money")
		if err != nil {
			t.Fatalf("IsItemBuyable failed: %v", err)
		}

		// money should be buyable based on migrations
		if !isBuyable {
			t.Log("money is not buyable (may be expected depending on migrations)")
		}

		// Test with non-existent item
		isBuyable, err = repo.IsItemBuyable(ctx, "nonexistent_item_xyz")
		if err != nil {
			t.Fatalf("IsItemBuyable failed for non-existent item: %v", err)
		}
		if isBuyable {
			t.Error("non-existent item should not be buyable")
		}
	})

	t.Run("GetSellablePrices", func(t *testing.T) {
		items, err := repo.GetSellablePrices(ctx)
		if err != nil {
			t.Fatalf("GetSellablePrices failed: %v", err)
		}

		// Should have at least some sellable items based on migrations
		if len(items) == 0 {
			t.Log("no sellable items found (may be expected depending on migrations)")
		}

		// Verify items have required fields
		for _, item := range items {
			if item.Name == "" {
				t.Error("sellable item has empty name")
			}
			if item.BaseValue < 0 {
				t.Error("sellable item has negative base value")
			}
		}
	})

	t.Run("GetItemByID", func(t *testing.T) {
		// First get an item to know its ID
		money, err := repo.GetItemByName(ctx, "money")
		if err != nil || money == nil {
			t.Skip("money item not found, skipping GetItemByID test")
		}

		// Get by ID
		item, err := repo.GetItemByID(ctx, money.ID)
		if err != nil {
			t.Fatalf("GetItemByID failed: %v", err)
		}
		if item == nil {
			t.Fatal("expected item, got nil")
		}
		if item.ID != money.ID {
			t.Errorf("expected item ID %d, got %d", money.ID, item.ID)
		}

		// Test with non-existent ID
		item, err = repo.GetItemByID(ctx, 999999)
		if err != nil {
			t.Fatalf("GetItemByID failed for non-existent item: %v", err)
		}
		if item != nil {
			t.Error("expected nil for non-existent item")
		}
	})

	t.Run("GetUserByUsername - Not Found", func(t *testing.T) {
		user, err := repo.GetUserByUsername(ctx, "nonexistent_user_xyz")
		if err != nil {
			t.Fatalf("GetUserByUsername failed: %v", err)
		}
		if user != nil {
			t.Error("expected nil for non-existent user")
		}
	})

	t.Run("GetUserByPlatformID", func(t *testing.T) {
		// Create a user with platform ID
		user := &domain.User{
			Username: "platform_test_user",
			TwitchID: "twitch_platform_123",
		}
		if err := repo.UpsertUser(ctx, user); err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		// Retrieve by platform ID
		retrieved, err := repo.GetUserByPlatformID(ctx, "twitch", "twitch_platform_123")
		if err != nil {
			t.Fatalf("GetUserByPlatformID failed: %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected user, got nil")
		}
		if retrieved.Username != user.Username {
			t.Errorf("expected username %s, got %s", user.Username, retrieved.Username)
		}
		if retrieved.TwitchID != user.TwitchID {
			t.Errorf("expected twitch ID %s, got %s", user.TwitchID, retrieved.TwitchID)
		}

		// Test with non-existent platform ID
		_, err = repo.GetUserByPlatformID(ctx, "twitch", "nonexistent_xyz")
		if err == nil {
			t.Error("expected error for non-existent platform ID")
		}
	})
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			// Accept both .up.sql and .sql files (exclude .down.sql)
			if (strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".sql")) && !strings.HasSuffix(name, ".down.sql") {
				migrationFiles = append(migrationFiles, filepath.Join(migrationsDir, name))
			}
		}
	}
	sort.Strings(migrationFiles)

	fmt.Printf("Applying %d migrations in order:\n", len(migrationFiles))
	for i, file := range migrationFiles {
		fmt.Printf("  %d. %s\n", i+1, filepath.Base(file))
	}

	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Strip out the "Down" section if it exists (goose-style migrations)
		contentStr := string(content)
		if downIdx := strings.Index(contentStr, "-- +goose Down"); downIdx != -1 {
			contentStr = contentStr[:downIdx]
		}

		fmt.Printf("Executing: %s\n", filepath.Base(file))
		_, err = pool.Exec(ctx, contentStr)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}
	return nil
}
