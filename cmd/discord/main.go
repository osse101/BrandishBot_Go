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
	devChannelID := os.Getenv("DISCORD_DEV_CHANNEL_ID")
	gameChannelID := os.Getenv("DISCORD_DIGGING_GAME_CHANNEL_ID")
	webhookPort := os.Getenv("DISCORD_WEBHOOK_PORT")
	if webhookPort == "" {
		webhookPort = "8082"
	}
	githubToken := os.Getenv("GITHUB_TOKEN")
	githubRepo := os.Getenv("GITHUB_OWNER_REPO")

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
	slog.Info("Configured API URL", "url", apiURL)

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		slog.Warn("API_KEY not set, discord bot requests may fail")
	}

	cfg := discord.Config{
		Token:                token,
		AppID:                appID,
		APIURL:               apiURL,
		APIKey:               apiKey,
		DevChannelID:         devChannelID,
		DiggingGameChannelID: gameChannelID,
		GithubToken:          githubToken,
		GithubOwnerRepo:      githubRepo,
	}

	bot, err := discord.New(cfg)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	// Start Internal HTTP Server
	httpServer := discord.NewHTTPServer(webhookPort, bot)
	httpServer.Start()
	defer httpServer.Stop()

	// Start Scheduled Jobs
	bot.StartDailyCommitChecker()

	// Register commands
	cmd, handler := discord.PingCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.ProfileCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.SearchCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.InventoryCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.UseItemCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.GambleStartCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.GambleJoinCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.VoteCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminUnlockCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminUnlockAllCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminRelockCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminInstantResolveCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminResetTreeCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminTreeStatusCommand()
	bot.Registry.Register(cmd, handler)

	// New progression commands
	cmd, handler = discord.UnlockProgressCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.EngagementCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.VotingSessionCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminStartVotingCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminEndVotingCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminAddContributionCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminReloadWeightsCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.InfoCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.JobBonusCommand()
	bot.Registry.Register(cmd, handler)

	// Economy commands
	cmd, handler = discord.BuyCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.SellCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.PricesCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.SellPricesCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.GiveCommand()
	bot.Registry.Register(cmd, handler)

	// Crafting commands
	cmd, handler = discord.UpgradeCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.DisassembleCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.RecipesCommand()
	bot.Registry.Register(cmd, handler)

	// Stats commands
	cmd, handler = discord.LeaderboardCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.StatsCommand()
	bot.Registry.Register(cmd, handler)

	// Admin commands
	cmd, handler = discord.AddItemCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.RemoveItemCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.AdminAwardXPCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.CheckTimeoutCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.ReloadCommand(bot)
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.LinkCommand()
	bot.Registry.Register(cmd, handler)

	cmd, handler = discord.UnlinkCommand()
	bot.Registry.Register(cmd, handler)

	// Register with Discord API
	// By default, only updates if commands have changed
	// Set DISCORD_FORCE_COMMAND_UPDATE=true to force full registration
	forceUpdate := os.Getenv("DISCORD_FORCE_COMMAND_UPDATE") == "true"
	if forceUpdate {
		slog.Info("Force command update enabled via environment variable")
	}

	if err := bot.RegisterCommands(bot.Registry, forceUpdate); err != nil {
		slog.Error("Failed to register commands", "error", err)
		// Don't exit - bot can still run if commands are already registered
	}

	if err := bot.Run(); err != nil {
		slog.Error("Bot failed", "error", err)
		os.Exit(1)
	}
}
