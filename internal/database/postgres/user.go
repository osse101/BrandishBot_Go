package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// UserRepository implements the user repository for PostgreSQL
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// UserTx implements transactional operations
type UserTx struct {
	tx pgx.Tx
}

// BeginTx starts a new transaction
func (r *UserRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &UserTx{tx: tx}, nil
}

// GetInventory retrieves inventory within a transaction with an exclusive lock
func (t *UserTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	query := `SELECT inventory_data FROM user_inventory WHERE user_id = $1 FOR UPDATE`
	var inventory domain.Inventory
	err := t.tx.QueryRow(ctx, query, userID).Scan(&inventory)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}
	return &inventory, nil
}

// UpdateInventory updates inventory within a transaction
func (t *UserTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	query := `
		INSERT INTO user_inventory (user_id, inventory_data)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET inventory_data = EXCLUDED.inventory_data
	`
	_, err := t.tx.Exec(ctx, query, userID, inventory)
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

// GetLastCooldownForUpdate retrieves the last time a user performed an action, locking the row for update.
// If the row does not exist, it inserts a dummy record (with zero time) to establish a lock.
func (t *UserTx) GetLastCooldownForUpdate(ctx context.Context, userID, action string) (*time.Time, error) {
	// We use an atomic "Insert if not exists, otherwise return existing" pattern with locking.
	// The dummy time for new rows is time.Time{} (zero value).
	// We use ON CONFLICT DO UPDATE to ensure we lock the row even if we don't change it.
	query := `
		INSERT INTO user_cooldowns (user_id, action_name, last_used_at)
		VALUES ($1, $2, '0001-01-01 00:00:00')
		ON CONFLICT (user_id, action_name) DO UPDATE
		SET last_used_at = user_cooldowns.last_used_at
		RETURNING last_used_at
	`
	var lastUsed time.Time
	err := t.tx.QueryRow(ctx, query, userID, action).Scan(&lastUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to get and lock cooldown: %w", err)
	}

	// If the time is zero (or very close to it), treat it as nil (never used)
	if lastUsed.IsZero() || lastUsed.Year() < 2000 {
		return nil, nil
	}

	return &lastUsed, nil
}

// UpdateCooldown updates the cooldown timestamp within the transaction
func (t *UserTx) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	query := `
		UPDATE user_cooldowns
		SET last_used_at = $3
		WHERE user_id = $1 AND action_name = $2
	`
	_, err := t.tx.Exec(ctx, query, userID, action, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update cooldown: %w", err)
	}
	return nil
}

// UpsertUser inserts a new user or updates existing user and their platform links
func (r *UserRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer SafeRollback(ctx, tx)

	// 1. Upsert User Core Data
	var userID string
	if user.ID == "" {
		// Insert new user
		query := `
			INSERT INTO users (username, created_at, updated_at)
			VALUES ($1, NOW(), NOW())
			RETURNING user_id
		`
		err := tx.QueryRow(ctx, query, user.Username).Scan(&userID)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
		user.ID = userID
	} else {
		// Update existing user
		query := `
			UPDATE users 
			SET username = $1, updated_at = NOW()
			WHERE user_id = $2
		`
		_, err := tx.Exec(ctx, query, user.Username, user.ID)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		userID = user.ID
	}

	// 2. Upsert Platform Links
	platforms := map[string]string{
		"twitch":  user.TwitchID,
		"youtube": user.YoutubeID,
		"discord": user.DiscordID,
	}

	for platformName, externalID := range platforms {
		if externalID == "" {
			continue
		}

		// Get Platform ID
		var platformID int
		err := tx.QueryRow(ctx, "SELECT platform_id FROM platforms WHERE name = $1", platformName).Scan(&platformID)
		if err != nil {
			return fmt.Errorf("failed to get platform id for %s: %w", platformName, err)
		}

		// Upsert Link
		linkQuery := `
			INSERT INTO user_platform_links (user_id, platform_id, platform_user_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, platform_id) DO UPDATE
			SET platform_user_id = EXCLUDED.platform_user_id
		`
		_, err = tx.Exec(ctx, linkQuery, userID, platformID, externalID)
		if err != nil {
			return fmt.Errorf("failed to upsert link for %s: %w", platformName, err)
		}
	}

	return tx.Commit(ctx)
}

