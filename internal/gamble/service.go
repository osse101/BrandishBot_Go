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
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Service defines the interface for gamble operations
type Service interface {
	StartGamble(ctx context.Context, platform, platformID, username string, bets []domain.LootboxBet) (*domain.Gamble, error)
	JoinGamble(ctx context.Context, gambleID uuid.UUID, platform, platformID, username string) error
	GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error)
	ExecuteGamble(ctx context.Context, id uuid.UUID) (*domain.GambleResult, error)
	GetActiveGamble(ctx context.Context) (*domain.Gamble, error)
	Shutdown(ctx context.Context) error
}

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}

// ResilientPublisher defines the interface for resilient event publishing
type ResilientPublisher interface {
	PublishWithRetry(ctx context.Context, evt event.Event)
}

type service struct {
	repo               repository.Gamble
	eventBus           event.Bus
	resilientPublisher ResilientPublisher
	lootboxSvc         lootbox.Service
	progressionSvc     ProgressionService
	namingResolver     naming.Resolver
	joinDuration       time.Duration
	rng                func(int) int
	wg                 sync.WaitGroup // Tracks async goroutines for graceful shutdown
}

// NewService creates a new gamble service
func NewService(repo repository.Gamble, eventBus event.Bus, resilientPublisher ResilientPublisher, lootboxSvc lootbox.Service, joinDuration time.Duration, progressionSvc ProgressionService, namingResolver naming.Resolver, rng func(int) int) Service {
	if rng == nil {
		rng = utils.SecureRandomInt
	}
	return &service{
		repo:               repo,
		eventBus:           eventBus,
		resilientPublisher: resilientPublisher,
		lootboxSvc:         lootboxSvc,
		progressionSvc:     progressionSvc,
		namingResolver:     namingResolver,
		joinDuration:       joinDuration,
		rng:                rng,
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

	// Create a local copy of bets to avoid modifying the caller's slice and race conditions
	gambleBets := make([]domain.LootboxBet, len(bets))
	copy(gambleBets, bets)

	// Consume bet items from inventory using resolved IDs
	for i := range gambleBets {
		itemID := resolvedItemIDs[i]
		qualityLevel, err := consumeItem(inventory, itemID, gambleBets[i].Quantity)
		if err != nil {
			return nil, fmt.Errorf("%s (item %d): %w", ErrContextFailedToConsumeBet, itemID, err)
		}
		gambleBets[i].QualityLevel = qualityLevel
	}

	participant := &domain.Participant{
		GambleID:    gamble.ID,
		UserID:      user.ID,
		LootboxBets: gambleBets,
		Username:    username,
	}

	if err := s.executeGambleStartTx(ctx, user.ID, inventory, gamble, participant); err != nil {
		return nil, err
	}

	s.publishGambleStartedEvent(ctx, gamble)
	s.publishGambleParticipatedEvent(ctx, gamble.ID.String(), user.ID, calculateTotalLootboxes(gambleBets), "start")

	return gamble, nil
}

// JoinGamble adds a user to an existing gamble
func (s *service) JoinGamble(ctx context.Context, gambleID uuid.UUID, platform, platformID, username string) error {
	log := logger.FromContext(ctx)
	log.Info(LogMsgJoinGambleCalled, "gambleID", gambleID, "username", username)

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

	// Get initiator's bets to use for this joiner
	var initiatorBets []domain.LootboxBet
	for _, p := range gamble.Participants {
		if p.UserID == gamble.InitiatorID {
			initiatorBets = p.LootboxBets
			break
		}
	}

	if len(initiatorBets) == 0 {
		return fmt.Errorf("failed to find initiator bets for gamble %s: %w", gambleID, domain.ErrGambleNotFound)
	}

	// Create a deep copy of bets to use for this joiner to avoid side effects
	bets := make([]domain.LootboxBet, len(initiatorBets))
	copy(bets, initiatorBets)

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

	// Publish gamble participated event (job handler awards XP)
	s.publishGambleParticipatedEvent(ctx, gambleID.String(), user.ID, calculateTotalLootboxes(bets), "join")

	return nil
}

// executeGambleJoinTx encapsulates the transactional logic for joining a gamble
func (s *service) executeGambleJoinTx(ctx context.Context, userID string, gambleID uuid.UUID, username string, bets []domain.LootboxBet, resolvedItemIDs []int) error {
	tx, err := s.repo.BeginGambleTx(ctx)
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
	for i := range bets {
		itemID := resolvedItemIDs[i]
		qualityLevel, err := consumeItem(inventory, itemID, bets[i].Quantity)
		if err != nil {
			return fmt.Errorf("%s (item %d): %w", ErrContextFailedToConsumeBet, itemID, err)
		}
		bets[i].QualityLevel = qualityLevel
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

	// Determine critical failures (before determining winner)
	critFailUsers := s.determineCriticalFailures(userValues, totalGambleValue)

	if err := tx.SaveOpenedItems(ctx, allOpenedItems); err != nil {
		return nil, fmt.Errorf("failed to save opened items: %w", err)
	}

	winnerID, highestValue, tieBreakLostUsers := s.determineGambleWinners(userValues)
	nearMissUsers := s.determineNearMisses(winnerID, highestValue, userValues)

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

	// Publish gamble completion event with per-participant outcomes
	participants := s.buildParticipantOutcomes(gamble, userValues, winnerID, critFailUsers, tieBreakLostUsers, nearMissUsers)
	s.publishGambleCompletedEvent(ctx, result, len(gamble.Participants), participants)

	return result, nil
}

// validateGambleBets validates bets and resolves item names to IDs
// Returns a slice of resolved item IDs corresponding to each bet
func (s *service) validateGambleBets(ctx context.Context, bets []domain.LootboxBet) ([]int, error) {
	resolvedItemIDs := make([]int, len(bets))
	for i, bet := range bets {
		if bet.Quantity > domain.MaxTransactionQuantity {
			return nil, fmt.Errorf("%w: max is %d", domain.ErrQuantityTooHigh, domain.MaxTransactionQuantity)
		}
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
	// Allow execution within grace period of deadline to handle clock skew/network delays
	deadlineWithGrace := gamble.JoinDeadline.Add(-ExecutionGracePeriod)
	if time.Now().Before(deadlineWithGrace) {
		return fmt.Errorf("%s (deadline: %v, grace_period: %v)", ErrMsgCannotExecuteBeforeDeadline, gamble.JoinDeadline, ExecutionGracePeriod)
	}
	return nil
}

func (s *service) transitionToOpeningState(ctx context.Context, tx repository.GambleTx, id uuid.UUID) error {
	rowsAffected, err := tx.UpdateGambleStateIfMatches(ctx, id, domain.GambleStateJoining, domain.GambleStateOpening)
	if err != nil {
		return fmt.Errorf("failed to transition gamble state: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New(ErrMsgGambleAlreadyExecuted)
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

			drops, err := s.lootboxSvc.OpenLootbox(ctx, lootboxItem.InternalName, bet.Quantity, bet.QualityLevel)
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
					GambleID:     gamble.ID,
					UserID:       p.UserID,
					ItemID:       drop.ItemID,
					Quantity:     drop.Quantity,
					Value:        totalValue,
					QualityLevel: drop.QualityLevel,
				})

				userValues[p.UserID] += totalValue
				totalGambleValue += totalValue
			}
		}
	}
	return userValues, allOpenedItems, totalGambleValue
}

