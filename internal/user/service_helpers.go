package user

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// getItemByNameCached retrieves an item from cache or DB
func (s *service) getItemByNameCached(ctx context.Context, name string) (*domain.Item, error) {
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
