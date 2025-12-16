package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// BuyCommand returns the buy command definition and handler
func BuyCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "buy",
		Description: "Purchase an item from the shop",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to buy",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to buy (default: 1)",
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

		user := i.Member.User
		if user == nil {
			user = i.User
		}

		options := i.ApplicationCommandData().Options
		itemName := options[0].StringValue()
		quantity := 1
		if len(options) > 1 {
			quantity = int(options[1].IntValue())
		}

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.BuyItem(domain.PlatformDiscord, user.ID, user.Username, itemName, quantity)
		if err != nil {
			slog.Error("Failed to buy item", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to buy item: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "💰 Purchase Complete",
			Description: msg,
			Color:       0x2ecc71, // Green
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

// SellCommand returns the sell command definition and handler
func SellCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "sell",
		Description: "Sell an item from your inventory",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to sell",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to sell (default: 1)",
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

		user := i.Member.User
		if user == nil {
			user = i.User
		}

		options := i.ApplicationCommandData().Options
		itemName := options[0].StringValue()
		quantity := 1
		if len(options) > 1 {
			quantity = int(options[1].IntValue())
		}

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.SellItem(domain.PlatformDiscord, user.ID, user.Username, itemName, quantity)
		if err != nil {
			slog.Error("Failed to sell item", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to sell item: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "💵 Sale Complete",
			Description: msg,
			Color:       0xf39c12, // Orange
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

// PricesCommand returns the prices command definition and handler
func PricesCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "prices",
		Description: "View current market prices for buyable items",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		msg, err := client.GetPrices()
		if err != nil {
			slog.Error("Failed to get prices", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to get prices: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🏪 Market Prices",
			Description: msg,
			Color:       0xf1c40f, // Yellow
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

// GiveCommand returns the give command definition and handler
func GiveCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "give",
		Description: "Give an item to another user",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to give item to",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to give",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to give (default: 1)",
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

		fromUser := i.Member.User
		if fromUser == nil {
			fromUser = i.User
		}

		options := i.ApplicationCommandData().Options
		toUser := options[0].UserValue(s)
		itemName := options[1].StringValue()
		quantity := 1
		if len(options) > 2 {
			quantity = int(options[2].IntValue())
		}

		// Ensure users exist
		_, err := client.RegisterUser(fromUser.Username, fromUser.ID)
		if err != nil {
			slog.Error("Failed to register from user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		_, err = client.RegisterUser(toUser.Username, toUser.ID)
		if err != nil {
			slog.Error("Failed to register to user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.GiveItem(
			domain.PlatformDiscord, fromUser.ID,
			domain.PlatformDiscord, toUser.ID, toUser.Username,
			itemName, quantity,
		)
		if err != nil {
			slog.Error("Failed to give item", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to give item: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🎁 Gift Sent",
			Description: msg,
			Color:       0xe91e63, // Pink
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
