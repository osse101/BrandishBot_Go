package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// LinkCommand returns the link command definition and handler
func LinkCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "link",
		Description: "Link your account to another platform",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "token",
				Description: "Link token from another platform (leave empty to generate one)",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "confirm",
				Description: "Confirm a pending link",
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
		var token string
		var confirm bool

		for _, opt := range options {
			switch opt.Name {
			case "token":
				token = opt.StringValue()
			case "confirm":
				confirm = opt.BoolValue()
			}
		}

		var embed *discordgo.MessageEmbed

		if confirm {
			// Step 3: Confirm link
			result, err := client.ConfirmLink(user.ID)
			if err != nil {
				respondError(s, i, fmt.Sprintf("Failed to confirm link: %v", err))
				return
			}

			embed = createEmbed("✅ Accounts Linked!", fmt.Sprintf("Your accounts are now connected.\n\n**Linked Platforms:** %s\n\n_Success! Accounts linked._", strings.Join(result.LinkedPlatforms, ", ")), 0x2ecc71, "Use /profile to see linked accounts")
		} else if token != "" {
			// Step 2: Claim token from another platform
			result, err := client.ClaimLink(token, user.ID)
			if err != nil {
				respondError(s, i, fmt.Sprintf("Failed to claim token: %v", err))
				return
			}

			embed = createEmbed("📋 Token Claimed!", fmt.Sprintf("Received token from **%s**.\n\nReturn to **%s** and use `/link confirm` (or equivalent) to complete the link.", result.SourcePlatform, result.SourcePlatform), 0x3498db, "Waiting for confirmation from source platform")
		} else {
			// Step 1: Generate new token
			result, err := client.InitiateLink(user.ID)
			if err != nil {
				respondError(s, i, fmt.Sprintf("Failed to generate link token: %v", err))
				return
			}

			embed = createEmbed("🔗 Link Started", fmt.Sprintf("**Your link code:** `%s`\n\n"+
				"**1. Copy Code:** `%s`\n"+
				"**2. Go to External Chat:** Twitch or YouTube chat\n"+
				"**3. Type Command:** `!link %s`\n"+
				"**4. Return Here:** Come back to this channel\n"+
				"**5. Confirm:** Type `/link confirm:true`\n\n"+
				"⏰ This code expires in **%d minutes**.",
				result.Token, result.Token, result.Token, result.ExpiresIn/60), 0xf1c40f, "Code is case-insensitive")
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send response", "error", err)
		}
	}

	return cmd, handler
}

// UnlinkCommand returns the unlink command definition and handler
func UnlinkCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "unlink",
		Description: "Unlink a platform from your account",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "platform",
				Description: "Platform to unlink",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Twitch", Value: "twitch"},
					{Name: "YouTube", Value: "youtube"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionBoolean,
				Name:        "confirm",
				Description: "Confirm the unlink",
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
		platform := options[0].StringValue()
		confirm := false
		if len(options) > 1 {
			confirm = options[1].BoolValue()
		}

		var embed *discordgo.MessageEmbed

		if confirm {
			// Confirm unlink
			err := client.ConfirmUnlink(user.ID, platform)
			if err != nil {
				respondError(s, i, fmt.Sprintf("Failed to unlink: %v", err))
				return
			}

			embed = createEmbed("✅ Platform Unlinked", fmt.Sprintf("Your **%s** account has been unlinked.\n\nYour Discord account keeps all inventory and stats.", cases.Title(language.English).String(platform)), 0x2ecc71, "")
		} else {
			// Initiate unlink
			err := client.InitiateUnlink(user.ID, platform)
			if err != nil {
				respondError(s, i, fmt.Sprintf("Failed to initiate unlink: %v", err))
				return
			}

			embed = createEmbed("⚠️ Confirm Unlink", fmt.Sprintf("Are you sure you want to unlink your **%s** account?\n\n"+
				"**Warning:** The %s account will lose access to your shared inventory.\n\n"+
				"To confirm, use:\n```/unlink platform:%s confirm:true```",
				cases.Title(language.English).String(platform), cases.Title(language.English).String(platform), platform), 0xe74c3c, "Confirm within 60 seconds")
		}

		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
