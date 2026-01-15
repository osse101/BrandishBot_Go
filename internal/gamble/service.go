package gamble

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Service defines the interface for gamble operations
type Service interface {
	StartGamble(ctx context.Context, platform, platformID, username string, bets []domain.LootboxBet) (*domain.Gamble, error)
	JoinGamble(ctx context.Context, gambleID uuid.UUID, platform, platformID, username string, bets []domain.LootboxBet) error
	GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error)
	ExecuteGamble(ctx context.Context, id uuid.UUID) (*domain.GambleResult, error)
	GetActiveGamble(ctx context.Context) (*domain.Gamble, error)
	Shutdown(ctx context.Context) error
}

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
}

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}


type service struct {
	repo           repository.Gamble
	eventBus       event.Bus
	lootboxSvc     lootbox.Service
	jobService     JobService
	progressionSvc ProgressionService
	statsSvc       stats.Service
	namingResolver naming.Resolver
	joinDuration   time.Duration
	wg             sync.WaitGroup // Tracks async goroutines for graceful shutdown
}

// NewService creates a new gamble service
func NewService(repo repository.Gamble, eventBus event.Bus, lootboxSvc lootbox.Service, statsSvc stats.Service, joinDuration time.Duration, jobService JobService, progressionSvc ProgressionService, namingResolver naming.Resolver) Service {
	return &service{
		repo:           repo,
		eventBus:       eventBus,
		lootboxSvc:     lootboxSvc,
		jobService:     jobService,
		progressionSvc: progressionSvc,
		statsSvc:       statsSvc,
		namingResolver: namingResolver,
		joinDuration:   joinDuration,
	}
}

// resolveItemName attempts to resolve a user-provided item name to its internal name.
// It first tries the naming resolver, then falls back to using the input as-is.
// This allows users to use either public names ("junkbox") or internal names ("lootbox_tier0").
func (s *service) resolveItemName(ctx context.Context, itemName string) (string, error) {
	// Try naming resolver first (handles public names)
	if s.namingResolver != nil {
		if internalName, ok := s.namingResolver.ResolvePublicName(itemName); ok {
			return internalName, nil
		}
	}

	// Fall back - assume it's already an internal name
	// Validate by checking if item exists
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf("%s '%s': %w", ErrContextFailedToResolveItemName, itemName, err)
	}
	if item == nil {
		return "", fmt.Errorf("%w: %s (%s)", domain.ErrItemNotFound, itemName, ErrMsgItemNotFoundAsPublicOrInternalName)
	}

	return itemName, nil
}

// resolveLootboxBet resolves a bet's item name to its item ID
// Returns the resolved item ID or an error
func (s *service) resolveLootboxBet(ctx context.Context, bet domain.LootboxBet) (int, error) {
	// Resolve name to internal name
	internalName, err := s.resolveItemName(ctx, bet.ItemName)
	if err != nil {
		return 0, fmt.Errorf("%s '%s': %w", ErrContextFailedToResolveItemName, bet.ItemName, err)
	}

	// Get item by internal name to get ID
	item, err := s.repo.GetItemByName(ctx, internalName)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", ErrContextFailedToGetItem, err)
	}
	if item == nil {
		return 0, fmt.Errorf("%w: %s", domain.ErrItemNotFound, internalName)
	}

	// Validate it's a lootbox
	if len(item.InternalName) < LootboxPrefixLength || item.InternalName[:LootboxPrefixLength] != LootboxPrefix {
		return 0, fmt.Errorf("%w: %s (id:%d)", domain.ErrNotALootbox, item.InternalName, item.ID)
	}

	return item.ID, nil
}

