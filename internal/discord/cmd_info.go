package discord

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// InfoCommand returns the info command definition and handler
func InfoCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "info",
		Description: "Get information about BrandishBot features",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "feature",
				Description: "Specific feature to learn about (optional)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Overview", Value: "overview"},
					{Name: "Economy", Value: "economy"},
					{Name: "Inventory", Value: "inventory"},
					{Name: "Crafting", Value: "crafting"},
					{Name: "Gamble", Value: "gamble"},
					{Name: "Progression", Value: "progression"},
					{Name: "Stats", Value: "stats"},
					{Name: "Commands", Value: "commands"},
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

		// Get feature name from options, default to "overview"
		featureName := "overview"
		options := i.ApplicationCommandData().Options
		if len(options) > 0 {
			featureName = options[0].StringValue()
		}

		// Load info text from file
		infoText, err := loadInfoText(featureName)
		if err != nil {
			slog.Error("Failed to load info text", "feature", featureName, "error", err)
			errorMsg := fmt.Sprintf("Error loading information for: %s", featureName)
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &errorMsg,
			}); err != nil {
				slog.Error("Failed to edit interaction response", "error", err)
			}
			return
		}

		// Create appropriate embed based on feature
		embed := createInfoEmbed(featureName, infoText)

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send info", "error", err)
		}
	}

	return cmd, handler
}

// InfoDir is the directory containing info text files (can be changed for testing)
var InfoDir = "configs/discord/info"

// loadInfoText loads the info text from a file
func loadInfoText(featureName string) (string, error) {
	// Sanitize feature name to prevent directory traversal
	featureName = strings.ToLower(strings.TrimSpace(featureName))
	
	// Prevent directory traversal
	if strings.Contains(featureName, "..") || strings.Contains(featureName, "/") || strings.Contains(featureName, "\\") {
		return "", fmt.Errorf("invalid feature name: %s", featureName)
	}

	// Use Clean to ensure path is normalized
	filename := filepath.Clean(filepath.Join(InfoDir, featureName+".txt"))

	// Verify the resolved path is still within InfoDir
	absInfoDir, err := filepath.Abs(InfoDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute info directory: %w", err)
	}
	
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute filename: %w", err)
	}

	if !strings.HasPrefix(absFilename, absInfoDir) {
		return "", fmt.Errorf("access denied: path traversal attempt detected")
	}

	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read info file: %w", err)
	}
	
	return string(data), nil
}

// createInfoEmbed creates an embed based on the feature name
func createInfoEmbed(featureName, content string) *discordgo.MessageEmbed {
	// Map feature names to titles and colors
	featureConfig := map[string]struct {
		title string
		color int
		icon  string
	}{
		"overview":    {"BrandishBot Overview", 0x9B59B6, "ℹ️"},
		"economy":     {"Economy System", 0xF1C40F, "💰"},
		"inventory":   {"Inventory Management", 0x3498DB, "🎒"},
		"crafting":    {"Crafting & Upgrades", 0xE67E22, "🔨"},
		"gamble":      {"Gambling System", 0xE74C3C, "🎲"},
		"progression": {"Progression Tree", 0x2ECC71, "🌳"},
		"stats":       {"Stats & Leaderboards", 0x1ABC9C, "📊"},
		"commands":    {"Available Commands", 0x95A5A6, "⚙️"},
	}

	config, ok := featureConfig[featureName]
	if !ok {
		config = featureConfig["overview"]
	}

	return &discordgo.MessageEmbed{
		Title:       config.icon + " " + config.title,
		Description: content,
		Color:       config.color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "BrandishBot • Use /info [feature] for specific topics",
		},
	}
}
