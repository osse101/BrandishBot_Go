package expedition

import (
	"math/rand"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Engine runs the expedition turn loop. It is pure logic with no DB or service dependencies.
type Engine struct {
	config       *EncounterConfig
	rng          *rand.Rand
	party        []*domain.PartyMemberState
	fatigue      int
	purse        int
	turn         int
	rewardPool   []string
	journal      []domain.ExpeditionTurn
	weightMods   map[domain.EncounterType]float64 // dynamic weight adjustments
	initialParty int
	maxJobLevel  int
}

// NewEngine creates a new expedition engine
func NewEngine(config *EncounterConfig, party []*domain.PartyMemberState, seed int64) *Engine {
	//nolint:gosec // G404: math/rand is acceptable for game mechanics, not for cryptographic purposes
	return &Engine{
		config:       config,
		rng:          rand.New(rand.NewSource(seed)),
		party:        party,
		fatigue:      0,
		purse:        config.Settings.StartingPurse,
		turn:         0,
		rewardPool:   make([]string, 0),
		journal:      make([]domain.ExpeditionTurn, 0, config.Settings.MaxTurns+1),
		weightMods:   make(map[domain.EncounterType]float64),
		initialParty: len(party),
		maxJobLevel:  FindMaxJobLevel(party),
	}
}

// Run executes the full expedition and returns the result
func (e *Engine) Run() *domain.ExpeditionResult {
	// Turn 0: intro narrative
	e.appendIntro()

	// Turn loop
	for e.turn = 1; e.turn <= e.config.Settings.MaxTurns; e.turn++ {
		if !e.runTurn() {
			break
		}
	}

	won := e.turn > e.config.Settings.MaxTurns
	allKO := !e.hasConsciousMembers()

	return &domain.ExpeditionResult{
		TotalTurns:    e.turn - 1,
		Won:           won,
		AllKnockedOut: allKO,
		FinalFatigue:  e.fatigue,
		PartyRewards:  e.calculateRewards(won),
		Journal:       e.journal,
	}
}

func (e *Engine) appendIntro() {
	narrative := e.config.IntroNarratives[e.rng.Intn(len(e.config.IntroNarratives))]
	e.journal = append(e.journal, domain.ExpeditionTurn{
		TurnNumber:    0,
		EncounterType: "",
		Outcome:       "",
		Narrative:     narrative,
		Fatigue:       e.fatigue,
		PurseAfter:    e.purse,
	})
}

// runTurn executes a single turn. Returns false if the expedition should end.
func (e *Engine) runTurn() bool {
	// 1. Apply base fatigue
	e.fatigue += e.config.Settings.BaseFatiguePerTurn

	// 2. Roll encounter
	encounterKey := e.rollEncounter()
	encounter := e.config.Encounters[string(encounterKey)]

	// 3. Roll outcome
	outcomeType := e.rollOutcome(encounter)

	// 4. Pick a random skill from the encounter's skill list
	skill := encounter.Skills[e.rng.Intn(len(encounter.Skills))]

	// 5. Resolve skill check
	consciousMembers := e.getConsciousMembers()
	passed, actingMember := ResolveSkillCheck(e.rng, skill, consciousMembers, e.maxJobLevel, e.config.Settings.TempSkillBonus)

	// 6. Handle debuff: if acting member is debuffed, force fail, clear debuff
	if actingMember != nil && actingMember.IsDebuffed {
		passed = false
		actingMember.IsDebuffed = false
	}

	// Consume temp skill if used (regardless of pass/fail, it was attempted)
	if actingMember != nil && hasTempSkill(actingMember, skill) {
		ConsumeTemporarySkill(actingMember, skill)
	}

	// 7. Apply effects
	outcomeDef := encounter.Outcomes[string(outcomeType)]
	var detail *OutcomeDetail
	if passed {
		detail = outcomeDef.SkillPass
	} else {
		detail = outcomeDef.SkillFail
	}
	e.applyEffects(detail.Effects, actingMember)

	// 8. Build narrative
	narrative := e.buildNarrative(detail, actingMember)

	// 9. Record turn
	primaryName := ""
	if actingMember != nil {
		primaryName = actingMember.Username
	}

	e.journal = append(e.journal, domain.ExpeditionTurn{
		TurnNumber:    e.turn,
		EncounterType: encounterKey,
		Outcome:       outcomeType,
		SkillChecked:  skill,
		SkillPassed:   passed,
		PrimaryMember: primaryName,
		Narrative:     narrative,
		Fatigue:       e.fatigue,
		PurseAfter:    e.purse,
	})

	// 10. Check end conditions
	if e.fatigue >= e.config.Settings.MaxFatigue {
		return false
	}
	if !e.hasConsciousMembers() {
		return false
	}

	return true
}

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

func (e *Engine) getConsciousMembers() []*domain.PartyMemberState {
	result := make([]*domain.PartyMemberState, 0)
	for _, m := range e.party {
		if m.IsConscious {
			result = append(result, m)
		}
	}
	return result
}

func (e *Engine) hasConsciousMembers() bool {
	for _, m := range e.party {
		if m.IsConscious {
			return true
		}
	}
	return false
}

func (e *Engine) calculateRewards(won bool) []domain.PartyMemberReward {
	rewards := make([]domain.PartyMemberReward, 0, len(e.party))

	// Divide purse among all participants with variance
	baseMoney := e.purse / len(e.party)
	variance := baseMoney / 5 // +-20% variance

	// Randomly assign pooled items to participants
	itemAssignments := make(map[int][]string) // index -> items
	for _, item := range e.rewardPool {
		idx := e.rng.Intn(len(e.party))
		itemAssignments[idx] = append(itemAssignments[idx], item)
	}

	for i, m := range e.party {
		money := baseMoney
		if variance > 0 {
			money += e.rng.Intn(variance*2+1) - variance
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
