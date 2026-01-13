package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// JobBonusCommand returns the job bonus command definition and handler
func JobBonusCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "job-bonus",
		Description: "Checkactive bonus for a specific job",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "job",
				Description:  "Job key (e.g., miner, warrior)",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "bonus_type",
				Description: "Type of bonus to check",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Drop Rate",
						Value: "drop_rate",
					},
					{
						Name:  "XP Gain",
						Value: "xp_gain",
					},
					{
						Name:  "Gold Gain",
						Value: "gold_gain",
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to check (optional, defaults to self)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}
		options := i.ApplicationCommandData().Options
		jobKey := options[0].StringValue()
		bonusType := options[1].StringValue()

		var targetUser *discordgo.User
		if len(options) > 2 {
			targetUser = options[2].UserValue(s)
		} else {
			targetUser = i.Interaction.Member.User
		}

		// Ensure user is registered/known
		_, err := client.RegisterUser(targetUser.Username, targetUser.ID)
		if err != nil {
			slog.Error("Failed to register/get user", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		bonusVal, err := client.GetJobBonus(targetUser.ID, jobKey, bonusType)
		if err != nil {
			slog.Error("Failed to get job bonus", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		// Format bonus value (percentage usually)
		bonusDisplay := fmt.Sprintf("%d%%", bonusVal)

		embed := &discordgo.MessageEmbed{
			Title:       "✨ Job Bonus",
			Description: fmt.Sprintf("Active **%s** bonus for **%s** as **%s**:", strings.ReplaceAll(bonusType, "_", " "), targetUser.Username, jobKey),
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Value",
					Value:  bonusDisplay,
					Inline: true,
				},
			},
			Color: 0xF1C40F, // Gold
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Jobs",
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
