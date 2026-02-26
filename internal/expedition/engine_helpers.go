package expedition

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

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
