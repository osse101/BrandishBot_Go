package user

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// HandleSearch performs a search action for a user with cooldown tracking
func (s *service) HandleSearch(ctx context.Context, platform, platformID, username, itemHint string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("HandleSearch called", "platform", platform, "platformID", platformID, "username", username, "itemHint", itemHint)

	// Get or create user
	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user or register", "error", err)
		return "", err
	}

	// Execute search with atomic cooldown enforcement
	var resultMessage string
	err = s.cooldownService.EnforceCooldown(ctx, user.ID, domain.ActionSearch, func() error {
		var err error
		resultMessage, err = s.executeSearch(ctx, user, itemHint)
		return err
	})

	if err != nil {
		return "", err
	}

	log.Info("Search completed", "username", username, "result", resultMessage)
	return resultMessage, nil
}

type searchParams struct {
	isFirstSearchDaily bool
	isDiminished       bool
	xpMultiplier       float64
	successThreshold   float64
	dailyCount         int
	streak             int
	region             *SearchRegion // resolved search region (nil = default/no regions)
}

// executeSearch performs the actual search logic (called within cooldown enforcement)
func (s *service) executeSearch(ctx context.Context, user *domain.User, itemHint string) (string, error) {
	log := logger.FromContext(ctx)
	params := s.calculateSearchParameters(ctx, user)

	// Resolve search region based on explorer level and optional item hint
	if len(s.searchRegions) > 0 {
		explorerLevel := 0
		if s.jobService != nil {
			if level, err := s.jobService.GetJobLevel(ctx, user.ID, domain.JobKeyExplorer); err == nil {
				explorerLevel = level
			} else {
				log.Warn("Failed to get explorer level for region resolution", "error", err)
			}
		}
		pubIndex := s.buildPublicNameIndex()
		params.region = resolveRegion(s.searchRegions, explorerLevel, itemHint, pubIndex)
		if params.region != nil {
			params.successThreshold += params.region.LootboxChanceModifier
			if params.successThreshold < 0.1 {
				params.successThreshold = 0.1
			}
			log.Debug("Search region resolved", "region", params.region.Name, "modifier", params.region.LootboxChanceModifier, "threshold", params.successThreshold)
		}
	}

	// Perform search roll
	roll := s.rnd()

	var resultMessage string
	isSuccess := roll <= params.successThreshold
	var isCritical, isNearMiss, isCritFail bool
	var itemName string
	var quantity int

	if isSuccess {
		var err error
		resultMessage, err = s.processSearchSuccess(ctx, user, roll, params)
		if err != nil {
			return "", err
		}
		isCritical = roll <= domain.SearchCriticalRate
		quantity = 1
		if isCritical {
			quantity = 2
		}
		itemName = domain.ItemLootbox0
	} else {
		failureType := determineSearchFailureType(roll, params.successThreshold)
		isNearMiss = failureType == searchFailureNearMiss
		isCritFail = failureType == searchFailureCritical
		resultMessage = s.processSearchFailure(roll, params.successThreshold, params)
	}

	xpAmount := int(float64(job.ExplorerXPPerItem) * params.xpMultiplier)
	if xpAmount < 1 {
		xpAmount = 1
	}

	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeSearchPerformed),
			Payload: domain.SearchPerformedPayload{
				UserID:         user.ID,
				Success:        isSuccess,
				IsCritical:     isCritical,
				IsNearMiss:     isNearMiss,
				IsCriticalFail: isCritFail,
				XPAmount:       xpAmount,
				ItemName:       itemName,
				Quantity:       quantity,
				Timestamp:      time.Now().Unix(),
			},
		})
	}

	return resultMessage, nil
}

