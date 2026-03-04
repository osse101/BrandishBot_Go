package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func getItemCategory(item *domain.Item) string {
	if item != nil && len(item.Types) > 0 {
		return item.Types[0]
	}
	return "Item"
}

func (s *service) getBuyEntities(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, fmt.Errorf(ErrMsgGetUserFailed, err)
	}
	if user == nil {
		return nil, nil, domain.ErrUserNotFound
	}

	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve item %q: %w", itemName, err)
	}

	item, err := s.repo.GetItemByName(ctx, resolvedName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get item %q: %w", resolvedName, err)
	}
	if item == nil {
		return nil, nil, fmt.Errorf("item not found: %q: %w", resolvedName, domain.ErrItemNotFound)
	}

	return user, item, nil
}

func (s *service) getSellEntities(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, *domain.Item, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetUserFailed, err)
	}
	if user == nil {
		return nil, nil, nil, domain.ErrUserNotFound
	}

	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, nil, nil, err
	}

	item, err := s.repo.GetItemByName(ctx, resolvedName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetItemFailed, err)
	}
	if item == nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgItemNotFoundFmt, resolvedName, domain.ErrItemNotFound)
	}

	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetMoneyItemFailed, err)
	}
	if moneyItem == nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgItemNotFoundFmt, domain.ItemMoney, domain.ErrItemNotFound)
	}

	return user, item, moneyItem, nil
}

func (s *service) resolveItemName(ctx context.Context, itemName string) (string, error) {
	if s.namingResolver != nil {
		if internalName, ok := s.namingResolver.ResolvePublicName(itemName); ok {
			return internalName, nil
		}
	}

	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf(ErrMsgResolveItemFailedFmt, itemName, err)
	}
	if item == nil {
		return "", fmt.Errorf(ErrMsgItemNotFoundPublicFmt, itemName, domain.ErrItemNotFound)
	}

	return itemName, nil
}
