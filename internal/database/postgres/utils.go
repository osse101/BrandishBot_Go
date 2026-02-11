package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

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

// ---- Common Helper Functions ----

// parseUserUUID parses a user ID string to uuid.UUID with consistent error message.
// Use this instead of repeating uuid.Parse + error wrapping throughout the codebase.
func parseUserUUID(userID string) (uuid.UUID, error) {
	u, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user id: %w", err)
	}
	return u, nil
}

// ptrTime converts a pgtype.Timestamp to *time.Time.
// Returns nil if the timestamp is not valid.
func ptrTime(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// ptrInt converts a pgtype.Int4 to *int.
// Returns nil if the int is not valid.
func ptrInt(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// numericToFloat64 safely converts pgtype.Numeric to float64.
// Returns (0, error) if conversion fails instead of silently ignoring errors.
func numericToFloat64(n pgtype.Numeric) (float64, error) {
	val, err := n.Float64Value()
	if err != nil {
		return 0, fmt.Errorf("failed to convert numeric to float64: %w", err)
	}
	return val.Float64, nil
}

// txHelper wraps common transaction begin logic.
// Returns a transaction and queries instance with the transaction applied.
type txHelper struct {
	tx pgx.Tx
	q  *generated.Queries
}

// beginTx starts a new transaction and returns a txHelper for common operations.
// Use SafeRollback in defer to ensure proper cleanup.
func beginTx(ctx context.Context, db *pgxpool.Pool, q *generated.Queries) (*txHelper, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &txHelper{
		tx: tx,
		q:  q.WithTx(tx),
	}, nil
}

// Commit commits the transaction
func (h *txHelper) Commit(ctx context.Context) error {
	return h.tx.Commit(ctx)
}

// Tx returns the underlying transaction for SafeRollback
func (h *txHelper) Tx() pgx.Tx {
	return h.tx
}

// Queries returns the transaction-bound queries
func (h *txHelper) Queries() *generated.Queries {
	return h.q
}

// ---- End Common Helper Functions ----

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
	return getInventoryInternal(ctx, q, userID, false)
}

// getInventoryForUpdate retrieves inventory with row locking (shared helper)
func getInventoryForUpdate(ctx context.Context, q *generated.Queries, userID string) (*domain.Inventory, error) {
	return getInventoryInternal(ctx, q, userID, true)
}

func getInventoryInternal(ctx context.Context, q *generated.Queries, userID string, forUpdate bool) (*domain.Inventory, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	var inventoryData []byte
	var fetchErr error
	if forUpdate {
		inventoryData, fetchErr = q.GetInventoryForUpdate(ctx, userUUID)
	} else {
		inventoryData, fetchErr = q.GetInventory(ctx, userUUID)
	}

	if fetchErr != nil {
		if errors.Is(fetchErr, pgx.ErrNoRows) {
			return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
		}
		op := "get inventory"
		if forUpdate {
			op = "get inventory for update"
		}
		return nil, fmt.Errorf("failed to %s: %w", op, fetchErr)
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

func getItemByName(ctx context.Context, q *generated.Queries, itemName string) (*domain.Item, error) {
	row, err := q.GetItemByName(ctx, itemName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil if item not found
		}
		return nil, fmt.Errorf("failed to get item by name: %w", err)
	}

	return mapItemFields(row.ItemID, row.InternalName, row.PublicName, row.DefaultDisplay, row.ItemDescription, row.BaseValue, row.Handler, row.ContentType, row.Types), nil
}

func getItemByID(ctx context.Context, q *generated.Queries, id int) (*domain.Item, error) {
	row, err := q.GetItemByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item by id: %w", err)
	}

	return mapItemFields(row.ItemID, row.InternalName, row.PublicName, row.DefaultDisplay, row.ItemDescription, row.BaseValue, row.Handler, row.ContentType, row.Types), nil
}

func getItemsByIDs(ctx context.Context, q *generated.Queries, itemIDs []int) ([]domain.Item, error) {
	if len(itemIDs) == 0 {
		return []domain.Item{}, nil
	}

	ids := make([]int32, len(itemIDs))
	for i, id := range itemIDs {
		ids[i] = int32(id)
	}

	rows, err := q.GetItemsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get items by ids: %w", err)
	}

	items := make([]domain.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapItemFields(row.ItemID, row.InternalName, row.PublicName, row.DefaultDisplay, row.ItemDescription, row.BaseValue, row.Handler, row.ContentType, row.Types))
	}
	return items, nil
}

func mapItemFields(itemID int32, internalName string, publicName, defaultDisplay, itemDescription pgtype.Text, baseValue pgtype.Int4, handler pgtype.Text, contentType []string, types []string) *domain.Item {
	return &domain.Item{
		ID:             int(itemID),
		InternalName:   internalName,
		PublicName:     publicName.String,
		DefaultDisplay: defaultDisplay.String,
		Description:    itemDescription.String,
		BaseValue:      int(baseValue.Int32),
		Handler:        textToPtr(handler),
		ContentType:    contentType,
		Types:          types,
	}
}

func mapProgressionNodeFields(id int32, nodeKey, nodeType, displayName string, description pgtype.Text, maxLevel, unlockCost pgtype.Int4, tier int32, size, category string, sortOrder pgtype.Int4, createdAt pgtype.Timestamp) *domain.ProgressionNode {
	return &domain.ProgressionNode{
		ID:          int(id),
		NodeKey:     nodeKey,
		NodeType:    nodeType,
		DisplayName: displayName,
		Description: description.String,
		MaxLevel:    int(maxLevel.Int32),
		UnlockCost:  int(unlockCost.Int32),
		Tier:        int(tier),
		Size:        size,
		Category:    category,
		SortOrder:   int(sortOrder.Int32),
		CreatedAt:   createdAt.Time,
	}
}