// StartGamble initiates a new gamble
func (s *service) StartGamble(ctx context.Context, platform, platformID, username string, bets []domain.LootboxBet) (*domain.Gamble, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgStartGambleCalled, "platform", platform, "platformID", platformID, "username", username, "bets", bets)

	if err := s.validateGambleStartInput(bets); err != nil {
		return nil, err
	}

	user, err := s.getAndValidateGambleUser(ctx, platform, platformID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureNoActiveGamble(ctx); err != nil {
		return nil, err
	}

	gamble := s.createGambleRecord(user.ID)

	// Validate bets and resolve item names to IDs
	resolvedItemIDs, err := s.validateGambleBets(ctx, bets)
	if err != nil {
		return nil, err
	}

	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToGetInventory, err)
	}

	// Consume bet items from inventory using resolved IDs
	for i, bet := range bets {
		itemID := resolvedItemIDs[i]
		if err := consumeItem(inventory, itemID, bet.Quantity); err != nil {
			return nil, fmt.Errorf("%s (item %d): %w", ErrContextFailedToConsumeBet, itemID, err)
		}
	}

	participant := &domain.Participant{
		GambleID:    gamble.ID,
		UserID:      user.ID,
		LootboxBets: bets,
		Username:    username,
	}

	if err := s.executeGambleStartTx(ctx, user.ID, inventory, gamble, participant); err != nil {
		return nil, err
	}

	s.publishGambleStartedEvent(ctx, gamble)

	s.wg.Add(1)
	go s.awardGamblerXP(context.Background(), user.ID, calculateTotalLootboxes(bets), "start", false)

	return gamble, nil
}

// JoinGamble adds a user to an existing gamble
func (s *service) JoinGamble(ctx context.Context, gambleID uuid.UUID, platform, platformID, username string, bets []domain.LootboxBet) error {
	log := logger.FromContext(ctx)
	log.Info(LogMsgJoinGambleCalled, "gambleID", gambleID, "username", username)

	// Validate bets
	if len(bets) == 0 {
		return domain.ErrAtLeastOneLootboxRequired
	}

	// Get User
	user, err := s.getAndValidateGambleUser(ctx, platform, platformID)
	if err != nil {
		return err
	}

	// Get Gamble
	gamble, err := s.getAndValidateActiveGamble(ctx, gambleID)
	if err != nil {
		return err
	}

	// Note: Duplicate join prevention is enforced by database constraint
	// (idx_gamble_participants_unique_user on gamble_participants table)

	// Validate bets and resolve item names to IDs
	resolvedItemIDs, err := s.validateGambleBets(ctx, bets)
	if err != nil {
		return err
	}

	// Execute transaction
	if err := s.executeGambleJoinTx(ctx, user.ID, gamble.ID, username, bets, resolvedItemIDs); err != nil {
		return err
	}

	// Award Gambler XP for joining (async, don't block)
	// Run async with detached context to prevent cancellation affecting XP award
	s.wg.Add(1)
	go s.awardGamblerXP(context.Background(), user.ID, calculateTotalLootboxes(bets), "join", false)

	return nil
}

// executeGambleJoinTx encapsulates the transactional logic for joining a gamble
func (s *service) executeGambleJoinTx(ctx context.Context, userID string, gambleID uuid.UUID, username string, bets []domain.LootboxBet, resolvedItemIDs []int) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToBeginTx, err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get Inventory
	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToGetInventory, err)
	}

	// Consume Bets using resolved item IDs
	for i, bet := range bets {
		itemID := resolvedItemIDs[i]
		if err := consumeItem(inventory, itemID, bet.Quantity); err != nil {
			return fmt.Errorf("%s (item %d): %w", ErrContextFailedToConsumeBet, itemID, err)
		}
	}

	// Update Inventory
	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToUpdateInventory, err)
	}

	// Add Participant
	participant := &domain.Participant{
		GambleID:    gambleID,
		UserID:      userID,
		LootboxBets: bets,
		Username:    username,
	}
	if err := s.repo.JoinGamble(ctx, participant); err != nil {
		if errors.Is(err, domain.ErrUserAlreadyJoined) {
			return domain.ErrUserAlreadyJoined
		}
		return fmt.Errorf("%s: %w", ErrContextFailedToJoinGamble, err)
	}

	return tx.Commit(ctx)
}

