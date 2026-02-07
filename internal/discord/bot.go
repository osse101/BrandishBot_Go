package discord

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Bot represents the Discord bot
type Bot struct {
	Session               *discordgo.Session
	Client                *APIClient
	AppID                 string
	Registry              *CommandRegistry
	DevChannelID          string
	DiggingGameChannelID  string
	NotificationChannelID string
	GithubToken           string
	GithubOwnerRepo       string
	sseClient             *SSEClient
	sseNotifier           *SSENotifier
	ctx                   context.Context
	cancel                context.CancelFunc
	wg                    sync.WaitGroup
}

// Config holds the bot configuration
type Config struct {
	Token                 string
	AppID                 string
	APIURL                string
	APIKey                string
	DevChannelID          string
	DiggingGameChannelID  string
	NotificationChannelID string
	GithubToken           string
	GithubOwnerRepo       string
}

// New creates a new Discord bot
func New(cfg Config) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	bot := &Bot{
		Session:               s,
		Client:                NewAPIClient(cfg.APIURL, cfg.APIKey), // Pass API Key
		AppID:                 cfg.AppID,
		Registry:              NewCommandRegistry(),
		DevChannelID:          cfg.DevChannelID,
		DiggingGameChannelID:  cfg.DiggingGameChannelID,
		NotificationChannelID: cfg.NotificationChannelID,
		GithubToken:           cfg.GithubToken,
		GithubOwnerRepo:       cfg.GithubOwnerRepo,
	}

	// Initialize SSE client if notification channel is configured
	if cfg.NotificationChannelID != "" {
		bot.sseClient = NewSSEClient(cfg.APIURL, cfg.APIKey, []string{
			SSEEventTypeJobLevelUp,
			SSEEventTypeVotingStarted,
			SSEEventTypeCycleCompleted,
			SSEEventTypeAllUnlocked,
			SSEEventTypeGambleCompleted,
			SSEEventTypeExpeditionStarted,
			SSEEventTypeExpeditionTurn,
			SSEEventTypeExpeditionCompleted,
		})
	}

	return bot, nil
}

// Start starts the bot
func (b *Bot) Start() error {
	b.Session.AddHandler(b.ready)
	b.Session.AddHandler(b.interactionCreate)
	b.Session.AddHandler(b.messageCreate)

	// Add autocomplete handler
	b.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
			HandleAutocomplete(s, i, b.Client)
		}
	})

	if err := b.Session.Open(); err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	// Start SSE client for real-time notifications
	if b.sseClient != nil && b.NotificationChannelID != "" {
		b.sseNotifier = NewSSENotifier(b.Session, b.NotificationChannelID)
		b.sseNotifier.RegisterHandlers(b.sseClient)

		b.ctx, b.cancel = context.WithCancel(context.Background())
		b.sseClient.Start(b.ctx)
		slog.Info("SSE client started for real-time notifications",
			"channel_id", b.NotificationChannelID)
	} else {
		b.ctx, b.cancel = context.WithCancel(context.Background())
	}

	slog.Info("Discord bot is now running. Press CTRL-C to exit.")
	return nil
}

// Stop stops the bot
func (b *Bot) Stop() {
	slog.Info("Shutting down bot...")

	// Cancel context to stop all background tasks
	if b.cancel != nil {
		b.cancel()
	}

	// Stop SSE client
	if b.sseClient != nil {
		b.sseClient.Stop()
		slog.Info("SSE client stopped")
	}

	// Wait for all background goroutines to finish
	b.wg.Wait()
	slog.Info("All background tasks finished")

	b.Session.Close()
	slog.Info("Discord session closed")
}

// Run runs the bot until a signal is received
func (b *Bot) Run() error {
	if err := b.Start(); err != nil {
		return err
	}
	defer b.Stop()

	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return nil
}

func (b *Bot) ready(s *discordgo.Session, r *discordgo.Ready) {
	slog.Info("Bot is ready", "user", s.State.User.Username)
}

func (b *Bot) interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.Registry != nil {
		b.Registry.Handle(s, i, b.Client)
	}
}

// SendDevMessage sends an embed to the developer channel
func (b *Bot) SendDevMessage(embed *discordgo.MessageEmbed) error {
	if b.DevChannelID == "" {
		return fmt.Errorf("dev channel ID not configured")
	}
	_, err := b.Session.ChannelMessageSendEmbed(b.DevChannelID, embed)
	return err
}

// StartDailyPatchNotesChecker starts a ticker to check for patch notes every 24 hours.
func (b *Bot) StartDailyPatchNotesChecker() {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		slog.Info("Daily patch notes checker started")

		for {
			select {
			case <-ticker.C:
				if err := b.SendDailyPatchNotesReport(); err != nil {
					slog.Error("Failed to send daily patch notes report", "error", err)
				}
			case <-b.ctx.Done():
				slog.Info("Daily patch notes checker stopping")
				return
			}
		}
	}()
}

// SendDailyPatchNotesReport reads docs/patchnotes.md and sends it to the developer channel
func (b *Bot) SendDailyPatchNotesReport() error {
	content, err := os.ReadFile("docs/patchnotes.md")
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("docs/patchnotes.md not found, skipping report")
			return nil
		}
		return fmt.Errorf("failed to read patch notes: %w", err)
	}

	// Limit content length for Discord embed (max 4096)
	description := string(content)
	if len(description) > 4000 {
		description = description[:3997] + "..."
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Latest Patch Notes",
		Description: description,
		Color:       0x00FF99, // Greenish
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Weekly Automated Updates",
		},
	}

	return b.SendDevMessage(embed)
}

// SendDailyCommitReport is deprecated in favor of SendDailyPatchNotesReport
func (b *Bot) SendDailyCommitReport() error {
	slog.Warn("SendDailyCommitReport is deprecated, use SendDailyPatchNotesReport instead")
	return nil
}

func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore own messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Ignore bot messages
	if m.Author.Bot {
		return
	}

	// Send to server for processing
	// We don't reply here, just track engagement/process commands
	_, err := b.Client.HandleMessage(
		domain.PlatformDiscord,
		domain.DiscordBotID, // Use constant Platform ID for the bot interaction context
		m.Author.Username,
		m.Content,
	)

	if err != nil {
		slog.Error("Failed to handle message", "error", err, "user", m.Author.Username)
	}
}
