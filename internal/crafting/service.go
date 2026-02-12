package crafting

import (
	"context"
	"fmt"
	"math"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// RecipeInfo represents recipe information with lock status
type RecipeInfo struct {
	ItemName string              `json:"item_name"`
	Locked   bool                `json:"locked,omitempty"`
	BaseCost []domain.RecipeCost `json:"base_cost,omitempty"`
}

// Result contains the result of an upgrade operation
type Result struct {
	ItemName      string `json:"item_name"`
	Quantity      int    `json:"quantity"`
	IsMasterwork  bool   `json:"is_masterwork"`
	BonusQuantity int    `json:"bonus_quantity"`
}

// DisassembleResult contains the result of a disassemble operation
type DisassembleResult struct {
	Outputs           map[string]int `json:"outputs"`
	QuantityProcessed int            `json:"quantity_processed"`
	IsPerfectSalvage  bool           `json:"is_perfect_salvage"`
	Multiplier        float64        `json:"multiplier"`
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	PublishWithRetry(ctx context.Context, event event.Event)
}

// Service defines the interface for crafting operations
type Service interface {
	UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*Result, error)
	GetRecipe(ctx context.Context, itemName, platform, platformID, username string) (*RecipeInfo, error)
	GetUnlockedRecipes(ctx context.Context, platform, platformID, username string) ([]repository.UnlockedRecipeInfo, error)
	GetAllRecipes(ctx context.Context) ([]repository.RecipeListItem, error)
	DisassembleItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*DisassembleResult, error)
	Shutdown(ctx context.Context) error
}

// ProgressionService defines the interface for progression operations
type ProgressionService interface {
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}

// Crafting balance constants are defined in constants.go

type service struct {
	repo           repository.Crafting
	eventPublisher EventPublisher
	progressionSvc ProgressionService
	namingResolver naming.Resolver // For resolving public names to internal names
	rnd            func() float64  // For rolling RNG (does not need to be cryptographically secure)
}

// NewService creates a new crafting service
func NewService(repo repository.Crafting, eventPublisher EventPublisher, namingResolver naming.Resolver, progressionSvc ProgressionService) Service {
	return &service{
		repo:           repo,
		eventPublisher: eventPublisher,
		progressionSvc: progressionSvc,
		namingResolver: namingResolver,
		rnd:            utils.RandomFloat,
	}
}

func (s *service) validateUser(ctx context.Context, platform, platformID string) (*domain.User, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (s *service) validateItem(ctx context.Context, itemName string) (*domain.Item, error) {
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return nil, fmt.Errorf("item not found: %s | %w", itemName, domain.ErrItemNotFound)
	}
	return item, nil
}

func (s *service) validateQuantity(quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive (got %d): %w", quantity, domain.ErrInvalidQuantity)
	}
	return nil
}

func (s *service) validatePlatformInput(platform, platformID string) error {
	if platform == "" || platformID == "" {
		return fmt.Errorf("platform and platformID cannot be empty: %w", domain.ErrInvalidInput)
	}
	validPlatforms := []string{domain.PlatformTwitch, domain.PlatformDiscord, domain.PlatformYoutube}
	for _, p := range validPlatforms {
		if platform == p {
			return nil
		}
	}
	return fmt.Errorf("invalid platform '%s': %w", platform, domain.ErrInvalidPlatform)
}

func (s *service) validateItemName(itemName string) error {
	if itemName == "" {
		return fmt.Errorf("item name cannot be empty: %w", domain.ErrInvalidInput)
	}
	return nil
}

// resolveItemName attempts to resolve a user-provided item name to its internal name.
// It first tries the naming resolver, then falls back to using the input as-is.
// This allows users to use either public names ("junkbox") or internal names ("lootbox_tier0").
func (s *service) resolveItemName(ctx context.Context, itemName string) (string, error) {
	// Try naming resolver first (handles public names)
	if s.namingResolver != nil {
		if internalName, ok := s.namingResolver.ResolvePublicName(itemName); ok {
			return internalName, nil
		}
	}

	// Fall back - assume it's already an internal name
	// Validate by checking if item exists
	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve item name '%s': %w", itemName, err)
	}
	if item == nil {
		return "", fmt.Errorf("item not found: %s (not found as public or internal name) | %w", itemName, domain.ErrItemNotFound)
	}

	return itemName, nil
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

// UpgradeItem upgrades as many items as possible based on available materials
func (s *service) UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*Result, error) {
	log := logger.FromContext(ctx)
	log.Info("UpgradeItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	// 1. Validate and resolve inputs
	user, item, recipe, resolvedName, err := s.validateUpgradeInput(ctx, platform, platformID, itemName, quantity)
	if err != nil {
		return nil, err
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

// GetRecipe returns recipe information for an item
// If username is provided, includes lock status; otherwise returns base recipe
func (s *service) GetRecipe(ctx context.Context, itemName, platform, platformID, username string) (*RecipeInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("GetRecipe called", "itemName", itemName, "platform", platform, "platformID", platformID, "username", username)

	// Validate inputs
	if err := s.validateItemName(itemName); err != nil {
		return nil, err
	}

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, err
	}

	// Validate and get item
	item, err := s.validateItem(ctx, resolvedName)
	if err != nil {
		return nil, err
	}

	// Get recipe by target item ID
	recipe, err := s.repo.GetRecipeByTargetItemID(ctx, item.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	if recipe == nil {
		return nil, fmt.Errorf("no recipe found for item: %s | %w", itemName, domain.ErrRecipeNotFound)
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
			return nil, fmt.Errorf("failed to check recipe unlock: %w", err)
		}

		recipeInfo.Locked = !unlocked
	}

	log.Info("Recipe retrieved", "itemName", itemName, "locked", recipeInfo.Locked)
	return recipeInfo, nil
}

// GetUnlockedRecipes returns all recipes that a user has unlocked
func (s *service) GetUnlockedRecipes(ctx context.Context, platform, platformID, username string) ([]repository.UnlockedRecipeInfo, error) {
	log := logger.FromContext(ctx)
	log.Info("GetUnlockedRecipes called", "platform", platform, "platformID", platformID, "username", username)

	// Validate inputs
	if err := s.validatePlatformInput(platform, platformID); err != nil {
		return nil, err
	}

	user, err := s.validateUser(ctx, platform, platformID)
	if err != nil {
		return nil, err
	}

	unlockedRecipes, err := s.repo.GetUnlockedRecipesForUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unlocked recipes: %w", err)
	}

	log.Info("Unlocked recipes retrieved", "username", username, "count", len(unlockedRecipes))
	return unlockedRecipes, nil
}

// GetAllRecipes returns all valid crafting recipes
func (s *service) GetAllRecipes(ctx context.Context) ([]repository.RecipeListItem, error) {
	recipes, err := s.repo.GetAllRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all recipes: %w", err)
	}

	return recipes, nil
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

// Shutdown gracefully shuts down the crafting service by waiting for all async operations to complete
func (s *service) Shutdown(ctx context.Context) error {
	// No more async operations to wait for
	return nil
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
	perfectSalvageCount := s.calculatePerfectSalvage(actualQuantity)

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

func (s *service) calculatePerfectSalvage(quantity int) int {
	// Get modified perfect salvage chance (base 0.10 = 10%)
	// Note: Using same modifier key as masterwork since they're both "crafting success"
	salvageChance := PerfectSalvageChance
	if s.progressionSvc != nil {
		// Use background context since we don't have ctx in this helper
		ctx := context.Background()
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
