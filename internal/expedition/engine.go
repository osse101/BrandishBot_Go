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