// ExecuteGamble runs the gamble logic
func (s *service) ExecuteGamble(ctx context.Context, id uuid.UUID) (*domain.GambleResult, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgExecuteGambleCalled, "gambleID", id)

	gamble, err := s.repo.GetGamble(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToGetGamble, err)
	}
	if gamble == nil {
		return nil, domain.ErrGambleNotFound
	}

	if gamble.State == domain.GambleStateCompleted {
		log.Info(LogMsgGambleAlreadyCompleted, "gambleID", id)
		return nil, nil
	}

	if err := s.validateGambleExecution(gamble); err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginGambleTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToBeginTx, err)
	}
	defer repository.SafeRollback(ctx, tx)

	if err := s.transitionToOpeningState(ctx, tx, id); err != nil {
		return nil, err
	}

	userValues, allOpenedItems, totalGambleValue := s.openParticipantsLootboxes(ctx, gamble)
	s.trackCriticalFailures(ctx, id, userValues, totalGambleValue)

	if err := tx.SaveOpenedItems(ctx, allOpenedItems); err != nil {
		return nil, fmt.Errorf("failed to save opened items: %w", err)
	}

	winnerID, highestValue := s.determineGambleWinners(ctx, id, userValues)
	s.trackNearMisses(ctx, id, winnerID, highestValue, userValues)

	if winnerID != "" {
		if err := s.awardItemsToWinner(ctx, tx, winnerID, allOpenedItems); err != nil {
			return nil, err
		}
	}

	result := &domain.GambleResult{
		GambleID:   id,
		WinnerID:   winnerID,
		TotalValue: totalGambleValue,
		Items:      allOpenedItems,
	}

	if err := tx.CompleteGamble(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to complete gamble: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToCommitTx, err)
	}

	if winnerID != "" {
		s.wg.Add(1)
		go s.awardGamblerXP(context.Background(), winnerID, 0, "win", true)
	}

	return result, nil
}

// validateGambleBets validates bets and resolves item names to IDs
// Returns a slice of resolved item IDs corresponding to each bet
func (s *service) validateGambleBets(ctx context.Context, bets []domain.LootboxBet) ([]int, error) {
	resolvedItemIDs := make([]int, len(bets))
	for i, bet := range bets {
		itemID, err := s.resolveLootboxBet(ctx, bet)
		if err != nil {
			return nil, err
		}
		resolvedItemIDs[i] = itemID
	}
	return resolvedItemIDs, nil
}

func (s *service) validateGambleExecution(gamble *domain.Gamble) error {
	if gamble.State != domain.GambleStateJoining {
		return fmt.Errorf("%w (current: %s)", domain.ErrNotInJoiningState, gamble.State)
	}
	if time.Now().Before(gamble.JoinDeadline) {
		return fmt.Errorf("%s (deadline: %v)", ErrMsgCannotExecuteBeforeDeadline, gamble.JoinDeadline)
	}
	return nil
}

