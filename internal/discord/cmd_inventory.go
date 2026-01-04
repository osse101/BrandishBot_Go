package discord

import (
	"fmt"
	"log/slog"
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

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondFriendlyError(s, i, err.Error())
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
		_, err = client.RegisterUser(targetUser.Username, targetUser.ID)
		if err != nil {
			slog.Error("Failed to register target user", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		// Get inventory based on whether targeting self or others
		var items []struct {
			Name     string
			Quantity int
		}
		if targetUser.ID == user.ID {
			//If querying self, use the standard method with platformId
			inventoryItems, err := client.GetInventory(domain.PlatformDiscord, targetUser.ID, targetUser.Username, filter)
			if err != nil {
				slog.Error("Failed to get inventory", "error", err)
				respondFriendlyError(s, i, err.Error())
				return
			}
			// Convert to simple struct
			for _, item := range inventoryItems {
				items = append(items, struct {
					Name     string
					Quantity int
				}{Name: item.Name, Quantity: item.Quantity})
			}
		} else {
			// If querying another user, use username-based method
			inventoryItems, err := client.GetInventoryByUsername(domain.PlatformDiscord, targetUser.Username, filter)
			if err != nil {
				slog.Error("Failed to get inventory", "error", err)
				respondFriendlyError(s, i, err.Error())
				return
			}
			// Convert to simple struct
			for _, item := range inventoryItems {
				items = append(items, struct {
					Name     string
					Quantity int
				}{Name: item.Name, Quantity: item.Quantity})
			}
		}

		var description string
		if len(items) == 0 {
			description = "Your inventory is empty."
		} else {
			var lines []string
			for _, item := range items {
				lines = append(lines, fmt.Sprintf("**%s** x%d", item.Name, item.Quantity))
			}
			description = strings.Join(lines, "\n")
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's Inventory", targetUser.Username),
			Description: description,
			Color:       0x9b59b6, // Purple
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send inventory embed", "error", err)
		}
	}

	return cmd, handler
}