// determineCriticalFailures returns the set of user IDs who had critical fail scores
func (s *service) determineCriticalFailures(userValues map[string]int64, totalGambleValue int64) map[string]bool {
	critFails := make(map[string]bool)
	if len(userValues) <= 1 || totalGambleValue <= 0 {
		return critFails
	}
	averageScore := float64(totalGambleValue) / float64(len(userValues))
	threshold := int64(averageScore * CriticalFailThreshold)
	for userID, val := range userValues {
		if val <= threshold {
			critFails[userID] = true
		}
	}
	return critFails
}

// determineGambleWinners returns the winner ID, highest score, and set of users who lost a tie-break
func (s *service) determineGambleWinners(userValues map[string]int64) (string, int64, map[string]bool) {
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

	tieBreakLost := make(map[string]bool)

	if len(winners) == 0 {
		return "", 0, tieBreakLost
	}

	if len(winners) > 1 {
		sort.Strings(winners)
		idx := s.rng(len(winners))
		winnerID := winners[idx]
		for _, uid := range winners {
			if uid != winnerID {
				tieBreakLost[uid] = true
			}
		}
		return winnerID, highestValue, tieBreakLost
	}
	return winners[0], highestValue, tieBreakLost
}

// determineNearMisses returns the set of user IDs who had near-miss scores (not the winner)
func (s *service) determineNearMisses(winnerID string, highestValue int64, userValues map[string]int64) map[string]bool {
	nearMiss := make(map[string]bool)
	if winnerID == "" || highestValue <= 0 {
		return nearMiss
	}
	threshold := int64(float64(highestValue) * NearMissThreshold)
	for userID, val := range userValues {
		if userID == winnerID || val == highestValue {
			continue
		}
		if val >= threshold {
			nearMiss[userID] = true
		}
	}
	return nearMiss
}

