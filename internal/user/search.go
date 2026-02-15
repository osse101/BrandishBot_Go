package user

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// HandleSearch performs a search action for a user with cooldown tracking
func (s *service) HandleSearch(ctx context.Context, platform, platformID, username string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("HandleSearch called", "platform", platform, "platformID", platformID, "username", username)

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
		resultMessage, err = s.executeSearch(ctx, user)
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
}

// executeSearch performs the actual search logic (called within cooldown enforcement)
func (s *service) executeSearch(ctx context.Context, user *domain.User) (string, error) {
	params := s.calculateSearchParameters(ctx, user)

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
		isCritical = roll <= SearchCriticalRate
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
		isDiminished:       (dailyCount >= SearchDailyDiminishmentThreshold),
		xpMultiplier:       1.0,
		successThreshold:   SearchSuccessRate,
		dailyCount:         dailyCount,
	}

	if params.isDiminished {
		params.successThreshold = SearchDiminishedSuccessRate
		params.xpMultiplier = SearchDiminishedXPMultiplier
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
	isCritical := roll <= SearchCriticalRate
	quantity := 1
	if isCritical {
		quantity = 2
	}

	// Grant reward
	qualityLevel := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
	if err := s.grantSearchReward(ctx, user, quantity, qualityLevel); err != nil {
		return "", err
	}

	// Get item for message formatting and event recording
	item, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
	if err != nil {
		return "", fmt.Errorf("failed to get reward item: %w", err)
	}

	// Format and return result message
	return s.formatSearchSuccessMessage(ctx, user, item, quantity, isCritical, params), nil
}

func (s *service) processSearchFailure(roll float64, successThreshold float64, params searchParams) string {
	// Determine failure type
	failureType := determineSearchFailureType(roll, successThreshold)

	// Format failure message
	resultMessage := formatSearchFailureMessage(failureType)

	// Append streak and exhausted status if applicable
	return s.formatSearchFailureMessageWithMeta(resultMessage, params)
}
