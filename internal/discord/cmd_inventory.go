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
		if len(i.ApplicationCommandData().Options) > 0 {
			filter = i.ApplicationCommandData().Options[0].StringValue()
		}

		items, err := client.GetInventory(domain.PlatformDiscord, user.ID, user.Username, filter)
		if err != nil {
			slog.Error("Failed to get inventory", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
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
			Title:       fmt.Sprintf("%s's Inventory", user.Username),
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
