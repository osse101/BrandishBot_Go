package expedition

import (
	"math/rand"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// SkillJobMap maps each expedition skill to its corresponding job key
var SkillJobMap = map[domain.ExpeditionSkill]string{
	domain.SkillFortitude:  "blacksmith",
	domain.SkillPerception: "explorer",
	domain.SkillSurvival:   "farmer",
	domain.SkillCunning:    "gambler",
	domain.SkillPersuasion: "merchant",
	domain.SkillKnowledge:  "scholar",
}

// SkillContribution represents one member's contribution to a skill check
type SkillContribution struct {
	Member       *domain.PartyMemberState
	Contribution float64
	CumStart     float64 // Start of this member's probability segment
	CumEnd       float64 // End of this member's probability segment
}

// FindMaxJobLevel returns the highest individual job level across all members and all jobs
func FindMaxJobLevel(members []*domain.PartyMemberState) int {
	maxLevel := 1 // minimum 1 to avoid division by zero
	for _, m := range members {
		for _, level := range m.JobLevels {
			if level > maxLevel {
				maxLevel = level
			}
		}
	}
	return maxLevel
}

// BuildSkillTable builds a probability table for a skill check
func BuildSkillTable(skill domain.ExpeditionSkill, members []*domain.PartyMemberState, maxLevel int, tempBonus float64) []SkillContribution {
	jobKey := SkillJobMap[skill]
	table := make([]SkillContribution, 0, len(members))
	cumulative := 0.0

	for _, m := range members {
		if !m.IsConscious {
			continue
		}

		contribution := float64(m.JobLevels[jobKey]) / float64(maxLevel)

		// Check for temporary skill bonus
		if hasTempSkill(m, skill) {
			contribution += tempBonus
		}

		if contribution <= 0 {
			continue
		}

		entry := SkillContribution{
			Member:       m,
			Contribution: contribution,
			CumStart:     cumulative,
			CumEnd:       cumulative + contribution,
		}
		table = append(table, entry)
		cumulative += contribution
	}

	return table
}

// ResolveSkillCheck performs a probability-based skill check
// Returns whether the check passed and which member acted
func ResolveSkillCheck(rng *rand.Rand, skill domain.ExpeditionSkill, members []*domain.PartyMemberState, maxLevel int, tempBonus float64) (passed bool, actingMember *domain.PartyMemberState) {
	table := BuildSkillTable(skill, members, maxLevel, tempBonus)
	if len(table) == 0 {
		// No conscious members with any contribution - guaranteed fail
		// Pick first conscious member as acting member for narrative
		for _, m := range members {
			if m.IsConscious {
				return false, m
			}
		}
		return false, nil
	}

	total := table[len(table)-1].CumEnd
	r := rng.Float64()

	if r >= total {
		// Fail - pick member with highest contribution for narrative
		best := table[0]
		for _, entry := range table[1:] {
			if entry.Contribution > best.Contribution {
				best = entry
			}
		}
		return false, best.Member
	}

	// Pass - find which member's segment the roll falls in
	// Scale roll to the total range
	scaledR := r
	for _, entry := range table {
		if scaledR >= entry.CumStart && scaledR < entry.CumEnd {
			return true, entry.Member
		}
	}

	// Fallback: first entry (shouldn't happen in practice)
	return true, table[0].Member
}

// ConsumeTemporarySkill removes a temporary skill from a member after use
func ConsumeTemporarySkill(member *domain.PartyMemberState, skill domain.ExpeditionSkill) {
	for i, s := range member.TempSkills {
		if s == skill {
			member.TempSkills = append(member.TempSkills[:i], member.TempSkills[i+1:]...)
			return
		}
	}
}

func hasTempSkill(member *domain.PartyMemberState, skill domain.ExpeditionSkill) bool {
	for _, s := range member.TempSkills {
		if s == skill {
			return true
		}
	}
	return false
}
