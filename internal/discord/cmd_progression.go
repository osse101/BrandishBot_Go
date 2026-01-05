package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// VoteCommand returns the vote command definition and handler
func VoteCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "vote",
		Description: "Vote for a progression node unlock",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "node",
				Description: "Node key to vote for (e.g., feature_buy, item_money)",
				Required:    true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		user := i.Member.User
		if user == nil {
			user = i.User
		}

		options := i.ApplicationCommandData().Options
		nodeKey := options[0].StringValue()

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.VoteForNode(domain.PlatformDiscord, user.ID, user.Username, nodeKey)
		if err != nil {
			slog.Error("Failed to vote", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "✅ Vote Recorded",
			Description: msg,
			Color:       0x3498db, // Blue
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send response", "error", err)
		}
	}

	return cmd, handler
}

// AdminUnlockCommand returns the admin unlock command definition and handler
func AdminUnlockCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-unlock",
		Description: "[Admin] Force unlock a progression node",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "node",
				Description: "Node key to unlock (e.g., feature_buy)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "level",
				Description: "Level to unlock (default: 1)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		options := i.ApplicationCommandData().Options
		nodeKey := options[0].StringValue()
		level := 1
		if len(options) > 1 {
			level = int(options[1].IntValue())
		}

		msg, err := client.AdminUnlockNode(nodeKey, level)
		if err != nil {
			slog.Error("Failed to unlock node", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to unlock: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🔓 Admin Unlock",
			Description: msg,
			Color:       0xe67e22, // Orange
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Admin",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send response", "error", err)
		}
	}

	return cmd, handler
}

// UnlockProgressCommand returns the unlock progress command handler
func UnlockProgressCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "unlock-progress",
		Description: "View progress towards the next community unlock",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		progress, err := client.GetUnlockProgress()
		if err != nil {
			slog.Error("Failed to get unlock progress", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		if progress == nil {
			slog.Info("No active unlock progress")
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &[]string{"No active unlock progress."}[0],
			}); err != nil {
				slog.Error("Failed to send response", "error", err)
			}
			return
		}

		p := *progress
		nodeName, _ := p["target_node_name"].(string)
		targetLevel, _ := p["target_level"].(float64) // JSON numbers are float64
		contribAcc, _ := p["contributions_accumulated"].(float64)
		targetCost, _ := p["target_unlock_cost"].(float64)
		percent, _ := p["completion_percentage"].(float64)

		description := "Current community contribution progress:"
		if nodeName != "" {
			description = fmt.Sprintf("Working towards unlocking **%s Level %.0f**", nodeName, targetLevel)
		}

		progressBar := createProgressBar(percent)

		embed := &discordgo.MessageEmbed{
			Title:       "🔓 Unlock Progress",
			Description: description,
			Color:       0x9b59b6, // Purple
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Progress",
					Value:  fmt.Sprintf("%s %.1f%%", progressBar, percent),
					Inline: false,
				},
				{
					Name:   "Contributions",
					Value:  fmt.Sprintf("%.0f / %.0f", contribAcc, targetCost),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Progression",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send response", "error", err)
		}
	}

	return cmd, handler
}

// EngagementCommand returns the engagement command handler
func EngagementCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "engagement",
		Description: "View your contribution points breakdown",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to view (optional)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		targetUser := i.Member.User
		if targetUser == nil {
			targetUser = i.User
		}

		options := i.ApplicationCommandData().Options
		if len(options) > 0 {
			targetUser = options[0].UserValue(s)
		}

		// Ensure user registered
		_, err := client.RegisterUser(targetUser.Username, targetUser.ID)
		if err != nil {
			slog.Error("Failed to register/get user", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		engagement, err := client.GetUserEngagement(targetUser.ID)
		if err != nil {
			slog.Error("Failed to get engagement", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("🌟 Engagement: %s", targetUser.Username),
			Description: fmt.Sprintf("Total Contribution Score: **%d**", engagement.TotalScore),
			Color:       0xf1c40f, // Gold
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Messages Sent", Value: fmt.Sprintf("%d", engagement.MessagesSent), Inline: true},
				{Name: "Commands Used", Value: fmt.Sprintf("%d", engagement.CommandsUsed), Inline: true},
				{Name: "Items Crafted", Value: fmt.Sprintf("%d", engagement.ItemsCrafted), Inline: true},
				{Name: "Items Used", Value: fmt.Sprintf("%d", engagement.ItemsUsed), Inline: true},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send response", "error", err)
		}
	}

	return cmd, handler
}

// VotingSessionCommand returns the voting session command handler
func VotingSessionCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "voting-session",
		Description: "View the current active voting session",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		session, err := client.GetVotingSession()
		if err != nil {
			slog.Error("Failed to get voting session", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		if session == nil {
			msg := "No active voting session currently."
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			}); err != nil {
				slog.Error("Failed to send response", "error", err)
			}
			return
		}

		var optionsList string
		for _, opt := range session.Options {
			name := "Unknown Node"
			if opt.NodeDetails != nil {
				name = opt.NodeDetails.DisplayName
			}
			optionsList += fmt.Sprintf("**%s** (Level %d) - %d votes (ID: `%s`)\n", name, opt.TargetLevel, opt.VoteCount, opt.NodeDetails.NodeKey)
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🗳️ Active Voting Session",
			Description: fmt.Sprintf("Voting ends: <t:%d:R>\n\n%s", session.VotingDeadline.Unix(), optionsList),
			Color:       0x3498db, // Blue
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Use /vote <node_key> to vote!",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send response", "error", err)
		}
	}

	return cmd, handler
}

func createProgressBar(percent float64) string {
	totalBars := 10
	filledBars := int((percent / 100) * float64(totalBars))
	if filledBars > totalBars {
		filledBars = totalBars
	}

	bar := "`["
	for i := 0; i < totalBars; i++ {
		if i < filledBars {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]`"
	return bar
}
