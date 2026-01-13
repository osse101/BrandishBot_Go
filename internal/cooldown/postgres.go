package cooldown

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// postgresBackend implements Service using PostgreSQL
type postgresBackend struct {
	db             *pgxpool.Pool
	config         Config
	progressionSvc ProgressionService
}

// NewPostgresService creates a new cooldown service with Postgres backend
func NewPostgresService(db *pgxpool.Pool, config Config, progressionSvc ProgressionService) Service {
	return &postgresBackend{
		db:             db,
		config:         config,
		progressionSvc: progressionSvc,
	}
}

// CheckCooldown checks if a user's action is on cooldown (unlocked read)
func (b *postgresBackend) CheckCooldown(ctx context.Context, userID, action string) (bool, time.Duration, error) {
	// Dev mode bypasses all cooldowns
	if b.config.DevMode {
		return false, 0, nil
	}

	lastUsed, err := b.getLastUsed(ctx, userID, action)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check cooldown: %w", err)
	}

	if lastUsed == nil {
		// Never used - not on cooldown
		return false, 0, nil
	}

	cooldownDuration, err := b.getEffectiveCooldown(ctx, action)
	if err != nil {
		return false, 0, fmt.Errorf("failed to get effective cooldown: %w", err)
	}

	onCooldown, remaining := b.checkCooldownInternal(lastUsed, cooldownDuration)
	return onCooldown, remaining, nil
}

// EnforceCooldown atomically checks cooldown and executes action if allowed
// Uses check-then-lock pattern for performance
func (b *postgresBackend) EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error {
	log := logger.FromContext(ctx)

	// PHASE 1: Cheap unlocked check - fast rejection for ~90% of requests
	onCooldown, remaining, err := b.CheckCooldown(ctx, userID, action)
	if err != nil {
		return err
	}
	if onCooldown {
		return ErrOnCooldown{Action: action, Remaining: remaining}
	}

	// Dev mode - just execute
	if b.config.DevMode {
		log.Debug("DEV_MODE: Bypassing cooldown enforcement", "action", action, "userID", userID)
		if err := fn(); err != nil {
			return err
		}
		// Still update cooldown for testing purposes
		return b.updateCooldown(ctx, userID, action, time.Now())
	}

	// PHASE 2: Transaction with advisory lock
	// Advisory locks work even when no row exists (unlike SELECT FOR UPDATE)
	tx, err := b.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Acquire advisory lock based on userID + action
	// This ensures mutual exclusion even when no cooldown row exists yet
	lockKey := hashUserAction(userID, action)
	_, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", lockKey)
	if err != nil {
		return fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	// Recheck cooldown with exclusive lock acquired
	// Use getLastUsedTx directly as we are already in a transaction
	lastUsed, err := b.getLastUsedTx(ctx, tx, userID, action)
	if err != nil {
		return fmt.Errorf("failed to get cooldown within transaction: %w", err)
	}

	if lastUsed != nil {
		cooldownDuration, err := b.getEffectiveCooldown(ctx, action)
		if err != nil {
			return fmt.Errorf("failed to get effective cooldown: %w", err)
		}

		onCooldown, remaining := b.checkCooldownInternal(lastUsed, cooldownDuration)
		if onCooldown {
			log.Debug("Race condition detected - concurrent request on cooldown",
				"action", action, "userID", userID, "remaining", remaining)
			return ErrOnCooldown{Action: action, Remaining: remaining}
		}
	}

	// Execute user function
	if err := fn(); err != nil {
		// User function failed - rollback, don't update cooldown
		return err
	}

	// Update cooldown within transaction
	now := time.Now()
	if err := b.updateCooldownTx(ctx, tx, userID, action, now); err != nil {
		return fmt.Errorf("failed to update cooldown: %w", err)
	}

	// Commit transaction (releases advisory lock automatically)
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit cooldown transaction: %w", err)
	}

	log.Debug("Cooldown enforced successfully", "action", action, "userID", userID)
	return nil
}

