package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// SeedFullyLoadedUser creates a user and populates many tables with dummy data for integration tests.
// This ensures that constraints (like ON DELETE CASCADE) are properly tested.
func SeedFullyLoadedUser(ctx context.Context, q *generated.Queries, userID uuid.UUID, username string) error {
	// 1. users
	// We use the provided userID directly
	_, err := q.CreateUserWithID(ctx, generated.CreateUserWithIDParams{
		UserID:   userID,
		Username: username,
	})
	if err != nil {
		return fmt.Errorf("failed to create user with id: %w", err)
	}

	// 2. platforms
	// Link to twitch, discord, youtube
	platforms := []string{string(domain.PlatformTwitch), string(domain.PlatformDiscord), string(domain.PlatformYoutube)}
	for _, pName := range platforms {
		pID, err := q.GetPlatformID(ctx, pName)
		if err == nil {
			err = q.UpsertUserPlatformLink(ctx, generated.UpsertUserPlatformLinkParams{
				UserID:         userID,
				PlatformID:     pID,
				PlatformUserID: fmt.Sprintf("%s_%s", pName, username),
				PlatformUsername: pgtype.Text{
					String: username,
					Valid:  true,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to upsert platform link for %s: %w", pName, err)
			}
		}
	}

	// 3. user_inventory
	inventoryData, _ := json.Marshal(map[string]interface{}{"slots": []interface{}{}})
	err = q.EnsureInventoryRow(ctx, generated.EnsureInventoryRowParams{
		UserID:        userID,
		InventoryData: inventoryData,
	})
	if err != nil {
		return fmt.Errorf("failed to ensure inventory row: %w", err)
	}

	// 4. user_jobs
	jobs, err := q.GetAllJobs(ctx)
	if err == nil && len(jobs) > 0 {
		err = q.UpsertUserJob(ctx, generated.UpsertUserJobParams{
			UserID:        userID,
			JobID:         jobs[0].ID,
			CurrentXp:     100,
			CurrentLevel:  5,
			XpGainedToday: pgtype.Int8{Int64: 10, Valid: true},
			LastXpGain:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to upsert user job: %w", err)
		}

		// 5. job_xp_events
		err = q.RecordJobXPEvent(ctx, generated.RecordJobXPEventParams{
			ID:         uuid.New(),
			UserID:     userID,
			JobID:      jobs[0].ID,
			XpAmount:   10,
			SourceType: "test_seed",
			RecordedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to record job xp event: %w", err)
		}
	}

	// 6. stats_events
	_, err = q.RecordEvent(ctx, generated.RecordEventParams{
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		EventType: "test_event",
		EventData: []byte(`{"detail": "seeded"}`),
		CreatedAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to record stats event: %w", err)
	}

	// 7. engagement_metrics
	err = q.RecordEngagement(ctx, generated.RecordEngagementParams{
		UserID:      userID.String(),
		MetricType:  "test_engagement",
		MetricValue: pgtype.Int4{Int32: 1, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to record engagement metric: %w", err)
	}

	// 8. user_progression
	err = q.UnlockUserProgression(ctx, generated.UnlockUserProgressionParams{
		UserID:          userID.String(),
		ProgressionType: "test_progression",
		ProgressionKey:  "test_key",
		Metadata:        []byte(`{}`),
	})
	if err != nil {
		return fmt.Errorf("failed to unlock user progression: %w", err)
	}

	// 9. user_cooldowns
	err = q.UpdateCooldown(ctx, generated.UpdateCooldownParams{
		UserID:     userID,
		ActionName: "test_cooldown",
		LastUsedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update user cooldown: %w", err)
	}

	return nil
}
