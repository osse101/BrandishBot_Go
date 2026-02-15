package crafting

import (
	"context"
	"fmt"
	"math"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// DisassembleItem disassembles items into their component materials
// Returns a map of item names to quantities and the number of items disassembled
func (s *service) DisassembleItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*DisassembleResult, error) {
	log := logger.FromContext(ctx)
	log.Info("DisassembleItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	// Validate inputs
	if err := s.validateQuantity(quantity); err != nil {
		return nil, err
	}
	if err := s.validatePlatformInput(platform, platformID); err != nil {
		return nil, err
	}
	if err := s.validateItemName(itemName); err != nil {
		return nil, err
	}

	user, item, recipe, err := s.validateDisassembleInput(ctx, platform, platformID, itemName)
	if err != nil {
		return nil, err
	}

	actualQuantity, perfectSalvageCount, outputMap, err := s.executeDisassembleTx(ctx, user.ID, item.ID, recipe, quantity, itemName)
	if err != nil {
		return nil, err
	}

	perfectSalvageTriggered := perfectSalvageCount > 0
	if perfectSalvageTriggered {
		log.Info("Perfect Salvage triggered!", "user_id", user.ID, "item", itemName, "quantity", actualQuantity, "perfect_count", perfectSalvageCount)
		// Stats event is now handled by event subscriber
	}

	// Publish event
	recipeKey := itemName // Use item name as fallback
	if recipe != nil && recipe.RecipeKey != "" {
		recipeKey = recipe.RecipeKey
	}
	evt := NewItemDisassembledEvent(user.ID, itemName, actualQuantity, recipeKey, perfectSalvageTriggered, perfectSalvageCount, PerfectSalvageMultiplier, outputMap)
	s.eventPublisher.PublishWithRetry(ctx, evt)

	log.Info("Items disassembled", "username", username, "item", itemName, "quantity", actualQuantity, "outputs", outputMap, "perfect_salvage", perfectSalvageTriggered)
	return &DisassembleResult{
		Outputs:           outputMap,
		QuantityProcessed: actualQuantity,
		IsPerfectSalvage:  perfectSalvageTriggered,
		Multiplier:        PerfectSalvageMultiplier,
	}, nil
}

func (s *service) validateDisassembleInput(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, *domain.DisassembleRecipe, error) {
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, nil, nil, err
	}

	user, err := s.validateUser(ctx, platform, platformID)
	if err != nil {
		return nil, nil, nil, err
	}

	item, err := s.validateItem(ctx, resolvedName)
	if err != nil {
		return nil, nil, nil, err
	}

	recipe, err := s.getAndValidateDisassembleRecipe(ctx, item.ID, user.ID, resolvedName)
	if err != nil {
		return nil, nil, nil, err
	}

	return user, item, recipe, nil
}

func (s *service) getAndValidateDisassembleRecipe(ctx context.Context, itemID int, userID string, itemName string) (*domain.DisassembleRecipe, error) {
	// Get disassemble recipe
	recipe, err := s.repo.GetDisassembleRecipeBySourceItemID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get disassemble recipe: %w", err)
	}
	if recipe == nil {
		return nil, fmt.Errorf("no disassemble recipe found for item: %s | %w", itemName, domain.ErrRecipeNotFound)
	}

	// Get associated upgrade recipe ID to check if unlocked
	upgradeRecipeID, err := s.repo.GetAssociatedUpgradeRecipeID(ctx, recipe.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get associated upgrade recipe: %w", err)
	}

	// Check if user has unlocked the associated upgrade recipe
	unlocked, err := s.repo.IsRecipeUnlocked(ctx, userID, upgradeRecipeID)
	if err != nil {
		return nil, fmt.Errorf("failed to check recipe unlock: %w", err)
	}
	if !unlocked {
		return nil, fmt.Errorf("disassemble recipe for %s is not unlocked | %w", itemName, domain.ErrRecipeLocked)
	}
	return recipe, nil
}

