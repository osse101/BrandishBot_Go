package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// UserRepository implements the user repository for PostgreSQL
type UserRepository struct {
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
		q:  generated.New(db),
	}
}

// UserTx implements transactional operations
type UserTx struct {
	tx pgx.Tx
	q  *generated.Queries
}

// BeginTx starts a new transaction
func (r *UserRepository) BeginTx(ctx context.Context) (repository.UserTx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &UserTx{
		tx: tx,
		q:  r.q.WithTx(tx),
	}, nil
}

// GetInventory retrieves inventory within a transaction
func (t *UserTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return getInventoryForUpdate(ctx, t.q, userID)
}

// UpdateInventory updates inventory within a transaction
func (t *UserTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return updateInventory(ctx, t.q, userID, inventory)
}

// Commit commits the transaction
func (t *UserTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *UserTx) Rollback(ctx context.Context) error {
	err := t.tx.Rollback(ctx)
	if errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("%w: %w", repository.ErrTxClosed, err)
	}
	return err
}

// UpsertUser inserts a new user or updates existing user and their platform links
func (r *UserRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer SafeRollback(ctx, tx)

	q := r.q.WithTx(tx)

	var userUUID uuid.UUID
	if user.ID == "" {
		userUUID, err = q.CreateUser(ctx, user.Username)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
		user.ID = userUUID.String()
	} else {
		userUUID, err = uuid.Parse(user.ID)
		if err != nil {
			return fmt.Errorf("invalid user id: %w", err)
		}
		err = q.UpdateUser(ctx, generated.UpdateUserParams{
			Username: user.Username,
			UserID:   userUUID,
		})
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	}

	platforms := map[string]string{
		domain.PlatformTwitch:  user.TwitchID,
		domain.PlatformYoutube: user.YoutubeID,
		domain.PlatformDiscord: user.DiscordID,
	}

	for platformName, externalID := range platforms {
		if externalID == "" {
			continue
		}

		platformID, err := q.GetPlatformID(ctx, platformName)
		if err != nil {
			return fmt.Errorf("failed to get platform id for %s: %w", platformName, err)
		}

		err = q.UpsertUserPlatformLink(ctx, generated.UpsertUserPlatformLinkParams{
			UserID:         userUUID,
			PlatformID:     platformID,
			PlatformUserID: externalID,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert link for %s: %w", platformName, err)
		}
	}

	// Ensure inventory row exists to allow row-level locking
	err = q.EnsureInventoryRow(ctx, generated.EnsureInventoryRowParams{
		UserID:        userUUID,
		InventoryData: []byte(`{"slots": []}`),
	})
	if err != nil {
		return fmt.Errorf("failed to ensure inventory row: %w", err)
	}

	return tx.Commit(ctx)
}

// GetUserByPlatformID finds a user by their platform-specific ID
func (r *UserRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	row, err := r.q.GetUserByPlatformID(ctx, generated.GetUserByPlatformIDParams{
		Name:           platform,
		PlatformUserID: platformID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user core data: %w", err)
	}

	return mapUserAndLinks(ctx, r.q, row.UserID, row.Username)
}

// GetUserByPlatformUsername finds a user by platform and username (case-insensitive)
func (r *UserRepository) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	row, err := r.q.GetUserByPlatformUsername(ctx, generated.GetUserByPlatformUsernameParams{
		Lower: strings.ToLower(username),
		Name:  platform,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return mapUserAndLinks(ctx, r.q, row.UserID, row.Username)
}

// GetInventory retrieves the user's inventory
func (r *UserRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return getInventory(ctx, r.q, userID)
}

// UpdateInventory updates the user's inventory
func (r *UserRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return updateInventory(ctx, r.q, userID, inventory)
}

// GetItemByName retrieves an item by its internal name
func (r *UserRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	return getItemByName(ctx, r.q, itemName)
}

// GetItemByPublicName retrieves an item by its public name
func (r *UserRepository) GetItemByPublicName(ctx context.Context, publicName string) (*domain.Item, error) {
	row, err := r.q.GetItemByPublicName(ctx, pgtype.Text{String: publicName, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item by public name: %w", err)
	}

	return mapItemFields(row.ItemID, row.InternalName, row.PublicName, row.DefaultDisplay, row.ItemDescription, row.BaseValue, row.Handler, row.ContentType, row.Types), nil
}

// GetItemsByIDs retrieves multiple items by their IDs
func (r *UserRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return getItemsByIDs(ctx, r.q, itemIDs)
}

// GetItemsByNames retrieves multiple items by their internal names
func (r *UserRepository) GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error) {
	if len(names) == 0 {
		return []domain.Item{}, nil
	}

	rows, err := r.q.GetItemsByNames(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("failed to get items by names: %w", err)
	}

	items := make([]domain.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapItemFields(row.ItemID, row.InternalName, row.PublicName, row.DefaultDisplay, row.ItemDescription, row.BaseValue, row.Handler, row.ContentType, row.Types))
	}
	return items, nil
}

// GetAllItems retrieves all items from the database
func (r *UserRepository) GetAllItems(ctx context.Context) ([]domain.Item, error) {
	rows, err := r.q.GetAllItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all items: %w", err)
	}

	items := make([]domain.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapItemFields(row.ItemID, row.InternalName, row.PublicName, row.DefaultDisplay, row.ItemDescription, row.BaseValue, row.Handler, row.ContentType, row.Types))
	}
	return items, nil
}

