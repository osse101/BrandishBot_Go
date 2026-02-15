package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// SubscriptionRepository implements the subscription repository for PostgreSQL
type SubscriptionRepository struct {
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewSubscriptionRepository creates a new SubscriptionRepository
func NewSubscriptionRepository(db *pgxpool.Pool) repository.Subscription {
	return &SubscriptionRepository{
		db: db,
		q:  generated.New(db),
	}
}

// GetUserSubscription retrieves a user's subscription for a specific platform
func (r *SubscriptionRepository) GetUserSubscription(ctx context.Context, userID, platform string) (*domain.SubscriptionWithTier, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	row, err := r.q.GetUserSubscription(ctx, generated.GetUserSubscriptionParams{
		UserID:   userUUID,
		Platform: platform,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscription: %w", err)
	}

	return &domain.SubscriptionWithTier{
		Subscription: domain.Subscription{
			UserID:         row.UserID.String(),
			Platform:       row.Platform,
			TierID:         int(row.TierID),
			Status:         row.Status,
			SubscribedAt:   row.SubscribedAt.Time,
			ExpiresAt:      row.ExpiresAt.Time,
			LastVerifiedAt: pgTimestamptzToTimePtr(row.LastVerifiedAt),
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
		},
		TierName:    row.TierName,
		DisplayName: row.DisplayName,
		TierLevel:   int(row.TierLevel),
	}, nil
}

// GetUserSubscriptions retrieves all subscriptions for a user
func (r *SubscriptionRepository) GetUserSubscriptions(ctx context.Context, userID string) ([]domain.SubscriptionWithTier, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUserSubscriptions(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscriptions: %w", err)
	}

	subscriptions := make([]domain.SubscriptionWithTier, 0, len(rows))
	for _, row := range rows {
		subscriptions = append(subscriptions, domain.SubscriptionWithTier{
			Subscription: domain.Subscription{
				UserID:         row.UserID.String(),
				Platform:       row.Platform,
				TierID:         int(row.TierID),
				Status:         row.Status,
				SubscribedAt:   row.SubscribedAt.Time,
				ExpiresAt:      row.ExpiresAt.Time,
				LastVerifiedAt: pgTimestamptzToTimePtr(row.LastVerifiedAt),
				CreatedAt:      row.CreatedAt.Time,
				UpdatedAt:      row.UpdatedAt.Time,
			},
			TierName:    row.TierName,
			DisplayName: row.DisplayName,
			TierLevel:   int(row.TierLevel),
		})
	}

	return subscriptions, nil
}

// CreateSubscription creates a new subscription (uses upsert internally)
func (r *SubscriptionRepository) CreateSubscription(ctx context.Context, sub domain.Subscription) error {
	return r.UpdateSubscription(ctx, sub)
}

// UpdateSubscription updates or creates a subscription
func (r *SubscriptionRepository) UpdateSubscription(ctx context.Context, sub domain.Subscription) error {
	userUUID, err := uuid.Parse(sub.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	err = r.q.UpsertSubscription(ctx, generated.UpsertSubscriptionParams{
		UserID:         userUUID,
		Platform:       sub.Platform,
		TierID:         int32(sub.TierID),
		Status:         sub.Status,
		SubscribedAt:   pgtype.Timestamptz{Time: sub.SubscribedAt, Valid: true},
		ExpiresAt:      pgtype.Timestamptz{Time: sub.ExpiresAt, Valid: true},
		LastVerifiedAt: timePtrToPgTimestamptz(sub.LastVerifiedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to upsert subscription: %w", err)
	}

	return nil
}

// DeleteSubscription deletes a user's subscription for a platform
func (r *SubscriptionRepository) DeleteSubscription(ctx context.Context, userID, platform string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	err = r.q.DeleteSubscription(ctx, generated.DeleteSubscriptionParams{
		UserID:   userUUID,
		Platform: platform,
	})
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}

// GetExpiringSubscriptions retrieves all subscriptions expiring before the given time
func (r *SubscriptionRepository) GetExpiringSubscriptions(ctx context.Context, before time.Time) ([]domain.SubscriptionWithTier, error) {
	rows, err := r.q.GetExpiringSubscriptions(ctx, pgtype.Timestamptz{Time: before, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get expiring subscriptions: %w", err)
	}

	subscriptions := make([]domain.SubscriptionWithTier, 0, len(rows))
	for _, row := range rows {
		subscriptions = append(subscriptions, domain.SubscriptionWithTier{
			Subscription: domain.Subscription{
				UserID:         row.UserID.String(),
				Platform:       row.Platform,
				TierID:         int(row.TierID),
				Status:         row.Status,
				SubscribedAt:   row.SubscribedAt.Time,
				ExpiresAt:      row.ExpiresAt.Time,
				LastVerifiedAt: pgTimestamptzToTimePtr(row.LastVerifiedAt),
				CreatedAt:      row.CreatedAt.Time,
				UpdatedAt:      row.UpdatedAt.Time,
			},
			TierName:    row.TierName,
			DisplayName: row.DisplayName,
			TierLevel:   int(row.TierLevel),
		})
	}

	return subscriptions, nil
}

// MarkSubscriptionExpired marks a subscription as expired
func (r *SubscriptionRepository) MarkSubscriptionExpired(ctx context.Context, userID, platform string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	err = r.q.MarkSubscriptionExpired(ctx, generated.MarkSubscriptionExpiredParams{
		UserID:   userUUID,
		Platform: platform,
	})
	if err != nil {
		return fmt.Errorf("failed to mark subscription expired: %w", err)
	}

	return nil
}

// GetTierByPlatformAndName retrieves a tier by platform and tier name
func (r *SubscriptionRepository) GetTierByPlatformAndName(ctx context.Context, platform, tierName string) (*domain.SubscriptionTier, error) {
	tier, err := r.q.GetTierByPlatformAndName(ctx, generated.GetTierByPlatformAndNameParams{
		Platform: platform,
		TierName: tierName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get tier: %w", err)
	}

	return &domain.SubscriptionTier{
		TierID:      int(tier.TierID),
		Platform:    tier.Platform,
		TierName:    tier.TierName,
		DisplayName: tier.DisplayName,
		TierLevel:   int(tier.TierLevel),
		CreatedAt:   tier.CreatedAt.Time,
	}, nil
}

// GetAllTiers retrieves all subscription tiers
func (r *SubscriptionRepository) GetAllTiers(ctx context.Context) ([]domain.SubscriptionTier, error) {
	rows, err := r.q.GetAllTiers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tiers: %w", err)
	}

	tiers := make([]domain.SubscriptionTier, 0, len(rows))
	for _, row := range rows {
		tiers = append(tiers, domain.SubscriptionTier{
			TierID:      int(row.TierID),
			Platform:    row.Platform,
			TierName:    row.TierName,
			DisplayName: row.DisplayName,
			TierLevel:   int(row.TierLevel),
			CreatedAt:   row.CreatedAt.Time,
		})
	}

	return tiers, nil
}

// RecordSubscriptionHistory records a subscription lifecycle event
func (r *SubscriptionRepository) RecordSubscriptionHistory(ctx context.Context, history domain.SubscriptionHistory) error {
	userUUID, err := uuid.Parse(history.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	// Convert metadata map to JSON
	var metadataJSON []byte
	if history.Metadata != nil {
		metadataJSON, err = json.Marshal(history.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	err = r.q.RecordSubscriptionHistory(ctx, generated.RecordSubscriptionHistoryParams{
		UserID:       userUUID,
		Platform:     history.Platform,
		TierID:       int32(history.TierID),
		EventType:    history.EventType,
		SubscribedAt: pgtype.Timestamptz{Time: history.SubscribedAt, Valid: true},
		ExpiresAt:    pgtype.Timestamptz{Time: history.ExpiresAt, Valid: true},
		Metadata:     metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to record subscription history: %w", err)
	}

	return nil
}

// GetUserSubscriptionHistory retrieves subscription history for a user
func (r *SubscriptionRepository) GetUserSubscriptionHistory(ctx context.Context, userID string, limit int) ([]domain.SubscriptionHistory, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUserSubscriptionHistory(ctx, generated.GetUserSubscriptionHistoryParams{
		UserID: userUUID,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription history: %w", err)
	}

	history := make([]domain.SubscriptionHistory, 0, len(rows))
	for _, row := range rows {
		var metadata map[string]interface{}
		if row.Metadata != nil {
			if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		history = append(history, domain.SubscriptionHistory{
			HistoryID:    row.HistoryID,
			UserID:       row.UserID.String(),
			Platform:     row.Platform,
			TierID:       int(row.TierID),
			EventType:    row.EventType,
			SubscribedAt: row.SubscribedAt.Time,
			ExpiresAt:    row.ExpiresAt.Time,
			Metadata:     metadata,
			CreatedAt:    row.CreatedAt.Time,
		})
	}

	return history, nil
}

// Helper functions

func pgTimestamptzToTimePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	return &ts.Time
}

func timePtrToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
