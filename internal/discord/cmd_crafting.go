package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// UpgradeCommand returns the upgrade command definition and handler
func UpgradeCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	return CreateItemQuantityCommand(ItemCommandConfig{
		Name:        "upgrade",
		Description: "Craft an item upgrade using a recipe",
		OptionName:  "recipe",
		OptionDesc:  "Recipe/Item to craft (start typing to search)",
		ResultTitle: "🔨 Upgrade Complete",
		ResultColor: 0xe67e22,
		Action:      func(c *APIClient) func(string, string, string, string, int) (string, error) { return c.UpgradeItem },
	})
}

// DisassembleCommand returns the disassemble command definition and handler
func DisassembleCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	return CreateItemQuantityCommand(ItemCommandConfig{
		Name:        "disassemble",
		Description: "Break down an item to get materials",
		OptionName:  "item",
		OptionDesc:  "Item name to disassemble",
		ResultTitle: "🔧 Disassemble Complete",
		ResultColor: 0x95a5a6,
		Action:      func(c *APIClient) func(string, string, string, string, int) (string, error) { return c.DisassembleItem },
	})
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
		})
	}

	return cmd, handler
}
