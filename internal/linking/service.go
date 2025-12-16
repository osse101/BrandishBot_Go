package linking

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

const (
	TokenLength     = 6
	TokenExpiration = 10 * time.Minute
	UnlinkTimeout   = 60 * time.Second
)

// Token states
const (
	StatePending   = "pending"   // Waiting for Step 2 (claim)
	StateClaimed   = "claimed"   // Waiting for Step 3 (confirm)
	StateConfirmed = "confirmed" // Link complete
	StateExpired   = "expired"   // Timed out
)

// LinkToken represents a pending link token
type LinkToken struct {
	Token            string    `json:"token"`
	SourcePlatform   string    `json:"source_platform"`
	SourcePlatformID string    `json:"source_platform_id"`
	TargetPlatform   string    `json:"target_platform,omitempty"`
	TargetPlatformID string    `json:"target_platform_id,omitempty"`
	State            string    `json:"state"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// Repository defines data access for linking
type Repository interface {
	CreateToken(ctx context.Context, token *LinkToken) error
	GetToken(ctx context.Context, tokenStr string) (*LinkToken, error)
	UpdateToken(ctx context.Context, token *LinkToken) error
	InvalidateTokensForSource(ctx context.Context, platform, platformID string) error
	GetClaimedTokenForSource(ctx context.Context, platform, platformID string) (*LinkToken, error)
	CleanupExpired(ctx context.Context) error
}

// UserService defines user operations needed for linking
type UserService interface {
	FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	RegisterUser(ctx context.Context, user domain.User) (domain.User, error)
	MergeUsers(ctx context.Context, primaryUserID, secondaryUserID string) error
	UnlinkPlatform(ctx context.Context, userID, platform string) error
	GetLinkedPlatforms(ctx context.Context, platform, platformID string) ([]string, error)
}

// Service defines the linking service interface
type Service interface {
	// InitiateLink generates a token for cross-platform linking (Step 1)
	InitiateLink(ctx context.Context, platform, platformID string) (*LinkToken, error)

	// ClaimLink claims a token from another platform (Step 2)
	ClaimLink(ctx context.Context, tokenStr, platform, platformID string) (*LinkToken, error)

	// ConfirmLink confirms the link from the source platform (Step 3)
	ConfirmLink(ctx context.Context, platform, platformID string) (*LinkResult, error)

	// InitiateUnlink starts the unlink confirmation process
	InitiateUnlink(ctx context.Context, platform, platformID, targetPlatform string) error

	// ConfirmUnlink completes the unlink
	ConfirmUnlink(ctx context.Context, platform, platformID, targetPlatform string) error

	// GetStatus returns current link status
	GetStatus(ctx context.Context, platform, platformID string) (*LinkStatus, error)
}

// LinkResult represents the result of a confirmed link
type LinkResult struct {
	Success         bool     `json:"success"`
	LinkedPlatforms []string `json:"linked_platforms"`
}

// LinkStatus represents current linking status
type LinkStatus struct {
	LinkedPlatforms []string   `json:"linked_platforms"`
	PendingToken    *LinkToken `json:"pending_token,omitempty"`
}

type service struct {
	repo        Repository
	userService UserService
	unlinkCache map[string]time.Time // platform:platformID:targetPlatform -> expiry
}

// NewService creates a new linking service
func NewService(repo Repository, userService UserService) Service {
	return &service{
		repo:        repo,
		userService: userService,
		unlinkCache: make(map[string]time.Time),
	}
}

// InitiateLink generates a token for cross-platform linking (Step 1)
func (s *service) InitiateLink(ctx context.Context, platform, platformID string) (*LinkToken, error) {
	log := logger.FromContext(ctx)

	// Invalidate any existing tokens for this source
	if err := s.repo.InvalidateTokensForSource(ctx, platform, platformID); err != nil {
		log.Warn("Failed to invalidate old tokens", "error", err)
	}

	// Generate new token
	tokenStr, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	token := &LinkToken{
		Token:            tokenStr,
		SourcePlatform:   platform,
		SourcePlatformID: platformID,
		State:            StatePending,
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(TokenExpiration),
	}

	if err := s.repo.CreateToken(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	log.Info("Link token created", "platform", platform, "token", tokenStr)
	return token, nil
}

// ClaimLink claims a token from another platform (Step 2)
func (s *service) ClaimLink(ctx context.Context, tokenStr, platform, platformID string) (*LinkToken, error) {
	log := logger.FromContext(ctx)

	token, err := s.repo.GetToken(ctx, strings.ToUpper(tokenStr))
	if err != nil {
		return nil, fmt.Errorf("token not found")
	}

	// Validate token state
	if token.State != StatePending {
		return nil, fmt.Errorf("token already used or expired")
	}

	if time.Now().After(token.ExpiresAt) {
		token.State = StateExpired
		s.repo.UpdateToken(ctx, token)
		return nil, fmt.Errorf("token expired")
	}

	// Can't link to same platform
	if token.SourcePlatform == platform && token.SourcePlatformID == platformID {
		return nil, fmt.Errorf("cannot link same account to itself")
	}

	// Update token with target info
	token.TargetPlatform = platform
	token.TargetPlatformID = platformID
	token.State = StateClaimed

	if err := s.repo.UpdateToken(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to claim token: %w", err)
	}

	log.Info("Link token claimed", "token", tokenStr, "target_platform", platform)
	return token, nil
}

// ConfirmLink confirms the link from the source platform (Step 3)
func (s *service) ConfirmLink(ctx context.Context, platform, platformID string) (*LinkResult, error) {
	log := logger.FromContext(ctx)

	// Find claimed token for this source
	token, err := s.repo.GetClaimedTokenForSource(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("no pending link to confirm")
	}

	// Verify not expired
	if time.Now().After(token.ExpiresAt) {
		token.State = StateExpired
		s.repo.UpdateToken(ctx, token)
		return nil, fmt.Errorf("link token expired")
	}

	// Find or create users for both platforms
	sourceUser, err := s.userService.FindUserByPlatformID(ctx, token.SourcePlatform, token.SourcePlatformID)
	if err != nil {
		// Create new user for source
		sourceUser = &domain.User{}
		setPlatformID(sourceUser, token.SourcePlatform, token.SourcePlatformID)
	}

	targetUser, err := s.userService.FindUserByPlatformID(ctx, token.TargetPlatform, token.TargetPlatformID)
	if err != nil {
		// Target doesn't exist - just add platform ID to source user
		setPlatformID(sourceUser, token.TargetPlatform, token.TargetPlatformID)
		updatedUser, err := s.userService.RegisterUser(ctx, *sourceUser)
		if err != nil {
			return nil, fmt.Errorf("failed to link accounts: %w", err)
		}

		token.State = StateConfirmed
		s.repo.UpdateToken(ctx, token)

		log.Info("Accounts linked", "user_id", updatedUser.ID, "platforms", []string{token.SourcePlatform, token.TargetPlatform})

		return &LinkResult{
			Success:         true,
			LinkedPlatforms: getLinkedPlatforms(&updatedUser),
		}, nil
	}

	// Both users exist - merge them (source is primary)
	if err := s.userService.MergeUsers(ctx, sourceUser.ID, targetUser.ID); err != nil {
		return nil, fmt.Errorf("failed to merge accounts: %w", err)
	}

	token.State = StateConfirmed
	s.repo.UpdateToken(ctx, token)

	log.Info("Accounts merged", "primary_id", sourceUser.ID, "secondary_id", targetUser.ID)

	// Get updated platforms list
	platforms, _ := s.userService.GetLinkedPlatforms(ctx, platform, platformID)

	return &LinkResult{
		Success:         true,
		LinkedPlatforms: platforms,
	}, nil
}

// setPlatformID sets the appropriate platform ID on a user
func setPlatformID(user *domain.User, platform, platformID string) {
	switch platform {
	case domain.PlatformDiscord:
		user.DiscordID = platformID
	case domain.PlatformTwitch:
		user.TwitchID = platformID
	case domain.PlatformYoutube:
		user.YoutubeID = platformID
	}
}

// getLinkedPlatforms returns list of linked platforms for a user
func getLinkedPlatforms(user *domain.User) []string {
	var platforms []string
	if user.DiscordID != "" {
		platforms = append(platforms, domain.PlatformDiscord)
	}
	if user.TwitchID != "" {
		platforms = append(platforms, domain.PlatformTwitch)
	}
	if user.YoutubeID != "" {
		platforms = append(platforms, domain.PlatformYoutube)
	}
	return platforms
}

// InitiateUnlink starts the unlink confirmation process
func (s *service) InitiateUnlink(ctx context.Context, platform, platformID, targetPlatform string) error {
	key := fmt.Sprintf("%s:%s:%s", platform, platformID, targetPlatform)
	s.unlinkCache[key] = time.Now().Add(UnlinkTimeout)
	return nil
}

// ConfirmUnlink completes the unlink
func (s *service) ConfirmUnlink(ctx context.Context, platform, platformID, targetPlatform string) error {
	log := logger.FromContext(ctx)

	key := fmt.Sprintf("%s:%s:%s", platform, platformID, targetPlatform)
	expiry, exists := s.unlinkCache[key]
	if !exists || time.Now().After(expiry) {
		return fmt.Errorf("no pending unlink confirmation")
	}

	delete(s.unlinkCache, key)

	// Find user and unlink
	user, err := s.userService.FindUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if err := s.userService.UnlinkPlatform(ctx, user.ID, targetPlatform); err != nil {
		return fmt.Errorf("failed to unlink: %w", err)
	}

	log.Info("Platform unlinked", "user_id", user.ID, "platform", targetPlatform)
	return nil
}

// GetStatus returns current link status
func (s *service) GetStatus(ctx context.Context, platform, platformID string) (*LinkStatus, error) {
	platforms, err := s.userService.GetLinkedPlatforms(ctx, platform, platformID)
	if err != nil {
		return nil, err
	}

	return &LinkStatus{
		LinkedPlatforms: platforms,
	}, nil
}

// generateToken creates a random 6-character alphanumeric token
func generateToken() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use base32 encoding (A-Z, 2-7) and take first 6 chars
	token := base32.StdEncoding.EncodeToString(bytes)[:TokenLength]
	return strings.ToUpper(token), nil
}
