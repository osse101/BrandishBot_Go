package crafting

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// UpgradeItem upgrades as many items as possible based on available materials
func (s *service) UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*Result, error) {
	log := logger.FromContext(ctx)
	log.Info("UpgradeItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	// 1. Validate and resolve inputs
	user, item, recipe, resolvedName, err := s.validateUpgradeInput(ctx, platform, platformID, itemName, quantity)
	if err != nil {
		return nil, err
	}

	// 1b. Check job level requirements (if any)
	if recipe.RequiredJobLevel > 0 {
		if s.jobService != nil {
			// Get user's Blacksmith level
			currentLevel, err := s.jobService.GetJobLevel(ctx, user.ID, "blacksmith")
			if err != nil {
				log.Error("Failed to check job level", "error", err, "userID", user.ID)
				// Fail safe: if we can't check level, don't allow crafting high-tier items
				return nil, fmt.Errorf("failed to verify job level requirements")
			}

			if currentLevel < recipe.RequiredJobLevel {
				return nil, fmt.Errorf("requires Blacksmith Level %d (you are Level %d)", recipe.RequiredJobLevel, currentLevel)
			}
		} else {
			// Should not happen in production if initialized correctly
			log.Warn("Job service not initialized in crafting service, skipping level check")
		}
	}

	// 2. Execute transaction
	result, actualQuantity, err := s.executeUpgradeTx(ctx, user.ID, item.ID, recipe, quantity, resolvedName)
	if err != nil {
		return nil, err
	}

	// 3. Publish event
	recipeKey := itemName
	if recipe != nil && recipe.RecipeKey != "" {
		recipeKey = recipe.RecipeKey
	}
	evt := NewItemUpgradedEvent(user.ID, itemName, actualQuantity, recipeKey, result.IsMasterwork, result.BonusQuantity)
	s.eventPublisher.PublishWithRetry(ctx, evt)

	log.Info("Items upgraded", "username", username, "item", itemName, "quantity", result.Quantity, "masterwork", result.IsMasterwork)
	return result, nil
}

func (s *service) validateUpgradeInput(ctx context.Context, platform, platformID, itemName string, quantity int) (*domain.User, *domain.Item, *domain.Recipe, string, error) {
	if err := s.validateQuantity(quantity); err != nil {
		return nil, nil, nil, "", err
	}
	if err := s.validatePlatformInput(platform, platformID); err != nil {
		return nil, nil, nil, "", err
	}
	if err := s.validateItemName(itemName); err != nil {
		return nil, nil, nil, "", err
	}

	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, nil, nil, "", err
	}

	user, err := s.validateUser(ctx, platform, platformID)
	if err != nil {
		return nil, nil, nil, "", err
	}

	item, err := s.validateItem(ctx, resolvedName)
	if err != nil {
		return nil, nil, nil, "", err
	}

	recipe, err := s.getAndValidateRecipe(ctx, item.ID, user.ID, resolvedName)
	if err != nil {
		return nil, nil, nil, "", err
	}

	return user, item, recipe, resolvedName, nil
}

func (s *service) executeUpgradeTx(ctx context.Context, userID string, itemID int, recipe *domain.Recipe, requestedQuantity int, resolvedName string) (*Result, int, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	actualQuantity := calculateMaxPossibleCrafts(inventory, recipe, requestedQuantity)
	if actualQuantity == 0 {
		return nil, 0, fmt.Errorf("insufficient materials | %w", domain.ErrInsufficientQuantity)
	}

	consumedMaterials, err := consumeRecipeMaterials(inventory, recipe, actualQuantity, s.rnd)
	if err != nil {
		return nil, 0, err
	}

	outputQuality := utils.CalculateAverageQuality(consumedMaterials)
	result := s.calculateUpgradeOutput(ctx, userID, resolvedName, actualQuantity)

	addItemToInventory(inventory, itemID, result.Quantity, outputQuality)

	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		return nil, 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, actualQuantity, nil
}

