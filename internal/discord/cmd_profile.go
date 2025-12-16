package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// ProfileCommand returns the profile command definition and handler
func ProfileCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "profile",
		Description: "View your profile stats",
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

		domainUser, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &[]string{"Failed to retrieve profile. Please try again later."}[0],
			}); err != nil {
				slog.Error("Failed to edit interaction response", "error", err)
			}
			return
		}

		stats, err := client.GetUserStats(domainUser.ID)
		if err != nil {
			slog.Error("Failed to get stats", "error", err)
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &[]string{"Failed to retrieve stats."}[0],
			}); err != nil {
				slog.Error("Failed to edit interaction response", "error", err)
			}
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's Profile", user.Username),
			Description: "Here are your stats:",
			Color:       0x00ff00,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: user.AvatarURL(""),
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Total Events",
					Value:  fmt.Sprintf("%d", stats.TotalEvents),
					Inline: true,
				},
				{
					Name:   "Internal ID",
					Value:  domainUser.ID,
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send profile embed", "error", err)
		}
	}

	return cmd, handler
}
