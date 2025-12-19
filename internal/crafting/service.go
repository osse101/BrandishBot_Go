package crafting

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Repository defines the interface for data access required by the crafting service
type Repository interface {
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
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
	UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (string, int, error)
	GetRecipe(ctx context.Context, itemName, platform, platformID, username string) (*RecipeInfo, error)
	GetUnlockedRecipes(ctx context.Context, platform, platformID, username string) ([]UnlockedRecipeInfo, error)
	DisassembleItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (map[string]int, int, error)
}

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
}

type service struct {
	repo       Repository
	jobService JobService
}

// NewService creates a new crafting service
func NewService(repo Repository, jobService JobService) Service {
	return &service{
		repo:       repo,
		jobService: jobService,
	}
}

func (s *service) validateUser(ctx context.Context, platform, platformID string) (*domain.User, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
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

// calculateMaxPossibleCrafts calculates the maximum number of crafts possible given available materials
func calculateMaxPossibleCrafts(inventory *domain.Inventory, recipe *domain.Recipe, requestedQuantity int) int {
	maxPossible := requestedQuantity
	for _, cost := range recipe.BaseCost {
		_, userQuantity := utils.FindSlot(inventory, cost.ItemID)
		if cost.Quantity > 0 {
			affordableWithThis := userQuantity / cost.Quantity
			if affordableWithThis < maxPossible {
				maxPossible = affordableWithThis
			}
		}
	}
	return maxPossible
}

// consumeRecipeMaterials removes the required materials from inventory for crafting
func consumeRecipeMaterials(inventory *domain.Inventory, recipe *domain.Recipe, actualQuantity int) error {
	for _, cost := range recipe.BaseCost {
		totalNeeded := cost.Quantity * actualQuantity
		i, slotQuantity := utils.FindSlot(inventory, cost.ItemID)
		if i == -1 || slotQuantity < totalNeeded {
			return fmt.Errorf("insufficient material (itemID: %d)", cost.ItemID)
		}

		// Remove the materials
		if slotQuantity == totalNeeded {
			inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
		} else {
			inventory.Slots[i].Quantity -= totalNeeded
		}
	}
	return nil
}

// addItemToInventory adds items to the inventory, creating a new slot if necessary
func addItemToInventory(inventory *domain.Inventory, itemID, quantity int) {
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			inventory.Slots[i].Quantity += quantity
			return
		}
	}
	// Item not found, add new slot
	inventory.Slots = append(inventory.Slots, domain.InventorySlot{
		ItemID:   itemID,
		Quantity: quantity,
	})
}

