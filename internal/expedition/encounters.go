package expedition

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// rollEncounter selects an encounter type using weighted random with progressive inversion
func (e *Engine) rollEncounter() domain.EncounterType {
	consciousCount := len(e.getConsciousMembers())
	progress := float64(e.turn) / float64(e.config.Settings.MaxTurns)

	// Build weight list with progressive inversion
	type weightedEncounter struct {
		key    string
		weight float64
	}

	// Calculate inverted weights (proportional to 1/baseWeight)
	var invertedSum float64
	baseWeights := make(map[string]float64)
	for name, enc := range e.config.Encounters {
		if enc.BaseWeight > 0 {
			baseWeights[name] = enc.BaseWeight
			invertedSum += 1.0 / enc.BaseWeight
		}
	}

	candidates := make([]weightedEncounter, 0, len(e.config.Encounters))
	var totalWeight float64

	for name, enc := range e.config.Encounters {
		// Skip encounters that require more party members than conscious
		if enc.MinParty > consciousCount {
			continue
		}

		base := enc.BaseWeight
		inverted := (1.0 / base) / invertedSum // normalized inverted weight

		// Linear interpolation between base and inverted weights
		effectiveWeight := base*(1.0-progress) + inverted*progress

		if effectiveWeight > 0 {
			candidates = append(candidates, weightedEncounter{
				key:    name,
				weight: effectiveWeight,
			})
			totalWeight += effectiveWeight
		}
	}

	// Weighted random selection
	r := e.rng.Float64() * totalWeight
	cumulative := 0.0
	for _, c := range candidates {
		cumulative += c.weight
		if r < cumulative {
			return domain.EncounterType(c.key)
		}
	}

	// Fallback: return last candidate
	if len(candidates) > 0 {
		return domain.EncounterType(candidates[len(candidates)-1].key)
	}

	return domain.EncounterExplore
}

// rollOutcome selects an outcome type (positive/neutral/negative) using weighted random
func (e *Engine) rollOutcome(encounter *EncounterDef) domain.OutcomeType {
	type weightedOutcome struct {
		key    string
		weight float64
	}

	// Get weight shift modifier for this encounter type (shifts toward positive)
	// Positive shift_weights values increase positive outcome weight
	// and decrease negative outcome weight
	shift := 0.0
	for name := range e.config.Encounters {
		if mod, ok := e.weightMods[domain.EncounterType(name)]; ok {
			shift += mod
		}
	}
	// Normalize shift to per-encounter level
	if len(e.config.Encounters) > 0 {
		shift /= float64(len(e.config.Encounters))
	}

	outcomes := make([]weightedOutcome, 0, 3)
	var totalWeight float64

	for outcomeType, outcomeDef := range encounter.Outcomes {
		weight := outcomeDef.Weight

		// Apply shift: boost positive, reduce negative
		switch outcomeType {
		case string(domain.OutcomePositive):
			weight += shift
		case string(domain.OutcomeNegative):
			weight -= shift
		}

		if weight < 0.01 {
			weight = 0.01 // minimum weight
		}

		outcomes = append(outcomes, weightedOutcome{
			key:    outcomeType,
			weight: weight,
		})
		totalWeight += weight
	}

	// Weighted random selection
	r := e.rng.Float64() * totalWeight
	cumulative := 0.0
	for _, o := range outcomes {
		cumulative += o.weight
		if r < cumulative {
			return domain.OutcomeType(o.key)
		}
	}

	// Fallback
	if len(outcomes) > 0 {
		return domain.OutcomeType(outcomes[len(outcomes)-1].key)
	}

	return domain.OutcomeNeutral
}
