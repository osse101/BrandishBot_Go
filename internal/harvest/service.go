package harvest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

const (
	minHarvestInterval = 1.0 // Minimum 1 hour between harvests
)

// Service defines the harvest system business logic
type Service interface {
	// Harvest collects accumulated rewards for a user
	Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResponse, error)
}

type service struct {
	harvestRepo    repository.HarvestRepository
	userRepo       repository.User
	progressionSvc progression.Service
}

// NewService creates a new harvest service
func NewService(
	harvestRepo repository.HarvestRepository,
	userRepo repository.User,
	progressionSvc progression.Service,
) Service {
	return &service{
		harvestRepo:    harvestRepo,
		userRepo:       userRepo,
		progressionSvc: progressionSvc,
	}
}

// Harvest collects accumulated rewards for a user
func (s *service) Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("Harvest called", "platform", platform, "platformID", platformID, "username", username)

	// 1. Get or register user
	user, err := s.userRepo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Register new user
			newUser := &domain.User{
				Username: username,
			}
			// Set platform-specific ID
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
			user, err = s.userRepo.GetUserByPlatformID(ctx, platform, platformID)
			if err != nil {
				return nil, fmt.Errorf("failed to get newly registered user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	}

	// 2. Initialize harvest state if first time
	harvestState, err := s.harvestRepo.GetHarvestState(ctx, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrHarvestStateNotFound) {
			// Create initial harvest state (last_harvested_at = NOW)
			harvestState, err = s.harvestRepo.CreateHarvestState(ctx, user.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to create harvest state: %w", err)
			}

			// First harvest - user just started, no rewards yet
			return &domain.HarvestResponse{
				ItemsGained:       map[string]int{},
				HoursSinceHarvest: 0,
				NextHarvestAt:     time.Now().Add(time.Hour),
				Message:           "Harvest tracking initialized! Come back in 1 hour for your first harvest.",
			}, nil
		}
		return nil, fmt.Errorf("failed to get harvest state: %w", err)
	}

	// 3. Begin transaction
	tx, err := s.harvestRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// 4. Get harvest state with lock (FOR UPDATE)
	harvestState, err = tx.GetHarvestStateWithLock(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get harvest state with lock: %w", err)
	}

	// 5. Calculate elapsed hours
	now := time.Now()
	elapsed := now.Sub(harvestState.LastHarvestedAt)
	hoursElapsed := elapsed.Hours()

	log.Info("Harvest state retrieved", "lastHarvested", harvestState.LastHarvestedAt, "elapsed", elapsed, "hours", hoursElapsed)

	// 6. Validate minimum time (1 hour)
	if hoursElapsed < minHarvestInterval {
		nextHarvest := harvestState.LastHarvestedAt.Add(time.Hour)
		return nil, fmt.Errorf("%w: next harvest available at %s", domain.ErrHarvestTooSoon, nextHarvest.Format(time.RFC3339))
	}

	// 7. Calculate rewards (accumulate across tiers)
	rewards, err := s.calculateRewards(ctx, hoursElapsed)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate rewards: %w", err)
	}

	// If no rewards (all items locked), still update timestamp but warn user
	if len(rewards) == 0 {
		log.Warn("No rewards available - all items locked by progression")

		// Update harvest state timestamp
		if err := tx.UpdateHarvestState(ctx, user.ID, now); err != nil {
			return nil, fmt.Errorf("failed to update harvest state: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		return &domain.HarvestResponse{
			ItemsGained:       rewards,
			HoursSinceHarvest: hoursElapsed,
			NextHarvestAt:     now.Add(time.Hour),
			Message:           "No rewards available - unlock progression nodes to receive harvest items!",
		}, nil
	}

	// 8. Get inventory and add rewards (within transaction)
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Get item names to get their IDs
	itemNames := make([]string, 0, len(rewards))
	for itemName := range rewards {
		itemNames = append(itemNames, itemName)
	}

	// Get item details to convert names to IDs
	items, err := s.userRepo.GetItemsByNames(ctx, itemNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get item details: %w", err)
	}

	// Create map of item name -> item ID
	itemNameToID := make(map[string]int)
	for _, item := range items {
		itemNameToID[item.InternalName] = item.ID
	}

	// Add each reward item to inventory
	for itemName, quantity := range rewards {
		if quantity <= 0 {
			continue
		}

		itemID, ok := itemNameToID[itemName]
		if !ok {
			log.Warn("Item not found in database, skipping", "itemName", itemName)
			continue
		}

		// Find existing slot or create new one
		slotIndex, _ := utils.FindSlot(inventory, itemID)
		if slotIndex != -1 {
			inventory.Slots[slotIndex].Quantity += quantity
		} else {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{
				ItemID:   itemID,
				Quantity: quantity,
			})
		}
	}

	// Update inventory in database
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	// 9. Update harvest state timestamp
	if err := tx.UpdateHarvestState(ctx, user.ID, now); err != nil {
		return nil, fmt.Errorf("failed to update harvest state: %w", err)
	}

	// 10. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Harvest successful", "userID", user.ID, "rewards", rewards, "hours", hoursElapsed)

	return &domain.HarvestResponse{
		ItemsGained:       rewards,
		HoursSinceHarvest: hoursElapsed,
		NextHarvestAt:     now.Add(time.Hour),
		Message:           "Harvest successful!",
	}, nil
}

// calculateRewards calculates the total rewards for a given elapsed time
// Accumulates ALL items from all tiers up to and including the current tier
func (s *service) calculateRewards(ctx context.Context, hoursElapsed float64) (map[string]int, error) {
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
		return rewards, nil
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

	return rewards, nil
}
