package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
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
	desiredCmds := make([]*discordgo.ApplicationCommand, 0, len(registry.Commands))
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
	if a.Name != b.Name || a.Description != b.Description {
		return false
	}

	// Compare permissions
	if (a.DefaultMemberPermissions == nil) != (b.DefaultMemberPermissions == nil) {
		return false
	}
	if a.DefaultMemberPermissions != nil && b.DefaultMemberPermissions != nil {
		if *a.DefaultMemberPermissions != *b.DefaultMemberPermissions {
			return false
		}
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

// respondError sends a generic error message.
// Use for system-level errors or when detailed error message would confuse users.
//
// Usage:
//
//	respondError(s, i, "Error connecting to game server.")
func respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	}); err != nil {
		slog.Error("Failed to edit interaction response", "error", err)
	}
}

// ResponseConfig defines the visual properties of a command response embed
type ResponseConfig struct {
	Title string
	Color int
}

// handleEmbedResponse encapsulates the common logic of:
// 1. Deferring the response (optional)
// 2. Executing an action (API call)
// 3. Handling errors
// 4. Sending a success embed response
func handleEmbedResponse(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
	action func() (string, error),
	config ResponseConfig,
) {
	if !deferResponse(s, i) {
		return
	}

	msg, err := action()
	if err != nil {
		slog.Error("Action failed", "title", config.Title, "error", err)
		respondFriendlyError(s, i, err.Error())
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       config.Title,
		Description: msg,
		Color:       config.Color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "BrandishBot",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		slog.Error("Failed to send response", "error", err)
	}
}

// handleItemQuantityAction encapsulates the pattern of:
// 1. Getting item name and quantity from options
// 2. Registering/Ensuring user exists
// 3. Performing an action with the client
func handleItemQuantityAction(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient, title string, color int, action func(string, string, string, string, int) (string, error)) {
	handleEmbedResponse(s, i, func() (string, error) {
		user := getInteractionUser(i)
		options := getOptions(i)
		if len(options) == 0 {
			return "", fmt.Errorf("missing required item argument")
		}

		itemName := options[0].StringValue()
		quantity := 1
		if len(options) > 1 {
			quantity = int(options[1].IntValue())
		}

		// Ensure user exists
		_, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			return "", fmt.Errorf("failed to register user: %w", err)
		}

		return action(domain.PlatformDiscord, user.ID, user.Username, itemName, quantity)
	}, ResponseConfig{
		Title: title,
		Color: color,
	})
}

// ItemCommandConfig defines the configuration for a standard item+quantity command
type ItemCommandConfig struct {
	Name        string
	Description string
	OptionName  string // e.g. "item", "recipe"
	OptionDesc  string
	ResultTitle string
	ResultColor int
	Action      func(client *APIClient) func(string, string, string, string, int) (string, error)
}

// CreateItemQuantityCommand returns a standardized item+quantity command and handler
func CreateItemQuantityCommand(cfg ItemCommandConfig) (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        cfg.Name,
		Description: cfg.Description,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         cfg.OptionName,
				Description:  cfg.OptionDesc,
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity (default: 1)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		handleItemQuantityAction(s, i, client, cfg.ResultTitle, cfg.ResultColor, cfg.Action(client))
	}

	return cmd, handler
}

// deferResponse acknowledges an interaction with a deferred message.
// Required before any async operations that might take longer than 3 seconds.
// Returns false if deferral failed (should return early from handler).
//
// Usage:
//
//	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
//	    if !deferResponse(s, i) {
//	        return
//	    }
//	    // Perform slow operations...
//	}
func deferResponse(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		slog.Error("Failed to send deferred response", "error", err)
		return false
	}
	return true
}

// getInteractionUser extracts the user from an interaction.
// Handles both guild (i.Member.User) and DM (i.User) contexts.
// Always returns a non-nil *discordgo.User.
//
// Usage:
//
//	user := getInteractionUser(i)
//	if !ensureUserRegistered(s, i, client, user, true) {
//	    return
//	}
func getInteractionUser(i *discordgo.InteractionCreate) *discordgo.User {
	user := i.Member.User
	if user == nil {
		user = i.User
	}
	return user
}

// getOptions extracts command options from an interaction.
// Convenience helper to avoid calling i.ApplicationCommandData().Options repeatedly.
// Returns slice of ApplicationCommandInteractionDataOption for parsing command arguments.
//
// Usage:
//
//	options := getOptions(i)
//	itemName := options[0].StringValue()
//	quantity := 1
//	if len(options) > 1 {
//	    quantity = int(options[1].IntValue())
//	}
func getOptions(i *discordgo.InteractionCreate) []*discordgo.ApplicationCommandInteractionDataOption {
	return i.ApplicationCommandData().Options
}

