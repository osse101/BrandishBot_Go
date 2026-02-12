package harvest

import (
	"context"
	"errors"
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

const (
	minHarvestInterval = 1.0   // Minimum 1 hour between harvests
	farmerXPThreshold  = 5.0   // Minimum 5 hours for Farmer XP
	farmerXPPerHour    = 8     // Base XP per hour of waiting
	spoiledThreshold   = 336.0 // 168h (max tier) + 168h (1 week)
)

// Service defines the harvest system business logic
type Service interface {
	// Harvest collects accumulated rewards for a user
	Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResponse, error)
	// Shutdown gracefully shuts down the service
	Shutdown(ctx context.Context) error
}

type service struct {
	harvestRepo    repository.HarvestRepository
	userRepo       repository.User
	progressionSvc progression.Service
	publisher      *event.ResilientPublisher
	wg             sync.WaitGroup
}

// NewService creates a new harvest service
func NewService(
	harvestRepo repository.HarvestRepository,
	userRepo repository.User,
	progressionSvc progression.Service,
	publisher *event.ResilientPublisher,
) Service {
	return &service{
		harvestRepo:    harvestRepo,
		userRepo:       userRepo,
		progressionSvc: progressionSvc,
		publisher:      publisher,
	}
}

// Harvest collects accumulated rewards for a user
func (s *service) Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("Harvest called", "platform", platform, "platformID", platformID, "username", username)

	// 1. Get or register user
	user, err := s.ensureUser(ctx, platform, platformID, username)
	if err != nil {
		return nil, err
	}

	// 2. Check if harvest feature (farming) is unlocked
	if err := s.checkFarmingUnlocked(ctx); err != nil {
		return nil, err
	}

	// 3. Initialize harvest state if first time
	if initialResp, err := s.initializeHarvestStateIfNeeded(ctx, user.ID); err != nil || initialResp != nil {
		return initialResp, err
	}

	// 4. Begin transaction
	tx, err := s.harvestRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// 5. Get harvest state with lock (FOR UPDATE)
	harvestState, err := tx.GetHarvestStateWithLock(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get harvest state with lock: %w", err)
	}

	// 6. Calculate elapsed hours
	now := time.Now()
	hoursElapsed := now.Sub(harvestState.LastHarvestedAt).Hours()

	// 7. Validate minimum time (1 hour)
	if hoursElapsed < minHarvestInterval {
		nextHarvest := harvestState.LastHarvestedAt.Add(time.Hour)
		return nil, fmt.Errorf("%w: next harvest available at %s", domain.ErrHarvestTooSoon, nextHarvest.Format(time.RFC3339))
	}

	// 8. Calculate rewards (handle spoiled)
	rewards, message := s.calculateHarvestRewards(ctx, hoursElapsed)

	// 9. Award Farmer XP (Async)
	// Must be done before potential early return in handleEmptyHarvest
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// Use WithoutCancel to preserve logger/values but detach cancellation
		asyncCtx := context.WithoutCancel(ctx)
		s.awardFarmerXP(asyncCtx, user.ID, hoursElapsed)
	}()

	// 10. Update inventory and harvest state
	if len(rewards) == 0 {
		return s.handleEmptyHarvest(ctx, tx, user.ID, now, hoursElapsed)
	}

	if err := s.applyHarvestRewards(ctx, tx, user.ID, rewards); err != nil {
		return nil, err
	}

	if err := tx.UpdateHarvestState(ctx, user.ID, now); err != nil {
		return nil, fmt.Errorf("failed to update harvest state: %w", err)
	}

	// 11. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Harvest successful", "userID", user.ID, "rewards", rewards, "hours", hoursElapsed)

	return &domain.HarvestResponse{
		ItemsGained:       rewards,
		HoursSinceHarvest: hoursElapsed,
		NextHarvestAt:     now.Add(time.Hour),
		Message:           message,
	}, nil
}

func (s *service) ensureUser(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	user, err := s.userRepo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			newUser := &domain.User{Username: username}
			switch platform {
			case "discord":
				newUser.DiscordID = platformID
			case "twitch":
				newUser.TwitchID = platformID
			case "youtube":
				newUser.YoutubeID = platformID
			}
			if err := s.userRepo.UpsertUser(ctx, newUser); err != nil {
				return nil, fmt.Errorf("failed to register user: %w", err)
			}
			return s.userRepo.GetUserByPlatformID(ctx, platform, platformID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *service) checkFarmingUnlocked(ctx context.Context) error {
	unlocked, err := s.progressionSvc.IsFeatureUnlocked(ctx, "feature_farming")
	if err != nil {
		return fmt.Errorf("failed to check farming feature unlock: %w", err)
	}
	if !unlocked {
		return fmt.Errorf("harvest requires farming feature to be unlocked: %w", domain.ErrFeatureLocked)
	}
	return nil
}

func (s *service) initializeHarvestStateIfNeeded(ctx context.Context, userID string) (*domain.HarvestResponse, error) {
	_, err := s.harvestRepo.GetHarvestState(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrHarvestStateNotFound) {
			if _, err = s.harvestRepo.CreateHarvestState(ctx, userID); err != nil {
				return nil, fmt.Errorf("failed to create harvest state: %w", err)
			}
			return &domain.HarvestResponse{
				ItemsGained:       map[string]int{},
				HoursSinceHarvest: 0,
				NextHarvestAt:     time.Now().Add(time.Hour),
				Message:           "Harvest tracking initialized! Come back in 1 hour for your first harvest.",
			}, nil
		}
		return nil, fmt.Errorf("failed to get harvest state: %w", err)
	}
	return nil, nil
}

