package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/info"
)

// InfoCommand returns the info command definition and handler
func InfoCommand(loader *info.Loader) (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "info",
		Description: "Get information about BrandishBot features",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "feature",
				Description:  "Specific feature or topic to learn about (optional)",
				Required:     false,
				Autocomplete: true, // Enable autocomplete for better UX
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		// Handle autocomplete requests
		if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
			data := i.ApplicationCommandData()
			var query string
			for _, opt := range data.Options {
				if opt.Focused {
					query = strings.ToLower(opt.StringValue())
					break
				}
			}

			choices := make([]*discordgo.ApplicationCommandOptionChoice, 0)

			// Get all features available
			features := loader.GetAllFeatures()

			// Filter features matching query
			for name := range features {
				if strings.Contains(strings.ToLower(name), query) {
					choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
						Name:  cases.Title(language.English).String(name), // Simple title case
						Value: name,
					})
				}
			}

			// Send choices (limit to 25 Discord max)
			if len(choices) > 25 {
				choices = choices[:25]
			}

			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionApplicationCommandAutocompleteResult,
				Data: &discordgo.InteractionResponseData{
					Choices: choices,
				},
			})
			return
		}

		if !deferResponse(s, i) {
			return
		}

		// Get feature/topic name from options
		options := getOptions(i)
		targetName := ""
		if len(options) > 0 {
			targetName = strings.ToLower(strings.TrimSpace(options[0].StringValue()))
		}

		formatter := info.NewFormatter()

		// Case 1: Overview (no argument)
		if targetName == "" || targetName == "overview" {
			features := loader.GetAllFeatures()
			content := formatter.FormatFeatureList(features, domain.PlatformDiscord, "https://github.com/osse101/BrandishBot_Go") // Use repo link for now

			embed := createInfoEmbed("overview", "BrandishBot Overview", content, 0x9B59B6, "ℹ️")
			sendEmbed(s, i, embed)
			return
		}

		// Case 2: Specific Feature
		if feature, ok := loader.GetFeature(targetName); ok {
			content := formatter.FormatFeature(feature, domain.PlatformDiscord)
			title := feature.Title
			if title == "" {
				title = cases.Title(language.English).String(feature.Name)
			}

			// Resolve color/icon from known map or defaults
			// For now, reuse existing map logic or default
			embed := createInfoEmbed(feature.Name, title, content, 0, "")
			sendEmbed(s, i, embed)
			return
		}

		// Case 3: Specific Topic (Search)
		if topic, featureName, found := loader.SearchTopic(targetName); found {
			content := formatter.FormatTopic(topic, domain.PlatformDiscord)
			title := cases.Title(language.English).String(targetName)

			// Use feature name to derive color/icon style
			embed := createInfoEmbed(featureName, title, content, 0, "")
			sendEmbed(s, i, embed)
			return
		}

		// Not found
		respondFriendlyError(s, i, fmt.Sprintf("Info not found for: '%s'", targetName))
	}

	return cmd, handler
}

// createInfoEmbed creates an embed based on the feature/topic data
func createInfoEmbed(featureName, title, content string, overrideColor int, overrideIcon string) *discordgo.MessageEmbed {
	// Map feature names to default colors/icons
	featureConfig := map[string]struct {
		color int
		icon  string
	}{
		"overview":    {0x9B59B6, "ℹ️"},
		"economy":     {0xF1C40F, "💰"},
		"inventory":   {0x3498DB, "🎒"},
		"crafting":    {0xE67E22, "🔨"},
		"gamble":      {0xE74C3C, "🎲"},
		"expeditions": {0x9B59B6, "🗺️"},
		"quests":      {0x3498DB, "📜"},
		"farming":     {0x2ECC71, "🌾"},
		"jobs":        {0xE67E22, "🛠️"},
		"progression": {0x2ECC71, "🌳"},
		"stats":       {0x1ABC9C, "📊"},
		"commands":    {0x95A5A6, "⚙️"},
	}

	color := 0x9B59B6 // Default purple
	icon := "ℹ️"

	// Look up config by feature name (or fallback to passed overrides)
	if config, ok := featureConfig[featureName]; ok {
		color = config.color
		icon = config.icon
	}

	if overrideColor != 0 {
		color = overrideColor
	}
	if overrideIcon != "" {
		icon = overrideIcon
	}

	fullTitle := fmt.Sprintf("%s %s", icon, title)

	return &discordgo.MessageEmbed{
		Title:       fullTitle,
		Description: content,
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "BrandishBot • Use /info [feature] for specific topics",
		},
	}
}
