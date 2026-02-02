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
func (s *service) grantSearchReward(ctx context.Context, user *domain.User, quantity int, shineLevel string) error {
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
		return s.addItemToTx(ctx, tx, user.ID, item.ID, quantity, shineLevel)
	})
}

var searchShineLevels = []string{
	"CURSED",    // 0
	"JUNK",      // 1
	"POOR",      // 2
	"COMMON",    // 3
	"UNCOMMON",  // 4
	"RARE",      // 5
	"EPIC",      // 6
	"LEGENDARY", // 7
}

// calculateSearchShine determines the shine level for search results based on a point system
func (s *service) calculateSearchShine(isCritical bool, params searchParams) string {
	// 1. Determine base index based on daily count (current count includes the one we just did)
	// Bracket mapping: 1=uncommon, 2-5=common, 6-9=poor, 10-14=junk, 15+=cursed
	baseIndex := 0
	switch {
	case params.dailyCount == 0:
		baseIndex = 4 // 1st search: UNCOMMON
	case params.dailyCount >= 1 && params.dailyCount <= 4:
		baseIndex = 3 // 2nd-5th search: COMMON
	case params.dailyCount >= 5 && params.dailyCount <= 8:
		baseIndex = 2 // 6th-9th search: POOR
	case params.dailyCount >= 9 && params.dailyCount <= 13:
		baseIndex = 1 // 10th-14th search: JUNK
	default:
		baseIndex = 0 // 15th+ search: CURSED
	}

	// 2. Add points for bonuses
	points := 0
	if isCritical {
		points += 2
	}
	// Streak milestone (multiple of 5)
	if params.streak > 0 && params.streak%5 == 0 {
		points += 1
	}

	// 3. Calculate final index and clamp
	finalIndex := baseIndex + points
	if finalIndex >= len(searchShineLevels) {
		finalIndex = len(searchShineLevels) - 1
	}
	if finalIndex < 0 {
		finalIndex = 0
	}

	return searchShineLevels[finalIndex]
}

// formatSearchSuccessMessage builds the success message with appropriate formatting
func (s *service) formatSearchSuccessMessage(ctx context.Context, item *domain.Item, quantity int, isCritical bool, params searchParams) string {
	log := logger.FromContext(ctx)

	actualShine := s.calculateSearchShine(isCritical, params)

	// Use naming resolver to get name WITHOUT shine prefix for user display
	displayName := s.namingResolver.GetDisplayName(item.InternalName, "COMMON")
	var resultMessage string

	if isCritical {
		resultMessage = fmt.Sprintf("%s You found %dx %s", domain.MsgSearchCriticalSuccess, quantity, displayName)
		log.Info("Search CRITICAL success", "item", item.InternalName, "quantity", quantity, "shine", actualShine)
	} else {
		resultMessage = fmt.Sprintf("You have found %dx %s", quantity, displayName)
		log.Info("Search successful - lootbox found", "item", item.InternalName, "shine", actualShine)
	}

	// Show streak only on the first search of the day
	if params.isFirstSearchDaily && params.streak > 0 && params.streak%5 == 0 {
		resultMessage += fmt.Sprintf(domain.MsgStreakBonus, params.streak)
	}

	// Show (Exhausted) only on the VERY FIRST search that hits the threshold
	if params.dailyCount == SearchDailyDiminishmentThreshold {
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

// formatSearchFailureMessageWithMeta appends streak and exhausted status to failure messages
func (s *service) formatSearchFailureMessageWithMeta(message string, params searchParams) string {
	result := message

	// Show streak only on the first search of the day
	if params.isFirstSearchDaily && params.streak > 0 && params.streak%5 == 0 {
		result += fmt.Sprintf(domain.MsgStreakBonus, params.streak)
	}

	// Show (Exhausted) only on the VERY FIRST search that hits the threshold
	if params.dailyCount == SearchDailyDiminishmentThreshold {
		result += " (Exhausted)"
	}

	return result
}
