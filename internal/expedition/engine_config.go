package expedition

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// EncounterConfig represents the full encounter configuration
type EncounterConfig struct {
	Version         string                   `json:"version"`
	Settings        EngineSettings           `json:"settings"`
	IntroNarratives []string                 `json:"intro_narratives"`
	Encounters      map[string]*EncounterDef `json:"encounters"`
}

// EngineSettings holds tunable parameters for the expedition engine
type EngineSettings struct {
	BaseFatiguePerTurn int     `json:"base_fatigue_per_turn"`
	MaxFatigue         int     `json:"max_fatigue"`
	MaxTurns           int     `json:"max_turns"`
	StartingPurse      int     `json:"starting_purse"`
	SkillCheckBonus    int     `json:"skill_check_bonus_money"`
	XPFormulaDivisor   int     `json:"xp_formula_divisor"`
	LeaderBonusReward  string  `json:"leader_bonus_reward"`
	WinBonusReward     string  `json:"win_bonus_reward"`
	WinBonusMoney      int     `json:"win_bonus_money"`
	PartyScaleDivisor  int     `json:"party_scale_divisor"`
	TempSkillBonus     float64 `json:"temp_skill_bonus"`
}

// EncounterDef defines a single encounter type
type EncounterDef struct {
	DisplayName string                   `json:"display_name"`
	BaseWeight  float64                  `json:"base_weight"`
	Skills      []domain.ExpeditionSkill `json:"skills"`
	MinParty    int                      `json:"min_party"`
	Outcomes    map[string]*OutcomeDef   `json:"outcomes"`
}

// OutcomeDef defines one outcome category (positive/neutral/negative)
type OutcomeDef struct {
	Weight    float64        `json:"weight"`
	SkillPass *OutcomeDetail `json:"skill_pass"`
	SkillFail *OutcomeDetail `json:"skill_fail"`
}

// OutcomeDetail defines effects and narratives for a specific skill pass/fail
type OutcomeDetail struct {
	Effects    EffectsDef     `json:"effects"`
	Narratives []NarrativeDef `json:"narratives"`
}

// EffectsDef defines the effects of an encounter outcome
type EffectsDef struct {
	FatigueDelta  int     `json:"fatigue_delta"`
	PurseDelta    int     `json:"purse_delta"`
	Reward        string  `json:"reward"`
	KOScale       int     `json:"ko_scale"`
	ReviveScale   int     `json:"revive_scale"`
	DebuffPrimary bool    `json:"debuff_primary"`
	TempSkill     string  `json:"temp_skill"`
	ShiftWeights  float64 `json:"shift_weights"`
}

// NarrativeDef defines the 3-part narrative for an encounter outcome
type NarrativeDef struct {
	Surprise string `json:"surprise"`
	Action   string `json:"action"`
	Outcome  string `json:"outcome"`
}

// LoadEncounterConfig loads and validates the encounter configuration from a JSON file
func LoadEncounterConfig(path string) (*EncounterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read encounter config: %w", err)
	}

	var config EncounterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse encounter config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid encounter config: %w", err)
	}

	return &config, nil
}

func validateConfig(cfg *EncounterConfig) error {
	if len(cfg.IntroNarratives) == 0 {
		return fmt.Errorf("no intro narratives defined")
	}

	if len(cfg.Encounters) == 0 {
		return fmt.Errorf("no encounters defined")
	}

	if cfg.Settings.MaxTurns <= 0 {
		return fmt.Errorf("max_turns must be positive")
	}

	if cfg.Settings.MaxFatigue <= 0 {
		return fmt.Errorf("max_fatigue must be positive")
	}

	if cfg.Settings.PartyScaleDivisor <= 0 {
		return fmt.Errorf("party_scale_divisor must be positive")
	}

	// Validate each encounter
	for name, enc := range cfg.Encounters {
		if len(enc.Skills) == 0 {
			return fmt.Errorf("encounter %q has no skills", name)
		}

		if len(enc.Outcomes) == 0 {
			return fmt.Errorf("encounter %q has no outcomes", name)
		}

		// Validate outcome weights sum approximately to 1.0
		weightSum := 0.0
		for outcomeType, outcome := range enc.Outcomes {
			weightSum += outcome.Weight

			if outcome.SkillPass == nil || outcome.SkillFail == nil {
				return fmt.Errorf("encounter %q outcome %q missing skill_pass or skill_fail", name, outcomeType)
			}

			if len(outcome.SkillPass.Narratives) == 0 {
				return fmt.Errorf("encounter %q outcome %q skill_pass has no narratives", name, outcomeType)
			}

			if len(outcome.SkillFail.Narratives) == 0 {
				return fmt.Errorf("encounter %q outcome %q skill_fail has no narratives", name, outcomeType)
			}
		}

		if math.Abs(weightSum-1.0) > 0.01 {
			return fmt.Errorf("encounter %q outcome weights sum to %.2f (expected ~1.0)", name, weightSum)
		}
	}

	return nil
}
