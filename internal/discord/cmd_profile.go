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

		// Get inventory to show item count
		inventory, err := client.GetInventory("discord", user.ID, user.Username)
		var itemCount int
		if err == nil {
			itemCount = len(inventory)
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's Profile", user.Username),
			Description: "Your BrandishBot profile",
			Color:       0x00ff00,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: user.AvatarURL(""),
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "User ID",
					Value:  domainUser.ID,
					Inline: true,
				},
				{
					Name:   "Items",
					Value:  fmt.Sprintf("%d", itemCount),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Use /stats for detailed statistics",
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
