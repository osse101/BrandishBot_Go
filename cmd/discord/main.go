package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/osse101/BrandishBot_Go/internal/discord"
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	token := os.Getenv("DISCORD_TOKEN")
	appID := os.Getenv("DISCORD_APP_ID")
	apiURL := os.Getenv("API_URL")

	if token == "" {
		slog.Error("DISCORD_TOKEN is required")
		os.Exit(1)
	}
	if appID == "" {
		slog.Error("DISCORD_APP_ID is required")
		os.Exit(1)
	}
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	cfg := discord.Config{
		Token:  token,
		AppID:  appID,
		APIURL: apiURL,
	}

	bot, err := discord.New(cfg)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	// Register commands
	cmd, handler := discord.PingCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.ProfileCommand()
	bot.Registry.Register(cmd, handler)

	// Register with Discord API on startup
	// Note: In production, you might want to do this separately or check if needed to avoid rate limits
	if err := bot.RegisterCommands(bot.Registry); err != nil {
		slog.Error("Failed to register commands", "error", err)
		// Don't exit, bot can still run if commands are already registered
	}

	if err := bot.Run(); err != nil {
		slog.Error("Bot failed", "error", err)
		os.Exit(1)
	}
}
