package user

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// useItemInternal handles item usage logic within a transaction
func (s *service) useItemInternal(ctx context.Context, user *domain.User, platform, itemName string, quantity int, targetName string) (string, error) {
	log := logger.FromContext(ctx)

	itemToUse, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return "", domain.ErrFailedToGetItem
	}
	if itemToUse == nil {
		log.Warn("Item not found", "itemName", itemName)
		return "", domain.ErrItemNotFound
	}

	var message string
	err = s.withTx(ctx, func(tx repository.UserTx) error {
		inventory, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "userID", user.ID)
			return domain.ErrFailedToGetInventory
		}

		// Find item in inventory using random selection (in case multiple slots exist with different quality levels)
		itemSlotIndex, slotQty := utils.FindRandomSlot(inventory, itemToUse.ID, s.rnd)
		if itemSlotIndex == -1 {
			return domain.ErrNotInInventory
		}
		if slotQty < quantity {
			return domain.ErrInsufficientQuantity
		}

		// Execute item handler
		handler := s.handlerRegistry.GetHandler(itemName)
		if handler == nil {
			log.Warn("No handler for item", "itemName", itemName)
			return domain.ErrItemNotHandled
		}

		handlerArgs := ItemHandlerArgs{
			Username: user.Username,
			Platform: platform,
		}
		if targetName != "" {
			handlerArgs.TargetUsername = targetName
			handlerArgs.JobName = targetName
		}
		message, err = handler.Handle(ctx, s, user, inventory, itemToUse, quantity, handlerArgs)
		if err != nil {
			log.Error("Handler error", "error", err, "itemName", itemName)
			return err
		}

		if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
			log.Error("Failed to update inventory after use", "error", err, "userID", user.ID)
			return domain.ErrFailedToUpdateInventory
		}

		return nil
	})

	return message, err
}

func (s *service) UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetName string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("UseItem called",
		"platform", platform, "platformID", platformID, "username", username,
		"itemName", itemName, "quantity", quantity, "targetName", targetName)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user or register", "error", err)
		return "", domain.ErrFailedToGetUser
	}

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		log.Error("Failed to resolve item name", "error", err)
		return "", domain.ErrInvalidInput
	}

	return s.useItemInternal(ctx, user, platform, resolvedName, quantity, targetName)
}

// resolveItemName attempts to resolve a user-provided item name to its internal name.
// It first tries the naming resolver, then falls back to using the input as-is.
// This allows users to use either public names ("junkbox") or internal names ("lootbox_tier0").
func (s *service) resolveItemName(ctx context.Context, itemName string) (string, error) {
	log := logger.FromContext(ctx)
	// Try naming resolver first (handles public names)
	if s.namingResolver != nil {
		if internalName, ok := s.namingResolver.ResolvePublicName(itemName); ok {
			return internalName, nil
		}
	}

	// Fall back - assume it's already an internal name
	item, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Warn("Failed to resolve item name", "error", err)
		return "", domain.ErrItemNotFound
	}
	if item == nil {
		log.Warn("Item not found", "itemName", itemName)
		return "", domain.ErrItemNotFound
	}

	return itemName, nil
}

// validateItem validates an item exists by name
// Helper methods
func (s *service) validateItem(ctx context.Context, itemName string) (*domain.Item, error) {
	log := logger.FromContext(ctx)
	item, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item by name", "error", err)
		return nil, domain.ErrItemNotFound
	}
	if item == nil {
		log.Error("Item not found", "itemName", itemName)
		return nil, domain.ErrItemNotFound
	}
	return item, nil
}
