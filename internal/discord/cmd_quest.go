package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// QuestsCommand returns the quests command definition and handler
func QuestsCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "quests",
		Description: "View your active weekly quests and progress",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)

		// Ensure user exists
		if !ensureUserRegistered(s, i, client, user, true) {
			return
		}

		// Get active quests
		quests, err := client.GetActiveQuests()
		if err != nil {
			slog.Error("Failed to get active quests", "error", err)
			respondFriendlyError(s, i, fmt.Sprintf("Failed to load quests: %v", err))
			return
		}

		if len(quests) == 0 {
			respondFriendlyError(s, i, "No active quests available this week.")
			return
		}

		// Get user's quest progress
		userID := user.ID
		progress, err := client.GetUserQuestProgress(userID)
		if err != nil {
			slog.Error("Failed to get user quest progress", "error", err, "user_id", userID)
			respondFriendlyError(s, i, fmt.Sprintf("Failed to load your quest progress: %v", err))
			return
		}

		// Create progress map for quick lookup
		progressMap := make(map[int]*domain.QuestProgress)
		for i := range progress {
			if progress[i].QuestID > 0 {
				progressMap[progress[i].QuestID] = &progress[i]
			}
		}

		// Build embed with quest details
		embed := &discordgo.MessageEmbed{
			Title:       "📜 Weekly Quests",
			Description: "Complete quests to earn money and Merchant XP!",
			Color:       0x3498db, // Blue
			Fields:      make([]*discordgo.MessageEmbedField, 0, len(quests)),
			Timestamp:   time.Now().Format(time.RFC3339),
		}

		for _, quest := range quests {
			qp, exists := progressMap[quest.QuestID]
			if !exists {
				// Quest not started yet, shouldn't happen but handle gracefully
				continue
			}

			// Build status indicator
			status := "🔄 In Progress"
			if qp.CompletedAt != nil && qp.ClaimedAt == nil {
				status = "✅ Ready to Claim!"
			} else if qp.ClaimedAt != nil {
				status = "🎁 Claimed"
			}

			// Build progress bar (10 segments)
			progressBar := buildProgressBar(qp.ProgressCurrent, qp.ProgressRequired, 10)

			// Build field value
			fieldValue := fmt.Sprintf(
				"%s\n%s `%d/%d`\n💰 %d money | ⭐ %d Merchant XP",
				status,
				progressBar,
				qp.ProgressCurrent,
				qp.ProgressRequired,
				qp.RewardMoney,
				qp.RewardXp,
			)

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("**%s** (ID: %d)", qp.Description, quest.QuestID),
				Value:  fieldValue,
				Inline: false,
			})
		}

		// Add footer with hint
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: "Use /claimquest to claim completed quest rewards",
		}

		editInteractionResponse(s, i, embed)
	}

	return cmd, handler
}

// ClaimQuestCommand returns the claimquest command definition and handler
func ClaimQuestCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "claimquest",
		Description: "Claim rewards from a completed weekly quest",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quest_id",
				Description: "The ID of the quest to claim (shown in /quests)",
				Required:    true,
				MinValue:    func(i float64) *float64 { return &i }(1),
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)

		// Ensure user exists
		if !ensureUserRegistered(s, i, client, user, true) {
			return
		}

		options := getOptions(i)
		questID := int(options[0].IntValue())

		// Claim quest reward
		result, err := client.ClaimQuestReward(user.ID, questID)
		if err != nil {
			slog.Error("Failed to claim quest reward", "error", err, "user_id", user.ID, "quest_id", questID)
			respondFriendlyError(s, i, fmt.Sprintf("Failed to claim quest reward: %v", err))
			return
		}

		// Extract money and XP from result
		moneyEarned := int64(0)
		xpEarned := int64(0)

		if money, ok := result["money_earned"]; ok {
			if m, okm := money.(float64); okm {
				moneyEarned = int64(m)
			}
		}

		if xp, ok := result["xp_earned"]; ok {
			if x, okx := xp.(float64); okx {
				xpEarned = int64(x)
			}
		}

		// Build response embed
		embed := &discordgo.MessageEmbed{
			Title:       "🎉 Quest Reward Claimed!",
			Description: "You've successfully claimed your quest reward!",
			Color:       0x2ecc71, // Green
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "💰 Money Earned",
					Value:  fmt.Sprintf("%d", moneyEarned),
					Inline: true,
				},
				{
					Name:   "⭐ Merchant XP Earned",
					Value:  fmt.Sprintf("%d", xpEarned),
					Inline: true,
				},
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}

		editInteractionResponse(s, i, embed)
	}

	return cmd, handler
}

// buildProgressBar creates a visual progress bar using Unicode characters
func buildProgressBar(current, required, length int) string {
	if required <= 0 {
		return strings.Repeat("░", length)
	}

	filled := (current * length) / required
	if filled > length {
		filled = length
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", length-filled)
	return "[" + bar + "]"
}

// editInteractionResponse sends a deferred response with an embed
func editInteractionResponse(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}
