package expedition

import (
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func testConfig() *EncounterConfig {
	return &EncounterConfig{
		Version: "1.0",
		Settings: EngineSettings{
			BaseFatiguePerTurn: 5,
			MaxFatigue:         100,
			MaxTurns:           50,
			StartingPurse:      2000,
			SkillCheckBonus:    50,
			XPFormulaDivisor:   4,
			LeaderBonusReward:  "lootbox_tier2",
			WinBonusReward:     "xp_rarecandy",
			WinBonusMoney:      500,
			PartyScaleDivisor:  3,
			TempSkillBonus:     0.3,
		},
		IntroNarratives: []string{
			"The expedition begins.",
		},
		Encounters: map[string]*EncounterDef{
			"explore": {
				DisplayName: "Exploration",
				BaseWeight:  0.5,
				Skills:      []domain.ExpeditionSkill{domain.SkillPerception, domain.SkillSurvival},
				MinParty:    1,
				Outcomes: map[string]*OutcomeDef{
					"positive": {
						Weight: 0.30,
						SkillPass: &OutcomeDetail{
							Effects: EffectsDef{FatigueDelta: 0, PurseDelta: 100, Reward: "lootbox_tier1"},
							Narratives: []NarrativeDef{
								{Surprise: "A cache", Action: "{{primary}} opens it", Outcome: "Supplies found"},
							},
						},
						SkillFail: &OutcomeDetail{
							Effects: EffectsDef{FatigueDelta: 0, PurseDelta: 50},
							Narratives: []NarrativeDef{
								{Surprise: "A cache", Action: "{{primary}} reaches in", Outcome: "Something useful"},
							},
						},
					},
					"neutral": {
						Weight: 0.50,
						SkillPass: &OutcomeDetail{
							Effects:    EffectsDef{FatigueDelta: 0},
							Narratives: []NarrativeDef{{Surprise: "Path splits", Action: "{{primary}} examines", Outcome: "Correct path found"}},
						},
						SkillFail: &OutcomeDetail{
							Effects:    EffectsDef{FatigueDelta: 3},
							Narratives: []NarrativeDef{{Surprise: "Path splits", Action: "{{primary}} picks random", Outcome: "Minor detour"}},
						},
					},
					"negative": {
						Weight: 0.20,
						SkillPass: &OutcomeDetail{
							Effects:    EffectsDef{FatigueDelta: 5},
							Narratives: []NarrativeDef{{Surprise: "Ground gives way", Action: "{{primary}} dodges", Outcome: "Energy wasted"}},
						},
						SkillFail: &OutcomeDetail{
							Effects:    EffectsDef{FatigueDelta: 10, PurseDelta: -100, KOScale: 1},
							Narratives: []NarrativeDef{{Surprise: "Ground gives way", Action: "{{primary}} tumbles", Outcome: "Supplies lost"}},
						},
					},
				},
			},
			"combat_boss": {
				DisplayName: "Boss Fight",
				BaseWeight:  0.5,
				Skills:      []domain.ExpeditionSkill{domain.SkillFortitude},
				MinParty:    3,
				Outcomes: map[string]*OutcomeDef{
					"positive": {
						Weight: 0.30,
						SkillPass: &OutcomeDetail{
							Effects: EffectsDef{FatigueDelta: 8, PurseDelta: 500, Reward: "lootbox_tier3"},
							Narratives: []NarrativeDef{
								{Surprise: "A boss appears", Action: "{{primary}} attacks", Outcome: "Victory!"},
							},
						},
						SkillFail: &OutcomeDetail{
							Effects: EffectsDef{FatigueDelta: 12, PurseDelta: 200},
							Narratives: []NarrativeDef{
								{Surprise: "A boss appears", Action: "{{primary}} charges", Outcome: "Narrow win"},
							},
						},
					},
					"neutral": {
						Weight: 0.40,
						SkillPass: &OutcomeDetail{
							Effects:    EffectsDef{FatigueDelta: 10},
							Narratives: []NarrativeDef{{Surprise: "Boss blocks", Action: "{{primary}} negotiates", Outcome: "Stalemate"}},
						},
						SkillFail: &OutcomeDetail{
							Effects:    EffectsDef{FatigueDelta: 15, KOScale: 1},
							Narratives: []NarrativeDef{{Surprise: "Boss blocks", Action: "{{primary}} falters", Outcome: "Costly fight"}},
						},
					},
					"negative": {
						Weight: 0.30,
						SkillPass: &OutcomeDetail{
							Effects:    EffectsDef{FatigueDelta: 12, KOScale: 2, DebuffPrimary: true},
							Narratives: []NarrativeDef{{Surprise: "Boss rages", Action: "{{primary}} retreats", Outcome: "Heavy losses"}},
						},
						SkillFail: &OutcomeDetail{
							Effects:    EffectsDef{FatigueDelta: 20, KOScale: 3, DebuffPrimary: true},
							Narratives: []NarrativeDef{{Surprise: "Boss rages", Action: "{{primary}} freezes", Outcome: "Devastation"}},
						},
					},
				},
			},
		},
	}
}

func testParty(count int) []*domain.PartyMemberState {
	party := make([]*domain.PartyMemberState, count)
	for i := 0; i < count; i++ {
		party[i] = &domain.PartyMemberState{
			UserID:      uuid.New(),
			Username:    "player" + string(rune('A'+i)),
			JobLevels:   map[string]int{"blacksmith": 5, "explorer": 10, "farmer": 3, "gambler": 7, "merchant": 2, "scholar": 8},
			IsConscious: true,
		}
	}
	return party
}

func TestEngineRun_BasicExecution(t *testing.T) {
	cfg := testConfig()
	party := testParty(3)
	engine := NewEngine(cfg, party, 42)

	result := engine.Run()

	assert.NotNil(t, result)
	assert.Greater(t, result.TotalTurns, 0)
	assert.NotEmpty(t, result.Journal)
	// First entry should be intro (turn 0)
	assert.Equal(t, 0, result.Journal[0].TurnNumber)
	assert.Equal(t, "The expedition begins.", result.Journal[0].Narrative)
}

func TestEngineRun_FatigueAccumulation(t *testing.T) {
	cfg := testConfig()
	cfg.Settings.MaxFatigue = 30 // Low fatigue cap to end quickly
	party := testParty(1)
	engine := NewEngine(cfg, party, 42)

	result := engine.Run()

	assert.False(t, result.Won)
	assert.GreaterOrEqual(t, result.FinalFatigue, cfg.Settings.MaxFatigue)
}

func TestEngineRun_MaxTurnsWin(t *testing.T) {
	cfg := testConfig()
	cfg.Settings.MaxTurns = 3
	cfg.Settings.MaxFatigue = 10000 // Very high so fatigue won't end it
	cfg.Settings.BaseFatiguePerTurn = 0
	// Make all outcomes neutral with 0 fatigue delta to guarantee we reach max turns
	for _, enc := range cfg.Encounters {
		for _, out := range enc.Outcomes {
			out.SkillPass.Effects.FatigueDelta = 0
			out.SkillFail.Effects.FatigueDelta = 0
			out.SkillPass.Effects.KOScale = 0
			out.SkillFail.Effects.KOScale = 0
		}
	}

	party := testParty(3)
	engine := NewEngine(cfg, party, 42)

	result := engine.Run()

	assert.True(t, result.Won)
	assert.Equal(t, 3, result.TotalTurns)
}

func TestEngineRun_AllKnockedOut(t *testing.T) {
	cfg := testConfig()
	cfg.Settings.MaxFatigue = 10000 // Won't end from fatigue
	// Make all outcomes cause KOs
	for _, enc := range cfg.Encounters {
		for _, out := range enc.Outcomes {
			out.SkillPass.Effects.KOScale = 5
			out.SkillFail.Effects.KOScale = 5
			out.SkillPass.Effects.FatigueDelta = 0
			out.SkillFail.Effects.FatigueDelta = 0
		}
	}

	party := testParty(2)
	engine := NewEngine(cfg, party, 42)

	result := engine.Run()

	assert.True(t, result.AllKnockedOut)
	assert.False(t, result.Won)
}

func TestFindMaxJobLevel(t *testing.T) {
	party := testParty(2)
	party[0].JobLevels = map[string]int{"blacksmith": 15, "explorer": 3}
	party[1].JobLevels = map[string]int{"blacksmith": 5, "explorer": 20}

	max := FindMaxJobLevel(party)
	assert.Equal(t, 20, max)
}

func TestFindMaxJobLevel_AllZero(t *testing.T) {
	party := []*domain.PartyMemberState{
		{JobLevels: map[string]int{"blacksmith": 0}},
	}
	max := FindMaxJobLevel(party)
	assert.Equal(t, 1, max) // Minimum 1 to avoid division by zero
}

func TestResolveSkillCheck_HighLevel_AlwaysPass(t *testing.T) {
	party := []*domain.PartyMemberState{
		{
			UserID:      uuid.New(),
			Username:    "tank",
			JobLevels:   map[string]int{"blacksmith": 100},
			IsConscious: true,
		},
	}
	maxLevel := 100

	passCount := 0
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		passed, member := ResolveSkillCheck(rng, domain.SkillFortitude, party, maxLevel, 0.3)
		if passed {
			passCount++
		}
		assert.NotNil(t, member)
	}

	// With contribution = 100/100 = 1.0, all checks should pass
	assert.Equal(t, 1000, passCount)
}

