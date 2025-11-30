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

// RegisterCommands registers commands with Discord
func (b *Bot) RegisterCommands(registry *CommandRegistry) error {
	slog.Info("Registering commands...")
	for _, v := range registry.Commands {
		_, err := b.Session.ApplicationCommandCreate(b.AppID, "", v)
		if err != nil {
			return fmt.Errorf("cannot create command %v: %w", v.Name, err)
		}
		slog.Info("Registered command", "name", v.Name)
	}
	return nil
}

// PingCommand returns the ping command definition and handler
func PingCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Check if the bot is alive",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pong! 🏓",
			},
		})
	}

	return cmd, handler
}

// ProfileCommand returns the profile command definition and handler
func ProfileCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "profile",
		Description: "View your profile stats",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		// Acknowledge interaction (loading state)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})

		user := i.Member.User
		if user == nil {
			user = i.User // Fallback for DM
		}

		// 1. Register/Get User
		domainUser, err := client.RegisterUser(user.Username, user.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &[]string{"Failed to retrieve profile. Please try again later."}[0],
			})
			return
		}

		// 2. Get Stats
		stats, err := client.GetUserStats(domainUser.ID)
		if err != nil {
			slog.Error("Failed to get stats", "error", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &[]string{"Failed to retrieve stats."}[0],
			})
			return
		}

		// 3. Send Embed
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's Profile", user.Username),
			Description: "Here are your stats:",
			Color:       0x00ff00, // Green
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: user.AvatarURL(""),
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Total Events",
					Value:  fmt.Sprintf("%d", stats.TotalEvents),
					Inline: true,
				},
				{
					Name:   "Internal ID",
					Value:  domainUser.ID,
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot",
			},
		}

		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
	}

	return cmd, handler
}
