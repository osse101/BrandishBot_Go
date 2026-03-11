package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestUserRepository_MergeUsers_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()

	// Use shared pool and migrations
	ensureMigrations(t)

	repo := NewUserRepository(testPool)
	q := repo.q

	t.Run("MergeUsersInTransaction With Fully Seeded Users", func(t *testing.T) {
		primaryID := uuid.New()
		secondaryID := uuid.New()

		// 1. Seed users (now handles creation via CreateUserWithID)
		err := SeedFullyLoadedUser(ctx, q, primaryID, "PrimaryMergeUser")
		if err != nil {
			t.Fatalf("failed to seed primary user: %v", err)
		}

		err = SeedFullyLoadedUser(ctx, q, secondaryID, "SecondaryMergeUser")
		if err != nil {
			t.Fatalf("failed to seed secondary user: %v", err)
		}

		// 3. Setup Merged Data
		// Suppose secondary has discord, primary has twitch. Merged should have both.
		mergedUser := domain.User{
			TwitchID:  "primary_twitch",
			DiscordID: "secondary_discord",
			PlatformUsernames: map[string]string{
				domain.PlatformTwitch:  "PrimaryMergeUser",
				domain.PlatformDiscord: "SecondaryMergeUser",
			},
		}

		mergedInventory := domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 99},
			},
		}

		// 4. Execute Merge
		// If explicit deletes or ON DELETE CASCADEs are missing from any of the seeded tables this transaction will fail with an SQL constraint error.
		err = repo.MergeUsersInTransaction(ctx, primaryID.String(), secondaryID.String(), mergedUser, mergedInventory)
		if err != nil {
			t.Fatalf("MergeUsersInTransaction failed on fully seeded user: %v", err)
		}

		// 5. Verify Secondary is gone
		_, err = repo.GetUserByID(ctx, secondaryID.String())
		if err == nil {
			t.Errorf("Expected secondary user to be deleted, but it was found")
		}

		// 6. Verify Primary has merged data
		finalPrimary, err := repo.GetUserByID(ctx, primaryID.String())
		if err != nil {
			t.Fatalf("failed to get primary user after merge: %v", err)
		}

		if finalPrimary.TwitchID != "primary_twitch" {
			t.Errorf("Expected primary TwitchID to be primary_twitch, got %s", finalPrimary.TwitchID)
		}
		if finalPrimary.DiscordID != "secondary_discord" {
			t.Errorf("Expected primary DiscordID to be secondary_discord, got %s", finalPrimary.DiscordID)
		}
	})

	t.Run("DeleteUser With Fully Seeded User", func(t *testing.T) {
		deletionID := uuid.New()

		// 1. Fully seed the user (handles creation)
		err := SeedFullyLoadedUser(ctx, q, deletionID, "DeletionUser")
		if err != nil {
			t.Fatalf("failed to seed deletion user: %v", err)
		}

		// 2. Execute DeleteUser (now transactional and handles cleanup)
		err = repo.DeleteUser(ctx, deletionID.String())
		if err != nil {
			t.Fatalf("DeleteUser failed on fully seeded user: %v", err)
		}

		// 3. Verify user is gone
		_, err = repo.GetUserByID(ctx, deletionID.String())
		if err == nil {
			t.Errorf("Expected user to be deleted, but it was found")
		}
	})
}
