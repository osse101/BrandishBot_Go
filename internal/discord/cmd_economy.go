package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// BuyCommand returns the buy command definition and handler
func BuyCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	return CreateItemQuantityCommand(ItemCommandConfig{
		Name:        "buy",
		Description: "Purchase an item from the shop",
		OptionName:  "item",
		OptionDesc:  "Item name to buy",
		ResultTitle: "💰 Purchase Complete",
		ResultColor: 0x2ecc71,
		Action:      func(c *APIClient) func(string, string, string, string, int) (string, error) { return c.BuyItem },
	})
}

// SellCommand returns the sell command definition and handler
func SellCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	return CreateItemQuantityCommand(ItemCommandConfig{
		Name:        "sell",
		Description: "Sell an item from your inventory",
		OptionName:  "item",
		OptionDesc:  "Item name to sell",
		ResultTitle: "💵 Sale Complete",
		ResultColor: 0xf39c12,
		Action:      func(c *APIClient) func(string, string, string, string, int) (string, error) { return c.SellItem },
	})
}

// PricesCommand returns the prices command definition and handler (buy prices)
func PricesCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "prices",
		Description: "View buy prices (cost to purchase items)",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		handleEmbedResponse(s, i, func() (string, error) {
			return client.GetBuyPrices()
		}, ResponseConfig{
			Title: "🏪 Buy Prices",
			Color: 0x3498db, // Blue
		})
	}

	return cmd, handler
}

// SellPricesCommand returns the sell prices command definition and handler
func SellPricesCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "prices-sell",
		Description: "View sell prices (what you get when selling)",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		handleEmbedResponse(s, i, func() (string, error) {
			return client.GetSellPrices()
		}, ResponseConfig{
			Title: "💰 Sell Prices",
			Color: 0xf1c40f, // Yellow
		})
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
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "item",
				Description:  "Item name to give",
				Required:     true,
				Autocomplete: true,
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
		if !deferResponse(s, i) {
			return
		}

		fromUser := getInteractionUser(i)
		options := getOptions(i)
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
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := createEmbed("🎁 Gift Sent", msg, 0xe91e63, "")
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
