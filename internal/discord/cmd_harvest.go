package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// HarvestCommand returns the harvest command definition and handler
func HarvestCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "harvest",
		Description: "Harvest your accumulated rewards",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)

		// Ensure user exists
		if !ensureUserRegistered(s, i, client, user, false) {
			return
		}

		// Call harvest API
		resp, err := client.Harvest(domain.PlatformDiscord, user.ID, user.Username)
		if err != nil {
			slog.Error("Failed to harvest", "error", err, "user", user.Username)
			respondFriendlyError(s, i, err.Error())
			return
		}

		// Build items gained message
		var itemsList []string
		totalItems := 0
		for itemName, quantity := range resp.ItemsGained {
			if quantity > 0 {
				itemsList = append(itemsList, fmt.Sprintf("%s x%d", itemName, quantity))
				totalItems += quantity
			}
		}

		// Create embed based on whether items were gained
		var embed *discordgo.MessageEmbed
		if totalItems == 0 {
			// First harvest or no rewards
			embed = createEmbed("Harvest", resp.Message, 0x3498db, "")
		} else {
			// Successful harvest with rewards
			itemsGainedStr := strings.Join(itemsList, ", ")
			description := fmt.Sprintf("**Items Gained:** %s\n\n**Time Since Last Harvest:** %.1f hours\n\n%s",
				itemsGainedStr,
				resp.HoursSinceHarvest,
				resp.Message)

			embed = createEmbed("Harvest Complete!", description, 0x2ecc71, "") // Green color
		}

		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
