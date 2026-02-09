package slots

import (
	"context"
	"fmt"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// CooldownService defines the interface for cooldown operations
type CooldownService interface {
	EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error
}

// StatsService defines the interface for stats operations
type StatsService interface {
	RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error
}

// Service defines the interface for slots operations
type Service interface {
	SpinSlots(ctx context.Context, platform, platformID, username string, betAmount int) (*domain.SlotsResult, error)
	Shutdown(ctx context.Context) error
}

type service struct {
	userRepo           repository.User
	jobService         job.Service
	progressionService progression.Service
	cooldownSvc        CooldownService
	statsService       StatsService
	eventBus           event.Bus
	resilientPublisher *event.ResilientPublisher
	namingResolver     naming.Resolver
	rng                func(int) int // Injectable for testing
	wg                 sync.WaitGroup
	shutdown           chan struct{}
}

// NewService creates a new slots service
func NewService(
	userRepo repository.User,
	jobService job.Service,
	progressionService progression.Service,
	cooldownSvc CooldownService,
	statsService StatsService,
	eventBus event.Bus,
	resilientPublisher *event.ResilientPublisher,
	namingResolver naming.Resolver,
) Service {
	return &service{
		userRepo:           userRepo,
		jobService:         jobService,
		progressionService: progressionService,
		cooldownSvc:        cooldownSvc,
		statsService:       statsService,
		eventBus:           eventBus,
		resilientPublisher: resilientPublisher,
		namingResolver:     namingResolver,
		rng:                utils.SecureRandomInt,
		shutdown:           make(chan struct{}),
	}
}

// SpinSlots processes a slots spin with the given bet amount
func (s *service) SpinSlots(ctx context.Context, platform, platformID, username string, betAmount int) (*domain.SlotsResult, error) {
	log := logger.FromContext(ctx)

	// Validate bet amount
	if betAmount < MinBetAmount {
		return nil, fmt.Errorf("minimum bet is %d money", MinBetAmount)
	}
	if betAmount > MaxBetAmount {
		return nil, fmt.Errorf("maximum bet is %d money", MaxBetAmount)
	}

	// Get user
	user, err := s.userRepo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check feature unlock
	isUnlocked, err := s.progressionService.IsFeatureUnlocked(ctx, progression.FeatureSlots)
	if err != nil {
		log.Warn("Failed to check feature lock", "error", err)
	}
	if !isUnlocked {
		return nil, fmt.Errorf("slots feature is not yet unlocked")
	}

	// Store result outside the callback
	var result *domain.SlotsResult

	// Enforce cooldown - wrap the actual spin logic
	err = s.cooldownSvc.EnforceCooldown(ctx, user.ID, domain.ActionSlots, func() error {
		var spinErr error
		result, spinErr = s.executeSpin(ctx, user, username, betAmount)
		return spinErr
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// executeSpin performs the actual spin logic (called within cooldown enforcement)
func (s *service) executeSpin(ctx context.Context, user *domain.User, username string, betAmount int) (*domain.SlotsResult, error) {
	// Begin transaction
	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get money item
	moneyItem, err := s.userRepo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		return nil, fmt.Errorf("failed to get money item: %w", err)
	}
	if moneyItem == nil {
		return nil, fmt.Errorf("money item not found")
	}

	// Get inventory
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Find money slot and verify balance using FindRandomSlot
	moneySlotIndex, currentMoney := utils.FindRandomSlot(inventory, moneyItem.ID, func() float64 {
		return float64(s.rng(1000)) / 1000.0
	})

	if currentMoney < betAmount {
		return nil, fmt.Errorf("insufficient funds. You have %d money", currentMoney)
	}

	// Spin reels
	reel1, reel2, reel3 := s.spinReels()

	// Calculate payout
	payoutAmount, payoutMultiplier, triggerType := s.calculatePayout(reel1, reel2, reel3, betAmount)

	// Update inventory (deduct bet, add winnings)
	netChange := payoutAmount - betAmount
	newBalance := currentMoney + netChange

	if newBalance < 0 {
		return nil, fmt.Errorf("transaction would result in negative balance")
	}

	// Update money quantity in inventory
	if moneySlotIndex != -1 {
		if newBalance == 0 {
			// Remove slot if balance is zero
			inventory.Slots = append(inventory.Slots[:moneySlotIndex], inventory.Slots[moneySlotIndex+1:]...)
		} else {
			inventory.Slots[moneySlotIndex].Quantity = newBalance
		}
	}

	// Save inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Build result
	isWin := (reel1 == reel2 && reel2 == reel3)
	isNearMiss := false

	result := &domain.SlotsResult{
		UserID:           user.ID,
		Username:         username,
		Reel1:            reel1,
		Reel2:            reel2,
		Reel3:            reel3,
		BetAmount:        betAmount,
		PayoutAmount:     payoutAmount,
		PayoutMultiplier: payoutMultiplier,
		IsWin:            isWin,
		IsNearMiss:       isNearMiss,
		TriggerType:      triggerType,
		Message:          s.formatMessage(reel1, reel2, reel3, betAmount, payoutAmount, triggerType),
	}

	// Record engagement (async, non-blocking)
	s.wg.Add(1)
	go s.recordAllEngagement(ctx, user.ID, result)

	// Record stats (async, non-blocking)
	s.wg.Add(1)
	go s.recordSlotsStats(ctx, user.ID, result)

	// Award Gambler XP (async, non-blocking)
	s.wg.Add(1)
	go s.awardGamblerXP(ctx, user.ID, betAmount, payoutAmount, triggerType)

	// Publish event (async with retry)
	payload := domain.SlotsCompletedPayload{
		UserID:           user.ID,
		Username:         username,
		BetAmount:        betAmount,
		Reel1:            reel1,
		Reel2:            reel2,
		Reel3:            reel3,
		PayoutAmount:     payoutAmount,
		PayoutMultiplier: payoutMultiplier,
		TriggerType:      triggerType,
		IsWin:            isWin,
		IsNearMiss:       isNearMiss,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventSlotsCompleted),
		Payload: payload,
	}
	s.resilientPublisher.PublishWithRetry(ctx, evt)

	return result, nil
}

// spinReels generates three random symbols using weighted distribution
func (s *service) spinReels() (string, string, string) {
	return s.selectWeightedSymbol(), s.selectWeightedSymbol(), s.selectWeightedSymbol()
}

// selectWeightedSymbol performs weighted random selection of a symbol
func (s *service) selectWeightedSymbol() string {
	totalWeight := 1000 // Sum of all weights

	roll := s.rng(totalWeight)

	cumulative := 0
	for _, symbol := range []string{SymbolLemon, SymbolCherry, SymbolBell, SymbolBar, SymbolSeven, SymbolDiamond, SymbolStar} {
		cumulative += SymbolWeights[symbol]
		if roll < cumulative {
			return symbol
		}
	}

	// Fallback (should never happen)
	return SymbolLemon
}

// calculatePayout determines the payout amount, multiplier, and trigger type
func (s *service) calculatePayout(reel1, reel2, reel3 string, betAmount int) (payoutAmount int, multiplier float64, triggerType string) {
	// Check for 3 matching symbols
	if reel1 == reel2 && reel2 == reel3 {
		multiplier = PayoutMultipliers[reel1]
		payoutAmount = int(float64(betAmount) * multiplier)
		triggerType = s.determineWinType(multiplier)
		return
	}

	// Check for 2 matching symbols (consolation prize)
	if reel1 == reel2 || reel2 == reel3 || reel1 == reel3 {
		multiplier = TwoMatchMultiplier
		payoutAmount = int(float64(betAmount) * multiplier)
		triggerType = TriggerNormal
		return
	}

	// No match - total loss
	return 0, 0.0, TriggerNormal
}

// determineWinType classifies the win based on multiplier
func (s *service) determineWinType(multiplier float64) string {
	switch {
	case multiplier >= 100.0:
		return TriggerMegaJackpot
	case multiplier >= JackpotThreshold:
		return TriggerJackpot
	case multiplier >= BigWinThreshold:
		return TriggerBigWin
	default:
		return TriggerNormal
	}
}

// formatMessage creates a user-facing message for the result
func (s *service) formatMessage(reel1, reel2, reel3 string, betAmount, payoutAmount int, triggerType string) string {
	if payoutAmount == 0 {
		return fmt.Sprintf("Better luck next time! You lost %d money.", betAmount)
	}

	netWin := payoutAmount - betAmount

	switch triggerType {
	case TriggerMegaJackpot:
		return fmt.Sprintf("🌟 MEGA JACKPOT! 🌟 You won %d money (net +%d)!", payoutAmount, netWin)
	case TriggerJackpot:
		return fmt.Sprintf("💎 JACKPOT! 💎 You won %d money (net +%d)!", payoutAmount, netWin)
	case TriggerBigWin:
		return fmt.Sprintf("🎉 BIG WIN! You won %d money (net +%d)!", payoutAmount, netWin)
	default:
		if netWin > 0 {
			return fmt.Sprintf("You won %d money (net +%d)!", payoutAmount, netWin)
		}
		if netWin == 0 {
			return fmt.Sprintf("You broke even! %d money returned.", payoutAmount)
		}
		if (reel1 == reel2 || reel2 == reel3 || reel1 == reel3) && (reel1 != reel2 || reel2 != reel3) {
			// Consolation prize (2 symbols match, but not 3)
			return fmt.Sprintf("Consolation! You got %d back. (net %d)", payoutAmount, netWin)
		}
		return fmt.Sprintf("No luck! You won %d money (net %d).", payoutAmount, netWin)
	}
}

// recordAllEngagement tracks all relevant engagement metrics
func (s *service) recordAllEngagement(ctx context.Context, userID string, result *domain.SlotsResult) {
	defer s.wg.Done()

	log := logger.FromContext(ctx)

	// Always track spin
	if err := s.progressionService.RecordEngagement(ctx, userID, MetricSlotsSpin, 1); err != nil {
		log.Warn("Failed to record slots spin engagement", "error", err)
	}

	// Track outcome-specific engagement
	if result.IsWin {
		if err := s.progressionService.RecordEngagement(ctx, userID, MetricSlotsWin, 1); err != nil {
			log.Warn("Failed to record slots win engagement", "error", err)
		}
	}

	if result.PayoutMultiplier >= BigWinThreshold {
		if err := s.progressionService.RecordEngagement(ctx, userID, MetricSlotsBigWin, 1); err != nil {
			log.Warn("Failed to record slots big win engagement", "error", err)
		}
	}

	if result.PayoutMultiplier >= JackpotThreshold {
		if err := s.progressionService.RecordEngagement(ctx, userID, MetricSlotsJackpot, 1); err != nil {
			log.Warn("Failed to record slots jackpot engagement", "error", err)
		}
	}
}

// recordSlotsStats records statistics for slots spins
func (s *service) recordSlotsStats(ctx context.Context, userID string, result *domain.SlotsResult) {
	defer s.wg.Done()

	if s.statsService == nil {
		return
	}

	log := logger.FromContext(ctx)

	// Record every spin
	metadata := map[string]interface{}{
		"bet_amount":        result.BetAmount,
		"payout_amount":     result.PayoutAmount,
		"payout_multiplier": result.PayoutMultiplier,
		"net_profit":        result.PayoutAmount - result.BetAmount,
		"is_win":            result.IsWin,
		"is_near_miss":      result.IsNearMiss,
		"trigger_type":      result.TriggerType,
		"reel1":             result.Reel1,
		"reel2":             result.Reel2,
		"reel3":             result.Reel3,
	}

	if err := s.statsService.RecordUserEvent(ctx, userID, domain.EventSlotsSpin, metadata); err != nil {
		log.Warn("Failed to record slots spin stats", "error", err)
	}

	// Record wins separately for easier querying
	if result.IsWin {
		if err := s.statsService.RecordUserEvent(ctx, userID, domain.EventSlotsWin, metadata); err != nil {
			log.Warn("Failed to record slots win stats", "error", err)
		}
	}

	// Record mega jackpots
	if result.TriggerType == TriggerMegaJackpot {
		if err := s.statsService.RecordUserEvent(ctx, userID, domain.EventSlotsMegaJackpot, metadata); err != nil {
			log.Warn("Failed to record slots mega jackpot stats", "error", err)
		}
	}
}

// awardGamblerXP awards XP to the Gambler job based on spin results
func (s *service) awardGamblerXP(ctx context.Context, userID string, betAmount, payoutAmount int, triggerType string) {
	defer s.wg.Done()

	if s.jobService == nil {
		return
	}

	log := logger.FromContext(ctx)

	// Base XP: bet amount / 10
	xp := betAmount / 10

	// Bonuses
	if payoutAmount > betAmount {
		xp += 20 // Win bonus
	}
	if triggerType == TriggerJackpot || triggerType == TriggerMegaJackpot {
		xp += 100
	}

	metadata := map[string]interface{}{
		"source":        "slots",
		"bet_amount":    betAmount,
		"payout_amount": payoutAmount,
		"trigger_type":  triggerType,
	}

	result, err := s.jobService.AwardXP(ctx, userID, job.JobKeyGambler, xp, "slots", metadata)
	if err != nil {
		log.Warn("Failed to award Gambler XP", "error", err)
	} else if result != nil && result.LeveledUp {
		log.Info("Gambler leveled up", "user_id", userID, "new_level", result.NewLevel)
	}
}

// Shutdown gracefully stops the service
func (s *service) Shutdown(ctx context.Context) error {
	close(s.shutdown)

	// Wait for all async operations to complete
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
