package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
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
	r  *UserRepository
}

// BeginTx starts a new transaction
func (r *UserRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &UserTx{
		tx: tx,
		q:  r.q.WithTx(tx),
		r:  r,
	}, nil
}

// GetInventory retrieves inventory within a transaction with an exclusive lock
func (t *UserTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	emptyInventory := domain.Inventory{Slots: []domain.InventorySlot{}}
	inventoryJSON, err := json.Marshal(emptyInventory)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal empty inventory: %w", err)
	}

	err = t.q.EnsureInventoryRow(ctx, generated.EnsureInventoryRowParams{
		UserID:        userUUID,
		InventoryData: inventoryJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to ensure inventory row exists: %w", err)
	}

	inventoryData, err := t.q.GetInventoryForUpdate(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	var inventory domain.Inventory
	if err := json.Unmarshal(inventoryData, &inventory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal inventory: %w", err)
	}
	return &inventory, nil
}

// UpdateInventory updates inventory within a transaction
func (t *UserTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	inventoryJSON, err := json.Marshal(inventory)
	if err != nil {
		return fmt.Errorf("failed to marshal inventory: %w", err)
	}

	err = t.q.UpdateInventory(ctx, generated.UpdateInventoryParams{
		UserID:        userUUID,
		InventoryData: inventoryJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}
	return nil
}

// Commit commits the transaction
func (t *UserTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *UserTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
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
		"twitch":  user.TwitchID,
		"youtube": user.YoutubeID,
		"discord": user.DiscordID,
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

	return tx.Commit(ctx)
}

// GetUserByPlatformID finds a user by their platform-specific ID
func (r *UserRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	row, err := r.q.GetUserByPlatformID(ctx, generated.GetUserByPlatformIDParams{
		Name:           platform,
		PlatformUserID: platformID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user core data: %w", err)
	}

	user := domain.User{
		ID:       row.UserID.String(),
		Username: row.Username,
	}

	links, err := r.q.GetUserPlatformLinks(ctx, row.UserID)
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

// GetUserByPlatformUsername finds a user by platform and username (case-insensitive)
func (r *UserRepository) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	row, err := r.q.GetUserByPlatformUsername(ctx, generated.GetUserByPlatformUsernameParams{
		Lower: username,
		Name:  platform,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	user := domain.User{
		ID:       row.UserID.String(),
		Username: row.Username,
	}

	links, err := r.q.GetUserPlatformLinks(ctx, row.UserID)
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

// GetInventory retrieves the user's inventory
func (r *UserRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	inventoryData, err := r.q.GetInventory(ctx, userUUID)
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
	return &inventory, nil
}

// UpdateInventory updates the user's inventory
func (r *UserRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	inventoryJSON, err := json.Marshal(inventory)
	if err != nil {
		return fmt.Errorf("failed to marshal inventory: %w", err)
	}

	err = r.q.UpdateInventory(ctx, generated.UpdateInventoryParams{
		UserID:        userUUID,
		InventoryData: inventoryJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}
	return nil
}

func textToPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

// GetItemByName retrieves an item by its internal name
func (r *UserRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	row, err := r.q.GetItemByName(ctx, itemName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Return nil if item not found, as per original contract
		}
		return nil, fmt.Errorf("failed to get item by name: %w", err)
	}

	return &domain.Item{
		ID:             int(row.ItemID),
		InternalName:   row.InternalName,
		PublicName:     row.PublicName.String,
		DefaultDisplay: row.DefaultDisplay.String,
		Description:    row.ItemDescription.String,
		BaseValue:      int(row.BaseValue.Int32),
		Handler:        textToPtr(row.Handler),
		Types:          row.Types,
	}, nil
}

// GetItemByPublicName retrieves an item by its public name
func (r *UserRepository) GetItemByPublicName(ctx context.Context, publicName string) (*domain.Item, error) {
	row, err := r.q.GetItemByPublicName(ctx, pgtype.Text{String: publicName, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item by public name: %w", err)
	}

	return &domain.Item{
		ID:             int(row.ItemID),
		InternalName:   row.InternalName,
		PublicName:     row.PublicName.String,
		DefaultDisplay: row.DefaultDisplay.String,
		Description:    row.ItemDescription.String,
		BaseValue:      int(row.BaseValue.Int32),
		Handler:        textToPtr(row.Handler),
		Types:          row.Types,
	}, nil
}

// GetItemsByIDs retrieves multiple items by their IDs
func (r *UserRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	if len(itemIDs) == 0 {
		return []domain.Item{}, nil
	}

	ids := make([]int32, len(itemIDs))
	for i, id := range itemIDs {
		ids[i] = int32(id)
	}

	rows, err := r.q.GetItemsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get items by ids: %w", err)
	}

	var items []domain.Item
	for _, row := range rows {
		items = append(items, domain.Item{
			ID:             int(row.ItemID),
			InternalName:   row.InternalName,
			PublicName:     row.PublicName.String,
			DefaultDisplay: row.DefaultDisplay.String,
			Description:    row.ItemDescription.String,
			BaseValue:      int(row.BaseValue.Int32),
			Handler:        textToPtr(row.Handler),
			Types:          row.Types,
		})
	}
	return items, nil
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

	var items []domain.Item
	for _, row := range rows {
		items = append(items, domain.Item{
			ID:             int(row.ItemID),
			InternalName:   row.InternalName,
			PublicName:     row.PublicName.String,
			DefaultDisplay: row.DefaultDisplay.String,
			Description:    row.ItemDescription.String,
			BaseValue:      int(row.BaseValue.Int32),
			Handler:        textToPtr(row.Handler),
			Types:          row.Types,
		})
	}
	return items, nil
}

// GetItemByID retrieves an item by its ID
func (r *UserRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	row, err := r.q.GetItemByID(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item by id: %w", err)
	}

	return &domain.Item{
		ID:             int(row.ItemID),
		InternalName:   row.InternalName,
		PublicName:     row.PublicName.String,
		DefaultDisplay: row.DefaultDisplay.String,
		Description:    row.ItemDescription.String,
		BaseValue:      int(row.BaseValue.Int32),
		Handler:        textToPtr(row.Handler),
		Types:          row.Types,
	}, nil
}

// GetUserByUsername retrieves a user by their username
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	row, err := r.q.GetUserByUsername(ctx, username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &domain.User{
		ID:        row.UserID.String(),
		Username:  row.Username,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

// GetSellablePrices retrieves all sellable items with their prices
func (r *UserRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	rows, err := r.q.GetSellablePrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query sellable items: %w", err)
	}

	var items []domain.Item
	for _, row := range rows {
		items = append(items, domain.Item{
			ID:           int(row.ItemID),
			InternalName: row.InternalName,
			Description:  row.ItemDescription.String,
			BaseValue:    int(row.BaseValue.Int32),
		})
	}

	return items, nil
}

// IsItemBuyable checks if an item has the 'buyable' type
func (r *UserRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	return r.q.IsItemBuyable(ctx, itemName)
}

// GetRecipeByTargetItemID retrieves a recipe by its target item ID
func (r *UserRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	row, err := r.q.GetRecipeByTargetItemID(ctx, int32(itemID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get recipe by target item id: %w", err)
	}

	recipe := domain.Recipe{
		ID:           int(row.RecipeID),
		TargetItemID: int(row.TargetItemID),
		// BaseCost is []RecipeCost here
		CreatedAt: row.CreatedAt.Time,
	}

	// Unmarshal BaseCost which is []byte (JSONB)
	if len(row.BaseCost) > 0 {
		if err := json.Unmarshal(row.BaseCost, &recipe.BaseCost); err != nil {
			return nil, fmt.Errorf("failed to unmarshal base cost: %w", err)
		}
	} else {
		recipe.BaseCost = []domain.RecipeCost{}
	}

	return &recipe, nil
}

// IsRecipeUnlocked checks if a user has unlocked a specific recipe
func (r *UserRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user id: %w", err)
	}
	return r.q.IsRecipeUnlocked(ctx, generated.IsRecipeUnlockedParams{
		UserID:   userUUID,
		//nolint:gosec // DB IDs fit in int32
		RecipeID: int32(recipeID),
	})
}

// UnlockRecipe unlocks a recipe for a user
func (r *UserRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	err = r.q.UnlockRecipe(ctx, generated.UnlockRecipeParams{
		UserID:   userUUID,
		//nolint:gosec // DB IDs fit in int32
		RecipeID: int32(recipeID),
	})
	if err != nil {
		return fmt.Errorf("failed to unlock recipe: %w", err)
	}
	return nil
}

// GetUnlockedRecipesForUser retrieves all recipes unlocked by a specific user
func (r *UserRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]crafting.UnlockedRecipeInfo, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	rows, err := r.q.GetUnlockedRecipesForUser(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unlocked recipes: %w", err)
	}

	var recipes []crafting.UnlockedRecipeInfo
	for _, row := range rows {
		recipes = append(recipes, crafting.UnlockedRecipeInfo{
			ItemName: row.ItemName,
			ItemID:   int(row.ItemID),
		})
	}
	return recipes, nil
}

// GetAllRecipes retrieves all crafting recipes
func (r *UserRepository) GetAllRecipes(ctx context.Context) ([]crafting.RecipeListItem, error) {
	rows, err := r.q.GetAllRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query all recipes: %w", err)
	}

	var recipes []crafting.RecipeListItem
	for _, row := range rows {
		recipes = append(recipes, crafting.RecipeListItem{
			ItemName: row.ItemName,
			ItemID:   int(row.ItemID),
		})
	}
	return recipes, nil
}

// GetDisassembleRecipeBySourceItemID retrieves a disassemble recipe for a given source item
func (r *UserRepository) GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error) {
	//nolint:gosec // DB IDs fit in int32
	row, err := r.q.GetDisassembleRecipeBySourceItemID(ctx, int32(itemID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query disassemble recipe: %w", err)
	}

	recipe := domain.DisassembleRecipe{
		ID:               int(row.RecipeID),
		SourceItemID:     int(row.SourceItemID),
		QuantityConsumed: int(row.QuantityConsumed),
		CreatedAt:        row.CreatedAt.Time,
	}

	outputs, err := r.q.GetDisassembleOutputs(ctx, row.RecipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query disassemble outputs: %w", err)
	}

	for _, out := range outputs {
		recipe.Outputs = append(recipe.Outputs, domain.RecipeOutput{
			ItemID:   int(out.ItemID),
			Quantity: int(out.Quantity),
		})
	}

	return &recipe, nil
}

// GetAssociatedUpgradeRecipeID retrieves the upgrade recipe ID associated with a disassemble recipe
func (r *UserRepository) GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error) {
	//nolint:gosec // DB IDs fit in int32
	id, err := r.q.GetAssociatedUpgradeRecipeID(ctx, int32(disassembleRecipeID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("no associated upgrade recipe found for disassemble recipe %d", disassembleRecipeID)
		}
		return 0, fmt.Errorf("failed to query associated upgrade recipe: %w", err)
	}
	return int(id), nil
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
		if err == pgx.ErrNoRows {
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
		if err == pgx.ErrNoRows {
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
