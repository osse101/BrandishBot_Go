package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GambleStartCommand returns the gamble start command definition and handler
func GambleStartCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "gamble-start",
		Description: "Start a new gamble with lootbox items",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "item",
				Description:  "Lootbox item to wager (e.g., lootbox, goldbox)",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Number of items to wager",
				Required:    true,
				MinValue:    &[]float64{1}[0],
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
		quantity := int(options[1].IntValue())

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.StartGamble(domain.PlatformDiscord, user.ID, user.Username, itemName, quantity)
		if err != nil {
			slog.Error("Failed to start gamble", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🎲 Gamble Started!",
			Description: msg,
			Color:       0xe74c3c, // Red
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

// GambleJoinCommand returns the gamble join command definition and handler
func GambleJoinCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "gamble-join",
		Description: "Join an active gamble with lootbox items",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "gamble-id",
				Description: "ID of the gamble to join",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "item",
				Description:  "Lootbox item to wager (e.g., lootbox, goldbox)",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Number of items to wager",
				Required:    true,
				MinValue:    &[]float64{1}[0],
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
		gambleID := options[0].StringValue()
		itemName := options[1].StringValue()
		quantity := int(options[2].IntValue())

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.JoinGamble(domain.PlatformDiscord, user.ID, user.Username, gambleID, itemName, quantity)
		if err != nil {
			slog.Error("Failed to join gamble", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🎲 Joined Gamble!",
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
