package user

import (
	"context"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/search"
)

// Compile-time checks: service implements search dependency interfaces
var _ search.UserResolver = (*service)(nil)
var _ search.ItemLookup = (*service)(nil)
var _ search.RewardGranter = (*service)(nil)

// GetUserOrRegister satisfies search.UserResolver by delegating to the
// existing getUserOrRegister method (which is unexported).
func (s *service) GetUserOrRegister(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	return s.getUserOrRegister(ctx, platform, platformID, username)
}

// GetItemByName is already declared in effect_context.go and satisfies search.ItemLookup.

// BuildPublicNameIndex creates a map from public_name -> internal_name
// using the service's item cache.
func (s *service) BuildPublicNameIndex() map[string]string {
	s.itemCacheMu.RLock()
	defer s.itemCacheMu.RUnlock()

	index := make(map[string]string, len(s.itemCacheByName))
	for internalName, item := range s.itemCacheByName {
		if item.PublicName != "" {
			index[strings.ToLower(item.PublicName)] = internalName
		}
	}
	return index
}

// GrantSearchReward adds a lootbox to inventory within a transaction.
func (s *service) GrantSearchReward(ctx context.Context, user *domain.User, quantity int, qualityLevel domain.QualityLevel) error {
	log := logger.FromContext(ctx)

	item, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
	if err != nil {
		log.Error("Failed to get lootbox0 item", "error", err)
		return domain.ErrFailedToGetItem
	}
	if item == nil {
		log.Error("Lootbox0 item not found in database")
		return domain.ErrItemNotFound
	}

	return s.withTx(ctx, func(txCtx context.Context, tx repository.UserTx) error {
		return s.addItemToTx(txCtx, tx, user.ID, item.ID, quantity, qualityLevel)
	})
}

// GrantItemReward adds a specific item to inventory within a transaction.
func (s *service) GrantItemReward(ctx context.Context, user *domain.User, item *domain.Item, quantity int, qualityLevel domain.QualityLevel) error {
	return s.withTx(ctx, func(txCtx context.Context, tx repository.UserTx) error {
		return s.addItemToTx(txCtx, tx, user.ID, item.ID, quantity, qualityLevel)
	})
}
