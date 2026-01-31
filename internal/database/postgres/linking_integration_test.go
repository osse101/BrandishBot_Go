package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/repository"
)

func TestLinkingRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()

	// Use shared pool and migrations
	ensureMigrations(t)

	repo := NewLinkingRepository(testPool)

	t.Run("CreateAndGetToken", func(t *testing.T) {
		token := &repository.LinkToken{
			Token:            "TEST1234",
			SourcePlatform:   "twitch",
			SourcePlatformID: "user1",
			State:            "pending",
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(1 * time.Hour),
		}

		err := repo.CreateToken(ctx, token)
		if err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}

		retrieved, err := repo.GetToken(ctx, "TEST1234")
		if err != nil {
			t.Fatalf("GetToken failed: %v", err)
		}

		if retrieved.Token != token.Token {
			t.Errorf("Expected token %s, got %s", token.Token, retrieved.Token)
		}
		if retrieved.SourcePlatform != token.SourcePlatform {
			t.Errorf("Expected platform %s, got %s", token.SourcePlatform, retrieved.SourcePlatform)
		}
	})

	t.Run("UpdateToken", func(t *testing.T) {
		token := &repository.LinkToken{
			Token:            "UPDATE12",
			SourcePlatform:   "twitch",
			SourcePlatformID: "user2",
			State:            "pending",
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(1 * time.Hour),
		}

		err := repo.CreateToken(ctx, token)
		if err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}

		token.TargetPlatform = "discord"
		token.TargetPlatformID = "discord_user2"
		token.State = "claimed"

		err = repo.UpdateToken(ctx, token)
		if err != nil {
			t.Fatalf("UpdateToken failed: %v", err)
		}

		retrieved, err := repo.GetToken(ctx, "UPDATE12")
		if err != nil {
			t.Fatalf("GetToken failed: %v", err)
		}

		if retrieved.State != "claimed" {
			t.Errorf("Expected state claimed, got %s", retrieved.State)
		}
		if retrieved.TargetPlatform != "discord" {
			t.Errorf("Expected target platform discord, got %s", retrieved.TargetPlatform)
		}
	})

	t.Run("InvalidateTokens", func(t *testing.T) {
		token := &repository.LinkToken{
			Token:            "INVALID1",
			SourcePlatform:   "youtube",
			SourcePlatformID: "user3",
			State:            "pending",
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(1 * time.Hour),
		}

		err := repo.CreateToken(ctx, token)
		if err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}

		err = repo.InvalidateTokensForSource(ctx, "youtube", "user3")
		if err != nil {
			t.Fatalf("InvalidateTokensForSource failed: %v", err)
		}

		retrieved, err := repo.GetToken(ctx, "INVALID1")
		if err != nil {
			t.Fatalf("GetToken failed: %v", err)
		}

		if retrieved.State != "expired" {
			t.Errorf("Expected state expired, got %s", retrieved.State)
		}
	})

	t.Run("GetClaimedToken", func(t *testing.T) {
		token := &repository.LinkToken{
			Token:            "CLAIMED1",
			SourcePlatform:   "discord",
			SourcePlatformID: "user4",
			State:            "claimed",
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(1 * time.Hour),
		}

		err := repo.CreateToken(ctx, token)
		if err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}

		retrieved, err := repo.GetClaimedTokenForSource(ctx, "discord", "user4")
		if err != nil {
			t.Fatalf("GetClaimedTokenForSource failed: %v", err)
		}

		if retrieved.Token != "CLAIMED1" {
			t.Errorf("Expected token CLAIMED1, got %s", retrieved.Token)
		}
	})

	t.Run("CleanupExpired", func(t *testing.T) {
		token := &repository.LinkToken{
			Token:            "EXPIRED1",
			SourcePlatform:   "twitch",
			SourcePlatformID: "user5",
			State:            "pending",
			CreatedAt:        time.Now().Add(-2 * time.Hour),
			ExpiresAt:        time.Now().Add(-1*time.Hour - 1*time.Minute), // Expired > 1 hour ago
		}

		err := repo.CreateToken(ctx, token)
		if err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}

		err = repo.CleanupExpired(ctx)
		if err != nil {
			t.Fatalf("CleanupExpired failed: %v", err)
		}

		_, err = repo.GetToken(ctx, "EXPIRED1")
		if err == nil {
			t.Error("Expected token to be deleted, but it was found")
		}
	})
}