// UpgradeItem upgrades as many items as possible based on available materials
func (s *service) UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (string, int, error) {
	log := logger.FromContext(ctx)
	log.Info("UpgradeItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	// Validate user and item
	user, err := s.validateUser(ctx, platform, platformID)
	if err != nil {
		return "", 0, err
	}

	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return "", 0, err
	}

	// Get and validate recipe
	recipe, err := s.repo.GetRecipeByTargetItemID(ctx, item.ID)
	if err != nil {
		log.Error("Failed to get recipe", "error", err)
		return "", 0, fmt.Errorf("failed to get recipe: %w", err)
	}
	if recipe == nil {
		return "", 0, fmt.Errorf("no recipe found for item: %s", itemName)
	}

	// Check if user has unlocked this recipe
	unlocked, err := s.repo.IsRecipeUnlocked(ctx, user.ID, recipe.ID)
	if err != nil {
		log.Error("Failed to check recipe unlock", "error", err)
		return "", 0, fmt.Errorf("failed to check recipe unlock: %w", err)
	}
	if !unlocked {
		return "", 0, fmt.Errorf("recipe for %s is not unlocked", itemName)
	}

	// Begin transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return "", 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get inventory
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Calculate maximum possible upgrades
	maxPossible := calculateMaxPossibleCrafts(inventory, recipe, quantity)
	if maxPossible == 0 {
		return "", 0, fmt.Errorf("insufficient materials to craft %s", itemName)
	}

	actualQuantity := maxPossible
	if actualQuantity > quantity {
		actualQuantity = quantity
	}

	// Consume materials and add crafted items
	if err := consumeRecipeMaterials(inventory, recipe, actualQuantity); err != nil {
		return "", 0, err
	}
	addItemToInventory(inventory, item.ID, actualQuantity)

	// Update inventory and commit
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return "", 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Award Blacksmith XP (don't fail upgrade if XP award fails)
	go s.awardBlacksmithXP(context.Background(), user.ID, actualQuantity, "upgrade", itemName)

	log.Info("Items upgraded", "username", username, "item", itemName, "quantity", actualQuantity)
	return itemName, actualQuantity, nil
}

// GetRecipe returns recipe information for an item
// If username is provided, includes lock status; otherwise returns base recipe
func (s *service) GetRecipe(ctx context.Context, itemName, platform, platformID, username string) (*RecipeInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("GetRecipe called", "itemName", itemName, "platform", platform, "platformID", platformID, "username", username)

	// Validate and get item
	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return nil, err
	}

	// Get recipe by target item ID
	recipe, err := s.repo.GetRecipeByTargetItemID(ctx, item.ID)
	if err != nil {
		log.Error("Failed to get recipe", "error", err)
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	if recipe == nil {
		return nil, fmt.Errorf("no recipe found for item: %s", itemName)
	}

	recipeInfo := &RecipeInfo{
		ItemName: itemName,
		BaseCost: recipe.BaseCost,
	}

	// If user info provided, check lock status
	if platform != "" && platformID != "" {
		user, err := s.validateUser(ctx, platform, platformID)
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
func (s *service) GetUnlockedRecipes(ctx context.Context, platform, platformID, username string) ([]UnlockedRecipeInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("GetUnlockedRecipes called", "platform", platform, "platformID", platformID, "username", username)

	user, err := s.validateUser(ctx, platform, platformID)
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

// processDisassembleOutputs adds disassemble outputs to inventory and builds result map
func (s *service) processDisassembleOutputs(ctx context.Context, inventory *domain.Inventory, outputs []domain.RecipeOutput, actualQuantity int) (map[string]int, error) {
	outputMap := make(map[string]int)
	
	for _, output := range outputs {
		totalOutput := output.Quantity * actualQuantity

		// Get item name for the output
		outputItem, err := s.repo.GetItemByID(ctx, output.ItemID)
		if err != nil {
			return nil, fmt.Errorf("failed to get output item: %w", err)
		}
		outputMap[outputItem.InternalName] = totalOutput

		// Add to inventory
		addItemToInventory(inventory, output.ItemID, totalOutput)
	}
	
	return outputMap, nil
}

// DisassembleItem disassembles items into their component materials
// Returns a map of item names to quantities and the number of items disassembled
func (s *service) DisassembleItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (map[string]int, int, error) {
	log := logger.FromContext(ctx)
	log.Info("DisassembleItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	// Validate user and item
	user, err := s.validateUser(ctx, platform, platformID)
	if err != nil {
		return nil, 0, err
	}

	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return nil, 0, err
	}

	// Get disassemble recipe
	recipe, err := s.repo.GetDisassembleRecipeBySourceItemID(ctx, item.ID)
	if err != nil {
		log.Error("Failed to get disassemble recipe", "error", err)
		return nil, 0, fmt.Errorf("failed to get disassemble recipe: %w", err)
	}
	if recipe == nil {
		return nil, 0, fmt.Errorf("no disassemble recipe found for item: %s", itemName)
	}

	// Get associated upgrade recipe ID to check if unlocked
	upgradeRecipeID, err := s.repo.GetAssociatedUpgradeRecipeID(ctx, recipe.ID)
	if err != nil {
		log.Error("Failed to get associated upgrade recipe", "error", err)
		return nil, 0, fmt.Errorf("failed to get associated upgrade recipe: %w", err)
	}

	// Check if user has unlocked the associated upgrade recipe
	unlocked, err := s.repo.IsRecipeUnlocked(ctx, user.ID, upgradeRecipeID)
	if err != nil {
		log.Error("Failed to check recipe unlock", "error", err)
		return nil, 0, fmt.Errorf("failed to check recipe unlock: %w", err)
	}
	if !unlocked {
		return nil, 0, fmt.Errorf("disassemble recipe for %s is not unlocked", itemName)
	}

	// Begin transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return nil, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get inventory
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Find source items and calculate max possible
	sourceSlotIndex, userQuantity := utils.FindSlot(inventory, item.ID)
	maxPossible := userQuantity / recipe.QuantityConsumed
	if maxPossible == 0 {
		return nil, 0, fmt.Errorf("insufficient items to disassemble %s (need %d, have %d)", itemName, recipe.QuantityConsumed, userQuantity)
	}

	actualQuantity := maxPossible
	if actualQuantity > quantity {
		actualQuantity = quantity
	}

	// Remove source items
	totalConsumed := recipe.QuantityConsumed * actualQuantity
	if inventory.Slots[sourceSlotIndex].Quantity == totalConsumed {
		inventory.Slots = append(inventory.Slots[:sourceSlotIndex], inventory.Slots[sourceSlotIndex+1:]...)
	} else {
		inventory.Slots[sourceSlotIndex].Quantity -= totalConsumed
	}

	// Process outputs
	outputMap, err := s.processDisassembleOutputs(ctx, inventory, recipe.Outputs, actualQuantity)
	if err != nil {
		return nil, 0, err
	}

	// Update inventory and commit
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return nil, 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Award Blacksmith XP (don't fail disassemble if XP award fails)
	go s.awardBlacksmithXP(context.Background(), user.ID, actualQuantity, "disassemble", itemName)

	log.Info("Items disassembled", "username", username, "item", itemName, "quantity", actualQuantity, "outputs", outputMap)
	return outputMap, actualQuantity, nil
}

// awardBlacksmithXP awards Blacksmith job XP for crafting operations
func (s *service) awardBlacksmithXP(ctx context.Context, userID string, quantity int, source, itemName string) {
	if s.jobService == nil {
		return // Job system not enabled
	}

	// Use exported constant for XP per item
	totalXP := job.BlacksmithXPPerItem * quantity

	metadata := map[string]interface{}{
		"source":    source,
		"item_name": itemName,
		"quantity":  quantity,
	}

	result, err := s.jobService.AwardXP(ctx, userID, job.JobKeyBlacksmith, totalXP, source, metadata)
	if err != nil {
		// Log but don't fail the operation
		logger.FromContext(ctx).Warn("Failed to award Blacksmith XP", "error", err, "user_id", userID)
	} else if result != nil && result.LeveledUp {
		logger.FromContext(ctx).Info("Blacksmith leveled up!", "user_id", userID, "new_level", result.NewLevel)
	}
}
