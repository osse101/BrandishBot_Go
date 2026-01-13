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
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)

		// Ensure user exists
		domainUser, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		// Get inventory to calculate net worth
		inventory, err := client.GetInventory("discord", user.ID, user.Username, "")
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

// CheckTimeoutCommand returns the check timeout command definition and handler
func CheckTimeoutCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "check-timeout",
		Description: "Check if a user (or yourself) is timed out",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to check (optional)",
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
		var targetUser *discordgo.User

		if len(options) > 0 {
			targetUser = options[0].UserValue(s)
		} else {
			if i.Member != nil {
				targetUser = i.Member.User
			} else {
				targetUser = i.User
			}
		}

		isTimedOut, remainingSeconds, err := client.GetUserTimeout(targetUser.Username)
		if err != nil {
			slog.Error("Failed to check timeout", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		var description string
		var color int
		if isTimedOut {
			description = fmt.Sprintf("🔴 **%s** is currently timed out.\nRemaining time: **%.0fs**", targetUser.Username, remainingSeconds)
			color = 0xe74c3c // Red
		} else {
			description = fmt.Sprintf("🟢 **%s** is NOT timed out.", targetUser.Username)
			color = 0x2ecc71 // Green
		}

		embed := &discordgo.MessageEmbed{
			Title:       "⏱️ Timeout Status",
			Description: description,
			Color:       color,
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
