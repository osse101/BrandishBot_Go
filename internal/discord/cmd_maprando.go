package discord

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"regexp"
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
				Description:  "The preset to use (e.g., S4)",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: true,
			},
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

		// Get params
		options := getOptions(i)
		if len(options) < 1 {
			respondError(s, i, "Missing required preset name")
			return
		}
		presetName := options[0].StringValue()
		presetFormalName := randoClient.PresetDescription(presetName)
		if presetFormalName == "" {
			presetFormalName = presetName
		}

		// Give feedback immediately
		initialMsg := "Generating seed..."
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &initialMsg,
		})

		// Call client
		seedURL, err := randoClient.Randomize(presetName, func(pos int) {
			waitingMsg := fmt.Sprintf("Server is busy. Your seed is in the queue (Position %d). Please wait...", pos)
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &waitingMsg,
			})
		})
		if err != nil {
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

// MapRandoUnlockCommand returns the /maprandounlock command definition and handler
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

		// Input validation: ensure the seed name is strictly alphanumeric/dashes
		var validSeedRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
		if !validSeedRegex.MatchString(seedName) {
			respondError(s, i, "Invalid seed name. Only alphanumeric characters, dashes, and underscores are allowed.")
			return
		}

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
