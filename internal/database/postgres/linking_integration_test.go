package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

func TestLinkingRepository_Integration(t *testing.T) {
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
	pool, err := database.NewPool(connStr, 10, 30*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Apply migrations
	if err := applyMigrations(t, ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	repo := NewLinkingRepository(pool)

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
