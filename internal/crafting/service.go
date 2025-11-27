package crafting

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Repository defines the interface for data access required by the crafting service
type Repository interface {
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error)
	IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error)
	UnlockRecipe(ctx context.Context, userID string, recipeID int) error
	GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]UnlockedRecipeInfo, error)
	BeginTx(ctx context.Context) (repository.Tx, error)

	// Disassemble methods
	GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error)
	GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error)
}

// RecipeInfo represents recipe information with lock status
type RecipeInfo struct {
	ItemName string              `json:"item_name"`
	Locked   bool                `json:"locked,omitempty"`
	BaseCost []domain.RecipeCost `json:"base_cost,omitempty"`
}

// UnlockedRecipeInfo represents an unlocked recipe
type UnlockedRecipeInfo struct {
	ItemName string `json:"item_name"`
	ItemID   int    `json:"item_id"`
}

// Service defines the interface for crafting operations
type Service interface {
	UpgradeItem(ctx context.Context, username, platform, itemName string, quantity int) (string, int, error)
	GetRecipe(ctx context.Context, itemName, username string) (*RecipeInfo, error)
	GetUnlockedRecipes(ctx context.Context, username string) ([]UnlockedRecipeInfo, error)
	DisassembleItem(ctx context.Context, username, platform, itemName string, quantity int) (map[string]int, int, error)
}

type service struct {
	repo        Repository
	lockManager *concurrency.LockManager
}

// NewService creates a new crafting service
func NewService(repo Repository, lockManager *concurrency.LockManager) Service {
	return &service{
		repo:        repo,
		lockManager: lockManager,
	}
}

func (s *service) validateUser(ctx context.Context, username string) (*domain.User, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	return user, nil
}

func (s *service) validateItem(ctx context.Context, itemName string) (*domain.Item, error) {
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return nil, fmt.Errorf("item not found: %s", itemName)
	}
	return item, nil
}

// UpgradeItem upgrades as many items as possible based on available materials
func (s *service) UpgradeItem(ctx context.Context, username, platform, itemName string, quantity int) (string, int, error) {
	log := logger.FromContext(ctx)
	log.Info("UpgradeItem called", "username", username, "item", itemName, "quantity", quantity)

	// Validate user
	user, err := s.validateUser(ctx, username)
	if err != nil {
		return "", 0, err
	}

	// Validate target item
	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return "", 0, err
	}

	// Get recipe for target item
	recipe, err := s.repo.GetRecipeByTargetItemID(ctx, item.ID)
	if err != nil {
		log.Error("Failed to get recipe", "error", err, "itemID", item.ID)
		return "", 0, fmt.Errorf("failed to get recipe: %w", err)
	}
	if recipe == nil {
		log.Warn("No recipe found for item", "itemName", itemName)
		return "", 0, fmt.Errorf("no recipe found for item: %s", itemName)
	}

	// Check if user has unlocked this recipe
	unlocked, err := s.repo.IsRecipeUnlocked(ctx, user.ID, recipe.ID)
	if err != nil {
		log.Error("Failed to check recipe unlock", "error", err, "recipeID", recipe.ID)
		return "", 0, fmt.Errorf("failed to check recipe unlock: %w", err)
	}
	if !unlocked {
		log.Warn("Recipe not unlocked", "username", username, "recipeID", recipe.ID)
		return "", 0, fmt.Errorf("recipe for %s is not unlocked", itemName)
	}

	// Begin transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return "", 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get user's inventory
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Calculate max possible upgrades based on available materials
	maxPossible := quantity
	for _, cost := range recipe.BaseCost {
		// Find how many of this material the user has
		// Find how many of this material the user has
		_, userQuantity := utils.FindSlot(inventory, cost.ItemID)

		// Calculate how many upgrades this material allows
		if cost.Quantity > 0 {
			affordableWithThis := userQuantity / cost.Quantity
			if affordableWithThis < maxPossible {
				maxPossible = affordableWithThis
			}
		}
	}

	if maxPossible == 0 {
		log.Warn("Insufficient materials", "username", username, "item", itemName)
		return "", 0, fmt.Errorf("insufficient materials to craft %s", itemName)
	}

	// Actual quantity to upgrade
	actualQuantity := maxPossible
	if actualQuantity > quantity {
		actualQuantity = quantity
	}

	// Consume materials
	for _, cost := range recipe.BaseCost {
		totalNeeded := cost.Quantity * actualQuantity

		// Find the slot with this material
		// Find the slot with this material
		i, slotQuantity := utils.FindSlot(inventory, cost.ItemID)
		if i == -1 || slotQuantity < totalNeeded {
			// This should not happen due to our earlier check, but handle it anyway
			return "", 0, fmt.Errorf("insufficient material (itemID: %d)", cost.ItemID)
		}

		// Remove the materials
		if slotQuantity == totalNeeded {
			// Remove the slot entirely
			inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
		} else {
			inventory.Slots[i].Quantity -= totalNeeded
		}
	}

	// Add upgraded items
	found := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			inventory.Slots[i].Quantity += actualQuantity
			found = true
			break
		}
	}
	if !found {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{
			ItemID:   item.ID,
			Quantity: actualQuantity,
		})
	}

	// Update inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return "", 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return "", 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Items upgraded", "username", username, "item", itemName, "quantity", actualQuantity)
	return itemName, actualQuantity, nil
}

