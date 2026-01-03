package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// AddItemCommand returns the add item command definition and handler (admin only)
func AddItemCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "add-item",
		Description: "[ADMIN] Add items to a user's inventory",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to add item to",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to add",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to add",
				Required:    true,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		options := i.ApplicationCommandData().Options
		targetUser := options[0].UserValue(s)
		itemName := options[1].StringValue()
		quantity := int(options[2].IntValue())

		// Ensure target user exists
		_, err := client.RegisterUser(targetUser.Username, targetUser.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.AddItem(domain.PlatformDiscord, targetUser.ID, itemName, quantity)
		if err != nil {
			slog.Error("Failed to add item", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to add item: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "✅ Items Added",
			Description: fmt.Sprintf("Added %d x %s to %s\n\n%s", quantity, itemName, targetUser.Username, msg),
			Color:       0x2ecc71, // Green
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Admin Action",
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

// RemoveItemCommand returns the remove item command definition and handler (admin only)
func RemoveItemCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "remove-item",
		Description: "[ADMIN] Remove items from a user's inventory",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to remove item from",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to remove",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to remove",
				Required:    true,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		options := i.ApplicationCommandData().Options
		targetUser := options[0].UserValue(s)
		itemName := options[1].StringValue()
		quantity := int(options[2].IntValue())

		// Ensure target user exists
		_, err := client.RegisterUser(targetUser.Username, targetUser.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.RemoveItem(domain.PlatformDiscord, targetUser.ID, itemName, quantity)
		if err != nil {
			slog.Error("Failed to remove item", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to remove item: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🗑️ Items Removed",
			Description: fmt.Sprintf("Removed %d x %s from %s\n\n%s", quantity, itemName, targetUser.Username, msg),
			Color:       0xe74c3c, // Red
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Admin Action",
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

// AdminAwardXPCommand returns the award XP command definition and handler (admin only)
func AdminAwardXPCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-award-xp",
		Description: "[ADMIN] Award job XP to a user",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "platform",
				Description: "Platform (discord, twitch, youtube)",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Discord", Value: domain.PlatformDiscord},
					{Name: "Twitch", Value: domain.PlatformTwitch},
					{Name: "YouTube", Value: domain.PlatformYoutube},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "username",
				Description: "Username on the specified platform",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "job",
				Description:  "Job to award XP to",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of XP to award (1-10000)",
				Required:    true,
				MinValue:    floatPtr(1.0),
				MaxValue:    10000.0,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		options := i.ApplicationCommandData().Options
		platform := options[0].StringValue()
		username := options[1].StringValue()
		jobKey := options[2].StringValue()
		amount := int(options[3].IntValue())

		// Call API to award XP
		result, err := client.AdminAwardXP(platform, username, jobKey, amount)
		if err != nil {
			slog.Error("Failed to award XP", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to award XP: %v", err))
			return
		}

		// Build response message
		description := fmt.Sprintf("Awarded **%d XP** to **%s** (@%s) for job **%s**",
			amount, platform, username, jobKey)

		if result.LeveledUp {
			description += fmt.Sprintf("\n\n🎉 **Level Up!** %s → %d",
				jobKey, result.NewLevel)
		}

		embed := &discordgo.MessageEmbed{
			Title:       "✅ XP Awarded",
			Description: description,
			Color:       0x3498db, // Blue
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Platform",
					Value:  platform,
					Inline: true,
				},
				{
					Name:   "Username",
					Value:  username,
					Inline: true,
				},
				{
					Name:   "Job",
					Value:  jobKey,
					Inline: true,
				},
				{
					Name:   "XP Awarded",
					Value:  fmt.Sprintf("%d", amount),
					Inline: true,
				},
				{
					Name:   "New Level",
					Value:  fmt.Sprintf("%d", result.NewLevel),
					Inline: true,
				},
				{
					Name:   "Total XP",
					Value:  fmt.Sprintf("%d", result.NewXP),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Admin Action by %s", i.Member.User.Username),
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

func floatPtr(v float64) *float64 {
	return &v
}
