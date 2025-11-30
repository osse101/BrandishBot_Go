package discord

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

// Bot represents the Discord bot
type Bot struct {
	Session  *discordgo.Session
	Client   *APIClient
	AppID    string
	Registry *CommandRegistry
}

// Config holds the bot configuration
type Config struct {
	Token  string
	AppID  string
	APIURL string
}

// New creates a new Discord bot
func New(cfg Config) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	return &Bot{
		Session:  s,
		Client:   NewAPIClient(cfg.APIURL, ""), // API Key support can be added to Config
		AppID:    cfg.AppID,
		Registry: NewCommandRegistry(),
	}, nil
}

// Start starts the bot
func (b *Bot) Start() error {
	b.Session.AddHandler(b.ready)
	b.Session.AddHandler(b.interactionCreate)

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
