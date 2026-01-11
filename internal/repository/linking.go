package repository

import (
	"context"
	"time"
)

// Linking defines data access for linking
type Linking interface {
	CreateToken(ctx context.Context, token *LinkToken) error
	GetToken(ctx context.Context, tokenStr string) (*LinkToken, error)
	UpdateToken(ctx context.Context, token *LinkToken) error
	InvalidateTokensForSource(ctx context.Context, platform, platformID string) error
	GetClaimedTokenForSource(ctx context.Context, platform, platformID string) (*LinkToken, error)
	CleanupExpired(ctx context.Context) error
}

// LinkToken represents a pending link token (moved from linking package)
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
