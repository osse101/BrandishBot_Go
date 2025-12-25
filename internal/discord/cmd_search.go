package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// SearchCommand returns the search command definition and handler
func SearchCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "search",
		Description: "Search for items",
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

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &[]string{"Error connecting to game server."}[0],
			}); err != nil {
				slog.Error("Failed to edit interaction response", "error", err)
			}
			return
		}

		msg, err := client.Search(domain.PlatformDiscord, user.ID, user.Username)
		if err != nil {
			slog.Error("Failed to search", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Search Result",
			Description: msg,
			Color:       0x3498db, // Blue
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send search results", "error", err)
		}
	}

	return cmd, handler
}
