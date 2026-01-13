package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// UpgradeCommand returns the upgrade command definition and handler
func UpgradeCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "upgrade",
		Description: "Craft an item upgrade using a recipe",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "recipe",
				Description:  "Recipe/Item to craft (start typing to search)",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to craft (default: 1)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		handleEmbedResponse(s, i, func() (string, error) {
			user := getInteractionUser(i)
			options := getOptions(i)
			itemName := options[0].StringValue()
			quantity := 1
			if len(options) > 1 {
				quantity = int(options[1].IntValue())
			}

			// Ensure user exists
			_, err := client.RegisterUser(user.Username, user.ID)
			if err != nil {
				return "", fmt.Errorf("failed to register user: %w", err)
			}

			return client.UpgradeItem(domain.PlatformDiscord, user.ID, user.Username, itemName, quantity)
		}, ResponseConfig{
			Title: "🔨 Upgrade Complete",
			Color: 0xe67e22, // Orange
		}, true)
	}

	return cmd, handler
}

// DisassembleCommand returns the disassemble command definition and handler
func DisassembleCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "disassemble",
		Description: "Break down an item to get materials",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "item",
				Description:  "Item name to disassemble",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to disassemble (default: 1)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		handleEmbedResponse(s, i, func() (string, error) {
			user := getInteractionUser(i)
			options := getOptions(i)
			itemName := options[0].StringValue()
			quantity := 1
			if len(options) > 1 {
				quantity = int(options[1].IntValue())
			}

			// Ensure user exists
			_, err := client.RegisterUser(user.Username, user.ID)
			if err != nil {
				return "", fmt.Errorf("failed to register user: %w", err)
			}

			return client.DisassembleItem(domain.PlatformDiscord, user.ID, user.Username, itemName, quantity)
		}, ResponseConfig{
			Title: "🔧 Disassemble Complete",
			Color: 0x95a5a6, // Gray
		}, true)
	}

	return cmd, handler
}

// RecipesCommand returns the recipes command definition and handler
func RecipesCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "recipes",
		Description: "View all available crafting recipes",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		handleEmbedResponse(s, i, func() (string, error) {
			recipes, err := client.GetRecipes()
			if err != nil {
				return "", err
			}

			// Format recipes list
			var sb string
			if len(recipes) == 0 {
				sb = "No recipes available."
			} else {
				for _, r := range recipes {
					sb += fmt.Sprintf("• **%s**\n", r.ItemName)
				}
			}
			return sb, nil
		}, ResponseConfig{
			Title: "📜 Crafting Recipes",
			Color: 0x9b59b6, // Purple
		}, true)
	}

	return cmd, handler
}
