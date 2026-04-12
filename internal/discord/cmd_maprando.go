package discord

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// MapRandoCommand returns the /maprando command definition and handler
func MapRandoCommand(randoClient *MapRandoClient) (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "maprando",
		Description: "Generate a Super Metroid Map Randomizer seed",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "preset",
				Description:  "The preset to use (e.g., S4). Optional if uploading a file.",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     false,
				Autocomplete: true,
			},
			{
				Name:        "preset_file",
				Description: "Upload a custom JSON preset file (overrides base preset)",
				Type:        discordgo.ApplicationCommandOptionAttachment,
				Required:    false,
			},
			{Name: "override1_key", Description: "Key (e.g. game_options.race_mode)", Type: discordgo.ApplicationCommandOptionString, Required: false},
			{Name: "override1_val", Description: "Value (e.g. true)", Type: discordgo.ApplicationCommandOptionString, Required: false},
			{Name: "override2_key", Description: "Key", Type: discordgo.ApplicationCommandOptionString, Required: false},
			{Name: "override2_val", Description: "Value", Type: discordgo.ApplicationCommandOptionString, Required: false},
			{Name: "override3_key", Description: "Key", Type: discordgo.ApplicationCommandOptionString, Required: false},
			{Name: "override3_val", Description: "Value", Type: discordgo.ApplicationCommandOptionString, Required: false},
			{Name: "override4_key", Description: "Key", Type: discordgo.ApplicationCommandOptionString, Required: false},
			{Name: "override4_val", Description: "Value", Type: discordgo.ApplicationCommandOptionString, Required: false},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		// Require defer due to potential slow external API
		if !deferResponse(s, i) {
			return
		}

		user := getInteractionUser(i)
		if remaining, onCooldown := randoClient.CheckCooldown(user.ID); onCooldown {
			respondError(s, i, fmt.Sprintf("Please wait %d seconds before generating another seed.", int(math.Ceil(remaining.Seconds()))))
			return
		}

		// Parse options using helper to reduce complexity
		presetName, presetFileURL, overrides := parseMapRandoOptions(i)

		if presetName == "" && presetFileURL == "" {
			respondError(s, i, "You must provide either a base `preset` or upload a `preset_file`.")
			return
		}

		presetFormalName := presetName
		if presetName != "" {
			desc := randoClient.PresetDescription(presetName)
			if desc != "" {
				presetFormalName = desc
			}
		} else {
			presetFormalName = "Custom Upload"
		}

		// Give feedback immediately
		initialMsg := "Generating seed..."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &initialMsg,
		})

		// Call client
		seedURL, err := randoClient.RandomizeWithOverrides(presetName, presetFileURL, overrides, func(pos int) {
			waitingMsg := fmt.Sprintf("Server is busy. Your seed is in the queue (Position %d). Please wait...", pos)
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &waitingMsg,
			})
		})
		if err != nil {
			randoClient.ClearCooldown(user.ID)
			slog.Error("Failed to randomize seed", "error", err, "preset", presetName)
			respondError(s, i, fmt.Sprintf("Failed to generate seed: %v", err))
			return
		}

		seedName := ""
		// Parse out seed name from end of URL. URL looks like http://domain.com/seed/{seedname}
		const prefix = "/seed/"
		idx := strings.LastIndex(seedURL, prefix)
		if idx != -1 {
			seedName = seedURL[idx+len(prefix):]
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Map Rando Seed Generated!",
			Description: fmt.Sprintf("**Preset:** %s\n\n%s", presetFormalName, seedURL),
			Color:       0x00FF00,
		}

		// Add thumbnail attachment
		var files []*discordgo.File
		const thumbPath = "media/images/MapRando/map_station_transparent.png"
		if file, err := os.Open(thumbPath); err == nil {
			// We cannot defer file.Close() if we are handing it to InteractionResponseEdit,
			// the Discord client closes it internally after reading.
			files = []*discordgo.File{
				{
					Name:   "thumb.png",
					Reader: file,
				},
			}
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: "attachment://thumb.png",
			}
		} else {
			slog.Warn("Failed to load maprando thumbnail", "path", thumbPath, "error", err)
		}

		emptyStr := ""
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &emptyStr,
			Embeds:  &[]*discordgo.MessageEmbed{embed},
			Files:   files, // DiscordGo closes files internally
			Components: &[]discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Unlock Spoiler Log",
							Style:    discordgo.SuccessButton,
							CustomID: "maprando_unlock_" + seedName,
							Emoji: &discordgo.ComponentEmoji{
								Name: "🔓",
							},
						},
					},
				},
			},
		})
		if err != nil {
			slog.Error("Failed to send maprando embed with components", "error", err)
		}
	}

	return cmd, handler
}