func (s *service) transitionToOpeningState(ctx context.Context, tx repository.GambleTx, id uuid.UUID) error {
	rowsAffected, err := tx.UpdateGambleStateIfMatches(ctx, id, domain.GambleStateJoining, domain.GambleStateOpening)
	if err != nil {
		return fmt.Errorf("failed to transition gamble state: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf(ErrMsgGambleAlreadyExecuted)
	}
	return nil
}

func (s *service) openParticipantsLootboxes(ctx context.Context, gamble *domain.Gamble) (map[string]int64, []domain.GambleOpenedItem, int64) {
	userValues := make(map[string]int64)
	var allOpenedItems []domain.GambleOpenedItem
	var totalGambleValue int64

	for _, p := range gamble.Participants {
		for _, bet := range p.LootboxBets {
			// Resolve bet item name to ID to get lootbox item
			itemID, err := s.resolveLootboxBet(ctx, bet)
			if err != nil {
				continue
			}

			lootboxItem, err := s.repo.GetItemByID(ctx, itemID)
			if err != nil || lootboxItem == nil {
				continue
			}

			drops, err := s.lootboxSvc.OpenLootbox(ctx, lootboxItem.InternalName, bet.Quantity)
			if err != nil {
				continue
			}

			for _, drop := range drops {
				totalValue := int64(drop.Value * drop.Quantity)
				if s.progressionSvc != nil {
					modifiedValue, err := s.progressionSvc.GetModifiedValue(ctx, ProgressionFeatureGambleWinBonus, float64(totalValue))
					if err == nil {
						totalValue = int64(modifiedValue)
					}
				}

				allOpenedItems = append(allOpenedItems, domain.GambleOpenedItem{
					GambleID:   gamble.ID,
					UserID:     p.UserID,
					ItemID:     drop.ItemID,
					Value:      totalValue,
					ShineLevel: drop.ShineLevel,
				})

				userValues[p.UserID] += totalValue
				totalGambleValue += totalValue
			}
		}
	}
	return userValues, allOpenedItems, totalGambleValue
}

func (s *service) trackCriticalFailures(ctx context.Context, id uuid.UUID, userValues map[string]int64, totalGambleValue int64) {
	if len(userValues) <= 1 || totalGambleValue <= 0 || s.statsSvc == nil {
		return
	}
	averageScore := float64(totalGambleValue) / float64(len(userValues))
	criticalFailThreshold := int64(averageScore * CriticalFailThreshold)
	for userID, val := range userValues {
		if val <= criticalFailThreshold {
			_ = s.statsSvc.RecordUserEvent(ctx, userID, domain.EventGambleCriticalFail, map[string]interface{}{
				"gamble_id":     id,
				"score":         val,
				"average_score": averageScore,
				"threshold":     criticalFailThreshold,
			})
		}
	}
}

func (s *service) determineGambleWinners(ctx context.Context, id uuid.UUID, userValues map[string]int64) (string, int64) {
	var highestValue int64 = InitialHighestValue
	var winners []string

	for userID, val := range userValues {
		if val > highestValue {
			highestValue = val
			winners = []string{userID}
		} else if val == highestValue {
			winners = append(winners, userID)
		}
	}

	if len(winners) == 0 {
		return "", 0
	}

	if len(winners) > 1 {
		idx := utils.SecureRandomInt(len(winners))
		winnerID := winners[idx]
		if s.statsSvc != nil {
			for _, uid := range winners {
				if uid != winnerID {
					_ = s.statsSvc.RecordUserEvent(ctx, uid, domain.EventGambleTieBreakLost, map[string]interface{}{
						"gamble_id": id,
						"score":     highestValue,
					})
				}
			}
		}
		return winnerID, highestValue
	}
	return winners[0], highestValue
}

func (s *service) trackNearMisses(ctx context.Context, id uuid.UUID, winnerID string, highestValue int64, userValues map[string]int64) {
	if winnerID == "" || highestValue <= 0 || s.statsSvc == nil {
		return
	}
	threshold := int64(float64(highestValue) * NearMissThreshold)
	for userID, val := range userValues {
		if userID == winnerID || val == highestValue {
			continue
		}
		if val >= threshold {
			_ = s.statsSvc.RecordUserEvent(ctx, userID, domain.EventGambleNearMiss, map[string]interface{}{
				"gamble_id":    id,
				"score":        val,
				"winner_score": highestValue,
				"diff":         highestValue - val,
			})
		}
	}
}

func (s *service) awardItemsToWinner(ctx context.Context, tx repository.GambleTx, winnerID string, allOpenedItems []domain.GambleOpenedItem) error {
	inv, err := tx.GetInventory(ctx, winnerID)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToGetWinnerInv, err)
	}

	itemsToAdd := make(map[int]int)
	for _, item := range allOpenedItems {
		itemsToAdd[item.ItemID]++
	}

	for i, slot := range inv.Slots {
		if qty, ok := itemsToAdd[slot.ItemID]; ok {
			inv.Slots[i].Quantity += qty
			delete(itemsToAdd, slot.ItemID)
		}
	}

	var newItemIDs []int
	if len(itemsToAdd) > 0 {
		newItemIDs = make([]int, 0, len(itemsToAdd))
		for itemID := range itemsToAdd {
			newItemIDs = append(newItemIDs, itemID)
		}
		sort.Ints(newItemIDs)
	}

	for _, itemID := range newItemIDs {
		inv.Slots = append(inv.Slots, domain.InventorySlot{ItemID: itemID, Quantity: itemsToAdd[itemID]})
	}

	if err := tx.UpdateInventory(ctx, winnerID, *inv); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToUpdateWinnerInv, err)
	}
	return nil
}

// GetGamble retrieves a gamble by ID
func (s *service) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	return s.repo.GetGamble(ctx, id)
}

// GetActiveGamble retrieves the current active gamble
func (s *service) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) {
	return s.repo.GetActiveGamble(ctx)
}

// Helper to consume item from inventory
func consumeItem(inventory *domain.Inventory, itemID, quantity int) error {
	for i := range inventory.Slots {
		if inventory.Slots[i].ItemID == itemID {
			if inventory.Slots[i].Quantity < quantity {
				return domain.ErrInsufficientQuantity
			}
			if inventory.Slots[i].Quantity == quantity {
				// Remove slot
				inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
			} else {
				// Reduce quantity
				inventory.Slots[i].Quantity -= quantity
			}
			return nil
		}
	}
	return domain.ErrItemNotFound
}

// calculateTotalLootboxes sums up lootbox quantities from bets
func calculateTotalLootboxes(bets []domain.LootboxBet) int {
	total := 0
	for _, bet := range bets {
		total += bet.Quantity
	}
	return total
}