// ResetCooldown manually resets a cooldown
func (b *postgresBackend) ResetCooldown(ctx context.Context, userID, action string) error {
	query := `DELETE FROM user_cooldowns WHERE user_id = $1 AND action_name = $2`
	_, err := b.db.Exec(ctx, query, userID, action)
	if err != nil {
		return fmt.Errorf("failed to reset cooldown: %w", err)
	}
	return nil
}

// GetLastUsed returns when action was last performed
func (b *postgresBackend) GetLastUsed(ctx context.Context, userID, action string) (*time.Time, error) {
	return b.getLastUsed(ctx, userID, action)
}

// getLastUsed retrieves last used time (unlocked read)
func (b *postgresBackend) getLastUsed(ctx context.Context, userID, action string) (*time.Time, error) {
	var lastUsed time.Time
	query := `
		SELECT last_used_at
		FROM user_cooldowns
		WHERE user_id = $1 AND action_name = $2
	`

	err := b.db.QueryRow(ctx, query, userID, action).Scan(&lastUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No cooldown record
		}
		return nil, fmt.Errorf("failed to get last used: %w", err)
	}
	return &lastUsed, nil
}

// updateCooldown updates cooldown outside transaction
func (b *postgresBackend) updateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	query := `
		INSERT INTO user_cooldowns (user_id, action_name, last_used_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, action_name) DO UPDATE
		SET last_used_at = EXCLUDED.last_used_at
	`

	_, err := b.db.Exec(ctx, query, userID, action, timestamp)
	return err
}

// updateCooldownTx updates cooldown within transaction
func (b *postgresBackend) updateCooldownTx(ctx context.Context, tx pgx.Tx, userID, action string, timestamp time.Time) error {
	query := `
		INSERT INTO user_cooldowns (user_id, action_name, last_used_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, action_name) DO UPDATE
		SET last_used_at = EXCLUDED.last_used_at
	`

	_, err := tx.Exec(ctx, query, userID, action, timestamp)
	return err
}

// getLastUsedTx retrieves last used time within a transaction (unlocked read)
func (b *postgresBackend) getLastUsedTx(ctx context.Context, tx pgx.Tx, userID, action string) (*time.Time, error) {
	var lastUsed time.Time
	query := `
		SELECT last_used_at
		FROM user_cooldowns
		WHERE user_id = $1 AND action_name = $2
	`

	err := tx.QueryRow(ctx, query, userID, action).Scan(&lastUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No cooldown record
		}
		return nil, fmt.Errorf("failed to get last used: %w", err)
	}
	return &lastUsed, nil
}

// hashUserAction creates a consistent int64 hash from userID + action for advisory locking
func hashUserAction(userID, action string) int64 {
	h := sha256.Sum256([]byte(userID + ":" + action))
	// Use first 8 bytes as int64, masking MSB to ensure positive value and avoid overflow warning
	return int64(binary.BigEndian.Uint64(h[:8]) & 0x7FFFFFFFFFFFFFFF)
}

func (b *postgresBackend) getEffectiveCooldown(ctx context.Context, action string) (time.Duration, error) {
	duration := b.config.GetCooldownDuration(action)

	// Apply progression modifiers (e.g., cooldown reduction for search)
	if b.progressionSvc != nil && action == "search" {
		modifiedDuration, err := b.progressionSvc.GetModifiedValue(ctx, "search_cooldown_reduction", float64(duration))
		if err == nil {
			return time.Duration(modifiedDuration), nil
		}
	}

	return duration, nil
}

func (b *postgresBackend) checkCooldownInternal(lastUsed *time.Time, duration time.Duration) (bool, time.Duration) {
	if lastUsed == nil {
		return false, 0
	}

	elapsed := time.Since(*lastUsed)
	if elapsed < duration {
		return true, duration - elapsed
	}

	return false, 0
}
