package harvest

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Harvest collects accumulated rewards for a user
func (s *service) Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("Harvest called", "platform", platform, "platformID", platformID, "username", username)

	user, err := s.ensureUser(ctx, platform, platformID, username)
	if err != nil {
		return nil, err
	}

	if err := s.checkFarmingUnlocked(ctx); err != nil {
		return nil, err
	}

	if initialResp, err := s.initializeHarvestStateIfNeeded(ctx, user.ID); err != nil || initialResp != nil {
		return initialResp, err
	}

	return s.performHarvestTransaction(ctx, user)
}

func (s *service) performHarvestTransaction(ctx context.Context, user *domain.User) (*domain.HarvestResponse, error) {
	tx, err := s.harvestRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	harvestState, err := tx.GetHarvestStateWithLock(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get harvest state with lock: %w", err)
	}

	now := time.Now()
	hoursElapsed := now.Sub(harvestState.LastHarvestedAt).Hours()

	if hoursElapsed < minHarvestInterval {
		nextHarvest := harvestState.LastHarvestedAt.Add(time.Hour)
		return nil, fmt.Errorf("%w: next harvest available at %s", domain.ErrHarvestTooSoon, nextHarvest.Format(time.RFC3339))
	}

	yieldMultiplier, growthMultiplier := s.calculateBonuses(ctx, user.ID)
	effectiveHours := hoursElapsed * growthMultiplier

	rewards, message := s.calculateHarvestRewards(ctx, effectiveHours, yieldMultiplier)

	s.fireAsyncEvents(ctx, user.ID, hoursElapsed)

	if len(rewards) == 0 {
		return s.handleEmptyHarvest(ctx, tx, user.ID, now, hoursElapsed)
	}

	if err := s.applyHarvestRewards(ctx, tx, user.ID, rewards); err != nil {
		return nil, err
	}

	if err := tx.UpdateHarvestState(ctx, user.ID, now); err != nil {
		return nil, fmt.Errorf("failed to update harvest state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.FromContext(ctx).Info("Harvest successful", "userID", user.ID, "rewards", rewards, "hours", hoursElapsed)

	return &domain.HarvestResponse{
		ItemsGained:       rewards,
		HoursSinceHarvest: hoursElapsed,
		NextHarvestAt:     now.Add(time.Hour),
		Message:           message,
	}, nil
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