// GetUserByPlatformID finds a user by their platform-specific ID
// GetUserByPlatformID finds a user by their platform-specific ID
func (r *UserRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	// 1. Find User ID
	query := `
		SELECT u.user_id, u.username
		FROM users u
		JOIN user_platform_links upl ON u.user_id = upl.user_id
		JOIN platforms p ON upl.platform_id = p.platform_id
		WHERE p.name = $1 AND upl.platform_user_id = $2
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, platform, platformID).Scan(&user.ID, &user.Username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user core data: %w", err)
	}

	// 2. Fetch all platform links for this user
	linksQuery := `
		SELECT p.name, upl.platform_user_id
		FROM user_platform_links upl
		JOIN platforms p ON upl.platform_id = p.platform_id
		WHERE upl.user_id = $1
	`
	rows, err := r.db.Query(ctx, linksQuery, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user links: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pName, extID string
		if err := rows.Scan(&pName, &extID); err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}
		switch pName {
		case "twitch":
			user.TwitchID = extID
		case "youtube":
			user.YoutubeID = extID
		case "discord":
			user.DiscordID = extID
		}
	}

	return &user, nil
}

// GetInventory retrieves the user's inventory
func (r *UserRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	query := `SELECT inventory_data FROM user_inventory WHERE user_id = $1`
	var inventory domain.Inventory
	err := r.db.QueryRow(ctx, query, userID).Scan(&inventory)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Return empty inventory if not found
			return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
		}
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}
	return &inventory, nil
}

// UpdateInventory updates the user's inventory
func (r *UserRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	query := `
		INSERT INTO user_inventory (user_id, inventory_data)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET inventory_data = EXCLUDED.inventory_data
	`
	_, err := r.db.Exec(ctx, query, userID, inventory)
	if err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}
	return nil
}

// GetItemByName retrieves an item by its internal name
func (r *UserRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	query := `
		SELECT item_id, internal_name, public_name, default_display, description, base_value, handler
		FROM items 
		WHERE internal_name = $1
	`
	var item domain.Item
	err := r.db.QueryRow(ctx, query, itemName).Scan(
		&item.ID, &item.InternalName, &item.PublicName, &item.DefaultDisplay,
		&item.Description, &item.BaseValue, &item.Handler,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item by name: %w", err)
	}
	return &item, nil
}

// GetItemByPublicName retrieves an item by its public name
func (r *UserRepository) GetItemByPublicName(ctx context.Context, publicName string) (*domain.Item, error) {
	query := `
		SELECT item_id, internal_name, public_name, default_display, description, base_value, handler
		FROM items 
		WHERE public_name = $1
	`
	var item domain.Item
	err := r.db.QueryRow(ctx, query, publicName).Scan(
		&item.ID, &item.InternalName, &item.PublicName, &item.DefaultDisplay,
		&item.Description, &item.BaseValue, &item.Handler,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item by public name: %w", err)
	}
	return &item, nil
}

// GetItemsByIDs retrieves multiple items by their IDs
func (r *UserRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	if len(itemIDs) == 0 {
		return []domain.Item{}, nil
	}

	query := `
		SELECT item_id, internal_name, public_name, default_display, description, base_value, handler
		FROM items
		WHERE item_id = ANY($1)
	`
	rows, err := r.db.Query(ctx, query, itemIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get items by ids: %w", err)
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(
			&item.ID, &item.InternalName, &item.PublicName, &item.DefaultDisplay,
			&item.Description, &item.BaseValue, &item.Handler,
		); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return items, nil
}

// GetItemByID retrieves an item by its ID
func (r *UserRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	query := `
		SELECT item_id, internal_name, public_name, default_display, description, base_value, handler
		FROM items 
		WHERE item_id = $1
	`
	var item domain.Item
	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.InternalName, &item.PublicName, &item.DefaultDisplay,
		&item.Description, &item.BaseValue, &item.Handler,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item by id: %w", err)
	}
	return &item, nil
}

// GetUserByUsername retrieves a user by their username
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `SELECT user_id, username, created_at, updated_at FROM users WHERE username = $1`
	var user domain.User
	err := r.db.QueryRow(ctx, query, username).Scan(&user.ID, &user.Username, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

// GetSellablePrices retrieves all sellable items with their prices
func (r *UserRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	query := `
		SELECT DISTINCT i.item_id, i.item_name, i.item_description, i.base_value
		FROM items i
		INNER JOIN item_type_assignments ita ON i.item_id = ita.item_id
		INNER JOIN item_types it ON ita.item_type_id = it.item_type_id
		WHERE it.type_name = 'sellable'
		ORDER BY i.item_name
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sellable items: %w", err)
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(&item.ID, &item.InternalName, &item.Description, &item.BaseValue); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return items, nil
}