// respondFriendlyError formats the error message to be more user-friendly before responding.
// Transforms technical errors (insufficient funds, item not found, cooldowns, etc.) into
// readable messages. Use for API/business logic errors users can understand and act on.
//
// Usage:
//
//	msg, err := client.BuyItem(...)
//	if err != nil {
//	    slog.Error("Failed to buy item", "error", err)
//	    respondFriendlyError(s, i, err.Error())
//	    return
//	}
func respondFriendlyError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	friendlyMsg := formatFriendlyError(message)
	respondError(s, i, friendlyMsg)
}

// formatFriendlyError cleans up technical error messages
func formatFriendlyError(msg string) string {
	// Remove "API error: " prefix if present (from client.go)
	if len(msg) > 11 && msg[:11] == "API error: " {
		msg = msg[11:]
	}

	// Map common technical errors to friendly messages
	// We check for containment because error messages might be wrapped or contain details
	switch {
	case strings.HasPrefix(msg, "LOCKED_NODES:"):
		nodes := strings.TrimPrefix(msg, "LOCKED_NODES:")
		return fmt.Sprintf("%s\nTo unlock this, you need to active: **%s**", MsgFeatureLocked, nodes)
	case strings.Contains(msg, domain.ErrMsgInsufficientFunds):
		return MsgInsufficientFunds
	case strings.Contains(msg, domain.ErrMsgItemNotFound):
		return MsgItemNotFound
	case strings.Contains(msg, domain.ErrMsgInventoryFull):
		return MsgInventoryFull
	case strings.Contains(msg, domain.ErrMsgUserNotFound):
		return MsgUserNotFound
	case strings.Contains(msg, "on cooldown"):
		// Extract remaining time if present (format: "action 'x' on cooldown: 4m 3s remaining")
		if parts := strings.Split(msg, "on cooldown: "); len(parts) > 1 {
			remaining := strings.TrimSuffix(parts[1], " remaining")
			return fmt.Sprintf("%s\nWait for: **%s**", MsgCooldownActive, remaining)
		}
		return MsgCooldownActive
	case strings.Contains(msg, domain.ErrMsgNotEnoughItems):
		return MsgNotEnoughItems
	default:
		// If it looks like a sentence, just return it, otherwise wrap it slightly
		return "❌ " + msg
	}
}

// sendEmbed sends an embed message with standardized error handling.
// Encapsulates the common pattern of sending InteractionResponseEdit with embeds.
// Logs errors internally - no need for callers to handle send errors.
//
// Usage:
//
//	embed := createEmbed("Title", "Description", 0x3498db, "")
//	sendEmbed(s, i, embed)
func sendEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) {
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		slog.Error("Failed to send response", "error", err)
	}
}

// Footer constants for standardized embed footers.
// Use these instead of magic strings to maintain consistency across all embeds.
// This allows updating footer text globally by changing one constant.
const (
	FooterBrandishBot      = "BrandishBot"       // Standard footer for user-facing commands
	FooterBrandishBotAdmin = "BrandishBot Admin" // Footer for admin commands
	FooterAdminAction      = "Admin Action"      // Footer for inventory management commands
)

// createEmbed creates a standard embed with optional footer customization.
// Handles default footer assignment and enforces consistent embed structure.
//
// Parameters:
//
//	title: Embed title (e.g., "💰 Purchase Complete")
//	description: Main embed content/description
//	color: Hex color code (e.g., 0x2ecc71 for green)
//	footerText: Custom footer text; empty string defaults to FooterBrandishBot
//
// Usage:
//
//	embed := createEmbed("Sale Complete", msg, 0xf39c12, "")
//	embed := createEmbed("Admin Action", msg, 0x95a5a6, FooterBrandishBotAdmin)
//	sendEmbed(s, i, embed)
func createEmbed(title, description string, color int, footerText string) *discordgo.MessageEmbed {
	if footerText == "" {
		footerText = FooterBrandishBot
	}
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
	}
}

// ensureUserRegistered registers a user and handles errors uniformly.
// Encapsulates the 6-line registration error handling pattern into a single check.
// Returns true if successful, false if registration failed (error already handled).
//
// Parameters:
//
//	friendlyError: true = show formatted error message from API, false = generic error message
//	Use true for operations like inventory/stats where users can understand the error
//	Use false for system operations where generic "Error connecting..." is better
//
// Usage (friendly error - show API details):
//
//	if !ensureUserRegistered(s, i, client, user, true) {
//	    return
//	}
//
// Usage (generic error - system operation):
//
//	if !ensureUserRegistered(s, i, client, user, false) {
//	    return
//	}
func ensureUserRegistered(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient, user *discordgo.User, friendlyError bool) bool {
	_, err := client.RegisterUser(user.Username, user.ID)
	if err != nil {
		slog.Error("Failed to register user", "error", err)
		if friendlyError {
			respondFriendlyError(s, i, err.Error())
		} else {
			respondError(s, i, "Error connecting to game server.")
		}
		return false
	}
	return true
}