// HandleButtonUnlock handles the "Unlock Spoiler Log" button click
func HandleButtonUnlock(s *discordgo.Session, i *discordgo.InteractionCreate, randoClient *MapRandoClient, seedName string) {
	if !deferResponse(s, i) {
		return
	}

	err := randoClient.Unlock(seedName, "")
	if err != nil {
		slog.Error("Failed to unlock seed via button", "error", err, "seed", seedName)
		respondError(s, i, fmt.Sprintf("Failed to unlock seed: %v", err))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Seed Unlocked!",
		Description: fmt.Sprintf("Spoiler log for seed `%s` has been unlocked.\n\n[View Seed](%s)", seedName, randoClient.SeedURL(seedName, "")),
		Color:       0x00FF00,
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		slog.Error("Failed to send maprandounlock button embed", "error", err)
	}
}

// parseMapRandoOptions extracts the preset and dynamic overrides from the interaction payload
func parseMapRandoOptions(i *discordgo.InteractionCreate) (presetName string, presetFileURL string, overrides map[string]string) {
	options := getOptions(i)
	overrides = make(map[string]string)

	overrideKeys := make(map[string]string)
	overrideVals := make(map[string]string)

	for _, opt := range options {
		switch opt.Name {
		case "preset":
			presetName = opt.StringValue()
		case "preset_file":
			attachmentID, ok := opt.Value.(string)
			if ok && i.ApplicationCommandData().Resolved != nil {
				if att, ok := i.ApplicationCommandData().Resolved.Attachments[attachmentID]; ok {
					presetFileURL = att.URL
				}
			}
		}

		if strings.HasPrefix(opt.Name, "override") {
			parts := strings.Split(opt.Name, "_")
			if len(parts) == 2 {
				if parts[1] == "key" {
					overrideKeys[parts[0]] = opt.StringValue()
				} else if parts[1] == "val" {
					overrideVals[parts[0]] = opt.StringValue()
				}
			}
		}
	}

	for prefix, key := range overrideKeys {
		if val, ok := overrideVals[prefix]; ok {
			overrides[key] = val
		}
	}

	return presetName, presetFileURL, overrides
}
func MapRandoUnlockCommand(randoClient *MapRandoClient) (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "maprandounlock",
		Description: "Unlock the spoiler log for a generated seed",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "seed",
				Description: "The seed name to unlock",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		if len(options) < 1 {
			respondError(s, i, "Missing required seed name")
			return
		}

		seedName := strings.TrimSpace(options[0].StringValue())

		err := randoClient.Unlock(seedName, "")
		if err != nil {
			slog.Error("Failed to unlock seed", "error", err, "seed", seedName)
			respondError(s, i, fmt.Sprintf("Failed to unlock seed: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Seed Unlocked!",
			Description: fmt.Sprintf("Spoiler log for seed `%s` has been unlocked.\n\n[View Seed](%s)", seedName, randoClient.SeedURL(seedName, "")),
			Color:       0x00FF00,
		}

		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			slog.Error("Failed to send maprandounlock embed", "error", err)
		}
	}

	return cmd, handler
}
