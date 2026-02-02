package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// ReloadCommand returns the reload command definition and handler (admin only)
func ReloadCommand(bot *Bot) (*discordgo.ApplicationCommand, CommandHandler) {
	// Create admin permission value
	adminPerm := int64(discordgo.PermissionAdministrator)

	cmd := &discordgo.ApplicationCommand{
		Name:        "reload",
		Description: "[ADMIN] Reload Discord commands (sync or remove)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "command-name",
				Description: "Specific command to remove. Omit to sync all commands.",
				Required:    false,
			},
		},
		DefaultMemberPermissions: &adminPerm,
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		var commandName string
		if len(options) > 0 {
			commandName = options[0].StringValue()
		}

		var resultMsg string
		var resultColor int

		if commandName == "" {
			// Register missing commands
			result, err := registerMissingCommands(s, bot)
			if err != nil {
				slog.Error("Failed to register missing commands", "error", err)
				resultMsg = fmt.Sprintf("❌ Error: %v", err)
				resultColor = 0xe74c3c // Red
			} else {
				resultMsg = result
				resultColor = 0x2ecc71 // Green
			}
		} else {
			// Remove specific command
			result, err := removeCommand(s, bot.AppID, commandName)
			if err != nil {
				slog.Error("Failed to remove command", "error", err, "command", commandName)
				resultMsg = fmt.Sprintf("❌ Error removing '%s': %v", commandName, err)
				resultColor = 0xe74c3c // Red
			} else {
				resultMsg = result
				resultColor = 0xf39c12 // Orange
			}
		}

		embed := &discordgo.MessageEmbed{
			Title:       "⚙️ Command Reload",
			Description: resultMsg,
			Color:       resultColor,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Admin Action",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to send response", "error", err)
		}
	}

	return cmd, handler
}

// registerMissingCommands adds any commands that aren't currently registered
func registerMissingCommands(s *discordgo.Session, bot *Bot) (string, error) {
	// Get currently registered commands from Discord
	existingCmds, err := s.ApplicationCommands(bot.AppID, "")
	if err != nil {
		return "", fmt.Errorf("failed to fetch existing commands: %w", err)
	}

	// Build map of existing command names
	existingMap := make(map[string]*discordgo.ApplicationCommand)
	for _, cmd := range existingCmds {
		existingMap[cmd.Name] = cmd
	}

	// Check which commands from our registry are missing
	var missingCmds []*discordgo.ApplicationCommand
	var updatedCmds []*discordgo.ApplicationCommand

	for name, cmd := range bot.Registry.Commands {
		if existing, exists := existingMap[name]; exists {
			// Check if it needs updating (description changed, etc.)
			if !commandEqual(existing, cmd) {
				updatedCmds = append(updatedCmds, cmd)
			}
		} else {
			missingCmds = append(missingCmds, cmd)
		}
	}

	if len(missingCmds) == 0 && len(updatedCmds) == 0 {
		return fmt.Sprintf("✅ All commands up to date!\n\nRegistered: %d commands", len(existingCmds)), nil
	}

	// Register missing commands
	registered := 0
	for _, cmd := range missingCmds {
		_, err := s.ApplicationCommandCreate(bot.AppID, "", cmd)
		if err != nil {
			slog.Error("Failed to create command", "name", cmd.Name, "error", err)
			continue
		}
		slog.Info("Registered missing command", "name", cmd.Name)
		registered++
	}

	// Update changed commands
	updated := 0
	for _, cmd := range updatedCmds {
		existing := existingMap[cmd.Name]
		_, err := s.ApplicationCommandEdit(bot.AppID, "", existing.ID, cmd)
		if err != nil {
			slog.Error("Failed to update command", "name", cmd.Name, "error", err)
			continue
		}
		slog.Info("Updated command", "name", cmd.Name)
		updated++
	}

	totalAfter := len(existingCmds) + registered

	var sb strings.Builder
	sb.WriteString("✅ Commands synchronized!\n\n")
	if registered > 0 {
		fmt.Fprintf(&sb, "➕ Registered: %d new command(s)\n", registered)
	}
	if updated > 0 {
		fmt.Fprintf(&sb, "🔄 Updated: %d command(s)\n", updated)
	}
	fmt.Fprintf(&sb, "\n📊 Total: %d commands active", totalAfter)

	return sb.String(), nil
}

// removeCommand removes a specific command by name
func removeCommand(s *discordgo.Session, appID, commandName string) (string, error) {
	// Get all commands
	commands, err := s.ApplicationCommands(appID, "")
	if err != nil {
		return "", fmt.Errorf("failed to fetch commands: %w", err)
	}

	// Find the command to delete
	var targetCmd *discordgo.ApplicationCommand
	for _, cmd := range commands {
		if cmd.Name == commandName {
			targetCmd = cmd
			break
		}
	}

	if targetCmd == nil {
		return fmt.Sprintf("⚠️ Command '%s' not found.\n\nAvailable commands: %d", commandName, len(commands)), nil
	}

	// Delete the command
	err = s.ApplicationCommandDelete(appID, "", targetCmd.ID)
	if err != nil {
		return "", fmt.Errorf("failed to delete command: %w", err)
	}

	slog.Info("Command removed", "name", commandName, "id", targetCmd.ID)

	return fmt.Sprintf("🗑️ Removed: `/%s`\n\nRemaining commands: %d\n\nRestart bot to re-register.", commandName, len(commands)-1), nil
}
