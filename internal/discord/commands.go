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
		RecordCommand() // Track command usage
		h(s, i, client)
	}
}

// RegisterCommands intelligently registers/updates commands with Discord
// Only performs updates if commands have changed to avoid rate limits
func (b *Bot) RegisterCommands(registry *CommandRegistry, forceUpdate bool) error {
	slog.Info("Checking Discord commands...")
	
	// Get currently registered commands from Discord
	existingCmds, err := b.Session.ApplicationCommands(b.AppID, "")
	if err != nil {
		return fmt.Errorf("failed to fetch existing commands: %w", err)
	}

	// Build desired commands list
	var desiredCmds []*discordgo.ApplicationCommand
	for _, cmd := range registry.Commands {
		desiredCmds = append(desiredCmds, cmd)
	}

	// If force update, use bulk overwrite
	if forceUpdate {
		slog.Info("Force update enabled - replacing all commands", "count", len(desiredCmds))
		_, err := b.Session.ApplicationCommandBulkOverwrite(b.AppID, "", desiredCmds)
		if err != nil {
			return fmt.Errorf("failed to bulk overwrite commands: %w", err)
		}
		slog.Info("Commands force updated successfully")
		return nil
	}

	// Check if commands have changed
	if commandsEqual(existingCmds, desiredCmds) {
		slog.Info("Commands unchanged, skipping registration", "count", len(existingCmds))
		return nil
	}

	// Commands have changed - update them
	slog.Info("Commands changed, updating...", 
		"existing", len(existingCmds), 
		"desired", len(desiredCmds))
	
	_, err = b.Session.ApplicationCommandBulkOverwrite(b.AppID, "", desiredCmds)
	if err != nil {
		return fmt.Errorf("failed to update commands: %w", err)
	}
	
	slog.Info("Commands updated successfully", "count", len(desiredCmds))
	return nil
}

// commandsEqual checks if two command sets are equivalent
func commandsEqual(existing, desired []*discordgo.ApplicationCommand) bool {
	if len(existing) != len(desired) {
		return false
	}

	// Build map of existing commands by name
	existingMap := make(map[string]*discordgo.ApplicationCommand)
	for _, cmd := range existing {
		existingMap[cmd.Name] = cmd
	}

	// Check each desired command exists and matches
	for _, desired := range desired {
		existing, ok := existingMap[desired.Name]
		if !ok {
			return false
		}
		if !commandEqual(existing, desired) {
			return false
		}
	}

	return true
}

// commandEqual checks if two commands are equivalent
func commandEqual(a, b *discordgo.ApplicationCommand) bool {
	// Compare basic fields
	if a.Name != b.Name || a.Description != b.Description {
		return false
	}

	// Compare options length
	if len(a.Options) != len(b.Options) {
		return false
	}

	// Compare each option
	for i := range a.Options {
		if !optionEqual(a.Options[i], b.Options[i]) {
			return false
		}
	}

	return true
}

// optionEqual checks if two command options are equivalent
func optionEqual(a, b *discordgo.ApplicationCommandOption) bool {
	if a.Type != b.Type || a.Name != b.Name || a.Description != b.Description || a.Required != b.Required {
		return false
	}

	// Compare choices if present
	if len(a.Choices) != len(b.Choices) {
		return false
	}

	for i := range a.Choices {
		if a.Choices[i].Name != b.Choices[i].Name || a.Choices[i].Value != b.Choices[i].Value {
			return false
		}
	}

	return true
}
