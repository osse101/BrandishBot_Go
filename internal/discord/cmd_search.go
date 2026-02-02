package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// SearchCommand returns the search command definition and handler
func SearchCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "search",
		Description: "Search for items",
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

		msg, err := client.Search(domain.PlatformDiscord, user.ID, user.Username)
		if err != nil {
			slog.Error("Failed to search", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		embed := createEmbed("Search Result", msg, 0x3498db, "")
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
