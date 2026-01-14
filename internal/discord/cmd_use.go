package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// UseItemCommand returns the use item command definition and handler
func UseItemCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "use",
		Description: "Use an item from your inventory",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "item",
				Description:  "Item name to use",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to use",
				Required:    false,
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
		quantity := 1
		if len(options) > 1 {
			quantity = int(options[1].IntValue())
		}

		// Ensure user exists
		if !ensureUserRegistered(s, i, client, user, false) {
			return
		}

		msg, err := client.UseItem(domain.PlatformDiscord, user.ID, user.Username, itemName, quantity)
		if err != nil {
			slog.Error("Failed to use item", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		// Format: <Effect>\n\n<Quantity> <Item> consumed
		description := fmt.Sprintf("%s\n\n_%d %s consumed_", msg, quantity, itemName)

		embed := createEmbed("🧪 Item Used", description, 0xf39c12, "")
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
