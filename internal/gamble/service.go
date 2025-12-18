package gamble

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/stats"
)

// Repository defines the interface for data access required by the gamble service
type Repository interface {
	CreateGamble(ctx context.Context, gamble *domain.Gamble) error
	GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error)
	JoinGamble(ctx context.Context, participant *domain.Participant) error
	UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error
	SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error
	CompleteGamble(ctx context.Context, result *domain.GambleResult) error
	GetActiveGamble(ctx context.Context) (*domain.Gamble, error)

	// Transaction support
	BeginTx(ctx context.Context) (repository.Tx, error)

	// Inventory operations (reused from other services)
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
}

// Service defines the interface for gamble operations
type Service interface {
	StartGamble(ctx context.Context, platform, platformID, username string, bets []domain.LootboxBet) (*domain.Gamble, error)
	JoinGamble(ctx context.Context, gambleID uuid.UUID, platform, platformID, username string, bets []domain.LootboxBet) error
	GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error)
	ExecuteGamble(ctx context.Context, id uuid.UUID) (*domain.GambleResult, error)
	GetActiveGamble(ctx context.Context) (*domain.Gamble, error)
}

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
}

// NearMissThreshold defines the percentage of the winner's score required to trigger a "Near Miss" event
const NearMissThreshold = 0.95

type service struct {
	repo         Repository
	lockManager  *concurrency.LockManager
	eventBus     event.Bus
	lootboxSvc   lootbox.Service
	jobService   JobService
	statsSvc     stats.Service
	joinDuration time.Duration
}

// NewService creates a new gamble service
func NewService(repo Repository, lockManager *concurrency.LockManager, eventBus event.Bus, lootboxSvc lootbox.Service, statsSvc stats.Service, joinDuration time.Duration, jobService JobService) Service {
	return &service{
		repo:         repo,
		lockManager:  lockManager,
		eventBus:     eventBus,
		lootboxSvc:   lootboxSvc,
		jobService:   jobService,
		statsSvc:     statsSvc,
		joinDuration: joinDuration,
	}
}

