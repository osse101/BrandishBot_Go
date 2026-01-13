package discord

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Bot represents the Discord bot
type Bot struct {
	Session              *discordgo.Session
	Client               *APIClient
	AppID                string
	Registry             *CommandRegistry
	DevChannelID         string
	DiggingGameChannelID string
	GithubToken          string
	GithubOwnerRepo      string
}

// Config holds the bot configuration
type Config struct {
	Token                string
	AppID                string
	APIURL               string
	APIKey               string
	DevChannelID         string
	DiggingGameChannelID string
	GithubToken          string
	GithubOwnerRepo      string
}

// New creates a new Discord bot
func New(cfg Config) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	return &Bot{
		Session:              s,
		Client:               NewAPIClient(cfg.APIURL, cfg.APIKey), // Pass API Key
		AppID:                cfg.AppID,
		Registry:             NewCommandRegistry(),
		DevChannelID:         cfg.DevChannelID,
		DiggingGameChannelID: cfg.DiggingGameChannelID,
		GithubToken:          cfg.GithubToken,
		GithubOwnerRepo:      cfg.GithubOwnerRepo,
	}, nil
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

	slog.Info("Discord bot is now running. Press CTRL-C to exit.")
	return nil
}

// Stop stops the bot
func (b *Bot) Stop() {
	b.Session.Close()
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

// StartDailyCommitChecker starts a ticker to check for commits every 24 hours
func (b *Bot) StartDailyCommitChecker() {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			if err := b.SendDailyCommitReport(); err != nil {
				slog.Error("Failed to send daily commit report", "error", err)
			}
		}
	}()
}

// SendDailyCommitReport queries GitHub and sends a summary of commits from the last 24h
func (b *Bot) SendDailyCommitReport() error {
	if b.GithubToken == "" || b.GithubOwnerRepo == "" {
		slog.Warn("GitHub not configured, skipping commit report")
		return nil
	}

	since := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits?since=%s", b.GithubOwnerRepo, since)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+b.GithubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var commits []struct {
		Sha    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
				Date string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		HtmlUrl string `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return err
	}

	if len(commits) == 0 {
		return nil // No commits to report
	}

	var sb strings.Builder
	// Optimization: Use strings.Builder for efficient string concatenation (O(n) vs O(n^2))
	for _, c := range commits {
		msg := c.Commit.Message
		if len(msg) > 50 {
			msg = msg[:47] + "..."
		}
		fmt.Fprintf(&sb, "• [`%s`](%s) %s - *%s*\n", c.Sha[:7], c.HtmlUrl, msg, c.Commit.Author.Name)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Daily Commit Summary",
		Description: sb.String(),
		Color:       0x0099FF, // Blue
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "GitHub Activity",
		},
	}

	return b.SendDevMessage(embed)
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
		domain.DiscordBotId, // Use constant Platform ID for the bot interaction context
		m.Author.Username,
		m.Content,
	)

	if err != nil {
		slog.Error("Failed to handle message", "error", err, "user", m.Author.Username)
	}
}
