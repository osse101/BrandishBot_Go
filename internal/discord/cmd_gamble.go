package discord

import (
	"fmt"
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
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)
		options := getOptions(i)
		itemName := options[0].StringValue()
		quantity := int(options[1].IntValue())

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		gambleID, err := client.StartGamble(domain.PlatformDiscord, user.ID, user.Username, itemName, quantity)
		if err != nil {
			slog.Error("Failed to start gamble", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		description := fmt.Sprintf("**Gamble ID:** `%s`\n\nOthers can join using `/gamble-join %s`\n\nThe gamble will execute shortly after the join deadline.", gambleID, gambleID)
		embed := createEmbed("🎲 Gamble Started!", description, 0xe74c3c, "")
		sendEmbed(s, i, embed)
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
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)
		options := getOptions(i)
		gambleID := options[0].StringValue()

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		msg, err := client.JoinGamble(domain.PlatformDiscord, user.ID, user.Username, gambleID)
		if err != nil {
			slog.Error("Failed to join gamble", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := createEmbed("🎲 Joined Gamble!", msg, 0x2ecc71, "")
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
