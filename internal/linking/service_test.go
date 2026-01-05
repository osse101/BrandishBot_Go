package linking

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock objects
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateToken(ctx context.Context, token *LinkToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}
func (m *MockRepository) GetToken(ctx context.Context, tokenStr string) (*LinkToken, error) {
	args := m.Called(ctx, tokenStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LinkToken), args.Error(1)
}
func (m *MockRepository) UpdateToken(ctx context.Context, token *LinkToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}
func (m *MockRepository) InvalidateTokensForSource(ctx context.Context, platform, platformID string) error {
	args := m.Called(ctx, platform, platformID)
	return args.Error(0)
}
func (m *MockRepository) GetClaimedTokenForSource(ctx context.Context, platform, platformID string) (*LinkToken, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LinkToken), args.Error(1)
}
func (m *MockRepository) CleanupExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *MockUserService) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(domain.User), args.Error(1)
}
func (m *MockUserService) MergeUsers(ctx context.Context, primaryUserID, secondaryUserID string) error {
	args := m.Called(ctx, primaryUserID, secondaryUserID)
	return args.Error(0)
}
func (m *MockUserService) UnlinkPlatform(ctx context.Context, userID, platform string) error {
	args := m.Called(ctx, userID, platform)
	return args.Error(0)
}
func (m *MockUserService) GetLinkedPlatforms(ctx context.Context, platform, platformID string) ([]string, error) {
	args := m.Called(ctx, platform, platformID)
	return args.Get(0).([]string), args.Error(1)
}

// TestUnlinkCache_RaceCondition validates that unlink cache access is thread-safe
func TestUnlinkCache_Concurrency(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Allow FindUserByPlatformID to be called with any args, return error to stop flow
	// This mocks the DB call that happens after cache check passes (if it does)
	userService.On("FindUserByPlatformID", mock.Anything, mock.Anything, mock.Anything).Return(nil, fmt.Errorf("mock error")).Maybe()

	// Concurrent writes (InitiateUnlink)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc.InitiateUnlink(ctx, domain.PlatformDiscord, fmt.Sprintf("id-%d", id), domain.PlatformTwitch)
		}(i)
	}

	// Concurrent reads/writes (ConfirmUnlink deletes from map)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc.ConfirmUnlink(ctx, domain.PlatformDiscord, fmt.Sprintf("id-%d", id), domain.PlatformTwitch)
		}(i)
	}

	wg.Wait()
}

// TestConfirmLink_SourceUserCreationFlow validates that when a new source user is created,
// it is persisted (RegisterUser) getting a valid ID *before* MergeUsers is called.
func TestConfirmLink_SourceUserCreationFlow(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	token := &LinkToken{
		Token:            "ABCDEF",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		TargetPlatform:   domain.PlatformTwitch,
		TargetPlatformID: "twitch-456",
		State:            StateClaimed,
		ExpiresAt:        time.Now().Add(1 * time.Hour),
	}

	repo.On("GetClaimedTokenForSource", ctx, domain.PlatformDiscord, "discord-123").Return(token, nil)
	repo.On("UpdateToken", ctx, mock.Anything).Return(nil)

	// Step 1: Source user search -> Not Found (Triggers creation flow)
	userService.On("FindUserByPlatformID", ctx, domain.PlatformDiscord, "discord-123").Return(nil, fmt.Errorf("not found"))

	// Step 2: RegisterUser MUST be called for the new source user
	createdUser := domain.User{ID: "new-source-id", DiscordID: "discord-123"}
	userService.On("RegisterUser", ctx, mock.MatchedBy(func(u domain.User) bool {
		return u.DiscordID == "discord-123"
	})).Return(createdUser, nil)

	// Step 3: Target user search -> Found
	targetUser := &domain.User{ID: "existing-user-id", TwitchID: "twitch-456"}
	userService.On("FindUserByPlatformID", ctx, domain.PlatformTwitch, "twitch-456").Return(targetUser, nil)

	// Step 4: MergeUsers called with the ID from Step 2 (new-source-id)
	// If the fix is missing, this would receive "" (empty string)
	userService.On("MergeUsers", ctx, "new-source-id", "existing-user-id").Return(nil)

	// Step 5: Get Status
	userService.On("GetLinkedPlatforms", ctx, domain.PlatformDiscord, "discord-123").Return([]string{domain.PlatformDiscord, domain.PlatformTwitch}, nil)

	_, err := svc.ConfirmLink(ctx, domain.PlatformDiscord, "discord-123")
	assert.NoError(t, err)

	userService.AssertExpectations(t)
}

