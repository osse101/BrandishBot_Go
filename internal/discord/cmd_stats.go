package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// LeaderboardCommand returns the leaderboard command definition and handler
func LeaderboardCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "leaderboard",
		Description: "View top players on the leaderboard",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "metric",
				Description: "Metric to rank by (default: engagement_score)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Engagement Score", Value: "engagement_score"},
					{Name: "Total Money", Value: "money"},
					{Name: "Items Found", Value: "items"},
					{Name: "Messages Sent", Value: "messages"},
					{Name: "Contribution Points", Value: "contribution"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "limit",
				Description: "Number of top players to show (default: 10)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		metric := "engagement_score"
		limit := 10

		for _, opt := range options {
			switch opt.Name {
			case "metric":
				metric = opt.StringValue()
			case "limit":
				limit = int(opt.IntValue())
			}
		}

		var msg string
		var err error

		if metric == "contribution" {
			msg, err = client.GetContributionLeaderboard(limit)
		} else {
			msg, err = client.GetLeaderboard(metric, limit)
		}

		if err != nil {
			slog.Error("Failed to get leaderboard", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🏆 Leaderboard",
			Description: msg,
			Color:       0x1abc9c, // Teal
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

// StatsCommand returns the stats command definition and handler
func StatsCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "stats",
		Description: "View user statistics",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to view stats for (default: yourself)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)
		options := getOptions(i)
		if len(options) > 0 {
			user = options[0].UserValue(s)
		}

		// Ensure user exists
		if !ensureUserRegistered(s, i, client, user, false) {
			return
		}

		msg, err := client.GetUserStats(domain.PlatformDiscord, user.ID)
		if err != nil {
			slog.Error("Failed to get stats", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := createEmbed(fmt.Sprintf("📊 Stats for %s", user.Username), msg, 0x3498db, "")
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