func (s *service) calculateSearchParameters(ctx context.Context, user *domain.User) searchParams {
	log := logger.FromContext(ctx)
	dailyCount := 0
	if s.statsService != nil {
		stats, err := s.statsService.GetUserStats(ctx, user.ID, domain.PeriodDaily)
		if err != nil {
			log.Warn("Failed to get search counts", "error", err)
		} else if stats != nil && stats.EventCounts != nil {
			dailyCount = stats.EventCounts[domain.StatsEventSearch]
		}
	}

	params := searchParams{
		isFirstSearchDaily: (dailyCount == 0),
		isDiminished:       (dailyCount >= domain.SearchDailyDiminishmentThreshold),
		xpMultiplier:       1.0,
		successThreshold:   domain.SearchSuccessRate,
		dailyCount:         dailyCount,
	}

	if params.isDiminished {
		params.xpMultiplier = domain.SearchDiminishedXPMultiplier
		log.Info(LogMsgDiminishedReturnsApplied, "username", user.Username, "dailyCount", dailyCount)
	}

	if params.isFirstSearchDaily && s.statsService != nil {
		streak, err := s.statsService.GetUserCurrentStreak(ctx, user.ID)
		if err != nil {
			log.Warn("Failed to get user streak", "error", err)
		} else {
			params.streak = streak
		}
	}

	return params
}

func (s *service) processSearchSuccess(ctx context.Context, user *domain.User, roll float64, params searchParams) (string, error) {
	isCritical := roll <= domain.SearchCriticalRate
	quantity := 1
	if isCritical {
		quantity = 2
	}

	// Determine if we grant a region item instead of a lootbox
	if params.region != nil && len(params.region.ItemDrops) > 0 {
		regionRoll := utils.RandomFloat()
		if regionRoll < domain.SearchRegionItemDropChance {
			return s.processRegionItemDrop(ctx, user, isCritical, quantity, params)
		}
	}

	// Default: grant lootbox
	qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
	if err := s.grantSearchReward(ctx, user, quantity, qualityLevel); err != nil {
		return "", err
	}

	// Get item for message formatting
	item, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
	if err != nil {
		return "", fmt.Errorf("failed to get reward item: %w", err)
	}

	msg := s.formatSearchSuccessMessage(ctx, user, item, quantity, isCritical, params)
	if params.region != nil && params.region.RequiredExplorerLevel > 0 {
		msg += fmt.Sprintf(" [%s]", params.region.Name)
	}
	return msg, nil
}

// processRegionItemDrop grants a region-specific item instead of a lootbox.
func (s *service) processRegionItemDrop(ctx context.Context, user *domain.User, isCritical bool, quantity int, params searchParams) (string, error) {
	log := logger.FromContext(ctx)
	droppedItemName := rollRegionItemDrop(params.region.ItemDrops)
	if droppedItemName == "" {
		// Fallback to lootbox if roll fails
		log.Warn("Region item drop roll returned empty, falling back to lootbox")
		qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
		if err := s.grantSearchReward(ctx, user, quantity, qualityLevel); err != nil {
			return "", err
		}
		item, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
		if err != nil {
			return "", fmt.Errorf("failed to get reward item: %w", err)
		}
		return s.formatSearchSuccessMessage(ctx, user, item, quantity, isCritical, params), nil
	}

	// Grant the region item
	item, err := s.getItemByNameCached(ctx, droppedItemName)
	if err != nil || item == nil {
		log.Error("Failed to get region drop item, falling back to lootbox", "item", droppedItemName, "error", err)
		qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
		if err := s.grantSearchReward(ctx, user, quantity, qualityLevel); err != nil {
			return "", err
		}
		lbItem, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
		if err != nil {
			return "", fmt.Errorf("failed to get reward item: %w", err)
		}
		return s.formatSearchSuccessMessage(ctx, user, lbItem, quantity, isCritical, params), nil
	}

	qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
	if err := s.grantItemReward(ctx, user, item, quantity, qualityLevel); err != nil {
		return "", err
	}

	msg := s.formatSearchSuccessMessage(ctx, user, item, quantity, isCritical, params)
	if params.region != nil {
		msg += fmt.Sprintf(" [%s]", params.region.Name)
	}
	log.Info("Region item drop granted", "item", droppedItemName, "region", params.region.Name, "quantity", quantity)
	return msg, nil
}

func (s *service) processSearchFailure(roll float64, successThreshold float64, params searchParams) string {
	// Determine failure type
	failureType := determineSearchFailureType(roll, successThreshold)

	// Format failure message
	resultMessage := formatSearchFailureMessage(failureType)

	// Append region name if applicable
	if params.region != nil && params.region.RequiredExplorerLevel > 0 {
		resultMessage += fmt.Sprintf(" [%s]", params.region.Name)
	}

	// Append streak and exhausted status if applicable
	return s.formatSearchFailureMessageWithMeta(resultMessage, params)
}