// ============================================================================
// PHASE 1: FUNCTIONAL TESTS
// ============================================================================

func TestInitiateLink_Success(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	repo.On("InvalidateTokensForSource", ctx, domain.PlatformDiscord, "discord-123").Return(nil)
	repo.On("CreateToken", ctx, mock.MatchedBy(func(token *LinkToken) bool {
		return token.SourcePlatform == domain.PlatformDiscord &&
			token.SourcePlatformID == "discord-123" &&
			token.State == StatePending &&
			len(token.Token) == TokenLength
	})).Return(nil)

	token, err := svc.InitiateLink(ctx, domain.PlatformDiscord, "discord-123")

	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, domain.PlatformDiscord, token.SourcePlatform)
	assert.Equal(t, "discord-123", token.SourcePlatformID)
	assert.Equal(t, StatePending, token.State)
	assert.Equal(t, TokenLength, len(token.Token))
	assert.True(t, token.ExpiresAt.After(time.Now()))
	repo.AssertExpectations(t)
}

func TestClaimLink_Success(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	pendingToken := &LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		State:            StatePending,
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}

	repo.On("GetToken", ctx, "ABC123").Return(pendingToken, nil)
	repo.On("UpdateToken", ctx, mock.MatchedBy(func(token *LinkToken) bool {
		return token.State == StateClaimed &&
			token.TargetPlatform == domain.PlatformTwitch &&
			token.TargetPlatformID == "twitch-456"
	})).Return(nil)

	token, err := svc.ClaimLink(ctx, "ABC123", domain.PlatformTwitch, "twitch-456")

	assert.NoError(t, err)
	assert.Equal(t, StateClaimed, token.State)
	assert.Equal(t, domain.PlatformTwitch, token.TargetPlatform)
	assert.Equal(t, "twitch-456", token.TargetPlatformID)
	repo.AssertExpectations(t)
}

