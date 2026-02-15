package user

import (
	"context"
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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
func (s *service) grantSearchReward(ctx context.Context, user *domain.User, quantity int, qualityLevel domain.QualityLevel) error {
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
		return s.addItemToTx(ctx, tx, user.ID, item.ID, quantity, qualityLevel)
	})
}

var searchQualityLevels = []domain.QualityLevel{
	domain.QualityCursed,    // 0
	domain.QualityJunk,      // 1
	domain.QualityPoor,      // 2
	domain.QualityCommon,    // 3
	domain.QualityUncommon,  // 4
	domain.QualityRare,      // 5
	domain.QualityEpic,      // 6
	domain.QualityLegendary, // 7
}

// calculateSearchQuality determines the quality level for search results based on a point system
func (s *service) calculateSearchQuality(ctx context.Context, userID string, isCritical bool, params searchParams) domain.QualityLevel {
	log := logger.FromContext(ctx)
	// 1. Determine base index based on daily count (current count includes the one we just did)
	// Bracket mapping: 1=uncommon, 2-5=common, 6-9=poor, 10-14=junk, 15+=cursed
	// 1. Determine base index based on daily count
	baseIndex := s.calculateBaseQualityIndex(params.dailyCount)

	// 2. Add points for bonuses
	points := 0
	if isCritical {
		points += 2
	}
	// Streak milestone (multiple of 5)
	if params.streak > 0 && params.streak%5 == 0 {
		points += 1
	}

	// 3. Add Explorer job level bonus (+1 point per 5 levels)
	if s.jobService != nil {
		explorerLevel, err := s.jobService.GetJobLevel(ctx, userID, "job_explorer")
		if err != nil {
			log.Warn("Failed to get Explorer job level for search quality", "error", err)
		} else if explorerLevel > 0 {
			explorerBonus := explorerLevel / 5
			points += explorerBonus
			log.Debug("Explorer bonus applied to search quality", "level", explorerLevel, "bonus", explorerBonus)
		}
	}

	// 4. Calculate final index and clamp
	finalIndex := baseIndex + points
	if finalIndex >= len(searchQualityLevels) {
		finalIndex = len(searchQualityLevels) - 1
	}
	if finalIndex < 0 {
		finalIndex = 0
	}

	return searchQualityLevels[finalIndex]
}

// formatSearchSuccessMessage builds the success message with appropriate formatting
func (s *service) formatSearchSuccessMessage(ctx context.Context, user *domain.User, item *domain.Item, quantity int, isCritical bool, params searchParams) string {
	log := logger.FromContext(ctx)

	actualQuality := s.calculateSearchQuality(ctx, user.ID, isCritical, params)

	// User Request: "Search should use the item's public name"
	displayName := cases.Title(language.English).String(item.PublicName)
	var resultMessage string

	if isCritical {
		resultMessage = fmt.Sprintf("%s You found %dx%s", domain.MsgSearchCriticalSuccess, quantity, displayName)
		log.Info("Search CRITICAL success", "item", item.InternalName, "quantity", quantity, "quality", actualQuality)
	} else {
		resultMessage = fmt.Sprintf("You have found %dx%s", quantity, displayName)
		log.Info("Search successful - lootbox found", "item", item.InternalName, "quality", actualQuality)
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

// calculateBaseQualityIndex determines the base quality index based on daily search count
func (s *service) calculateBaseQualityIndex(dailyCount int) int {
	// Bracket mapping: 1=uncommon, 2-5=common, 6-9=poor, 10-14=junk, 15+=cursed
	switch {
	case dailyCount == 0:
		return 4 // 1st search: UNCOMMON
	case dailyCount >= 1 && dailyCount <= 4:
		return 3 // 2nd-5th search: COMMON
	case dailyCount >= 5 && dailyCount <= 8:
		return 2 // 6th-9th search: POOR
	case dailyCount >= 9 && dailyCount <= 13:
		return 1 // 10th-14th search: JUNK
	default:
		return 0 // 15th+ search: CURSED
	}
}
