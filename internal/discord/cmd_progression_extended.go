package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// AdminRelockCommand returns the admin relock command definition and handler
func AdminRelockCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-relock",
		Description: "[Admin] Force relock a progression node",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "node",
				Description: "Node key to relock",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "level",
				Description: "Level to relock to (default: 0)",
				Required:    false,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		options := i.ApplicationCommandData().Options
		nodeKey := options[0].StringValue()
		level := 0
		if len(options) > 1 {
			level = int(options[1].IntValue())
		}

		msg, err := client.AdminRelockNode(nodeKey, level)
		if err != nil {
			slog.Error("Failed to relock node", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to relock: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🔒 Admin Relock",
			Description: msg,
			Color:       0x95a5a6, // Grey
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Admin",
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

// AdminInstantResolveCommand returns the admin instant resolve command
func AdminInstantResolveCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-instant-resolve",
		Description: "[Admin] Force end voting and unlock winner immediately",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		msg, err := client.AdminInstantUnlock()
		if err != nil {
			slog.Error("Failed to instant unlock", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to instant unlock: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "⚡ Admin Instant Resolve",
			Description: msg,
			Color:       0xf1c40f, // Yellow
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Admin",
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

// AdminResetTreeCommand returns the admin reset command
func AdminResetTreeCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-reset-tree",
		Description: "[Admin] Reset the entire progression tree",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "confirmation",
				Description: "Type 'CONFIRM RESET' to proceed",
				Required:    true,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		options := i.ApplicationCommandData().Options
		confirmation := options[0].StringValue()

		if confirmation != "CONFIRM RESET" {
			respondError(s, i, "Invalid confirmation. Type 'CONFIRM RESET' exactly.")
			return
		}

		user := i.Member.User
		if user == nil {
			user = i.User
		}

		msg, err := client.AdminResetProgression(user.Username, "Discord Admin Command", true)
		if err != nil {
			slog.Error("Failed to reset tree", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to reset tree: %v", err))
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "☢️ Progression Tree Reset",
			Description: msg,
			Color:       0xff0000, // Red
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Admin",
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

// AdminTreeStatusCommand returns the tree status command
func AdminTreeStatusCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-tree-status",
		Description: "[Admin] View full progression tree status",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		nodes, err := client.GetProgressionTree()
		if err != nil {
			slog.Error("Failed to get tree status", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to get tree: %v", err))
			return
		}

		statusText := formatTreeStatus(nodes)
		
		// Check length limit
		if len(statusText) > 4000 {
			// Send as file if too long
			reader := strings.NewReader(statusText)
			_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Files: []*discordgo.File{
					{
						Name:   "tree_status.txt",
						Reader: reader,
					},
				},
			})
			if err != nil {
				slog.Error("Failed to send file response", "error", err)
			}
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🌳 Progression Tree Status",
			Description: statusText,
			Color:       0x2ecc71,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Admin",
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

// AdminStartVotingCommand returns the start voting command
func AdminStartVotingCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-start-voting",
		Description: "[Admin] Start a new voting session",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		msg, err := client.AdminStartVoting()
		if err != nil {
			errorMsg := fmt.Sprintf("❌ Failed to start voting: %v", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &errorMsg,
			})
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🗳️ Admin Start Voting",
			Description: msg,
			Color:       0x9B59B6, // Purple
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Admin",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to edit interaction response", "error", err)
		}
	}

	return cmd, handler
}

// AdminEndVotingCommand returns the end voting command
func AdminEndVotingCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-end-voting",
		Description: "[Admin] End current voting session",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		}); err != nil {
			slog.Error("Failed to send deferred response", "error", err)
			return
		}

		msg, err := client.AdminEndVoting()
		if err != nil {
			errorMsg := fmt.Sprintf("❌ Failed to end voting: %v", err)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &errorMsg,
			})
			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🛑 Admin End Voting",
			Description: msg,
			Color:       0x9B59B6, // Purple
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Admin",
			},
		}

		if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		}); err != nil {
			slog.Error("Failed to edit interaction response", "error", err)
		}
	}

	return cmd, handler
}