func TestConfirmLink_MergeTwoExistingUsers(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	token := &LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		TargetPlatform:   domain.PlatformTwitch,
		TargetPlatformID: "twitch-456",
		State:            StateClaimed,
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}

	sourceUser := &domain.User{ID: "user-1", DiscordID: "discord-123"}
	targetUser := &domain.User{ID: "user-2", TwitchID: "twitch-456"}

	repo.On("GetClaimedTokenForSource", ctx, domain.PlatformDiscord, "discord-123").Return(token, nil)
	repo.On("UpdateToken", ctx, mock.Anything).Return(nil)
	userService.On("FindUserByPlatformID", ctx, domain.PlatformDiscord, "discord-123").Return(sourceUser, nil)
	userService.On("FindUserByPlatformID", ctx, domain.PlatformTwitch, "twitch-456").Return(targetUser, nil)
	userService.On("MergeUsers", ctx, "user-1", "user-2").Return(nil)
	userService.On("GetLinkedPlatforms", ctx, domain.PlatformDiscord, "discord-123").Return([]string{domain.PlatformDiscord, domain.PlatformTwitch}, nil)

	result, err := svc.ConfirmLink(ctx, domain.PlatformDiscord, "discord-123")

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, []string{domain.PlatformDiscord, domain.PlatformTwitch}, result.LinkedPlatforms)
	userService.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestConfirmLink_LinkToNewUser(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	token := &LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		TargetPlatform:   domain.PlatformTwitch,
		TargetPlatformID: "twitch-456",
		State:            StateClaimed,
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}

	existingUser := &domain.User{ID: "user-1", DiscordID: "discord-123"}

	repo.On("GetClaimedTokenForSource", ctx, domain.PlatformDiscord, "discord-123").Return(token, nil)
	repo.On("UpdateToken", ctx, mock.Anything).Return(nil)
	userService.On("FindUserByPlatformID", ctx, domain.PlatformDiscord, "discord-123").Return(existingUser, nil)
	userService.On("FindUserByPlatformID", ctx, domain.PlatformTwitch, "twitch-456").Return(nil, fmt.Errorf("not found"))
	userService.On("RegisterUser", ctx, mock.MatchedBy(func(u domain.User) bool {
		return u.ID == "user-1" && u.DiscordID == "discord-123" && u.TwitchID == "twitch-456"
	})).Return(domain.User{
		ID:        "user-1",
		DiscordID: "discord-123",
		TwitchID:  "twitch-456",
	}, nil)

	result, err := svc.ConfirmLink(ctx, domain.PlatformDiscord, "discord-123")

	assert.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.LinkedPlatforms, domain.PlatformDiscord)
	assert.Contains(t, result.LinkedPlatforms, domain.PlatformTwitch)
	userService.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestGetStatus_Success(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	linkedPlatforms := []string{domain.PlatformDiscord, domain.PlatformTwitch, domain.PlatformYoutube}
	userService.On("GetLinkedPlatforms", ctx, domain.PlatformDiscord, "discord-123").Return(linkedPlatforms, nil)

	status, err := svc.GetStatus(ctx, domain.PlatformDiscord, "discord-123")

	assert.NoError(t, err)
	assert.Equal(t, linkedPlatforms, status.LinkedPlatforms)
	userService.AssertExpectations(t)
}

// ============================================================================
// PHASE 2: SECURITY TESTS
// ============================================================================

func TestClaimLink_ExpiredToken(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	expiredToken := &LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		State:            StatePending,
		CreatedAt:        time.Now().Add(-20 * time.Minute),
		ExpiresAt:        time.Now().Add(-10 * time.Minute), // Expired
	}

	repo.On("GetToken", ctx, "ABC123").Return(expiredToken, nil)
	repo.On("UpdateToken", ctx, mock.MatchedBy(func(token *LinkToken) bool {
		return token.State == StateExpired
	})).Return(nil)

	token, err := svc.ClaimLink(ctx, "ABC123", domain.PlatformTwitch, "twitch-456")

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Contains(t, err.Error(), StateExpired)
	repo.AssertExpectations(t)
}

func TestClaimLink_AlreadyUsedToken(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	usedToken := &LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		State:            StateConfirmed, // Already confirmed
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}

	repo.On("GetToken", ctx, "ABC123").Return(usedToken, nil)

	token, err := svc.ClaimLink(ctx, "ABC123", domain.PlatformTwitch, "twitch-456")

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Contains(t, err.Error(), "already used")
	repo.AssertExpectations(t)
}

func TestClaimLink_SamePlatformRejection(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	token := &LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		State:            StatePending,
		ExpiresAt:        time.Now().Add(10 * time.Minute),
	}

	repo.On("GetToken", ctx, "ABC123").Return(token, nil)

	// Try to claim with same platform and ID
	result, err := svc.ClaimLink(ctx, "ABC123", domain.PlatformDiscord, "discord-123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot link same account")
	repo.AssertExpectations(t)
}

