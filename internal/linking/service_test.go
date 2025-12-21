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
			svc.InitiateUnlink(ctx, "discord", fmt.Sprintf("id-%d", id), "twitch")
		}(i)
	}

	// Concurrent reads/writes (ConfirmUnlink deletes from map)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc.ConfirmUnlink(ctx, "discord", fmt.Sprintf("id-%d", id), "twitch")
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
		Token: "ABCDEF",
		SourcePlatform: "discord",
		SourcePlatformID: "discord-123",
		TargetPlatform: "twitch",
		TargetPlatformID: "twitch-456",
		State: StateClaimed,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	repo.On("GetClaimedTokenForSource", ctx, "discord", "discord-123").Return(token, nil)
	repo.On("UpdateToken", ctx, mock.Anything).Return(nil)

	// Step 1: Source user search -> Not Found (Triggers creation flow)
	userService.On("FindUserByPlatformID", ctx, "discord", "discord-123").Return(nil, fmt.Errorf("not found"))
	
	// Step 2: RegisterUser MUST be called for the new source user
	createdUser := domain.User{ID: "new-source-id", DiscordID: "discord-123"}
	userService.On("RegisterUser", ctx, mock.MatchedBy(func(u domain.User) bool {
		return u.DiscordID == "discord-123"
	})).Return(createdUser, nil)

	// Step 3: Target user search -> Found
	targetUser := &domain.User{ID: "existing-user-id", TwitchID: "twitch-456"}
	userService.On("FindUserByPlatformID", ctx, "twitch", "twitch-456").Return(targetUser, nil)

	// Step 4: MergeUsers called with the ID from Step 2 (new-source-id)
	// If the fix is missing, this would receive "" (empty string)
	userService.On("MergeUsers", ctx, "new-source-id", "existing-user-id").Return(nil)
	
	// Step 5: Get Status
	userService.On("GetLinkedPlatforms", ctx, "discord", "discord-123").Return([]string{"discord", "twitch"}, nil)

	_, err := svc.ConfirmLink(ctx, "discord", "discord-123")
	assert.NoError(t, err)
	
	userService.AssertExpectations(t)
}