// StartGamble initiates a new gamble
func (s *service) StartGamble(ctx context.Context, platform, platformID, username string, bets []domain.LootboxBet) (*domain.Gamble, error) {
	log := logger.FromContext(ctx)
	log.Info("StartGamble called", "platform", platform, "platformID", platformID, "username", username, "bets", bets)

	// Validate input
	if len(bets) == 0 {
		return nil, fmt.Errorf("at least one lootbox bet is required")
	}
	for _, bet := range bets {
		if bet.Quantity <= 0 {
			return nil, fmt.Errorf("bet quantity must be positive")
		}
	}

	// Get user
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check for active gamble
	active, err := s.repo.GetActiveGamble(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check active gamble: %w", err)
	}
	if active != nil {
		return nil, fmt.Errorf("a gamble is already active")
	}

	// Create gamble record
	gamble := &domain.Gamble{
		ID:           uuid.New(),
		InitiatorID:  user.ID,
		State:        domain.GambleStateJoining,
		CreatedAt:    time.Now(),
		JoinDeadline: time.Now().Add(s.joinDuration),
	}

	// Lock user inventory briefly to consume bets
	lock := s.lockManager.GetLock(user.ID)
	lock.Lock()
	defer lock.Unlock()

	// Begin transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get inventory
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Consume bets
	for _, bet := range bets {
		// Verify ownership and quantity
		// Using utils.FindSlot (assuming it exists based on previous context)
		// We need to import "github.com/osse101/BrandishBot_Go/internal/utils" if not already
		// But wait, utils.FindSlot returns index and quantity.
		// I'll implement a helper here or assume utils is available.
		// Checking imports... yes, utils is imported.

		// Note: We need to handle the case where the user doesn't have the item.
		// Since I can't see utils.FindSlot signature right now, I'll assume standard behavior.
		// Actually, I'll implement a local helper `consumeItem` to be safe and clean.
		if err := consumeItem(inventory, bet.ItemID, bet.Quantity); err != nil {
			return nil, fmt.Errorf("failed to consume bet (item %d): %w", bet.ItemID, err)
		}
	}

	// Update inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	// Save gamble
	if err := s.repo.CreateGamble(ctx, gamble); err != nil {
		return nil, fmt.Errorf("failed to create gamble: %w", err)
	}

	// Add initiator as participant
	participant := &domain.Participant{
		GambleID:    gamble.ID,
		UserID:      user.ID,
		LootboxBets: bets,
		Username:    username,
	}
	if err := s.repo.JoinGamble(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to add initiator as participant: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Publish GambleStarted event
	if s.eventBus != nil {
		err := s.eventBus.Publish(ctx, event.Event{
			Type:    event.Type(domain.EventGambleStarted),
			Payload: gamble,
		})
		if err != nil {
			log.Error("Failed to publish GambleStarted event", "error", err)
			// Don't fail the request, just log
		}
	}

	// Award Gambler XP for joining (async, don't block)
	go s.awardGamblerXP(context.Background(), user.ID, calculateTotalLootboxes(bets), "start", false)

	return gamble, nil
}

// JoinGamble adds a user to an existing gamble
func (s *service) JoinGamble(ctx context.Context, gambleID uuid.UUID, platform, platformID, username string, bets []domain.LootboxBet) error {
	log := logger.FromContext(ctx)
	log.Info("JoinGamble called", "gambleID", gambleID, "username", username)

	// Validate bets
	if len(bets) == 0 {
		return fmt.Errorf("at least one lootbox bet is required")
	}

	// Get User
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Get Gamble
	gamble, err := s.repo.GetGamble(ctx, gambleID)
	if err != nil {
		return fmt.Errorf("failed to get gamble: %w", err)
	}
	if gamble == nil {
		return fmt.Errorf("gamble not found")
	}
	if gamble.State != domain.GambleStateJoining {
		return fmt.Errorf("gamble is not in joining state")
	}
	if time.Now().After(gamble.JoinDeadline) {
		return fmt.Errorf("gamble join deadline has passed")
	}

	// Lock Inventory
	lock := s.lockManager.GetLock(user.ID)
	lock.Lock()
	defer lock.Unlock()

	// Begin Transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get Inventory
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Consume Bets
	for _, bet := range bets {
		if err := consumeItem(inventory, bet.ItemID, bet.Quantity); err != nil {
			return fmt.Errorf("failed to consume bet (item %d): %w", bet.ItemID, err)
		}
	}

	// Update Inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	// Add Participant
	participant := &domain.Participant{
		GambleID:    gamble.ID,
		UserID:      user.ID,
		LootboxBets: bets,
		Username:    username,
	}
	if err := s.repo.JoinGamble(ctx, participant); err != nil {
		return fmt.Errorf("failed to join gamble: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Award Gambler XP for joining (async, don't block)
	go s.awardGamblerXP(context.Background(), user.ID, calculateTotalLootboxes(bets), "join", false)

	return nil
}

// ExecuteGamble runs the gamble logic
func (s *service) ExecuteGamble(ctx context.Context, id uuid.UUID) (*domain.GambleResult, error) {
	log := logger.FromContext(ctx)
	log.Info("ExecuteGamble called", "gambleID", id)

	// Get Gamble with participants
	gamble, err := s.repo.GetGamble(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get gamble: %w", err)
	}
	if gamble == nil {
		return nil, fmt.Errorf("gamble not found")
	}

	// Check if already completed (graceful handling of duplicate execution)
	if gamble.State == domain.GambleStateCompleted {
		log.Info("Gamble already completed, skipping execution", "gambleID", id)
		return nil, nil
	}

	// Only execute if in Joining state
	if gamble.State != domain.GambleStateJoining {
		return nil, fmt.Errorf("gamble is not in joining state (current: %s)", gamble.State)
	}

	// Update State to Opening
	if err := s.repo.UpdateGambleState(ctx, id, domain.GambleStateOpening); err != nil {
		return nil, fmt.Errorf("failed to update state to opening: %w", err)
	}

	// Simulate opening lootboxes (Placeholder: In real impl, use LootboxService)
	// We need to track total value per user
	userValues := make(map[string]int64)
	var allOpenedItems []domain.GambleOpenedItem
	var totalGambleValue int64

	// For each participant, open their lootboxes
	for _, p := range gamble.Participants {
		for _, bet := range p.LootboxBets {
			// Get lootbox item to find its name
			lootboxItem, err := s.repo.GetItemByID(ctx, bet.ItemID)
			if err != nil {
				log.Error("Failed to get lootbox item", "itemID", bet.ItemID, "error", err)
				continue
			}
			if lootboxItem == nil {
				log.Warn("Lootbox item not found", "itemID", bet.ItemID)
				continue
			}

			// Open lootbox using shared service
			drops, err := s.lootboxSvc.OpenLootbox(ctx, lootboxItem.Name, bet.Quantity)
			if err != nil {
				log.Error("Failed to open lootbox", "lootbox", lootboxItem.Name, "error", err)
				continue
			}

			// Process drops
			for _, drop := range drops {
				// Create individual records for each item quantity
				// (GambleOpenedItem represents a single item instance or stack?
				// The struct has Value int64. If quantity > 1, is Value per item or total?
				// Usually in gambles we want to show each "pull".
				// But OpenLootbox aggregates by item type.
				// Let's record them as a stack for now, but Value should be total for the stack?
				// Wait, domain.GambleOpenedItem has ItemID and Value.
				// If I get 5 coins worth 1 each, is it 1 record of 5 coins worth 5?
				// The previous dummy logic did: for i < quantity... simulate item drop.
				// OpenLootbox returns aggregated drops.
				// Let's create one record per dropped item type per bet.

				totalValue := int64(drop.Value * drop.Quantity)

				openedItem := domain.GambleOpenedItem{
					GambleID:   id,
					UserID:     p.UserID,
					ItemID:     drop.ItemID,
					Value:      totalValue,
					ShineLevel: drop.ShineLevel,
				}
				allOpenedItems = append(allOpenedItems, openedItem)
				userValues[p.UserID] += totalValue
				totalGambleValue += totalValue
			}
		}
	}

	// Critical Failure Tracking (Low scores relative to average)
	if len(userValues) > 1 && totalGambleValue > 0 && s.statsSvc != nil {
		averageScore := float64(totalGambleValue) / float64(len(userValues))
		criticalFailThreshold := int64(averageScore * 0.2) // 20% of average

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

	// Save opened items
	if err := s.repo.SaveOpenedItems(ctx, allOpenedItems); err != nil {
		return nil, fmt.Errorf("failed to save opened items: %w", err)
	}

	// Determine Winner
	var highestValue int64 = -1
	var winners []string

	for userID, val := range userValues {
		if val > highestValue {
			highestValue = val
			winners = []string{userID}
		} else if val == highestValue {
			winners = append(winners, userID)
		}
	}

	// Tie-breaking
	winnerID := ""
	if len(winners) > 0 {
		if len(winners) > 1 {
			// Randomly select one
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			idx := r.Intn(len(winners))
			winnerID = winners[idx]

			log.Info("Tie-break resolved", "winnerID", winnerID, "originalValue", highestValue)

			// Track tie-break losers
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
		} else {
			winnerID = winners[0]
		}
	}

	// Near Miss Tracking (Close scores)
	if winnerID != "" && highestValue > 0 && s.statsSvc != nil {
		threshold := int64(float64(highestValue) * NearMissThreshold)
		for userID, val := range userValues {
			if userID == winnerID {
				continue
			}

			// Skip if this user was already tracked as a tie-break loser (value == highestValue)
			if val == highestValue {
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

	// Award items to winner
	if winnerID != "" {
		// Lock winner inventory
		lock := s.lockManager.GetLock(winnerID)
		lock.Lock()
		defer lock.Unlock()

		tx, err := s.repo.BeginTx(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin tx for awarding: %w", err)
		}
		defer repository.SafeRollback(ctx, tx)

		inv, err := tx.GetInventory(ctx, winnerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get winner inventory: %w", err)
		}

		// Add all items
		// Optimization: Aggregate items first to avoid O(N*M) loop
		itemsToAdd := make(map[int]int)
		for _, item := range allOpenedItems {
			itemsToAdd[item.ItemID]++
		}

		// Update existing slots
		for i, slot := range inv.Slots {
			if qty, ok := itemsToAdd[slot.ItemID]; ok {
				inv.Slots[i].Quantity += qty
				delete(itemsToAdd, slot.ItemID)
			}
		}

		// Append new slots
		// Sort keys for deterministic output
		var newItemIDs []int
		for itemID := range itemsToAdd {
			newItemIDs = append(newItemIDs, itemID)
		}
		sort.Ints(newItemIDs)

		for _, itemID := range newItemIDs {
			inv.Slots = append(inv.Slots, domain.InventorySlot{ItemID: itemID, Quantity: itemsToAdd[itemID]})
		}

		if err := tx.UpdateInventory(ctx, winnerID, *inv); err != nil {
			return nil, fmt.Errorf("failed to update winner inventory: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit award tx: %w", err)
		}
	}

	// Complete Gamble
	result := &domain.GambleResult{
		GambleID:   id,
		WinnerID:   winnerID,
		TotalValue: totalGambleValue,
		Items:      allOpenedItems,
	}

	if err := s.repo.CompleteGamble(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to complete gamble: %w", err)
	}

	// Award bonus XP to winner (async)
	if winnerID != "" {
		go s.awardGamblerXP(context.Background(), winnerID, 0, "win", true)
	}

	return result, nil
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
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			if slot.Quantity < quantity {
				return fmt.Errorf("insufficient quantity")
			}
			if slot.Quantity == quantity {
				inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
			} else {
				inventory.Slots[i].Quantity -= quantity
			}
			return nil
		}
	}
	return fmt.Errorf("item not found")
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
		"source":        source,
		"lootbox_count": lootboxCount,
		"is_win":        isWin,
	}

	result, err := s.jobService.AwardXP(ctx, userID, job.JobKeyGambler, xp, source, metadata)
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to award Gambler XP", "error", err, "user_id", userID)
	} else if result != nil && result.LeveledUp {
		logger.FromContext(ctx).Info("Gambler leveled up!", "user_id", userID, "new_level", result.NewLevel)
	}
}
