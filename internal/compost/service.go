package compost

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Service defines the compost feature interface
type Service interface {
	Deposit(ctx context.Context, platform, platformID string, items []DepositItem) (*domain.CompostBin, error)
	Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResult, error)
	Shutdown(ctx context.Context) error
}

// DepositItem represents a single item deposit request
type DepositItem struct {
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

type service struct {
	repo           repository.CompostRepository
	userRepo       repository.User
	progressionSvc progression.Service
	publisher      *event.ResilientPublisher
	engine         *Engine
	wg             sync.WaitGroup
}

// NewService creates a new compost service
func NewService(
	repo repository.CompostRepository,
	userRepo repository.User,
	progressionSvc progression.Service,
	publisher *event.ResilientPublisher,
) Service {
	return &service{
		repo:           repo,
		userRepo:       userRepo,
		progressionSvc: progressionSvc,
		publisher:      publisher,
		engine:         NewEngine(),
	}
}

type resolvedDeposit struct {
	item     *domain.Item
	quantity int
}

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

// Harvest collects compost output, or returns status if not ready
func (s *service) Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResult, error) {
	if err := s.validateFeature(ctx); err != nil {
		return nil, err
	}

	user, bin, err := s.getUserAndBin(ctx, platform, platformID, false)
	if err != nil {
		return nil, err
	}

	// Handle idle/empty bin
	if bin == nil || bin.Status == domain.CompostBinStatusIdle {
		return s.idleHarvestResult(), nil
	}

	// Lazy status resolution
	s.resolveLazyBinStatus(bin)

	// If still composting, return status
	if bin.Status == domain.CompostBinStatusComposting {
		return s.compostingHarvestResult(bin), nil
	}

	// Ready or sludge - harvest!
	isSludge := bin.Status == domain.CompostBinStatusSludge

	// Get all items for output calculation
	allItems, err := s.userRepo.GetAllItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	// Calculate multiplier (base * progression bonuses)
	multiplier := DefaultMultiplier
	if bonus, err := s.progressionSvc.GetModifiedValue(ctx, progression.FeatureCompost, 1.0); err == nil {
		multiplier *= bonus
	}

	output := s.engine.CalculateOutput(bin.InputValue, bin.DominantType, isSludge, allItems, multiplier)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin harvest transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	if err := s.processHarvestItems(ctx, tx, user.ID, output); err != nil {
		return nil, err
	}

	if err := tx.ResetBin(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to reset bin: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit harvest: %w", err)
	}

	s.awardHarvestXP(ctx, user.ID, bin.InputValue, isSludge)

	return &domain.HarvestResult{
		Harvested: true,
		Output:    output,
	}, nil
}

func (s *service) validateFeature(ctx context.Context) error {
	unlocked, err := s.progressionSvc.IsFeatureUnlocked(ctx, progression.FeatureCompost)
	if err != nil {
		return fmt.Errorf("failed to check compost feature: %w", err)
	}
	if !unlocked {
		return fmt.Errorf("compost requires feature unlock: %w", domain.ErrFeatureLocked)
	}
	return nil
}

func (s *service) getUserAndBin(ctx context.Context, platform, platformID string, createIfMissing bool) (*domain.User, *domain.CompostBin, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	bin, err := s.repo.GetBin(ctx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bin: %w", err)
	}
	if bin == nil && createIfMissing {
		bin, err = s.repo.CreateBin(ctx, user.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create bin: %w", err)
		}
	}
	return user, bin, nil
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

func (s *service) processHarvestItems(ctx context.Context, tx repository.CompostTx, userID string, output *domain.CompostOutput) error {
	inv, err := tx.GetInventory(ctx, userID)
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
		slotIdx, _ := utils.FindSlot(inv, item.ID)
		if slotIdx >= 0 {
			inv.Slots[slotIdx].Quantity += qty
		} else {
			inv.Slots = append(inv.Slots, domain.InventorySlot{
				ItemID:       item.ID,
				Quantity:     qty,
				QualityLevel: domain.QualityCommon,
			})
		}
	}

	if err := tx.UpdateInventory(ctx, userID, *inv); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}
	return nil
}

func (s *service) awardHarvestXP(ctx context.Context, userID string, inputValue int, isSludge bool) {
	xpAmount := inputValue / 10
	if xpAmount < 1 {
		xpAmount = 1
	}

	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeCompostHarvested),
			Payload: domain.CompostHarvestedPayload{
				UserID:     userID,
				InputValue: inputValue,
				XPAmount:   xpAmount,
				IsSludge:   isSludge,
				Timestamp:  time.Now().Unix(),
			},
		})
	}
}

// Shutdown waits for async goroutines to complete
func (s *service) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return MsgReadyNow
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
