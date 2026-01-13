package user

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// getItemByNameCached retrieves an item from cache or DB
// Supports both internal names (lootbox_tier0) and public names (junkbox)
func (s *service) getItemByNameCached(ctx context.Context, name string) (*domain.Item, error) {
	// Try to resolve as public name first (e.g., "junkbox" -> "lootbox_tier0")
	if internalName, ok := s.namingResolver.ResolvePublicName(name); ok {
		name = internalName
	}

	s.itemCacheMu.RLock()
	if item, ok := s.itemCacheByName[name]; ok {
		s.itemCacheMu.RUnlock()
		return &item, nil
	}
	s.itemCacheMu.RUnlock()

	item, err := s.repo.GetItemByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if item != nil {
		s.itemCacheMu.Lock()
		s.itemCacheByName[name] = *item
		s.itemCache[item.ID] = *item // Update ID cache too
		s.itemCacheMu.Unlock()
	}
	return item, nil
}