// IsItemBuyable checks if an item has the 'buyable' type
func (r *UserRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM items i
			JOIN item_type_assignments ita ON i.item_id = ita.item_id
			JOIN item_types it ON ita.item_type_id = it.item_type_id
			WHERE i.item_name = $1 AND it.type_name = 'buyable'
		)
	`
	var isBuyable bool
	err := r.db.QueryRow(ctx, query, itemName).Scan(&isBuyable)
	if err != nil {
		return false, fmt.Errorf("failed to check if item is buyable: %w", err)
	}
	return isBuyable, nil
}

// GetRecipeByTargetItemID retrieves a recipe by its target item ID
func (r *UserRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	query := `SELECT recipe_id, target_item_id, base_cost, created_at FROM crafting_recipes WHERE target_item_id = $1`
	var recipe domain.Recipe
	err := r.db.QueryRow(ctx, query, itemID).Scan(&recipe.ID, &recipe.TargetItemID, &recipe.BaseCost, &recipe.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get recipe by target item id: %w", err)
	}
	return &recipe, nil
}

// IsRecipeUnlocked checks if a user has unlocked a specific recipe
func (r *UserRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM recipe_unlocks WHERE user_id = $1 AND recipe_id = $2)`
	var unlocked bool
	err := r.db.QueryRow(ctx, query, userID, recipeID).Scan(&unlocked)
	if err != nil {
		return false, fmt.Errorf("failed to check if recipe is unlocked: %w", err)
	}
	return unlocked, nil
}

// UnlockRecipe unlocks a recipe for a user
func (r *UserRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	query := `
		INSERT INTO recipe_unlocks (user_id, recipe_id, unlocked_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, recipe_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, userID, recipeID)
	if err != nil {
		return fmt.Errorf("failed to unlock recipe: %w", err)
	}
	return nil
}

// GetUnlockedRecipesForUser retrieves all recipes unlocked by a specific user
func (r *UserRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]crafting.UnlockedRecipeInfo, error) {
	query := `
		SELECT i.item_name, r.target_item_id
		FROM crafting_recipes r
		JOIN recipe_unlocks ru ON r.recipe_id = ru.recipe_id
		JOIN items i ON r.target_item_id = i.item_id
		WHERE ru.user_id = $1
		ORDER BY i.item_name
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unlocked recipes: %w", err)
	}
	defer rows.Close()

	var recipes []crafting.UnlockedRecipeInfo
	for rows.Next() {
		var recipe crafting.UnlockedRecipeInfo
		if err := rows.Scan(&recipe.ItemName, &recipe.ItemID); err != nil {
			return nil, fmt.Errorf("failed to scan recipe: %w", err)
		}
		recipes = append(recipes, recipe)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return recipes, nil
}

// GetDisassembleRecipeBySourceItemID retrieves a disassemble recipe for a given source item
func (r *UserRepository) GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error) {
	// Get the recipe details
	var recipe domain.DisassembleRecipe
	query := `
		SELECT recipe_id, source_item_id, quantity_consumed, created_at
		FROM disassemble_recipes
		WHERE source_item_id = $1
	`

	err := r.db.QueryRow(ctx, query, itemID).Scan(
		&recipe.ID,
		&recipe.SourceItemID,
		&recipe.QuantityConsumed,
		&recipe.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No recipe found
		}
		return nil, fmt.Errorf("failed to query disassemble recipe: %w", err)
	}

	// Get the outputs for this recipe
	outputQuery := `
		SELECT item_id, quantity
		FROM disassemble_outputs
		WHERE recipe_id = $1
		ORDER BY item_id
	`

	rows, err := r.db.Query(ctx, outputQuery, recipe.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query disassemble outputs: %w", err)
	}
	defer rows.Close()

	var outputs []domain.RecipeOutput
	for rows.Next() {
		var output domain.RecipeOutput
		if err := rows.Scan(&output.ItemID, &output.Quantity); err != nil {
			return nil, fmt.Errorf("failed to scan output: %w", err)
		}
		outputs = append(outputs, output)
	}

	recipe.Outputs = outputs
	return &recipe, nil
}

// GetAssociatedUpgradeRecipeID retrieves the upgrade recipe ID associated with a disassemble recipe
func (r *UserRepository) GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error) {
	var upgradeRecipeID int
	query := `
		SELECT upgrade_recipe_id
		FROM recipe_associations
		WHERE disassemble_recipe_id = $1
	`

	err := r.db.QueryRow(ctx, query, disassembleRecipeID).Scan(&upgradeRecipeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("no associated upgrade recipe found for disassemble recipe %d", disassembleRecipeID)
		}
		return 0, fmt.Errorf("failed to query associated upgrade recipe: %w", err)
	}

	return upgradeRecipeID, nil
}

// GetLastCooldown retrieves the last time a user performed an action
func (r *UserRepository) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	var lastUsed time.Time
	query := `
		SELECT last_used_at
		FROM user_cooldowns
		WHERE user_id = $1 AND action_name = $2
	`

	err := r.db.QueryRow(ctx, query, userID, action).Scan(&lastUsed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No cooldown record
		}
		return nil, fmt.Errorf("failed to get cooldown: %w", err)
	}
	return &lastUsed, nil
}

// UpdateCooldown updates or creates a cooldown record for a user action
func (r *UserRepository) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	query := `
		INSERT INTO user_cooldowns (user_id, action_name, last_used_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, action_name) DO UPDATE
		SET last_used_at = EXCLUDED.last_used_at
	`

	_, err := r.db.Exec(ctx, query, userID, action, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update cooldown: %w", err)
	}
	return nil
}
