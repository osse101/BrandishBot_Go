package gamble

import (
	"sort"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// calculateTotalLootboxes sums up lootbox quantities from bets
func calculateTotalLootboxes(bets []domain.LootboxBet) int {
	total := 0
	for _, bet := range bets {
		total += bet.Quantity
	}
	return total
}

// determineCriticalFailures returns the set of user IDs who had critical fail scores
func (s *service) determineCriticalFailures(userValues map[string]int64, totalGambleValue int64) map[string]bool {
	critFails := make(map[string]bool)
	if len(userValues) <= 1 || totalGambleValue <= 0 {
		return critFails
	}
	averageScore := float64(totalGambleValue) / float64(len(userValues))
	threshold := int64(averageScore * CriticalFailThreshold)
	for userID, val := range userValues {
		if val <= threshold {
			critFails[userID] = true
		}
	}
	return critFails
}

// determineGambleWinners returns the winner ID, highest score, and set of users who lost a tie-break
func (s *service) determineGambleWinners(userValues map[string]int64) (string, int64, map[string]bool) {
	var highestValue int64 = InitialHighestValue
	var winners []string

	for userID, val := range userValues {
		if val > highestValue {
			highestValue = val
			winners = []string{userID}
		} else if val == highestValue {
			winners = append(winners, userID)
		}
	}

	tieBreakLost := make(map[string]bool)

	if len(winners) == 0 {
		return "", 0, tieBreakLost
	}

	if len(winners) > 1 {
		sort.Strings(winners)
		idx := s.rng(len(winners))
		winnerID := winners[idx]
		for _, uid := range winners {
			if uid != winnerID {
				tieBreakLost[uid] = true
			}
		}
		return winnerID, highestValue, tieBreakLost
	}
	return winners[0], highestValue, tieBreakLost
}

// determineNearMisses returns the set of user IDs who had near-miss scores (not the winner)
func (s *service) determineNearMisses(winnerID string, highestValue int64, userValues map[string]int64) map[string]bool {
	nearMiss := make(map[string]bool)
	if winnerID == "" || highestValue <= 0 {
		return nearMiss
	}
	threshold := int64(float64(highestValue) * NearMissThreshold)
	for userID, val := range userValues {
		if userID == winnerID || val == highestValue {
			continue
		}
		if val >= threshold {
			nearMiss[userID] = true
		}
	}
	return nearMiss
}

// buildParticipantOutcomes constructs per-participant outcome data for the GambleCompletedPayloadV2
func (s *service) buildParticipantOutcomes(gamble *domain.Gamble, userValues map[string]int64, winnerID string, critFailUsers, tieBreakLostUsers, nearMissUsers map[string]bool) []domain.GambleParticipantOutcome {
	outcomes := make([]domain.GambleParticipantOutcome, 0, len(gamble.Participants))
	for _, p := range gamble.Participants {
		outcomes = append(outcomes, domain.GambleParticipantOutcome{
			UserID:         p.UserID,
			Score:          userValues[p.UserID],
			LootboxCount:   calculateTotalLootboxes(p.LootboxBets),
			IsWinner:       p.UserID == winnerID,
			IsNearMiss:     nearMissUsers[p.UserID],
			IsCritFail:     critFailUsers[p.UserID],
			IsTieBreakLost: tieBreakLostUsers[p.UserID],
		})
	}
	return outcomes
}