// GetRecipe returns recipe information for an item
// If username is provided, includes lock status; otherwise returns base recipe
func (s *service) GetRecipe(ctx context.Context, itemName, username string) (*RecipeInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("GetRecipe called", "itemName", itemName, "username", username)

	// Validate and get item
	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return nil, err
	}

	// Get recipe by target item ID
	recipe, err := s.repo.GetRecipeByTargetItemID(ctx, item.ID)
	if err != nil {
		log.Error("Failed to get recipe", "error", err, "itemID", item.ID)
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	if recipe == nil {
		log.Warn("No recipe found", "itemName", itemName)
		return nil, fmt.Errorf("no recipe found for item: %s", itemName)
	}

	recipeInfo := &RecipeInfo{
		ItemName: itemName,
		BaseCost: recipe.BaseCost,
	}

	// If username provided, check lock status
	if username != "" {
		user, err := s.validateUser(ctx, username)
		if err != nil {
			return nil, err
		}

		unlocked, err := s.repo.IsRecipeUnlocked(ctx, user.ID, recipe.ID)
		if err != nil {
			log.Error("Failed to check recipe unlock", "error", err)
			return nil, fmt.Errorf("failed to check recipe unlock: %w", err)
		}

		recipeInfo.Locked = !unlocked
	}

	log.Info("Recipe retrieved", "itemName", itemName, "locked", recipeInfo.Locked)
	return recipeInfo, nil
}

// GetUnlockedRecipes returns all recipes that a user has unlocked
func (s *service) GetUnlockedRecipes(ctx context.Context, username string) ([]UnlockedRecipeInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("GetUnlockedRecipes called", "username", username)

	user, err := s.validateUser(ctx, username)
	if err != nil {
		return nil, err
	}

	unlockedRecipes, err := s.repo.GetUnlockedRecipesForUser(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get unlocked recipes", "error", err)
		return nil, fmt.Errorf("failed to get unlocked recipes: %w", err)
	}

	log.Info("Unlocked recipes retrieved", "username", username, "count", len(unlockedRecipes))
	return unlockedRecipes, nil
}

