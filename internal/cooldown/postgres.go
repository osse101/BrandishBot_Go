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

	"github.com/osse101/BrandishBot_Go/internal/domain"
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
		return false, 0, fmt.Errorf(ErrMsgCheckCooldownFailed, err)
	}

	if lastUsed == nil {
		// Never used - not on cooldown
		return false, 0, nil
	}

	cooldownDuration := b.getEffectiveCooldown(ctx, action)

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
		log.Debug(LogMsgDevModeBypass, "action", action, "userID", userID)
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
		return fmt.Errorf(ErrMsgBeginTransactionFailed, err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Acquire advisory lock based on userID + action
	// This ensures mutual exclusion even when no cooldown row exists yet
	lockKey := hashUserAction(userID, action)
	_, err = tx.Exec(ctx, SQLAdvisoryLock, lockKey)
	if err != nil {
		return fmt.Errorf(ErrMsgAcquireLockFailed, err)
	}

	// Recheck cooldown with exclusive lock acquired
	// Use getLastUsedTx directly as we are already in a transaction
	lastUsed, err := b.getLastUsedTx(ctx, tx, userID, action)
	if err != nil {
		return fmt.Errorf(ErrMsgGetCooldownTxFailed, err)
	}

	if lastUsed != nil {
		cooldownDuration := b.getEffectiveCooldown(ctx, action)
		onCooldown, remaining := b.checkCooldownInternal(lastUsed, cooldownDuration)
		if onCooldown {
			log.Debug(LogMsgRaceConditionDetected,
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
		return fmt.Errorf(ErrMsgUpdateCooldownFailed, err)
	}

	// Commit transaction (releases advisory lock automatically)
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(ErrMsgCommitTransactionFailed, err)
	}

	log.Debug(LogMsgCooldownEnforced, "action", action, "userID", userID)
	return nil
}

// ResetCooldown manually resets a cooldown
func (b *postgresBackend) ResetCooldown(ctx context.Context, userID, action string) error {
	_, err := b.db.Exec(ctx, SQLDeleteCooldown, userID, action)
	if err != nil {
		return fmt.Errorf(ErrMsgResetCooldownFailed, err)
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

	err := b.db.QueryRow(ctx, SQLSelectLastUsed, userID, action).Scan(&lastUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No cooldown record
		}
		return nil, fmt.Errorf(ErrMsgGetLastUsedFailed, err)
	}
	return &lastUsed, nil
}

// updateCooldown updates cooldown outside transaction
func (b *postgresBackend) updateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	_, err := b.db.Exec(ctx, SQLUpsertCooldown, userID, action, timestamp)
	return err
}

// updateCooldownTx updates cooldown within transaction
func (b *postgresBackend) updateCooldownTx(ctx context.Context, tx pgx.Tx, userID, action string, timestamp time.Time) error {
	_, err := tx.Exec(ctx, SQLUpsertCooldown, userID, action, timestamp)
	return err
}

// getLastUsedTx retrieves last used time within a transaction (unlocked read)
func (b *postgresBackend) getLastUsedTx(ctx context.Context, tx pgx.Tx, userID, action string) (*time.Time, error) {
	var lastUsed time.Time

	err := tx.QueryRow(ctx, SQLSelectLastUsed, userID, action).Scan(&lastUsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No cooldown record
		}
		return nil, fmt.Errorf(ErrMsgGetLastUsedFailed, err)
	}
	return &lastUsed, nil
}

// hashUserAction creates a consistent int64 hash from userID + action for advisory locking
func hashUserAction(userID, action string) int64 {
	h := sha256.Sum256([]byte(userID + HashSeparator + action))
	// Use first 8 bytes as int64, masking MSB to ensure positive value and avoid overflow warning
	return int64(binary.BigEndian.Uint64(h[:8]) & HashMaskPositiveInt64)
}

func (b *postgresBackend) getEffectiveCooldown(ctx context.Context, action string) time.Duration {
	duration := b.config.GetCooldownDuration(action)

	// Apply progression modifiers (e.g., cooldown reduction for search)
	if b.progressionSvc != nil && action == domain.ActionSearch {
		modifiedDuration, err := b.progressionSvc.GetModifiedValue(ctx, FeatureKeySearchCooldownReduction, float64(duration))
		if err == nil {
			return time.Duration(modifiedDuration)
		}
	}

	return duration
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
