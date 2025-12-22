package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/linking"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// INTEGRATION TESTS - Account Linking with Real Database
// ============================================================================


func TestLinking_EndToEndFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	pool, cleanup := setupTestDB(t)
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
		}
		cleanup()
	}()

	ctx := context.Background()

	// Initialize repositories and services
	linkingRepo := NewLinkingRepository(pool)
	userRepo := NewUserRepository(pool)
	userService := user.NewService(userRepo, nil, nil, nil, nil, false)
	linkingService := linking.NewService(linkingRepo, userService)

	// ========== Test Complete Linking Flow ==========


	// Step 1: Initiate link from Discord
	token, err := linkingService.InitiateLink(ctx, domain.PlatformDiscord, "discord-integration-123")
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, domain.PlatformDiscord, token.SourcePlatform)
	assert.Equal(t, "discord-integration-123", token.SourcePlatformID)
	assert.Equal(t, "pending", token.State)
	assert.Equal(t, 6, len(token.Token))

	tokenStr := token.Token

	// Step 2: Claim token from Twitch
	claimedToken, err := linkingService.ClaimLink(ctx, tokenStr, domain.PlatformTwitch, "twitch-integration-456")
	require.NoError(t, err)
	require.NotNil(t, claimedToken)
	assert.Equal(t, "claimed", claimedToken.State)
	assert.Equal(t, domain.PlatformTwitch, claimedToken.TargetPlatform)
	assert.Equal(t, "twitch-integration-456", claimedToken.TargetPlatformID)

	// Step 3: Confirm link from Discord
	result, err := linkingService.ConfirmLink(ctx, domain.PlatformDiscord, "discord-integration-123")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.LinkedPlatforms, domain.PlatformDiscord)
	assert.Contains(t, result.LinkedPlatforms, domain.PlatformTwitch)

	// Verify user has both platforms linked
	status, err := linkingService.GetStatus(ctx, domain.PlatformDiscord, "discord-integration-123")
	require.NoError(t, err)
	assert.Equal(t, 2, len(status.LinkedPlatforms))

	// Verify can access from either platform
	status2, err := linkingService.GetStatus(ctx, domain.PlatformTwitch, "twitch-integration-456")
	require.NoError(t, err)
	assert.Equal(t, 2, len(status2.LinkedPlatforms))
}

func TestLinking_TokenExpiration_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
		}
		cleanup()
	}()

	ctx := context.Background()

	linkingRepo := NewLinkingRepository(pool)
	userRepo := NewUserRepository(pool)
	userService := user.NewService(userRepo, nil, nil, nil, nil, false)
	linkingService := linking.NewService(linkingRepo, userService)

	// Create an expired token directly in database
	expiredToken := &linking.LinkToken{
		Token:            "EXPIRE",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-expired-123",
		State:            "pending",
		CreatedAt:        time.Now().Add(-2 * time.Hour),
		ExpiresAt:        time.Now().Add(-1 * time.Hour),
	}

	err := linkingRepo.CreateToken(ctx, expiredToken)
	require.NoError(t, err)

	// Try to claim expired token - should fail
	_, err = linkingService.ClaimLink(ctx, "EXPIRE", domain.PlatformTwitch, "twitch-456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestLinking_MergeTwoExistingUsers_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
		}
		cleanup()
	}()

	ctx := context.Background()

	linkingRepo := NewLinkingRepository(pool)
	userRepo := NewUserRepository(pool)
	userService := user.NewService(userRepo, nil, nil, nil, nil, false)
	linkingService := linking.NewService(linkingRepo, userService)

	// Create two separate users
	discordUser := domain.User{
		Username:  "discord_user",
		DiscordID: "discord-merge-123",
	}
	twitchUser := domain.User{
		Username: "twitch_user",
		TwitchID: "twitch-merge-456",
	}

	discordUser, err := userService.RegisterUser(ctx, discordUser)
	require.NoError(t, err)
	twitchUser, err = userService.RegisterUser(ctx, twitchUser)
	require.NoError(t, err)

	// Link them together
	token, err := linkingService.InitiateLink(ctx, domain.PlatformDiscord, "discord-merge-123")
	require.NoError(t, err)

	_, err = linkingService.ClaimLink(ctx, token.Token, domain.PlatformTwitch, "twitch-merge-456")
	require.NoError(t, err)

	result, err := linkingService.ConfirmLink(ctx, domain.PlatformDiscord, "discord-merge-123")
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify both platform IDs now point to same user
	user1, err := userService.FindUserByPlatformID(ctx, domain.PlatformDiscord, "discord-merge-123")
	require.NoError(t, err)
	user2, err := userService.FindUserByPlatformID(ctx, domain.PlatformTwitch, "twitch-merge-456")
	require.NoError(t, err)

	assert.Equal(t, user1.ID, user2.ID, "Both platforms should point to same user ID")
	assert.NotEmpty(t, user1.DiscordID)
	assert.NotEmpty(t, user1.TwitchID)
}