// DisassembleItem disassembles items into their component materials
// Returns a map of item names to quantities and the number of items disassembled
func (s *service) DisassembleItem(ctx context.Context, username, platform, itemName string, quantity int) (map[string]int, int, error) {
	log := logger.FromContext(ctx)
	log.Info("DisassembleItem called", "username", username, "item", itemName, "quantity", quantity)

	// Validate user
	user, err := s.validateUser(ctx, username)
	if err != nil {
		return nil, 0, err
	}

	// Validate source item
	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return nil, 0, err
	}

	// Get disassemble recipe for source item
	recipe, err := s.repo.GetDisassembleRecipeBySourceItemID(ctx, item.ID)
	if err != nil {
		log.Error("Failed to get disassemble recipe", "error", err, "itemID", item.ID)
		return nil, 0, fmt.Errorf("failed to get disassemble recipe: %w", err)
	}
	if recipe == nil {
		log.Warn("No disassemble recipe found for item", "itemName", itemName)
		return nil, 0, fmt.Errorf("no disassemble recipe found for item: %s", itemName)
	}

	// Get associated upgrade recipe ID to check if unlocked
	upgradeRecipeID, err := s.repo.GetAssociatedUpgradeRecipeID(ctx, recipe.ID)
	if err != nil {
		log.Error("Failed to get associated upgrade recipe", "error", err, "disassembleRecipeID", recipe.ID)
		return nil, 0, fmt.Errorf("failed to get associated upgrade recipe: %w", err)
	}

	// Check if user has unlocked the associated upgrade recipe
	unlocked, err := s.repo.IsRecipeUnlocked(ctx, user.ID, upgradeRecipeID)
	if err != nil {
		log.Error("Failed to check recipe unlock", "error", err, "recipeID", upgradeRecipeID)
		return nil, 0, fmt.Errorf("failed to check recipe unlock: %w", err)
	}
	if !unlocked {
		log.Warn("Recipe not unlocked", "username", username, "recipeID", upgradeRecipeID)
		return nil, 0, fmt.Errorf("disassemble recipe for %s is not unlocked", itemName)
	}

	// Begin transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return nil, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get user's inventory
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Find how many source items the user has
	// Find how many source items the user has
	sourceSlotIndex, userQuantity := utils.FindSlot(inventory, item.ID)

	// Calculate max possible disassembles
	maxPossible := userQuantity / recipe.QuantityConsumed
	if maxPossible == 0 {
		log.Warn("Insufficient items", "username", username, "item", itemName, "available", userQuantity, "needed", recipe.QuantityConsumed)
		return nil, 0, fmt.Errorf("insufficient items to disassemble %s (need %d, have %d)", itemName, recipe.QuantityConsumed, userQuantity)
	}

	// Actual quantity to disassemble
	actualQuantity := maxPossible
	if actualQuantity > quantity {
		actualQuantity = quantity
	}

	// Calculate total consumed
	totalConsumed := recipe.QuantityConsumed * actualQuantity

	// Remove the source items
	if inventory.Slots[sourceSlotIndex].Quantity == totalConsumed {
		// Remove the slot entirely
		inventory.Slots = append(inventory.Slots[:sourceSlotIndex], inventory.Slots[sourceSlotIndex+1:]...)
	} else {
		inventory.Slots[sourceSlotIndex].Quantity -= totalConsumed
	}

	// Add output items and build result map
	outputMap := make(map[string]int)
	for _, output := range recipe.Outputs {
		totalOutput := output.Quantity * actualQuantity

		// Get item name for the output
		outputItem, err := s.repo.GetItemByID(ctx, output.ItemID)
		if err != nil {
			log.Error("Failed to get output item", "error", err, "itemID", output.ItemID)
			return nil, 0, fmt.Errorf("failed to get output item: %w", err)
		}
		outputMap[outputItem.Name] = totalOutput

		// Add to inventory
		found := false
		for i, slot := range inventory.Slots {
			if slot.ItemID == output.ItemID {
				inventory.Slots[i].Quantity += totalOutput
				found = true
				break
			}
		}
		if !found {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{
				ItemID:   output.ItemID,
				Quantity: totalOutput,
			})
		}
	}

	// Update inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return nil, 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Items disassembled", "username", username, "item", itemName, "quantity", actualQuantity, "outputs", outputMap)
	return outputMap, actualQuantity, nil
}
