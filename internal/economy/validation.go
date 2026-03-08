package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func validateQuantity(quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf(ErrMsgInvalidQuantityFmt, quantity, domain.ErrInvalidInput)
	}
	if quantity > domain.MaxTransactionQuantity {
		return fmt.Errorf(ErrMsgQuantityExceedsMaxFmt, quantity, domain.MaxTransactionQuantity, domain.ErrInvalidInput)
	}
	return nil
}

func (s *service) checkBuyEligibility(ctx context.Context, item *domain.Item) error {
	isBuyable, err := s.repo.IsItemBuyable(ctx, item.InternalName)
	if err != nil {
		return fmt.Errorf(ErrMsgCheckBuyableFailed, err)
	}
	if !isBuyable {
		return fmt.Errorf(ErrMsgItemNotBuyableFmt, item.InternalName, domain.ErrNotBuyable)
	}

	if s.progressionService != nil {
		unlocked, err := s.progressionService.IsItemUnlocked(ctx, item.InternalName)
		if err != nil {
			return fmt.Errorf("failed to check unlock status: %w", err)
		}
		if !unlocked {
			return domain.ErrItemLocked
		}
	}
	return nil
}
