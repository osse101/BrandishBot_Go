package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// PingCommand returns the ping command definition and handler
func PingCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Check if the bot is alive",
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pong! 🏓",
			},
		}); err != nil {
			slog.Error("Failed to respond to ping", "error", err)
		}
	}

	return cmd, handler
}
