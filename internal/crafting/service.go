package crafting

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/stats"
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

// Service defines the interface for crafting operations
type Service interface {
	UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*Result, error)
	GetRecipe(ctx context.Context, itemName, platform, platformID, username string) (*RecipeInfo, error)
	GetUnlockedRecipes(ctx context.Context, platform, platformID, username string) ([]repository.UnlockedRecipeInfo, error)
	GetAllRecipes(ctx context.Context) ([]repository.RecipeListItem, error)
	DisassembleItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*DisassembleResult, error)
	Shutdown(ctx context.Context) error
}

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
}

// ProgressionService defines the interface for progression operations
type ProgressionService interface {
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}

// Crafting balance constants are defined in constants.go

type service struct {
	repo           repository.Crafting
	jobService     JobService
	progressionSvc ProgressionService
	statsSvc       stats.Service
	namingResolver naming.Resolver // For resolving public names to internal names
	rnd            func() float64  // For rolling RNG (does not need to be cryptographically secure)
	wg             sync.WaitGroup  // Tracks async goroutines for graceful shutdown
}

// NewService creates a new crafting service
func NewService(repo repository.Crafting, jobService JobService, statsSvc stats.Service, namingResolver naming.Resolver, progressionSvc ProgressionService) Service {
	return &service{
		repo:           repo,
		jobService:     jobService,
		progressionSvc: progressionSvc,
		statsSvc:       statsSvc,
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
func consumeRecipeMaterials(inventory *domain.Inventory, recipe *domain.Recipe, actualQuantity int, rnd func() float64) error {
	for _, cost := range recipe.BaseCost {
		totalNeeded := cost.Quantity * actualQuantity
		// Use random selection in case multiple slots with different shine levels exist
		i, slotQuantity := utils.FindRandomSlot(inventory, cost.ItemID, rnd)
		if i == -1 || slotQuantity < totalNeeded {
			return fmt.Errorf("insufficient material (itemID: %d) | %w", cost.ItemID, domain.ErrInsufficientQuantity)
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
func (s *service) UpgradeItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*Result, error) {
	log := logger.FromContext(ctx)
	log.Info("UpgradeItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

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

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, err
	}

	// Validate user and item
	user, err := s.validateUser(ctx, platform, platformID)
	if err != nil {
		return nil, err
	}

	item, err := s.validateItem(ctx, resolvedName)
	if err != nil {
		return nil, err
	}

	// Get and validate recipe
	recipe, err := s.getAndValidateRecipe(ctx, item.ID, user.ID, resolvedName)
	if err != nil {
		return nil, err
	}

	// Begin transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get inventory and calculate actual quantity
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	maxPossible := calculateMaxPossibleCrafts(inventory, recipe, quantity)
	if maxPossible == 0 {
		return nil, fmt.Errorf("insufficient materials to craft %s | %w", itemName, domain.ErrInsufficientQuantity)
	}

	actualQuantity := maxPossible
	if actualQuantity > quantity {
		actualQuantity = quantity
	}

	// Consume materials
	if err := consumeRecipeMaterials(inventory, recipe, actualQuantity, s.rnd); err != nil {
		return nil, err
	}

	// Calculate output
	result := s.calculateUpgradeOutput(ctx, user.ID, itemName, actualQuantity)

	addItemToInventory(inventory, item.ID, result.Quantity)

	// Update inventory and commit
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Award Blacksmith XP (don't fail upgrade if XP award fails)
	// Run async with detached context to prevent cancellation affecting XP award
	s.wg.Add(1)
	go s.awardBlacksmithXP(context.Background(), user.ID, actualQuantity, "upgrade", itemName)

	log.Info("Items upgraded", "username", username, "item", itemName, "quantity", result.Quantity, "masterwork", result.IsMasterwork)

	return result, nil
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

		if s.statsSvc != nil {
			_ = s.statsSvc.RecordUserEvent(ctx, userID, domain.EventCraftingCriticalSuccess, map[string]interface{}{
				"item_name":         itemName,
				"original_quantity": actualQuantity,
				"masterwork_count":  masterworkCount,
				"bonus_quantity":    outputQuantity - actualQuantity,
			})
		}
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

// processDisassembleOutputs adds disassemble outputs to inventory and builds result map
func (s *service) processDisassembleOutputs(ctx context.Context, inventory *domain.Inventory, outputs []domain.RecipeOutput, actualQuantity int, perfectSalvageCount int) (map[string]int, error) {
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

		// Prepare for batch add
		itemsToAdd = append(itemsToAdd, domain.InventorySlot{
			ItemID:   output.ItemID,
			Quantity: totalOutput,
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
		s.recordPerfectSalvageEvent(ctx, user.ID, itemName, actualQuantity, perfectSalvageCount)
	}

	// Award Blacksmith XP (don't fail disassemble if XP award fails)
	s.wg.Add(1)
	go s.awardBlacksmithXP(context.Background(), user.ID, actualQuantity, "disassemble", itemName)

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

func (s *service) calculateDisassembleQuantity(inventory *domain.Inventory, itemID int, quantityConsumed int, quantity int, itemName string) (int, int, error) {
	// Use random selection in case multiple slots with different shine levels exist
	sourceSlotIndex, userQuantity := utils.FindRandomSlot(inventory, itemID, s.rnd)
	maxPossible := userQuantity / quantityConsumed
	if maxPossible == 0 {
		return 0, -1, fmt.Errorf("insufficient items to disassemble %s (need %d, have %d) | %w", itemName, quantityConsumed, userQuantity, domain.ErrInsufficientQuantity)
	}

	actualQuantity := maxPossible
	if actualQuantity > quantity {
		actualQuantity = quantity
	}
	return actualQuantity, sourceSlotIndex, nil
}

// awardBlacksmithXP awards Blacksmith job XP for crafting operations
// NOTE: Caller must call s.wg.Add(1) before launching this in a goroutine
func (s *service) awardBlacksmithXP(ctx context.Context, userID string, quantity int, source, itemName string) {
	defer s.wg.Done()

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

// Shutdown gracefully shuts down the crafting service by waiting for all async operations to complete
func (s *service) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Shutting down crafting service, waiting for async operations...")

	// Wait for all async XP awards to complete
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("Crafting service shutdown complete")
		return nil
	case <-ctx.Done():
		log.Warn("Crafting service shutdown forced by context cancellation")
		return ctx.Err()
	}
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

	actualQuantity, sourceSlotIndex, err := s.calculateDisassembleQuantity(inventory, itemID, recipe.QuantityConsumed, requestedQuantity, itemName)
	if err != nil {
		return 0, 0, nil, err
	}

	// Remove source items
	totalConsumed := recipe.QuantityConsumed * actualQuantity
	if inventory.Slots[sourceSlotIndex].Quantity == totalConsumed {
		inventory.Slots = append(inventory.Slots[:sourceSlotIndex], inventory.Slots[sourceSlotIndex+1:]...)
	} else {
		inventory.Slots[sourceSlotIndex].Quantity -= totalConsumed
	}

	// Calculate perfect salvage
	perfectSalvageCount := s.calculatePerfectSalvage(actualQuantity)

	// Process outputs
	outputMap, err := s.processDisassembleOutputs(ctx, inventory, recipe.Outputs, actualQuantity, perfectSalvageCount)
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

func (s *service) recordPerfectSalvageEvent(ctx context.Context, userID, itemName string, actualQuantity, perfectCount int) {
	logger.FromContext(ctx).Info("Perfect Salvage triggered!", "user_id", userID, "item", itemName, "quantity", actualQuantity, "perfect_count", perfectCount)

	if s.statsSvc != nil {
		_ = s.statsSvc.RecordUserEvent(ctx, userID, domain.EventCraftingPerfectSalvage, map[string]interface{}{
			"item_name":     itemName,
			"quantity":      actualQuantity,
			"perfect_count": perfectCount,
			"multiplier":    PerfectSalvageMultiplier,
		})
	}
}
