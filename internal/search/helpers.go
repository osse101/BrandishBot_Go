package search

import (
	"context"
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// searchFailureType categorizes the type of search failure
type searchFailureType int

const (
	searchFailureNearMiss searchFailureType = iota
	searchFailureCritical
	searchFailureNormal
)

const (
	MsgExhaustedSuffix = " (Exhausted)"
)

// searchQualityLevels maps point totals to quality levels.
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

// calculateSearchQuality determines the quality level for search results.
func (s *service) calculateSearchQuality(ctx context.Context, userID string, isCritical bool, params searchParams) domain.QualityLevel {
	log := logger.FromContext(ctx)
	baseIndex := calculateBaseQualityIndex(params.dailyCount)

	points := 0
	if isCritical {
		points += 2
	}
	if params.streak > 0 && params.streak%5 == 0 {
		points += 1
	}

	if s.deps.JobSvc != nil {
		explorerLevel, err := s.deps.JobSvc.GetJobLevel(ctx, userID, "job_explorer")
		if err != nil {
			log.Warn("Failed to get Explorer job level for search quality", "error", err)
		} else if explorerLevel > 0 {
			explorerBonus := explorerLevel / 5
			points += explorerBonus
			log.Debug("Explorer bonus applied to search quality", "level", explorerLevel, "bonus", explorerBonus)
		}
	}

	finalIndex := baseIndex + points

	if s.deps.ProgressionSvc != nil {
		if modifiedIndex, err := s.deps.ProgressionSvc.GetModifiedValue(ctx, userID, "search_quality", float64(finalIndex)); err == nil {
			finalIndex = int(modifiedIndex)
		} else {
			log.Warn("Failed to apply search_quality modifier", "error", err)
		}
	}

	if finalIndex >= len(searchQualityLevels) {
		finalIndex = len(searchQualityLevels) - 1
	}
	if finalIndex < 0 {
		finalIndex = 0
	}

	return searchQualityLevels[finalIndex]
}

// formatSearchSuccessMessage builds the success message.
func (s *service) formatSearchSuccessMessage(ctx context.Context, user *domain.User, item *domain.Item, quantity int, isCritical bool, params searchParams) string {
	log := logger.FromContext(ctx)
	actualQuality := s.calculateSearchQuality(ctx, user.ID, isCritical, params)
	displayName := cases.Title(language.English).String(item.PublicName)

	var resultMessage string
	if isCritical {
		resultMessage = fmt.Sprintf("%s You found %dx%s", domain.MsgSearchCriticalSuccess, quantity, displayName)
		log.Info("Search CRITICAL success", "item", item.InternalName, "quantity", quantity, "quality", actualQuality)
	} else {
		resultMessage = fmt.Sprintf("You found %dx%s", quantity, displayName)
		log.Info("Search successful - lootbox found", "item", item.InternalName, "quality", actualQuality)
	}

	if params.isFirstSearchDaily && params.streak > 0 && params.streak%5 == 0 {
		resultMessage += fmt.Sprintf(domain.MsgStreakBonus, params.streak)
	}

	if params.dailyCount == domain.SearchDailyDiminishmentThreshold {
		resultMessage += " (Exhausted)"
	}

	return resultMessage
}

// determineSearchFailureType categorizes the type of search failure based on roll.
func determineSearchFailureType(roll, successThreshold float64) searchFailureType {
	if roll <= successThreshold+domain.SearchNearMissRate {
		return searchFailureNearMiss
	}
	if roll > 1.0-domain.SearchCriticalFailRate {
		return searchFailureCritical
	}
	return searchFailureNormal
}

// formatSearchFailureMessage builds the failure message based on failure type.
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

// formatSearchFailureMessageWithMeta appends streak and exhausted status to failure messages.
func formatSearchFailureMessageWithMeta(message string, params searchParams) string {
	result := message

	if params.isFirstSearchDaily && params.streak > 0 && params.streak%5 == 0 {
		result += fmt.Sprintf(domain.MsgStreakBonus, params.streak)
	}

	if params.dailyCount == domain.SearchDailyDiminishmentThreshold {
		result += MsgExhaustedSuffix
	}

	return result
}

// calculateBaseQualityIndex determines the base quality index based on daily search count.
func calculateBaseQualityIndex(dailyCount int) int {
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