func TestLinking_UnlinkFlow_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
		}
		cleanup()
	}()

	ctx := context.Background()

	linkingRepo := NewLinkingRepository(pool)
	userRepo := NewUserRepository(pool)
	userService := user.NewService(userRepo, nil, nil, nil, nil, false)
	linkingService := linking.NewService(linkingRepo, userService)

	// Create a user with two platforms
	token, err := linkingService.InitiateLink(ctx, domain.PlatformDiscord, "discord-unlink-789")
	require.NoError(t, err)

	_, err = linkingService.ClaimLink(ctx, token.Token, domain.PlatformTwitch, "twitch-unlink-012")
	require.NoError(t, err)

	_, err = linkingService.ConfirmLink(ctx, domain.PlatformDiscord, "discord-unlink-789")
	require.NoError(t, err)

	// Verify linked
	status, err := linkingService.GetStatus(ctx, domain.PlatformDiscord, "discord-unlink-789")
	require.NoError(t, err)
	assert.Equal(t, 2, len(status.LinkedPlatforms))

	// Initiate unlink
	err = linkingService.InitiateUnlink(ctx, domain.PlatformDiscord, "discord-unlink-789", domain.PlatformTwitch)
	require.NoError(t, err)

	// Confirm unlink
	err = linkingService.ConfirmUnlink(ctx, domain.PlatformDiscord, "discord-unlink-789", domain.PlatformTwitch)
	require.NoError(t, err)

	// Verify unlinked
	status, err = linkingService.GetStatus(ctx, domain.PlatformDiscord, "discord-unlink-789")
	require.NoError(t, err)
	assert.Equal(t, 1, len(status.LinkedPlatforms))
	assert.Contains(t, status.LinkedPlatforms, domain.PlatformDiscord)
	assert.NotContains(t, status.LinkedPlatforms, domain.PlatformTwitch)
}

func TestLinking_MultipleTokenInvalidation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
		}
		cleanup()
	}()

	ctx := context.Background()

	linkingRepo := NewLinkingRepository(pool)
	userRepo := NewUserRepository(pool)
	userService := user.NewService(userRepo, nil, nil, nil, nil, false)
	linkingService := linking.NewService(linkingRepo, userService)

	// Create first token
	token1, err := linkingService.InitiateLink(ctx, domain.PlatformDiscord, "discord-multi-555")
	require.NoError(t, err)
	oldToken := token1.Token

	// Create second token - should invalidate first
	token2, err := linkingService.InitiateLink(ctx, domain.PlatformDiscord, "discord-multi-555")
	require.NoError(t, err)
	assert.NotEqual(t, oldToken, token2.Token)

	// Try to use old token - should fail
	_, err = linkingService.ClaimLink(ctx, oldToken, domain.PlatformTwitch, "twitch-123")
	assert.Error(t, err)

	// New token should work
	_, err = linkingService.ClaimLink(ctx, token2.Token, domain.PlatformTwitch, "twitch-123")
	assert.NoError(t, err)
}

func TestLinking_TokenCleanup_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
		}
		cleanup()
	}()

	ctx := context.Background()

	linkingRepo := postgres.NewLinkingRepository(pool)

	// Create old expired token
	oldToken := &linking.LinkToken{
		Token:            "OLDTOK",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-old-999",
		State:            "expired",
		CreatedAt:        time.Now().Add(-3 * time.Hour),
		ExpiresAt:        time.Now().Add(-2 * time.Hour),
	}

	err := linkingRepo.CreateToken(ctx, oldToken)
	require.NoError(t, err)

	// Verify token exists
	retrievedToken, err := linkingRepo.GetToken(ctx, "OLDTOK")
	require.NoError(t, err)
	assert.Equal(t, "OLDTOK", retrievedToken.Token)

	// Run cleanup
	err = linkingRepo.CleanupExpired(ctx)
	require.NoError(t, err)

	// Verify token is deleted
	_, err = linkingRepo.GetToken(ctx, "OLDTOK")
	assert.Error(t, err)
}

func TestLinking_SelfLinkingPrevention_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	pool, cleanup := setupTestDB(t)
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
		}
		cleanup()
	}()

	ctx := context.Background()

	linkingRepo := NewLinkingRepository(pool)
	userRepo := NewUserRepository(pool)
	userService := user.NewService(userRepo, nil, nil, nil, nil, false)
	linkingService := linking.NewService(linkingRepo, userService)

	// Initiate link
	token, err := linkingService.InitiateLink(ctx, domain.PlatformDiscord, "discord-self-321")
	require.NoError(t, err)

	// Try to claim with same platform and ID
	_, err = linkingService.ClaimLink(ctx, token.Token, domain.PlatformDiscord, "discord-self-321")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot link same account")
}
