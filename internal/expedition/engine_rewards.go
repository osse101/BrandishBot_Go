package expedition

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func (e *Engine) calculateRewards(won bool) []domain.PartyMemberReward {
	rewards := make([]domain.PartyMemberReward, 0, len(e.party))

	// Divide purse among all participants with variance
	baseMoney := e.purse / len(e.party)
	variance := baseMoney / 5 // +-20% variance

	// Randomly assign pooled items to participants
	itemAssignments := make(map[int][]string) // index -> items
	for _, item := range e.rewardPool {
		idx := e.rng.IntN(len(e.party))
		itemAssignments[idx] = append(itemAssignments[idx], item)
	}

	for i, m := range e.party {
		money := baseMoney
		if variance > 0 {
			money += e.rng.IntN(variance*2+1) - variance
		}
		if money < 0 {
			money = 0
		}

		items := itemAssignments[i]
		if items == nil {
			items = []string{}
		}

		isLeader := i == 0 // First member is the leader (initiator)

		// Leader bonus
		if isLeader && e.config.Settings.LeaderBonusReward != "" {
			items = append(items, e.config.Settings.LeaderBonusReward)
		}

		// Win bonus: active (conscious) members get extra reward
		if won && m.IsConscious {
			if e.config.Settings.WinBonusReward != "" {
				items = append(items, e.config.Settings.WinBonusReward)
			}
			money += e.config.Settings.WinBonusMoney
		}

		// XP: ceil(partySize / divisor) + 1
		xp := 0
		if e.config.Settings.XPFormulaDivisor > 0 {
			xp = (e.initialParty+e.config.Settings.XPFormulaDivisor-1)/e.config.Settings.XPFormulaDivisor + 1
		}

		rewards = append(rewards, domain.PartyMemberReward{
			UserID:   m.UserID,
			Username: m.Username,
			Money:    money,
			Items:    items,
			XP:       xp,
			IsLeader: isLeader,
		})

		// Store on member for reference
		m.PrizeMoney = money
		m.PrizeItems = items
	}

	return rewards
}
