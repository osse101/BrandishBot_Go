package expedition

import (
	"fmt"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// FormatJournal formats the complete journal as plain text
func FormatJournal(result *domain.ExpeditionResult) string {
	var sb strings.Builder

	for i, turn := range result.Journal {
		if i == 0 {
			// Intro narrative
			sb.WriteString(turn.Narrative)
			sb.WriteString("\n---\n")
			continue
		}

		sb.WriteString(turn.Narrative)
		sb.WriteString("\n")
	}

	// Summary
	sb.WriteString("---\n")
	if result.Won {
		sb.WriteString("The expedition has seen all there is to see!\n")
	} else if result.AllKnockedOut {
		sb.WriteString("The entire party was knocked out!\n")
	} else {
		sb.WriteString("The party collapsed from exhaustion.\n")
	}

	sb.WriteString(fmt.Sprintf("Turns: %d | Final Fatigue: %d\n", result.TotalTurns, result.FinalFatigue))

	// Rewards
	sb.WriteString("---\nRewards:\n")
	for _, reward := range result.PartyRewards {
		leaderTag := ""
		if reward.IsLeader {
			leaderTag = " (Leader)"
		}

		items := "none"
		if len(reward.Items) > 0 {
			items = strings.Join(reward.Items, ", ")
		}

		sb.WriteString(fmt.Sprintf("%s%s: %d money, %s\n", reward.Username, leaderTag, reward.Money, items))
	}

	return sb.String()
}

// JournalEntry is a structured entry for SSE streaming
type JournalEntry struct {
	TurnNumber    int    `json:"turn_number"`
	EncounterType string `json:"encounter_type,omitempty"`
	Outcome       string `json:"outcome,omitempty"`
	SkillChecked  string `json:"skill_checked,omitempty"`
	SkillPassed   bool   `json:"skill_passed"`
	PrimaryMember string `json:"primary_member,omitempty"`
	Narrative     string `json:"narrative"`
	Fatigue       int    `json:"fatigue"`
	Purse         int    `json:"purse"`
}

// FormatJournalEntries converts expedition turns to structured entries for SSE
func FormatJournalEntries(result *domain.ExpeditionResult) []JournalEntry {
	entries := make([]JournalEntry, 0, len(result.Journal))
	for _, turn := range result.Journal {
		entries = append(entries, JournalEntry{
			TurnNumber:    turn.TurnNumber,
			EncounterType: string(turn.EncounterType),
			Outcome:       string(turn.Outcome),
			SkillChecked:  string(turn.SkillChecked),
			SkillPassed:   turn.SkillPassed,
			PrimaryMember: turn.PrimaryMember,
			Narrative:     turn.Narrative,
			Fatigue:       turn.Fatigue,
			Purse:         turn.PurseAfter,
		})
	}
	return entries
}