func (s *service) getAndValidateRecipe(ctx context.Context, itemID int, userID string, itemName string) (*domain.Recipe, error) {
	recipe, err := s.repo.GetRecipeByTargetItemID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	if recipe == nil {
		return nil, fmt.Errorf("no recipe found for item: %s | %w", itemName, domain.ErrRecipeNotFound)
	}

	// Check if user has unlocked this recipe
	unlocked, err := s.repo.IsRecipeUnlocked(ctx, userID, recipe.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check recipe unlock: %w", err)
	}
	if !unlocked {
		return nil, fmt.Errorf("recipe for %s is not unlocked | %w", itemName, domain.ErrRecipeLocked)
	}
	return recipe, nil
}

func (s *service) calculateUpgradeOutput(ctx context.Context, userID string, itemName string, actualQuantity int) *Result {
	log := logger.FromContext(ctx)

	outputQuantity := 0
	masterworkCount := 0

	// Get modified masterwork chance (base 0.10 = 10%)
	masterworkChance := MasterworkChance
	if s.progressionSvc != nil {
		if modifiedChance, err := s.progressionSvc.GetModifiedValue(ctx, "crafting_success_rate", MasterworkChance); err == nil {
			masterworkChance = modifiedChance
		} else {
			log.Warn("Failed to apply crafting_success_rate modifier, using base chance", "error", err)
		}
	}

	for i := 0; i < actualQuantity; i++ {
		if s.rnd() < masterworkChance {
			masterworkCount++
			outputQuantity += MasterworkMultiplier
		} else {
			outputQuantity += 1
		}
	}

	masterworkTriggered := masterworkCount > 0
	if masterworkTriggered {
		log.Info("Masterwork craft triggered!", "user_id", userID, "item", itemName, "count", masterworkCount, "bonus", outputQuantity-actualQuantity)
		// Stats event is now handled by event subscriber
	}

	return &Result{
		ItemName:      itemName,
		Quantity:      outputQuantity,
		IsMasterwork:  masterworkTriggered,
		BonusQuantity: outputQuantity - actualQuantity,
	}
}

// calculateMaxPossibleCrafts calculates the maximum number of crafts possible given available materials
func calculateMaxPossibleCrafts(inventory *domain.Inventory, recipe *domain.Recipe, requestedQuantity int) int {
	maxPossible := requestedQuantity
	for _, cost := range recipe.BaseCost {
		userQuantity := utils.GetTotalQuantity(inventory, cost.ItemID)
		if cost.Quantity > 0 {
			affordableWithThis := userQuantity / cost.Quantity
			if affordableWithThis < maxPossible {
				maxPossible = affordableWithThis
			}
		}
	}
	return maxPossible
}

// consumeRecipeMaterials removes the required materials from inventory for crafting.
// Returns the consumed materials with their quality levels for calculating output quality.
func consumeRecipeMaterials(inventory *domain.Inventory, recipe *domain.Recipe, actualQuantity int, rnd func() float64) ([]domain.InventorySlot, error) {
	allConsumed := make([]domain.InventorySlot, 0)

	for _, cost := range recipe.BaseCost {
		totalNeeded := cost.Quantity * actualQuantity
		consumed, err := utils.ConsumeItemsWithTracking(inventory, cost.ItemID, totalNeeded, rnd)
		if err != nil {
			return nil, fmt.Errorf("insufficient material (itemID: %d) | %w", cost.ItemID, domain.ErrInsufficientQuantity)
		}
		allConsumed = append(allConsumed, consumed...)
	}

	return allConsumed, nil
}

// addItemToInventory adds items to the inventory with specified quality level.
// Only stacks with slots that have matching ItemID AND QualityLevel.
func addItemToInventory(inventory *domain.Inventory, itemID, quantity int, qualityLevel domain.QualityLevel) {
	// Find slot with matching ItemID and QualityLevel
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID && slot.QualityLevel == qualityLevel {
			inventory.Slots[i].Quantity += quantity
			return
		}
	}
	// Item not found with matching quality, add new slot
	inventory.Slots = append(inventory.Slots, domain.InventorySlot{
		ItemID:       itemID,
		Quantity:     quantity,
		QualityLevel: qualityLevel,
	})
}
