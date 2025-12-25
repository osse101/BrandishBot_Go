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
