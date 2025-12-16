package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// CommandHandler handles a slash command
type CommandHandler func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient)

// CommandRegistry holds the registered commands
type CommandRegistry struct {
	Commands map[string]*discordgo.ApplicationCommand
	Handlers map[string]CommandHandler
}

// NewCommandRegistry creates a new registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		Commands: make(map[string]*discordgo.ApplicationCommand),
		Handlers: make(map[string]CommandHandler),
	}
}

// Register adds a command to the registry
func (r *CommandRegistry) Register(cmd *discordgo.ApplicationCommand, handler CommandHandler) {
	r.Commands[cmd.Name] = cmd
	r.Handlers[cmd.Name] = handler
}

// Handle processes an interaction
func (r *CommandRegistry) Handle(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
	if h, ok := r.Handlers[i.ApplicationCommandData().Name]; ok {
		h(s, i, client)
	}
}

// RegisterCommands registers commands with Discord (BulkOverwrite clears stale commands)
func (b *Bot) RegisterCommands(registry *CommandRegistry) error {
	slog.Info("Registering commands...")
	var cmds []*discordgo.ApplicationCommand
	for _, v := range registry.Commands {
		cmds = append(cmds, v)
	}

	_, err := b.Session.ApplicationCommandBulkOverwrite(b.AppID, "", cmds)
	if err != nil {
		return fmt.Errorf("cannot bulk overwrite commands: %w", err)
	}
	slog.Info("Commands registered successfully", "count", len(cmds))
	return nil
}
