package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// platformChoices returns the common platform choices for Discord commands
func platformChoices() []*discordgo.ApplicationCommandOptionChoice {
	return []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Twitch", Value: domain.PlatformTwitch},
		{Name: "YouTube", Value: domain.PlatformYoutube},
		{Name: "Discord", Value: domain.PlatformDiscord},
	}
}

// AdminTimeoutClearCommand returns the timeout-clear command definition and handler (admin only)
func AdminTimeoutClearCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "timeout-clear",
		Description: "[ADMIN] Clear a user's timeout",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "platform",
				Description: "Platform the user is on",
				Required:    true,
				Choices:     platformChoices(),
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "username",
				Description: "Username to clear timeout for",
				Required:    true,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		platform := options[0].StringValue()
		username := options[1].StringValue()

		msg, err := client.AdminClearTimeout(platform, username)
		if err != nil {
			slog.Error("Failed to clear timeout", "error", err, "platform", platform, "username", username)
			respondError(s, i, fmt.Sprintf("Failed to clear timeout: %v", err))
			return
		}

		embed := createEmbed(
			"Timeout Cleared",
			fmt.Sprintf("Cleared timeout for **%s** on **%s**\n\n%s", username, platform, msg),
			0x2ecc71,
			FooterAdminAction,
		)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// AdminSetTimeoutCommand returns the timeout-set command definition and handler (admin only)
func AdminSetTimeoutCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "timeout-set",
		Description: "[ADMIN] Set or extend a user's timeout",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "platform",
				Description: "Platform the user is on",
				Required:    true,
				Choices:     platformChoices(),
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "username",
				Description: "Username to timeout",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "duration",
				Description: "Timeout duration in seconds (max 86400)",
				Required:    true,
				MinValue:    &[]float64{1}[0],
				MaxValue:    86400,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "Reason for the timeout",
				Required:    false,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		platform := options[0].StringValue()
		username := options[1].StringValue()
		duration := int(options[2].IntValue())

		reason := ""
		if len(options) > 3 {
			reason = options[3].StringValue()
		}

		msg, err := client.SetUserTimeout(platform, username, duration, reason)
		if err != nil {
			slog.Error("Failed to set timeout", "error", err, "platform", platform, "username", username)
			respondError(s, i, fmt.Sprintf("Failed to set timeout: %v", err))
			return
		}

		embed := createEmbed(
			"Timeout Applied",
			fmt.Sprintf("Applied %d second timeout to **%s** on **%s**\n\n%s", duration, username, platform, msg),
			0xe74c3c,
			FooterAdminAction,
		)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
