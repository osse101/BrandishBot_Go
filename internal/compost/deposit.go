package compost

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Deposit adds items to the user's compost bin, auto-starting if first deposit
func (s *service) Deposit(ctx context.Context, platform, platformID string, items []DepositItem) (*domain.CompostBin, error) {
	if err := s.validateFeature(ctx); err != nil {
		return nil, err
	}

	user, bin, err := s.getUserAndBin(ctx, platform, platformID, true)
	if err != nil {
		return nil, err
	}

	if err := s.checkDepositPossible(bin); err != nil {
		return nil, err
	}

	resolved, err := s.resolveDepositItems(ctx, items)
	if err != nil {
		return nil, err
	}

	if err := s.checkBinCapacity(bin, resolved); err != nil {
		return nil, err
	}

	if err := s.executeDepositTransaction(ctx, user.ID, bin, resolved); err != nil {
		return nil, err
	}

	return bin, nil
}

func (s *service) checkBinCapacity(bin *domain.CompostBin, resolved []resolvedDeposit) error {
	newItemTotal := 0
	for _, r := range resolved {
		newItemTotal += r.quantity
	}
	if bin.ItemCount+newItemTotal > bin.Capacity {
		return fmt.Errorf("%w: bin has %d/%d slots used, cannot add %d more", domain.ErrCompostBinFull, bin.ItemCount, bin.Capacity, newItemTotal)
	}
	return nil
}

func (s *service) executeDepositTransaction(ctx context.Context, userID string, bin *domain.CompostBin, resolved []resolvedDeposit) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	binLocked, err := tx.GetBinForUpdate(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to lock bin: %w", err)
	}
	// Copy data to the original bin pointer to maintain state for the caller
	*bin = *binLocked

	inv, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	for _, r := range resolved {
		slotIdx, qty := utils.FindSlot(inv, r.item.ID)
		if slotIdx < 0 || qty < r.quantity {
			return fmt.Errorf("%w: %s", domain.ErrInsufficientQuantity, r.item.PublicName)
		}
		utils.RemoveFromSlot(inv, slotIdx, r.quantity)
	}

	if err := tx.UpdateInventory(ctx, userID, *inv); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	s.updateBinWithDeposits(bin, resolved)

	if err := tx.UpdateBin(ctx, bin); err != nil {
		return fmt.Errorf("failed to update bin: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

func (s *service) checkDepositPossible(bin *domain.CompostBin) error {
	if bin.Status == domain.CompostBinStatusReady || bin.Status == domain.CompostBinStatusSludge {
		return domain.ErrCompostMustHarvest
	}
	if bin.Status == domain.CompostBinStatusComposting && bin.ReadyAt != nil && !time.Now().Before(*bin.ReadyAt) {
		return domain.ErrCompostMustHarvest
	}
	return nil
}

func (s *service) resolveDepositItems(ctx context.Context, items []DepositItem) ([]resolvedDeposit, error) {
	allRepoItems, err := s.userRepo.GetAllItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}
	itemsByName := make(map[string]*domain.Item, len(allRepoItems))
	for i := range allRepoItems {
		itemsByName[allRepoItems[i].InternalName] = &allRepoItems[i]
	}

	var resolved = make([]resolvedDeposit, 0, len(items))
	for _, di := range items {
		domItem := itemsByName[di.ItemName]
		if domItem == nil {
			for i := range allRepoItems {
				if allRepoItems[i].PublicName == di.ItemName {
					domItem = &allRepoItems[i]
					break
				}
			}
		}
		if domItem == nil {
			return nil, fmt.Errorf("%w: %s", domain.ErrItemNotFound, di.ItemName)
		}
		if !domain.HasTag(domItem.Types, domain.CompostableTag) {
			return nil, fmt.Errorf("%w: %s", domain.ErrCompostNotCompostable, di.ItemName)
		}
		if di.Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity must be positive", domain.ErrInvalidQuantity)
		}
		resolved = append(resolved, resolvedDeposit{item: domItem, quantity: di.Quantity})
	}
	return resolved, nil
}

func (s *service) updateBinWithDeposits(bin *domain.CompostBin, resolved []resolvedDeposit) {
	now := time.Now()
	for _, r := range resolved {
		bin.Items = append(bin.Items, domain.CompostBinItem{
			ItemID:       r.item.ID,
			ItemName:     r.item.InternalName,
			Quantity:     r.quantity,
			QualityLevel: domain.QualityCommon,
			BaseValue:    r.item.BaseValue,
			ContentTypes: r.item.ContentType,
		})
	}

	bin.ItemCount = s.engine.TotalItemCount(bin.Items)
	bin.InputValue = s.engine.CalculateInputValue(bin.Items)
	bin.DominantType = s.engine.DetermineDominantType(bin.Items)

	if bin.Status == domain.CompostBinStatusIdle {
		bin.Status = domain.CompostBinStatusComposting
		bin.StartedAt = &now
	}

	readyAt := s.engine.CalculateReadyAt(*bin.StartedAt, bin.ItemCount)
	bin.ReadyAt = &readyAt
	sludgeAt := s.engine.CalculateSludgeAt(readyAt)
	bin.SludgeAt = &sludgeAt
}
