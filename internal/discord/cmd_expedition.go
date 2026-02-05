package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// ExploreCommand returns the explore command definition and handler
// /explore is multi-purpose: start, join, or check status of expeditions
func ExploreCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "explore",
		Description: "Start or join an expedition, or check expedition status",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondError(s, i, "Error connecting to game server.")
			return
		}

		// Check expedition status first
		status, err := client.GetExpeditionStatus()
		if err != nil {
			slog.Error("Failed to get expedition status", "error", err)
			respondError(s, i, "Error checking expedition status.")
			return
		}

		// If on cooldown, show time remaining
		if status.OnCooldown && !status.HasActive {
			description := "An expedition was completed recently. Please wait before starting a new one."
			if status.CooldownExpires != nil {
				description += fmt.Sprintf("\n\nCooldown expires: `%s`", *status.CooldownExpires)
			}
			embed := createEmbed("Expedition Cooldown", description, 0x95A5A6, "")
			sendEmbed(s, i, embed)
			return
		}

		// If there's an active expedition
		if status.HasActive && status.ActiveDetails != nil {
			details := status.ActiveDetails

			switch details.State {
			case string(domain.ExpeditionStateRecruiting):
				// Try to join the expedition
				msg, err := client.JoinExpedition(domain.PlatformDiscord, user.ID, user.Username, details.ID)
				if err != nil {
					// If join fails (already joined, etc.), show status
					respondFriendlyError(s, i, err.Error())
					return
				}

				description := fmt.Sprintf("%s\n\n**Expedition ID:** `%s`\n**Join Deadline:** `%s`", msg, details.ID, details.JoinDeadline)
				embed := createEmbed("Joined Expedition!", description, 0x2ecc71, "")
				sendEmbed(s, i, embed)
				return

			case string(domain.ExpeditionStateInProgress):
				embed := createEmbed("Expedition In Progress", "An expedition is currently underway! Watch the updates in the notification channel.", 0xf39c12, "")
				sendEmbed(s, i, embed)
				return

			default:
				embed := createEmbed("Expedition Active", fmt.Sprintf("An expedition is currently in state: %s", details.State), 0x3498db, "")
				sendEmbed(s, i, embed)
				return
			}
		}

		// No active expedition and no cooldown: start a new one
		expeditionID, joinDeadline, err := client.StartExpedition(domain.PlatformDiscord, user.ID, user.Username, "standard")
		if err != nil {
			slog.Error("Failed to start expedition", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		description := fmt.Sprintf("**%s** has started an expedition!\n\n**Expedition ID:** `%s`\n**Join Deadline:** `%s`\n\nOthers can join using `/explore` before the deadline.", user.Username, expeditionID, joinDeadline)
		embed := createEmbed("Expedition Started!", description, 0x9b59b6, "Expedition System")
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// ExpeditionJournalCommand returns the expedition-journal command definition and handler
func ExpeditionJournalCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "expedition-journal",
		Description: "View the journal of a completed expedition",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "expedition-id",
				Description: "ID of the expedition (leave empty for most recent)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		var expeditionID string
		if len(options) > 0 {
			expeditionID = options[0].StringValue()
		}

		if expeditionID == "" {
			// Try to get the most recent expedition via status
			respondError(s, i, "Please provide an expedition ID. You can find it from expedition completion messages.")
			return
		}

		entries, err := client.GetExpeditionJournal(expeditionID)
		if err != nil {
			slog.Error("Failed to get expedition journal", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		if len(entries) == 0 {
			embed := createEmbed("Expedition Journal", "No journal entries found for this expedition.", 0x95A5A6, "")
			sendEmbed(s, i, embed)
			return
		}

		// Build journal text - show first page (up to 10 entries to fit in embed)
		var sb strings.Builder
		maxEntries := 10
		if len(entries) < maxEntries {
			maxEntries = len(entries)
		}

		for idx := 0; idx < maxEntries; idx++ {
			entry := entries[idx]
			if entry.TurnNumber == 0 {
				// Intro narrative
				sb.WriteString(fmt.Sprintf("*%s*\n---\n", entry.Narrative))
			} else {
				sb.WriteString(fmt.Sprintf("**Turn %d** | Fatigue: %d | Purse: %d\n%s\n\n", entry.TurnNumber, entry.Fatigue, entry.Purse, entry.Narrative))
			}
		}

		if len(entries) > maxEntries {
			sb.WriteString(fmt.Sprintf("\n*... and %d more turns*", len(entries)-maxEntries))
		}

		embed := createEmbed("Expedition Journal", sb.String(), 0x9b59b6, fmt.Sprintf("Expedition %s", expeditionID[:8]))
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
