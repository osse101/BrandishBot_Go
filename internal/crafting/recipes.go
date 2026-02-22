package crafting

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

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