// GetRecentlyActiveUsers retrieves users who recently had events
func (r *UserRepository) GetRecentlyActiveUsers(ctx context.Context, limit int) ([]domain.User, error) {
	rows, err := r.q.GetRecentlyActiveUsers(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get recently active users: %w", err)
	}

	users := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, domain.User{
			ID:        row.UserID.String(),
			Username:  row.Username,
			TwitchID:  renderPlatformID(row.Platform, row.PlatformUserID, domain.PlatformTwitch),
			YoutubeID: renderPlatformID(row.Platform, row.PlatformUserID, domain.PlatformYoutube),
			DiscordID: renderPlatformID(row.Platform, row.PlatformUserID, domain.PlatformDiscord),
		})
	}
	return users, nil
}

func renderPlatformID(platform, platformUserID, targetPlatform string) string {
	if platform == targetPlatform {
		return platformUserID
	}
	return ""
}

// GetItemByID retrieves an item by its ID
func (r *UserRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return getItemByID(ctx, r.q, id)
}

// GetLastCooldown retrieves the last time a user performed an action
func (r *UserRepository) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	lastUsed, err := r.q.GetLastCooldown(ctx, generated.GetLastCooldownParams{
		UserID:     userUUID,
		ActionName: action,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cooldown: %w", err)
	}

	t := lastUsed.Time
	return &t, nil
}

// GetLastCooldownForUpdate retrieves the last time a user performed an action with row-level lock
func (r *UserRepository) GetLastCooldownForUpdate(ctx context.Context, tx pgx.Tx, userID, action string) (*time.Time, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	q := r.q.WithTx(tx)
	lastUsed, err := q.GetLastCooldownForUpdate(ctx, generated.GetLastCooldownForUpdateParams{
		UserID:     userUUID,
		ActionName: action,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cooldown with lock: %w", err)
	}

	t := lastUsed.Time
	return &t, nil
}

// UpdateCooldown updates or creates a cooldown record for a user action
func (r *UserRepository) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	err = r.q.UpdateCooldown(ctx, generated.UpdateCooldownParams{
		UserID:     userUUID,
		ActionName: action,
		LastUsedAt: pgtype.Timestamptz{Time: timestamp, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update cooldown: %w", err)
	}
	return nil
}

// UpdateCooldownTx updates or creates a cooldown record within a transaction
func (r *UserRepository) UpdateCooldownTx(ctx context.Context, tx pgx.Tx, userID, action string, timestamp time.Time) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	q := r.q.WithTx(tx)
	err = q.UpdateCooldown(ctx, generated.UpdateCooldownParams{
		UserID:     userUUID,
		ActionName: action,
		LastUsedAt: pgtype.Timestamptz{Time: timestamp, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update cooldown: %w", err)
	}
	return nil
}
