package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// SafeRollback rolls back a transaction and logs any error that isn't ErrTxClosed
func SafeRollback(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		logger.FromContext(ctx).Error("Failed to rollback transaction", "error", err)
	}
}

func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

// mapUserAndLinks maps database rows to a User domain object
func mapUserAndLinks(ctx context.Context, q *generated.Queries, userID uuid.UUID, username string) (*domain.User, error) {
	user := domain.User{
		ID:       userID.String(),
		Username: username,
	}

	links, err := q.GetUserPlatformLinks(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user links: %w", err)
	}

	for _, link := range links {
		switch link.Name {
		case "twitch":
			user.TwitchID = link.PlatformUserID
		case "youtube":
			user.YoutubeID = link.PlatformUserID
		case "discord":
			user.DiscordID = link.PlatformUserID
		}
	}

	return &user, nil
}

// getInventory retrieves inventory (shared helper)
func getInventory(ctx context.Context, q *generated.Queries, userID string) (*domain.Inventory, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	inventoryData, err := q.GetInventory(ctx, userUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	var inventory domain.Inventory
	if err := json.Unmarshal(inventoryData, &inventory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inventory: %w", err)
	}

	if inventory.Slots == nil {
		inventory.Slots = []domain.InventorySlot{}
	}

	return &inventory, nil
}

// getInventoryForUpdate retrieves inventory with row locking (shared helper)
func getInventoryForUpdate(ctx context.Context, q *generated.Queries, userID string) (*domain.Inventory, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	inventoryData, err := q.GetInventoryForUpdate(ctx, userUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
		}
		return nil, fmt.Errorf("failed to get inventory for update: %w", err)
	}

	var inventory domain.Inventory
	if err := json.Unmarshal(inventoryData, &inventory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inventory: %w", err)
	}

	if inventory.Slots == nil {
		inventory.Slots = []domain.InventorySlot{}
	}

	return &inventory, nil
}

// updateInventory updates inventory (shared helper)
func updateInventory(ctx context.Context, q *generated.Queries, userID string, inventory domain.Inventory) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	inventoryJSON, err := json.Marshal(inventory)
	if err != nil {
		return fmt.Errorf("failed to marshal inventory: %w", err)
	}

	err = q.UpdateInventory(ctx, generated.UpdateInventoryParams{
		UserID:        userUUID,
		InventoryData: inventoryJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}
	return nil
}

// strToText converts a string to pgtype.Text
func strToText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// ptrToText converts a string pointer to pgtype.Text
func ptrToText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// intToInt4 converts an int to pgtype.Int4
func intToInt4(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(i), Valid: true}
}
