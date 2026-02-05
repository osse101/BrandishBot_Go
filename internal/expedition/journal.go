package expedition

import (
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// buildNarrative generates the 3-part narrative for an encounter turn
func (e *Engine) buildNarrative(detail *OutcomeDetail, actingMember *domain.PartyMemberState) string {
	if len(detail.Narratives) == 0 {
		return "The expedition continues..."
	}

	// Pick a random narrative from the available options
	narr := detail.Narratives[e.rng.Intn(len(detail.Narratives))]

	// Build the 3-part narrative
	parts := make([]string, 0, 3)

	if narr.Surprise != "" {
		parts = append(parts, narr.Surprise)
	}

	if narr.Action != "" {
		action := replacePlaceholders(narr.Action, actingMember, e.party, e.rng)
		parts = append(parts, action)
	}

	if narr.Outcome != "" {
		outcome := replacePlaceholders(narr.Outcome, actingMember, e.party, e.rng)
		parts = append(parts, outcome)
	}

	return strings.Join(parts, ". ") + "."
}

// replacePlaceholders replaces {{primary}} and {{secondary}} with member names
func replacePlaceholders(text string, primary *domain.PartyMemberState, party []*domain.PartyMemberState, rng interface{ Intn(int) int }) string {
	if primary != nil {
		text = strings.ReplaceAll(text, "{{primary}}", primary.Username)
	}

	// {{secondary}} = random other conscious member
	if strings.Contains(text, "{{secondary}}") {
		secondary := pickSecondary(primary, party, rng)
		if secondary != nil {
			text = strings.ReplaceAll(text, "{{secondary}}", secondary.Username)
		} else {
			text = strings.ReplaceAll(text, "{{secondary}}", "a companion")
		}
	}

	return text
}

func pickSecondary(exclude *domain.PartyMemberState, party []*domain.PartyMemberState, rng interface{ Intn(int) int }) *domain.PartyMemberState {
	candidates := make([]*domain.PartyMemberState, 0)
	for _, m := range party {
		if m != exclude && m.IsConscious {
			candidates = append(candidates, m)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[rng.Intn(len(candidates))]
}