func TestResolveSkillCheck_ZeroLevel_AlwaysFail(t *testing.T) {
	party := []*domain.PartyMemberState{
		{
			UserID:      uuid.New(),
			Username:    "noob",
			JobLevels:   map[string]int{"blacksmith": 0},
			IsConscious: true,
		},
	}
	maxLevel := 10

	passCount := 0
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		passed, _ := ResolveSkillCheck(rng, domain.SkillFortitude, party, maxLevel, 0.3)
		if passed {
			passCount++
		}
	}

	assert.Equal(t, 0, passCount)
}

func TestResolveSkillCheck_Distribution(t *testing.T) {
	party := []*domain.PartyMemberState{
		{UserID: uuid.New(), Username: "A", JobLevels: map[string]int{"blacksmith": 10}, IsConscious: true},
		{UserID: uuid.New(), Username: "B", JobLevels: map[string]int{"blacksmith": 5}, IsConscious: true},
	}
	maxLevel := 15 // max across all jobs

	actCounts := map[string]int{}
	passCount := 0
	rng := rand.New(rand.NewSource(42))
	iterations := 10000

	for i := 0; i < iterations; i++ {
		passed, member := ResolveSkillCheck(rng, domain.SkillFortitude, party, maxLevel, 0.0)
		if passed {
			passCount++
		}
		if member != nil {
			actCounts[member.Username]++
		}
	}

	// Expected pass rate: (10/15) + (5/15) = 1.0 => always pass
	assert.Equal(t, iterations, passCount)
	// A should act ~2/3 of the time, B ~1/3
	assert.InDelta(t, float64(iterations)*2.0/3.0, float64(actCounts["A"]), float64(iterations)*0.05)
	assert.InDelta(t, float64(iterations)*1.0/3.0, float64(actCounts["B"]), float64(iterations)*0.05)
}

