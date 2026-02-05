package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"github.com/osse101/BrandishBot_Go/internal/discord"
)

// Default values for optional configuration
const (
	DefaultWebhookPort = "8082"
	DefaultAPIURL      = "http://localhost:8080"
)

// CommandFactory creates a Discord command and its handler.
// Used to register all available commands in one place.
type CommandFactory func() (*discordgo.ApplicationCommand, discord.CommandHandler)

func main() {
	// Load .env file
	_ = godotenv.Load()

	// Setup logging
	setupLogger()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Configuration failed", "error", err)
		os.Exit(1)
	}

	// Create bot
	bot, err := discord.New(cfg)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	// Start services
	webhookPort := os.Getenv("DISCORD_WEBHOOK_PORT")
	if webhookPort == "" {
		webhookPort = DefaultWebhookPort
	}

	httpServer := discord.NewHTTPServer(webhookPort, bot)
	httpServer.Start()
	defer httpServer.Stop()

	bot.StartDailyCommitChecker()

	// Register all commands
	registerCommands(bot, getCommandFactories(bot))

	// Register with Discord API
	forceUpdate := os.Getenv("DISCORD_FORCE_COMMAND_UPDATE") == "true"
	if forceUpdate {
		slog.Info("Force command update enabled via environment variable")
	}

	if err := bot.RegisterCommands(bot.Registry, forceUpdate); err != nil {
		slog.Error("Failed to register commands", "error", err)
		// Don't exit - bot can still run if commands are already registered
	}

	// Run bot
	if err := bot.Run(); err != nil {
		slog.Error("Bot failed", "error", err)
		os.Exit(1)
	}
}

// setupLogger configures structured logging to stdout.
func setupLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
}

// loadConfig loads and validates Discord bot configuration from environment variables.
// Returns error if required variables are missing.
func loadConfig() (discord.Config, error) {
	// Load required environment variables
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return discord.Config{}, errors.New("DISCORD_TOKEN is required")
	}

	appID := os.Getenv("DISCORD_APP_ID")
	if appID == "" {
		return discord.Config{}, errors.New("DISCORD_APP_ID is required")
	}

	// Load optional environment variables with defaults
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	slog.Info("Configured API URL", "url", apiURL)

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		slog.Warn("API_KEY not set, discord bot requests may fail")
	}

	// Load optional environment variables without defaults
	devChannelID := os.Getenv("DISCORD_DEV_CHANNEL_ID")
	gameChannelID := os.Getenv("DISCORD_DIGGING_GAME_CHANNEL_ID")
	notificationChannelID := os.Getenv("DISCORD_NOTIFICATION_CHANNEL_ID")
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubRepo := os.Getenv("GITHUB_OWNER_REPO")

	if notificationChannelID != "" {
		slog.Info("SSE notifications enabled", "channel_id", notificationChannelID)
	}

	return discord.Config{
		Token:                 token,
		AppID:                 appID,
		APIURL:                apiURL,
		APIKey:                apiKey,
		DevChannelID:          devChannelID,
		DiggingGameChannelID:  gameChannelID,
		NotificationChannelID: notificationChannelID,
		GithubToken:           githubToken,
		GithubOwnerRepo:       githubRepo,
	}, nil
}

// getCommandFactories returns a list of all available Discord command factories.
// This provides a single place to see and manage all registered commands.
// Note: ReloadCommand requires the bot instance, so it's wrapped here.
func getCommandFactories(bot *discord.Bot) []CommandFactory {
	return []CommandFactory{
		// Core commands
		discord.PingCommand,
		discord.ProfileCommand,
		discord.SearchCommand,
		discord.HarvestCommand,
		discord.InfoCommand,
		discord.CheckTimeoutCommand,

		// Inventory commands
		discord.InventoryCommand,
		discord.UseItemCommand,

		// Gamble commands
		discord.GambleStartCommand,
		discord.GambleJoinCommand,

		// Expedition commands
		discord.ExploreCommand,
		discord.ExpeditionJournalCommand,

		// Progression commands
		discord.VoteCommand,
		discord.UnlockProgressCommand,
		discord.EngagementCommand,
		discord.VotingSessionCommand,

		// Admin progression commands
		discord.AdminUnlockCommand,
		discord.AdminUnlockAllCommand,
		discord.AdminRelockCommand,
		discord.AdminInstantResolveCommand,
		discord.AdminResetTreeCommand,
		discord.AdminTreeStatusCommand,
		discord.AdminStartVotingCommand,
		discord.AdminEndVotingCommand,
		discord.AdminAddContributionCommand,
		discord.AdminReloadWeightsCommand,

		// Economy commands
		discord.BuyCommand,
		discord.SellCommand,
		discord.PricesCommand,
		discord.SellPricesCommand,
		discord.GiveCommand,

		// Crafting commands
		discord.UpgradeCommand,
		discord.DisassembleCommand,
		discord.RecipesCommand,

		// Job commands
		discord.JobProgressCommand,

		// Stats commands
		discord.LeaderboardCommand,
		discord.StatsCommand,

		// Admin inventory commands
		discord.AddItemCommand,
		discord.RemoveItemCommand,
		discord.AdminAwardXPCommand,

		// Admin timeout commands
		discord.AdminTimeoutClearCommand,
		discord.AdminSetTimeoutCommand,

		// Linking commands
		discord.LinkCommand,
		discord.UnlinkCommand,

		// Utility commands
		func() (*discordgo.ApplicationCommand, discord.CommandHandler) {
			return discord.ReloadCommand(bot)
		},
	}
}

// registerCommands registers all provided command factories with the bot's registry.
// Each factory is called to create the command and handler, then registered.
func registerCommands(bot *discord.Bot, factories []CommandFactory) {
	for _, factory := range factories {
		cmd, handler := factory()
		bot.Registry.Register(cmd, handler)
	}
}