func (s *service) executeDisassembleTx(ctx context.Context, userID string, itemID int, recipe *domain.DisassembleRecipe, requestedQuantity int, itemName string) (int, int, map[string]int, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	actualQuantity, err := s.calculateDisassembleQuantity(inventory, itemID, recipe.QuantityConsumed, requestedQuantity, itemName)
	if err != nil {
		return 0, 0, nil, err
	}

	// Remove source items and track what was consumed for quality averaging
	totalConsumed := recipe.QuantityConsumed * actualQuantity
	consumedItems, err := utils.ConsumeItemsWithTracking(inventory, itemID, totalConsumed, s.rnd)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to consume disassemble items: %w", err)
	}

	// Calculate average quality from consumed source items
	outputQuality := utils.CalculateAverageQuality(consumedItems)

	// Calculate perfect salvage
	perfectSalvageCount := s.calculatePerfectSalvage(ctx, actualQuantity)

	// Process outputs with averaged quality from source materials
	outputMap, err := s.processDisassembleOutputs(ctx, inventory, recipe.Outputs, actualQuantity, perfectSalvageCount, outputQuality)
	if err != nil {
		return 0, 0, nil, err
	}

	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		return 0, 0, nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return actualQuantity, perfectSalvageCount, outputMap, nil
}

func (s *service) calculateDisassembleQuantity(inventory *domain.Inventory, itemID int, quantityConsumed int, quantity int, itemName string) (int, error) {
	// Use total quantity across all slots
	userQuantity := utils.GetTotalQuantity(inventory, itemID)
	maxPossible := userQuantity / quantityConsumed
	if maxPossible == 0 {
		return 0, fmt.Errorf("insufficient items to disassemble %s (need %d, have %d) | %w", itemName, quantityConsumed, userQuantity, domain.ErrInsufficientQuantity)
	}

	actualQuantity := maxPossible
	if actualQuantity > quantity {
		actualQuantity = quantity
	}
	return actualQuantity, nil
}

// processDisassembleOutputs adds disassemble outputs to inventory and builds result map.
// Outputs inherit the averaged quality level from the consumed source items.
func (s *service) processDisassembleOutputs(ctx context.Context, inventory *domain.Inventory, outputs []domain.RecipeOutput, actualQuantity int, perfectSalvageCount int, outputQuality domain.QualityLevel) (map[string]int, error) {
	outputMap := make(map[string]int)

	// Collect IDs
	itemIDs := make([]int, 0, len(outputs))
	for _, output := range outputs {
		itemIDs = append(itemIDs, output.ItemID)
	}

	// Batch fetch items
	items, err := s.repo.GetItemsByIDs(ctx, itemIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get output items: %w", err)
	}

	// Map items by ID for easy lookup
	itemsByID := make(map[int]*domain.Item, len(items))
	for i := range items {
		itemsByID[items[i].ID] = &items[i]
	}

	// Prepare items to add to inventory
	itemsToAdd := make([]domain.InventorySlot, 0, len(outputs))

	for _, output := range outputs {
		// Calculate output for regular items
		regularQuantity := (actualQuantity - perfectSalvageCount) * output.Quantity

		// Calculate output for perfect salvage items (apply multiplier)
		// Multiplier is applied per item, rounded up
		perfectQuantity := 0
		if perfectSalvageCount > 0 {
			basePerItem := output.Quantity
			perfectPerItem := int(math.Ceil(float64(basePerItem) * PerfectSalvageMultiplier))
			perfectQuantity = perfectSalvageCount * perfectPerItem
		}

		totalOutput := regularQuantity + perfectQuantity

		// Get item name for the output
		outputItem, ok := itemsByID[output.ItemID]
		if !ok {
			return nil, fmt.Errorf("output item not found: %d | %w", output.ItemID, domain.ErrItemNotFound)
		}
		outputMap[outputItem.InternalName] = totalOutput

		// Prepare for batch add - outputs inherit averaged quality from source items
		itemsToAdd = append(itemsToAdd, domain.InventorySlot{
			ItemID:       output.ItemID,
			Quantity:     totalOutput,
			QualityLevel: outputQuality,
		})
	}

	// Add all outputs to inventory using optimized helper
	utils.AddItemsToInventory(inventory, itemsToAdd, nil)

	return outputMap, nil
}

func (s *service) calculatePerfectSalvage(ctx context.Context, quantity int) int {
	// Get modified perfect salvage chance (base 0.10 = 10%)
	// Note: Using same modifier key as masterwork since they're both "crafting success"
	salvageChance := PerfectSalvageChance
	if s.progressionSvc != nil {
		if modifiedChance, err := s.progressionSvc.GetModifiedValue(ctx, "crafting_success_rate", PerfectSalvageChance); err == nil {
			salvageChance = modifiedChance
		}
		// Silently fall back to base chance on error (no logging in helper)
	}

	count := 0
	for i := 0; i < quantity; i++ {
		if s.rnd() < salvageChance {
			count++
		}
	}
	return count
}