func TestResolveSkillCheck_TempSkillBonus(t *testing.T) {
	party := []*domain.PartyMemberState{
		{
			UserID:      uuid.New(),
			Username:    "scout",
			JobLevels:   map[string]int{"explorer": 3},
			IsConscious: true,
			TempSkills:  []domain.ExpeditionSkill{domain.SkillPerception},
		},
	}
	maxLevel := 10

	// Without temp bonus: contribution = 3/10 = 0.3
	// With temp bonus 0.3: contribution = 0.6
	passCount := 0
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 10000; i++ {
		passed, _ := ResolveSkillCheck(rng, domain.SkillPerception, party, maxLevel, 0.3)
		if passed {
			passCount++
		}
	}

	// Should pass ~60% of the time
	assert.InDelta(t, 6000, passCount, 300)
}

func TestConsumeTemporarySkill(t *testing.T) {
	member := &domain.PartyMemberState{
		TempSkills: []domain.ExpeditionSkill{domain.SkillPerception, domain.SkillCunning},
	}

	ConsumeTemporarySkill(member, domain.SkillPerception)
	assert.Equal(t, 1, len(member.TempSkills))
	assert.Equal(t, domain.SkillCunning, member.TempSkills[0])

	// Consuming a skill that doesn't exist is a no-op
	ConsumeTemporarySkill(member, domain.SkillFortitude)
	assert.Equal(t, 1, len(member.TempSkills))
}