// buildParticipantOutcomes constructs per-participant outcome data for the GambleCompletedPayloadV2
func (s *service) buildParticipantOutcomes(gamble *domain.Gamble, userValues map[string]int64, winnerID string, critFailUsers, tieBreakLostUsers, nearMissUsers map[string]bool) []domain.GambleParticipantOutcome {
	outcomes := make([]domain.GambleParticipantOutcome, 0, len(gamble.Participants))
	for _, p := range gamble.Participants {
		outcomes = append(outcomes, domain.GambleParticipantOutcome{
			UserID:         p.UserID,
			Score:          userValues[p.UserID],
			LootboxCount:   calculateTotalLootboxes(p.LootboxBets),
			IsWinner:       p.UserID == winnerID,
			IsNearMiss:     nearMissUsers[p.UserID],
			IsCritFail:     critFailUsers[p.UserID],
			IsTieBreakLost: tieBreakLostUsers[p.UserID],
		})
	}
	return outcomes
}

// publishGambleParticipatedEvent publishes a gamble.participated event for XP tracking
func (s *service) publishGambleParticipatedEvent(ctx context.Context, gambleID, userID string, lootboxCount int, source string) {
	if s.resilientPublisher == nil {
		return
	}
	s.resilientPublisher.PublishWithRetry(ctx, event.Event{
		Version: EventSchemaVersion,
		Type:    event.Type(domain.EventTypeGambleParticipated),
		Payload: domain.GambleParticipatedPayload{
			GambleID:     gambleID,
			UserID:       userID,
			LootboxCount: lootboxCount,
			Source:       source,
			Timestamp:    time.Now().Unix(),
		},
	})
}

func (s *service) awardItemsToWinner(ctx context.Context, tx repository.GambleTx, winnerID string, allOpenedItems []domain.GambleOpenedItem) error {
	inv, err := tx.GetInventory(ctx, winnerID)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToGetWinnerInv, err)
	}

	itemsToAdd := make(map[int]int)
	for _, item := range allOpenedItems {
		itemsToAdd[item.ItemID] += item.Quantity
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

// Helper to consume item from inventory and return its quality level
func consumeItem(inventory *domain.Inventory, itemID, quantity int) (domain.QualityLevel, error) {
	for i := range inventory.Slots {
		if inventory.Slots[i].ItemID == itemID {
			if inventory.Slots[i].Quantity < quantity {
				return "", domain.ErrInsufficientQuantity
			}
			qualityLevel := inventory.Slots[i].QualityLevel
			if inventory.Slots[i].Quantity == quantity {
				// Remove slot
				inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
			} else {
				// Reduce quantity
				inventory.Slots[i].Quantity -= quantity
			}
			return qualityLevel, nil
		}
	}
	return domain.QualityLevel(""), domain.ErrItemNotFound
}

// calculateTotalLootboxes sums up lootbox quantities from bets
func calculateTotalLootboxes(bets []domain.LootboxBet) int {
	total := 0
	for _, bet := range bets {
		total += bet.Quantity
	}
	return total
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
	tx, err := s.repo.BeginGambleTx(ctx)
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
		if bet.Quantity > domain.MaxTransactionQuantity {
			return fmt.Errorf("%w: max is %d", domain.ErrQuantityTooHigh, domain.MaxTransactionQuantity)
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

func (s *service) publishGambleCompletedEvent(ctx context.Context, result *domain.GambleResult, participantCount int, participants []domain.GambleParticipantOutcome) {
	log := logger.FromContext(ctx)

	if s.resilientPublisher == nil {
		log.Error("Failed to publish GambleCompleted event", "reason", "resilientPublisher is nil")
		return
	}

	evt := event.NewGambleCompletedEvent(result.GambleID.String(), result.WinnerID, result.TotalValue, participantCount, participants)
	s.resilientPublisher.PublishWithRetry(ctx, evt)
}
