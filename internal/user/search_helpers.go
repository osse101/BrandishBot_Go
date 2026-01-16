package user

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// searchFailureType categorizes the type of search failure
type searchFailureType int

const (
	searchFailureNearMiss searchFailureType = iota
	searchFailureCritical
	searchFailureNormal
)

// grantSearchReward adds lootbox to inventory within a transaction
func (s *service) grantSearchReward(ctx context.Context, user *domain.User, quantity int) error {
	log := logger.FromContext(ctx)

	item, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
	if err != nil {
		log.Error("Failed to get lootbox0 item", "error", err)
		return fmt.Errorf("failed to get reward item: %w", err)
	}
	if item == nil {
		log.Error("Lootbox0 item not found in database")
		return fmt.Errorf("%w: %s", domain.ErrItemNotFound, domain.ItemLootbox0)
	}

	return s.withTx(ctx, func(tx repository.UserTx) error {
		return s.addItemToTx(ctx, tx, user.ID, item.ID, quantity)
	})
}

// formatSearchSuccessMessage builds the success message with appropriate formatting
func (s *service) formatSearchSuccessMessage(ctx context.Context, item *domain.Item, quantity int, isCritical bool, params searchParams) string {
	log := logger.FromContext(ctx)
	displayName := s.namingResolver.GetDisplayName(item.InternalName, "")
	var resultMessage string

	if isCritical {
		resultMessage = fmt.Sprintf("%s You found %dx %s", domain.MsgSearchCriticalSuccess, quantity, displayName)
		log.Info("Search CRITICAL success", "item", item.InternalName, "quantity", quantity)
	} else {
		resultMessage = fmt.Sprintf("You have found %dx %s", quantity, displayName)
		log.Info("Search successful - lootbox found", "item", item.InternalName)
	}

	if params.isFirstSearchDaily {
		resultMessage += domain.MsgFirstSearchBonus
	} else if params.isDiminished {
		resultMessage += " (Exhausted)"
	}

	return resultMessage
}

// recordSearchSuccessEvents records statistics for successful searches
func (s *service) recordSearchSuccessEvents(ctx context.Context, user *domain.User, item *domain.Item, quantity int, roll float64, isCritical bool) {
	if s.statsService == nil {
		return
	}

	if isCritical {
		_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearchCriticalSuccess, map[string]interface{}{
			"item":     item.InternalName,
			"quantity": quantity,
			"roll":     roll,
		})
	}
}

// determineSearchFailureType categorizes the type of search failure based on roll
func determineSearchFailureType(roll, successThreshold float64) searchFailureType {
	if roll <= successThreshold+SearchNearMissRate {
		return searchFailureNearMiss
	}
	if roll > 1.0-SearchCriticalFailRate {
		return searchFailureCritical
	}
	return searchFailureNormal
}

// formatSearchFailureMessage builds the failure message based on failure type
func formatSearchFailureMessage(failureType searchFailureType) string {
	switch failureType {
	case searchFailureNearMiss:
		return domain.MsgSearchNearMiss

	case searchFailureCritical:
		resultMessage := domain.MsgSearchCriticalFail
		if len(domain.SearchCriticalFailMessages) > 0 {
			idx := utils.SecureRandomIntRange(0, len(domain.SearchCriticalFailMessages)-1)
			resultMessage = fmt.Sprintf("%s %s", domain.MsgSearchCriticalFail, domain.SearchCriticalFailMessages[idx])
		}
		return resultMessage

	case searchFailureNormal:
		if len(domain.SearchFailureMessages) > 0 {
			idx := utils.SecureRandomIntRange(0, len(domain.SearchFailureMessages)-1)
			return domain.SearchFailureMessages[idx]
		}
		return domain.MsgSearchNothingFound

	default:
		return domain.MsgSearchNothingFound
	}
}

// recordSearchFailureEvents records statistics for failed searches
func (s *service) recordSearchFailureEvents(ctx context.Context, user *domain.User, roll, successThreshold float64, failureType searchFailureType) {
	log := logger.FromContext(ctx)

	if s.statsService == nil {
		return
	}

	switch failureType {
	case searchFailureNearMiss:
		_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearchNearMiss, map[string]interface{}{
			"roll":      roll,
			"threshold": successThreshold,
		})
		log.Info("Search NEAR MISS", "username", user.Username, "roll", roll)

	case searchFailureCritical:
		_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearchCriticalFail, map[string]interface{}{
			"roll": roll,
		})
		log.Info("Search CRITICAL FAIL", "username", user.Username, "roll", roll)

	case searchFailureNormal:
		log.Info("Search successful - nothing found", "username", user.Username)
	}
}

// appendStreakBonus appends streak information to the message if applicable
func (s *service) appendStreakBonus(ctx context.Context, user *domain.User, message string, isFirstSearchDaily bool) string {
	log := logger.FromContext(ctx)

	if !isFirstSearchDaily || s.statsService == nil {
		return message
	}

	streak, err := s.statsService.GetUserCurrentStreak(ctx, user.ID)
	if err != nil {
		log.Warn("Failed to get user streak", "error", err)
		return message
	}

	if streak > 1 {
		return message + fmt.Sprintf(domain.MsgStreakBonus, streak)
	}

	return message
}