func (s *service) calculateHarvestRewards(ctx context.Context, hoursElapsed float64) (map[string]int, string) {
	if hoursElapsed > spoiledThreshold {
		logger.FromContext(ctx).Info("Harvest spoiled", "hours", hoursElapsed)
		return map[string]int{
			"lootbox1": 1,
			"stick":    3,
		}, "Your crops spoiled! You salvaged 1 Decent Lootbox and 3 Sticks."
	}
	return s.calculateRewards(ctx, hoursElapsed), "Harvest successful!"
}

func (s *service) awardFarmerXP(ctx context.Context, userID string, hoursElapsed float64) {
	if hoursElapsed < farmerXPThreshold {
		return
	}

	xpAmount := int(hoursElapsed * farmerXPPerHour)
	spoiled := hoursElapsed > spoiledThreshold

	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeHarvestCompleted),
			Payload: domain.HarvestCompletedPayload{
				UserID:       userID,
				HoursElapsed: hoursElapsed,
				XPAmount:     xpAmount,
				Spoiled:      spoiled,
				Timestamp:    time.Now().Unix(),
			},
		})
	}
}

// Shutdown gracefully shuts down the service
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

func (s *service) handleEmptyHarvest(ctx context.Context, tx repository.HarvestTx, userID string, now time.Time, hoursElapsed float64) (*domain.HarvestResponse, error) {
	logger.FromContext(ctx).Warn("No rewards available - all items locked by progression")

	if err := tx.UpdateHarvestState(ctx, userID, now); err != nil {
		return nil, fmt.Errorf("failed to update harvest state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &domain.HarvestResponse{
		ItemsGained:       map[string]int{},
		HoursSinceHarvest: hoursElapsed,
		NextHarvestAt:     now.Add(time.Hour),
		Message:           "No rewards available - unlock progression nodes to receive harvest items!",
	}, nil
}

func (s *service) applyHarvestRewards(ctx context.Context, tx repository.HarvestTx, userID string, rewards map[string]int) error {
	log := logger.FromContext(ctx)

	// 1. Get inventory
	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// 2. Get item IDs
	itemNames := make([]string, 0, len(rewards))
	for name := range rewards {
		itemNames = append(itemNames, name)
	}

	items, err := s.userRepo.GetItemsByNames(ctx, itemNames)
	if err != nil {
		return fmt.Errorf("failed to get item details: %w", err)
	}

	itemNameToID := make(map[string]int)
	for _, item := range items {
		itemNameToID[item.InternalName] = item.ID
	}

	// 3. Update inventory structure
	for name, qty := range rewards {
		if qty <= 0 {
			continue
		}
		id, ok := itemNameToID[name]
		if !ok {
			log.Warn("Item not found in database, skipping", "itemName", name)
			continue
		}

		slotIndex, _ := utils.FindSlot(inventory, id)
		if slotIndex != -1 {
			inventory.Slots[slotIndex].Quantity += qty
		} else {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{
				ItemID:       id,
				Quantity:     qty,
				QualityLevel: domain.QualityCommon,
			})
		}
	}

	// 4. Update database
	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	return nil
}

// calculateRewards calculates the total rewards for a given elapsed time
// Accumulates ALL items from all tiers up to and including the current tier
func (s *service) calculateRewards(ctx context.Context, hoursElapsed float64) map[string]int {
	log := logger.FromContext(ctx)
	rewards := make(map[string]int)
	tiers := getRewardTiers()

	// Find the applicable tier (highest tier where hoursElapsed >= MaxHours)
	maxTierIndex := -1
	for i := range tiers {
		if hoursElapsed >= tiers[i].MaxHours {
			maxTierIndex = i
		} else {
			break // Tiers are ordered, so we can stop here
		}
	}

	// No tier reached
	if maxTierIndex < 0 {
		log.Info("No tier reached", "hoursElapsed", hoursElapsed)
		return rewards
	}

	log.Info("Calculating rewards", "hoursElapsed", hoursElapsed, "maxTier", maxTierIndex)

	// ACCUMULATE ALL ITEMS from all tiers up to and including current tier
	for i := 0; i <= maxTierIndex; i++ {
		tier := &tiers[i]

		for itemName, quantity := range tier.Items {
			// Check progression unlock for gated items
			if tier.RequiresUnlock[itemName] {
				unlocked, err := s.progressionSvc.IsItemUnlocked(ctx, itemName)
				if err != nil {
					log.Warn("Failed to check item unlock status", "item", itemName, "error", err)
					continue // Skip on error
				}
				if !unlocked {
					log.Info("Item locked by progression, skipping", "item", itemName, "tier", i)
					continue // Skip locked items
				}
			}

			// SUM all items (accumulate)
			rewards[itemName] += quantity
		}
	}

	log.Info("Rewards calculated", "rewards", rewards, "tierCount", maxTierIndex+1)

	return rewards
}
