package discord

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// InventoryCommand returns the inventory command definition and handler
func InventoryCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "inventory",
		Description: "View your inventory",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "View another user's inventory (optional)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "filter",
				Description: "Filter items (upgrade, sellable, consumable)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Upgradable",
						Value: domain.FilterTypeUpgrade,
					},
					{
						Name:  "Sellable",
						Value: domain.FilterTypeSellable,
					},
					{
						Name:  "Consumable",
						Value: domain.FilterTypeConsumable,
					},
				},
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)
		// Ensure user exists
		if !ensureUserRegistered(s, i, client, user, true) {
			return
		}

		var filter string
		var targetUser *discordgo.User
		for _, opt := range i.ApplicationCommandData().Options {
			switch opt.Name {
			case "filter":
				filter = opt.StringValue()
			case "user":
				targetUser = opt.UserValue(s)
			}
		}

		// If no target user specified, use the command caller
		if targetUser == nil {
			targetUser = user
		}

		// Ensure target user is registered
		if !ensureUserRegistered(s, i, client, targetUser, true) {
			return
		}

		// Get inventory based on whether targeting self or others
		var items []SimpleInventoryItem
		if targetUser.ID == user.ID {
			//If querying self, use the standard method with platformId
			inventoryItems, err := client.GetInventory(domain.PlatformDiscord, targetUser.ID, targetUser.Username, filter)
			if err != nil {
				slog.Error("Failed to get inventory", "error", err)
				respondFriendlyError(s, i, err.Error())
				return
			}
			items = ConvertToSimpleInventory(inventoryItems)
		} else {
			// If querying another user, use username-based method
			inventoryItems, err := client.GetInventoryByUsername(domain.PlatformDiscord, targetUser.Username, filter)
			if err != nil {
				slog.Error("Failed to get inventory", "error", err)
				respondFriendlyError(s, i, err.Error())
				return
			}
			items = ConvertToSimpleInventory(inventoryItems)
		}

		var description string
		if len(items) == 0 {
			description = "Your inventory is empty."
		} else {
			var sb strings.Builder
			for i, item := range items {
				if i > 0 {
					sb.WriteByte('\n')
				}
				sb.WriteString("**")
				sb.WriteString(item.Name)
				sb.WriteString("** x")
				sb.WriteString(strconv.Itoa(item.Quantity))
			}
			description = sb.String()
		}

		embed := createEmbed(fmt.Sprintf("%s's Inventory", targetUser.Username), description, 0x9b59b6, "")
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
