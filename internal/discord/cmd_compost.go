package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// CompostDepositCommand returns the compost deposit command definition and handler
func CompostDepositCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "compost-deposit",
		Description: "Deposit items into your compost bin",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "item",
				Description:  "Item to deposit",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to deposit (default: 1)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)

		if !ensureUserRegistered(s, i, client, user, false) {
			return
		}

		options := getOptions(i)
		if len(options) == 0 {
			respondFriendlyError(s, i, "Missing required item argument")
			return
		}

		itemName := options[0].StringValue()
		quantity := 1
		if len(options) > 1 {
			quantity = int(options[1].IntValue())
		}

		items := []map[string]interface{}{
			{
				"item_name": itemName,
				"quantity":  quantity,
			},
		}

		result, err := client.CompostDeposit(domain.PlatformDiscord, user.ID, items)
		if err != nil {
			slog.Error("Failed to deposit into compost", "error", err, "user", user.Username)
			respondFriendlyError(s, i, err.Error())
			return
		}

		description := fmt.Sprintf("**%s x%d** added to compost bin\n\n**Bin:** %d/%d items\n**Status:** %s",
			itemName, quantity, result.ItemCount, result.Capacity, result.Status)

		if result.ReadyAt != "" {
			description += fmt.Sprintf("\n**Ready at:** %s", result.ReadyAt)
		}

		embed := createEmbed("Compost Deposit", description, 0x8B4513, "") // Brown color
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// CompostHarvestCommand returns the compost harvest command definition and handler
func CompostHarvestCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "compost-harvest",
		Description: "Harvest your compost bin or check its status",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)

		if !ensureUserRegistered(s, i, client, user, false) {
			return
		}

		result, err := client.CompostHarvest(domain.PlatformDiscord, user.ID, user.Username)
		if err != nil {
			slog.Error("Failed to harvest compost", "error", err, "user", user.Username)
			respondFriendlyError(s, i, err.Error())
			return
		}

		var embed *discordgo.MessageEmbed
		if result.Harvested {
			// Build items list
			var itemsList []string
			for name, qty := range result.Items {
				itemsList = append(itemsList, fmt.Sprintf("%s x%d", name, qty))
			}
			itemsStr := strings.Join(itemsList, ", ")
			description := fmt.Sprintf("**Items Received:** %s\n\n%s", itemsStr, result.Message)
			embed = createEmbed("Compost Harvest Complete!", description, 0x2ecc71, "") // Green
		} else {
			// Status update
			description := result.Message
			if result.TimeLeft != "" {
				description = fmt.Sprintf("**Status:** %s\n**Time remaining:** %s", result.Status, result.TimeLeft)
			}
			color := 0x3498db // Blue for composting
			if result.Status == string(domain.CompostBinStatusIdle) {
				color = 0x95a5a6 // Gray for idle
			}
			embed = createEmbed("Compost Bin", description, color, "")
		}

		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// CompostStatusCommand returns the compost status command definition and handler
func CompostStatusCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "compost-status",
		Description: "Check the status of your compost bin",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)

		// Status check doesn't strictly need registration, but good for tracking
		if !ensureUserRegistered(s, i, client, user, false) {
			return
		}

		result, err := client.CompostStatus(domain.PlatformDiscord, user.ID)
		if err != nil {
			slog.Error("Failed to check compost status", "error", err, "user", user.Username)
			respondFriendlyError(s, i, err.Error())
			return
		}

		description := ""
		color := 0x95a5a6

		if result.Harvested {
			description = fmt.Sprintf("♻️ **Compost Harvested!**\n\n%s", result.Output.Message)
			color = 0x2ecc71
		} else if result.Status != nil {
			status := result.Status
			description = fmt.Sprintf("**Status:** %s\n**Capacity:** %d/%d\n",
				status.Status, status.ItemCount, status.Capacity)

			if status.TimeLeft != "" {
				description += fmt.Sprintf("**Time Remaining:** %s", status.TimeLeft)
			}

			if status.Status == domain.CompostBinStatusComposting {
				color = 0x3498db
			} else if status.Status == domain.CompostBinStatusReady {
				color = 0x2ecc71
				description += "\n\n**Ready to Harvest!**"
			}
		} else {
			description = "Unknown status"
		}

		embed := createEmbed("Compost Bin Status", description, color, "")
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
