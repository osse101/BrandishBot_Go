package discord

import (
	"fmt"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// formatTreeStatus formats the progression tree into a readable string for Discord
func formatTreeStatus(nodes []*domain.ProgressionTreeNode) string {
	var sb strings.Builder

	// Group by locked status
	unlocked := []string{}
	locked := []string{}

	for _, node := range nodes {
		status := "🔒"
		if node.IsUnlocked {
			status = "✅"
		}

		info := fmt.Sprintf("%s **%s** (`%s`) - Current Level: %d/%d",
			status, node.DisplayName, node.NodeKey, node.UnlockedLevel, node.MaxLevel)

		if node.IsUnlocked {
			unlocked = append(unlocked, info)
		} else {
			locked = append(locked, info)
		}
	}

	if len(unlocked) > 0 {
		sb.WriteString("**Unlocked Nodes**\n")
		sb.WriteString(strings.Join(unlocked, "\n"))
		sb.WriteString("\n\n")
	}

	if len(locked) > 0 {
		sb.WriteString("**Locked Nodes**\n")
		sb.WriteString(strings.Join(locked, "\n"))
	}

	return sb.String()
}

// formatVotingOptions formats voting session options into a readable string
// Format: "display_name(target_level) - Unlock Cost: unlock_cost Votes: vote_count |"
// Target level is omitted if it is 1
func formatVotingOptions(options []domain.ProgressionVotingOption) string {
	var formatted []string

	for _, opt := range options {
		if opt.NodeDetails == nil {
			continue
		}

		// Build level suffix - omit if target_level is 1
		levelStr := ""
		if opt.TargetLevel != 1 {
			levelStr = fmt.Sprintf("(%d)", opt.TargetLevel)
		}

		line := fmt.Sprintf("%s%s - Unlock Cost: %d Votes: %d |",
			opt.NodeDetails.DisplayName, levelStr, opt.NodeDetails.UnlockCost, opt.VoteCount)
		formatted = append(formatted, line)
	}

	return strings.Join(formatted, " ")
}