func TestScaleEffect(t *testing.T) {
	tests := []struct {
		name      string
		base      int
		partySize int
		divisor   int
		expected  int
	}{
		{"1 member, divisor 3", 1, 1, 3, 1},
		{"3 members, divisor 3", 1, 3, 3, 1},
		{"4 members, divisor 3", 1, 4, 3, 2},
		{"6 members, divisor 3", 2, 6, 3, 4},
		{"9 members, divisor 3", 1, 9, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScaleEffect(tt.base, tt.partySize, tt.divisor)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_DebuffMechanics(t *testing.T) {
	cfg := testConfig()
	cfg.Settings.MaxTurns = 1
	cfg.Settings.MaxFatigue = 10000
	cfg.Settings.BaseFatiguePerTurn = 0

	// Make all outcomes have debuff_primary
	for _, enc := range cfg.Encounters {
		for _, out := range enc.Outcomes {
			out.SkillPass.Effects.DebuffPrimary = true
			out.SkillFail.Effects.DebuffPrimary = true
			out.SkillPass.Effects.FatigueDelta = 0
			out.SkillFail.Effects.FatigueDelta = 0
			out.SkillPass.Effects.KOScale = 0
			out.SkillFail.Effects.KOScale = 0
		}
	}

	party := testParty(1)
	engine := NewEngine(cfg, party, 42)

	result := engine.Run()
	require.NotEmpty(t, result.Journal)

	// After the first turn, the member should be debuffed
	assert.True(t, party[0].IsDebuffed)
}

func TestEngine_Rewards(t *testing.T) {
	cfg := testConfig()
	cfg.Settings.MaxTurns = 2
	cfg.Settings.MaxFatigue = 10000
	cfg.Settings.BaseFatiguePerTurn = 0
	// Neutralize effects to ensure we reach max turns
	for _, enc := range cfg.Encounters {
		for _, out := range enc.Outcomes {
			out.SkillPass.Effects.FatigueDelta = 0
			out.SkillFail.Effects.FatigueDelta = 0
			out.SkillPass.Effects.KOScale = 0
			out.SkillFail.Effects.KOScale = 0
		}
	}

	party := testParty(3)
	engine := NewEngine(cfg, party, 42)

	result := engine.Run()

	assert.True(t, result.Won)
	assert.Equal(t, len(party), len(result.PartyRewards))

	// First member should be leader
	assert.True(t, result.PartyRewards[0].IsLeader)
	// Leader should have bonus reward
	hasLeaderBonus := false
	for _, item := range result.PartyRewards[0].Items {
		if item == "lootbox_tier2" {
			hasLeaderBonus = true
			break
		}
	}
	assert.True(t, hasLeaderBonus, "leader should have bonus lootbox_tier2")

	// All rewards should have positive XP
	for _, reward := range result.PartyRewards {
		assert.Greater(t, reward.XP, 0)
	}
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := testConfig()
		err := validateConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("no intro narratives", func(t *testing.T) {
		cfg := testConfig()
		cfg.IntroNarratives = nil
		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "intro narratives")
	})

	t.Run("no encounters", func(t *testing.T) {
		cfg := testConfig()
		cfg.Encounters = nil
		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no encounters")
	})

	t.Run("encounter missing narratives", func(t *testing.T) {
		cfg := testConfig()
		cfg.Encounters["explore"].Outcomes["positive"].SkillPass.Narratives = nil
		err := validateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no narratives")
	})
}

func TestFormatJournal(t *testing.T) {
	result := &domain.ExpeditionResult{
		TotalTurns:   2,
		Won:          true,
		FinalFatigue: 15,
		Journal: []domain.ExpeditionTurn{
			{TurnNumber: 0, Narrative: "The expedition begins."},
			{TurnNumber: 1, Narrative: "A cache found. Player opens it. Supplies found."},
			{TurnNumber: 2, Narrative: "Path splits. Player examines. Correct path found."},
		},
		PartyRewards: []domain.PartyMemberReward{
			{Username: "PlayerA", Money: 650, Items: []string{"lootbox_tier2", "lootbox_tier1"}, IsLeader: true},
			{Username: "PlayerB", Money: 580, Items: []string{"lootbox_tier1"}},
		},
	}

	text := FormatJournal(result)
	assert.Contains(t, text, "The expedition begins.")
	assert.Contains(t, text, "has seen all there is to see!")
	assert.Contains(t, text, "PlayerA (Leader)")
	assert.Contains(t, text, "PlayerB: 580 money")
}