func TestConfirmLink_ExpiredConfirmation(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	expiredToken := &LinkToken{
		Token:            "ABC123",
		SourcePlatform:   domain.PlatformDiscord,
		SourcePlatformID: "discord-123",
		TargetPlatform:   domain.PlatformTwitch,
		TargetPlatformID: "twitch-456",
		State:            StateClaimed,
		ExpiresAt:        time.Now().Add(-1 * time.Minute), // Expired
	}

	repo.On("GetClaimedTokenForSource", ctx, domain.PlatformDiscord, "discord-123").Return(expiredToken, nil)
	repo.On("UpdateToken", ctx, mock.MatchedBy(func(token *LinkToken) bool {
		return token.State == StateExpired
	})).Return(nil)

	result, err := svc.ConfirmLink(ctx, domain.PlatformDiscord, "discord-123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), StateExpired)
	repo.AssertExpectations(t)
}

func TestInitiateLink_InvalidatesPreviousTokens(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	// Should invalidate old tokens before creating new one
	repo.On("InvalidateTokensForSource", ctx, domain.PlatformDiscord, "discord-123").Return(nil)
	repo.On("CreateToken", ctx, mock.Anything).Return(nil)

	_, err := svc.InitiateLink(ctx, domain.PlatformDiscord, "discord-123")

	assert.NoError(t, err)
	repo.AssertCalled(t, "InvalidateTokensForSource", ctx, domain.PlatformDiscord, "discord-123")
	repo.AssertExpectations(t)
}

func TestUnlink_RequiresConfirmation(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	// Initiate unlink
	err := svc.InitiateUnlink(ctx, domain.PlatformDiscord, "discord-123", domain.PlatformTwitch)
	assert.NoError(t, err)

	// Confirm should work immediately
	user := &domain.User{ID: "user-1", DiscordID: "discord-123", TwitchID: "twitch-456"}
	userService.On("FindUserByPlatformID", ctx, domain.PlatformDiscord, "discord-123").Return(user, nil)
	userService.On("UnlinkPlatform", ctx, "user-1", domain.PlatformTwitch).Return(nil)

	err = svc.ConfirmUnlink(ctx, domain.PlatformDiscord, "discord-123", domain.PlatformTwitch)
	assert.NoError(t, err)
	userService.AssertExpectations(t)
}

func TestUnlink_ConfirmationTimeout(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	// Manually set an expired unlink request
	svc.(*service).mu.Lock()
	key := "discord:discord-123:twitch"
	svc.(*service).unlinkCache[key] = time.Now().Add(-1 * time.Minute) // Expired
	svc.(*service).mu.Unlock()

	// Confirm should fail
	err := svc.ConfirmUnlink(ctx, domain.PlatformDiscord, "discord-123", domain.PlatformTwitch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending unlink")
}

// ============================================================================
// PHASE 3: ERROR HANDLING TESTS
// ============================================================================

func TestClaimLink_InvalidToken(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	repo.On("GetToken", ctx, "INVALID").Return(nil, fmt.Errorf("not found"))

	token, err := svc.ClaimLink(ctx, "INVALID", domain.PlatformTwitch, "twitch-456")

	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Contains(t, err.Error(), "token not found")
	repo.AssertExpectations(t)
}

func TestConfirmLink_NoPendingLink(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	repo.On("GetClaimedTokenForSource", ctx, domain.PlatformDiscord, "discord-123").Return(nil, fmt.Errorf("not found"))

	result, err := svc.ConfirmLink(ctx, domain.PlatformDiscord, "discord-123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no pending link")
	repo.AssertExpectations(t)
}

func TestGetStatus_UserNotFound(t *testing.T) {
	repo := new(MockRepository)
	userService := new(MockUserService)
	svc := NewService(repo, userService)
	ctx := context.Background()

	userService.On("GetLinkedPlatforms", ctx, domain.PlatformDiscord, "nonexistent").Return([]string(nil), fmt.Errorf("user not found"))

	status, err := svc.GetStatus(ctx, domain.PlatformDiscord, "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, status)
	userService.AssertExpectations(t)
}