// awardGamblerXP awards  Gambler job XP for gambling operations
func (s *service) awardGamblerXP(ctx context.Context, userID string, lootboxCount int, source string, isWin bool) {
	defer s.wg.Done() // Signal completion when goroutine ends

	if s.jobService == nil {
		return // Job system not enabled
	}

	// Use exported constants for XP amounts
	xp := lootboxCount * job.GamblerXPPerLootbox
	if isWin {
		xp += job.GamblerWinBonus
	}

	if xp <= 0 {
		return
	}

	metadata := map[string]interface{}{
		MetadataKeySource:       source,
		MetadataKeyLootboxCount: lootboxCount,
		MetadataKeyIsWin:        isWin,
	}

	result, err := s.jobService.AwardXP(ctx, userID, job.JobKeyGambler, xp, source, metadata)
	if err != nil {
		logger.FromContext(ctx).Warn(LogMsgFailedToAwardGamblerXP, "error", err, "user_id", userID)
	} else if result != nil && result.LeveledUp {
		logger.FromContext(ctx).Info(LogMsgGamblerLeveledUp, "user_id", userID, "new_level", result.NewLevel)
	}
}

// Shutdown gracefully shuts down the gamble service by waiting for all async operations to complete
func (s *service) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info(LogMsgShuttingDownGambleService)

	// Wait for all async XP awards to complete
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info(LogMsgGambleServiceShutdownDone)
		return nil
	case <-ctx.Done():
		log.Warn(LogMsgGambleServiceShutdownForced)
		return ctx.Err()
	}
}

func (s *service) getAndValidateGambleUser(ctx context.Context, platform, platformID string) (*domain.User, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToGetUser, err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (s *service) ensureNoActiveGamble(ctx context.Context) error {
	active, err := s.repo.GetActiveGamble(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToCheckActive, err)
	}
	if active != nil {
		return domain.ErrGambleAlreadyActive
	}
	return nil
}

func (s *service) getAndValidateActiveGamble(ctx context.Context, gambleID uuid.UUID) (*domain.Gamble, error) {
	gamble, err := s.repo.GetGamble(ctx, gambleID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToGetGamble, err)
	}
	if gamble == nil {
		return nil, domain.ErrGambleNotFound
	}
	if gamble.State != domain.GambleStateJoining {
		return nil, domain.ErrNotInJoiningState
	}
	if time.Now().After(gamble.JoinDeadline) {
		return nil, domain.ErrJoinDeadlinePassed
	}
	return gamble, nil
}

func (s *service) executeGambleStartTx(ctx context.Context, userID string, inventory *domain.Inventory, gamble *domain.Gamble, participant *domain.Participant) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToBeginTx, err)
	}
	defer repository.SafeRollback(ctx, tx)

	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToUpdateInventory, err)
	}

	if err := s.repo.CreateGamble(ctx, gamble); err != nil {
		if errors.Is(err, domain.ErrGambleAlreadyActive) {
			return domain.ErrGambleAlreadyActive
		}
		return fmt.Errorf("%s: %w", ErrContextFailedToCreateGamble, err)
	}

	if err := s.repo.JoinGamble(ctx, participant); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToAddInitiator, err)
	}

	return tx.Commit(ctx)
}

func (s *service) validateGambleStartInput(bets []domain.LootboxBet) error {
	if len(bets) == 0 {
		return domain.ErrAtLeastOneLootboxRequired
	}
	for _, bet := range bets {
		if bet.Quantity <= 0 {
			return domain.ErrBetQuantityMustBePositive
		}
	}
	return nil
}

func (s *service) createGambleRecord(initiatorID string) *domain.Gamble {
	return &domain.Gamble{
		ID:           uuid.New(),
		InitiatorID:  initiatorID,
		State:        domain.GambleStateJoining,
		CreatedAt:    time.Now(),
		JoinDeadline: time.Now().Add(s.joinDuration),
	}
}

func (s *service) publishGambleStartedEvent(ctx context.Context, gamble *domain.Gamble) {
	if s.eventBus == nil {
		logger.FromContext(ctx).Error("Failed to publish "+LogContextGambleStartedEvent, "reason", LogReasonEventBusNil)
		return
	}
	err := s.eventBus.Publish(ctx, event.Event{
		Version: EventSchemaVersion,
		Type:    domain.EventGambleStarted,
		Payload: gamble,
	})
	if err != nil {
		logger.FromContext(ctx).Error("Failed to publish "+LogContextGambleStartedEvent, "error", err)
	}
}
