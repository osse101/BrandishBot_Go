package compost

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func (s *service) Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResult, error) {
	user, bin, err := s.getUserAndBin(ctx, platform, platformID, false)
	if err != nil {
		return nil, err
	}

	if err := s.validateFeature(ctx, user.ID); err != nil {
		return nil, err
	}

	if bin == nil || bin.Status == domain.CompostBinStatusIdle {
		return s.idleHarvestResult(), nil
	}

	s.resolveLazyBinStatus(bin)

	if bin.Status == domain.CompostBinStatusComposting {
		return s.compostingHarvestResult(bin), nil
	}

	isSludge := bin.Status == domain.CompostBinStatusSludge

	allItems, err := s.userRepo.GetAllItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	multiplier := DefaultMultiplier
	if bonus, err := s.progressionSvc.GetModifiedValue(ctx, "", progression.FeatureCompost, 1.0); err == nil {
		multiplier *= bonus
	}

	output := s.engine.CalculateOutput(bin.InputValue, bin.DominantType, isSludge, allItems, multiplier)

	transaction, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin harvest transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, transaction)

	if err := s.processHarvestItems(ctx, transaction, user.ID, output); err != nil {
		return nil, err
	}

	if err := transaction.ResetBin(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to reset bin: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit harvest: %w", err)
	}

	s.awardHarvestXP(ctx, user.ID, bin.ItemCount, bin.InputValue, isSludge)

	return &domain.HarvestResult{
		Harvested: true,
		Output:    output,
	}, nil
}

func (s *service) resolveLazyBinStatus(bin *domain.CompostBin) {
	now := time.Now()
	if bin.Status == domain.CompostBinStatusComposting {
		if bin.ReadyAt != nil && !now.Before(*bin.ReadyAt) {
			if bin.SludgeAt != nil && !now.Before(*bin.SludgeAt) {
				bin.Status = domain.CompostBinStatusSludge
			} else {
				bin.Status = domain.CompostBinStatusReady
			}
		}
	} else if bin.Status == domain.CompostBinStatusReady {
		if bin.SludgeAt != nil && !now.Before(*bin.SludgeAt) {
			bin.Status = domain.CompostBinStatusSludge
		}
	}
}

func (s *service) idleHarvestResult() *domain.HarvestResult {
	return &domain.HarvestResult{
		Harvested: false,
		Status: &domain.CompostStatusResponse{
			Status:    domain.CompostBinStatusIdle,
			Capacity:  DefaultCapacity,
			ItemCount: 0,
			Items:     []domain.CompostBinItem{},
			TimeLeft:  MsgBinEmpty,
		},
	}
}

func (s *service) compostingHarvestResult(bin *domain.CompostBin) *domain.HarvestResult {
	timeLeft := ""
	if bin.ReadyAt != nil {
		timeLeft = formatDuration(time.Until(*bin.ReadyAt))
	}
	return &domain.HarvestResult{
		Harvested: false,
		Status: &domain.CompostStatusResponse{
			Status:    bin.Status,
			Capacity:  bin.Capacity,
			ItemCount: bin.ItemCount,
			Items:     bin.Items,
			ReadyAt:   bin.ReadyAt,
			SludgeAt:  bin.SludgeAt,
			TimeLeft:  timeLeft,
		},
	}
}

func (s *service) processHarvestItems(ctx context.Context, transaction repository.CompostTx, userID string, output *domain.CompostOutput) error {
	inventory, err := transaction.GetInventory(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	outputItemNames := make([]string, 0, len(output.Items))
	for name := range output.Items {
		outputItemNames = append(outputItemNames, name)
	}
	outputItems, err := s.userRepo.GetItemsByNames(ctx, outputItemNames)
	if err != nil {
		return fmt.Errorf("failed to get output items: %w", err)
	}

	outputItemByName := make(map[string]*domain.Item, len(outputItems))
	for i := range outputItems {
		outputItemByName[outputItems[i].InternalName] = &outputItems[i]
	}

	log := logger.FromContext(ctx)
	for name, qty := range output.Items {
		item, ok := outputItemByName[name]
		if !ok {
			log.Warn("Output item not found, skipping", "item", name)
			continue
		}
		slotIdx, _ := utils.FindSlot(inventory, item.ID)
		if slotIdx >= 0 {
			inventory.Slots[slotIdx].Quantity += qty
		} else {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{
				ItemID:       item.ID,
				Quantity:     qty,
				QualityLevel: domain.QualityCommon,
			})
		}
	}

	if err := transaction.UpdateInventory(ctx, userID, *inventory); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}
	return nil
}

func (s *service) awardHarvestXP(ctx context.Context, userID string, itemCount int, inputValue int, isSludge bool) {
	experienceAmount := itemCount * 12
	if experienceAmount < 1 {
		experienceAmount = 1
	}

	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeCompostHarvested),
			Payload: domain.CompostHarvestedPayload{
				UserID:     userID,
				InputValue: inputValue,
				XPAmount:   experienceAmount,
				IsSludge:   isSludge,
				Timestamp:  time.Now().Unix(),
			},
		})
	}
}
