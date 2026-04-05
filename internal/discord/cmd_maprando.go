package discord

import (
	"fmt"
	"log/slog"
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

		// Call client
		seedURL, err := randoClient.Randomize(presetName)
		if err != nil {
			slog.Error("Failed to randomize seed", "error", err, "preset", presetName)
			respondError(s, i, fmt.Sprintf("Failed to generate seed: %v", err))
			return
		}

		seedName := ""
		// Parse out seed name from end of URL. URL looks like http://domain.com/seed/{seedname}
		// Just slice after "/seed/"
		const prefix = "/seed/"
		idx := strings.LastIndex(seedURL, prefix)
		if idx != -1 {
			seedName = seedURL[idx+len(prefix):]
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Map Rando Seed Generated!",
			Description: fmt.Sprintf("**Preset:** %s\n\n%s\n\n*Use `/maprandounlock %s` to unlock the spoiler log.*", presetFormalName, seedURL, seedName),
			Color:       0x00FF00,
		}

		// Add thumbnail attachment
		var files []*discordgo.File
		const thumbPath = "media/images/MapRando/map_station_transparent.png"
		if file, err := os.Open(thumbPath); err == nil {
			defer file.Close()
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

		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
			Files:  files,
		})
		if err != nil {
			slog.Error("Failed to send maprando embed", "error", err)
		}
	}

	return cmd, handler
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
		seedName := options[0].StringValue()

		err := randoClient.Unlock(seedName)
		if err != nil {
			slog.Error("Failed to unlock seed", "error", err, "seed", seedName)
			respondError(s, i, fmt.Sprintf("Failed to unlock seed: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Seed Unlocked!",
			Description: fmt.Sprintf("Spoiler log for seed `%s` has been unlocked.\n\n[View Seed](%s)", seedName, randoClient.SeedURL(seedName)),
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
