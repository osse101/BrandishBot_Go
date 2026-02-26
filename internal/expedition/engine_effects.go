package expedition

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func (e *Engine) applyEffects(effects EffectsDef, actingMember *domain.PartyMemberState) {
	// Apply fatigue
	e.fatigue += effects.FatigueDelta
	if e.fatigue < 0 {
		e.fatigue = 0
	}

	// Apply purse change
	e.purse += effects.PurseDelta
	if e.purse < 0 {
		e.purse = 0
	}

	// Add reward to pool
	if effects.Reward != "" {
		e.rewardPool = append(e.rewardPool, effects.Reward)
	}

	// Apply KOs (scaled by party size)
	if effects.KOScale > 0 {
		koCount := ScaleEffect(effects.KOScale, e.initialParty, e.config.Settings.PartyScaleDivisor)
		e.applyKOs(koCount, actingMember)
	}

	// Apply revives (scaled by party size)
	if effects.ReviveScale > 0 {
		reviveCount := ScaleEffect(effects.ReviveScale, e.initialParty, e.config.Settings.PartyScaleDivisor)
		e.applyRevives(reviveCount)
	}

	// Debuff primary
	if effects.DebuffPrimary && actingMember != nil {
		actingMember.IsDebuffed = true
	}

	// Grant temp skill
	if effects.TempSkill != "" && actingMember != nil {
		skill := domain.ExpeditionSkill(effects.TempSkill)
		if !hasTempSkill(actingMember, skill) {
			actingMember.TempSkills = append(actingMember.TempSkills, skill)
		}
	}

	// Shift outcome weights toward positive
	if effects.ShiftWeights != 0 {
		// We store this as a modifier applied to all encounters
		for name := range e.config.Encounters {
			key := domain.EncounterType(name)
			e.weightMods[key] += effects.ShiftWeights
		}
	}

	// Bonus money for passing skill check
	if actingMember != nil && effects.PurseDelta >= 0 {
		// Skill check bonus only on non-negative purse outcomes
		e.purse += e.config.Settings.SkillCheckBonus
	}
}

func (e *Engine) applyKOs(count int, exclude *domain.PartyMemberState) {
	conscious := e.getConsciousMembers()
	// Shuffle to randomize who gets KO'd
	e.rng.Shuffle(len(conscious), func(i, j int) {
		conscious[i], conscious[j] = conscious[j], conscious[i]
	})

	knocked := 0
	for _, m := range conscious {
		if knocked >= count {
			break
		}
		// Try not to KO the acting member (they're the hero of this turn)
		if m == exclude && len(conscious) > 1 {
			continue
		}
		m.IsConscious = false
		knocked++
	}

	// If we still need to KO more and skipped the acting member, KO them too
	if knocked < count && exclude != nil && exclude.IsConscious {
		exclude.IsConscious = false
	}
}

func (e *Engine) applyRevives(count int) {
	revived := 0
	for _, m := range e.party {
		if revived >= count {
			break
		}
		if !m.IsConscious {
			m.IsConscious = true
			m.IsDebuffed = false
			revived++
		}
	}
}
