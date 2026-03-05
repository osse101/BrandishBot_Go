package compost

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func (s *service) Deposit(ctx context.Context, platform, platformID string, items []DepositItem) (*domain.CompostBin, error) {
	user, bin, err := s.getUserAndBin(ctx, platform, platformID, true)
	if err != nil {
		return nil, err
	}

	if err := s.validateFeature(ctx, user.ID); err != nil {
		return nil, err
	}

	if err := s.checkDepositPossible(bin); err != nil {
		return nil, err
	}

	capacityFloat, _ := s.progressionSvc.GetModifiedValue(ctx, user.ID, featureCompostCapacity, 3.0)
	bin.Capacity = int(capacityFloat)

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
	transaction, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, transaction)

	binLocked, err := transaction.GetBinForUpdate(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to lock bin: %w", err)
	}

	currentCapacity := bin.Capacity

	*bin = *binLocked

	bin.Capacity = currentCapacity

	inventory, err := transaction.GetInventory(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	for _, r := range resolved {
		slotIdx, qty := utils.FindSlot(inventory, r.item.ID)
		if slotIdx < 0 || qty < r.quantity {
			return fmt.Errorf("%w: %s", domain.ErrInsufficientQuantity, r.item.PublicName)
		}
		utils.RemoveFromSlot(inventory, slotIdx, r.quantity)
	}

	if err := transaction.UpdateInventory(ctx, userID, *inventory); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	s.updateBinWithDeposits(ctx, bin, resolved)

	if err := transaction.UpdateBin(ctx, bin); err != nil {
		return fmt.Errorf("failed to update bin: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
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
		domainItem := itemsByName[di.ItemName]
		if domainItem == nil {
			for i := range allRepoItems {
				if allRepoItems[i].PublicName == di.ItemName {
					domainItem = &allRepoItems[i]
					break
				}
			}
		}
		if domainItem == nil {
			return nil, fmt.Errorf("%w: %s", domain.ErrItemNotFound, di.ItemName)
		}
		if !domain.HasTag(domainItem.Types, domain.CompostableTag) {
			return nil, fmt.Errorf("%w: %s", domain.ErrCompostNotCompostable, di.ItemName)
		}
		if di.Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity must be positive", domain.ErrInvalidQuantity)
		}
		resolved = append(resolved, resolvedDeposit{item: domainItem, quantity: di.Quantity})
	}
	return resolved, nil
}

func (s *service) updateBinWithDeposits(ctx context.Context, bin *domain.CompostBin, resolved []resolvedDeposit) {
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

	compostSpeedMult, _ := s.progressionSvc.GetModifiedValue(ctx, bin.UserID, "compost_speed", 0.0)
	sludgeExt, _ := s.progressionSvc.GetModifiedValue(ctx, bin.UserID, "sludge_extension", 0.0)

	readyAt := s.engine.CalculateReadyAt(*bin.StartedAt, bin.ItemCount, compostSpeedMult)
	bin.ReadyAt = &readyAt
	sludgeAt := s.engine.CalculateSludgeAt(readyAt, sludgeExt)
	bin.SludgeAt = &sludgeAt
}
