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
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
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

		embed := createEmbed("🔒 Admin Relock", msg, 0x95a5a6, FooterBrandishBotAdmin)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// AdminInstantResolveCommand returns the admin instant resolve command
func AdminInstantResolveCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-instant-resolve",
		Description:              "[Admin] Force end voting and unlock winner immediately",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		msg, err := client.AdminInstantUnlock()
		if err != nil {
			slog.Error("Failed to instant unlock", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to instant unlock: %v", err))
			return
		}

		embed := createEmbed("⚡ Admin Instant Resolve", msg, 0xf1c40f, FooterBrandishBotAdmin)
		sendEmbed(s, i, embed)
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
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		confirmation := options[0].StringValue()

		if confirmation != "CONFIRM RESET" {
			respondError(s, i, "Invalid confirmation. Type 'CONFIRM RESET' exactly.")
			return
		}

		user := getInteractionUser(i)

		msg, err := client.AdminResetProgression(user.Username, "Discord Admin Command", true)
		if err != nil {
			slog.Error("Failed to reset tree", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to reset tree: %v", err))
			return
		}

		embed := createEmbed("☢️ Progression Tree Reset", msg, 0xff0000, FooterBrandishBotAdmin)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// AdminTreeStatusCommand returns the tree status command
func AdminTreeStatusCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-tree-status",
		Description:              "[Admin] View full progression tree status",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
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

		embed := createEmbed("🌳 Progression Tree Status", statusText, 0x2ecc71, FooterBrandishBotAdmin)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// AdminStartVotingCommand returns the start voting command
func AdminStartVotingCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-start-voting",
		Description:              "[Admin] Start a new voting session",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := genericAdminCommandHandler(
		"🗳️ Admin Start Voting",
		0x9B59B6,
		"Failed to start voting",
		func(c *APIClient) (string, error) { return c.AdminStartVoting() },
	)

	return cmd, handler
}

// AdminEndVotingCommand returns the end voting command
func AdminEndVotingCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-end-voting",
		Description:              "[Admin] End current voting session",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := genericAdminCommandHandler(
		"🛑 Admin End Voting",
		0x9B59B6,
		"Failed to end voting",
		func(c *APIClient) (string, error) { return c.AdminEndVoting() },
	)

	return cmd, handler
}

// AdminAddContributionCommand returns the admin add contribution command
func AdminAddContributionCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-contribution",
		Description: "[Admin] Add progression contribution points",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of contribution points to add",
				Required:    true,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		amount := int(options[0].IntValue())

		msg, err := client.AdminAddContribution(amount)
		if err != nil {
			errorMsg := fmt.Sprintf("❌ Failed to add contribution: %v", err)
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &errorMsg,
			}); err != nil {
				slog.Error("Failed to send error response", "error", err)
			}
			return
		}

		embed := createEmbed("📈 Admin Contribution Added", fmt.Sprintf("Successfully added **%d** contribution points.\n\n%s", amount, msg), 0x2ecc71, FooterBrandishBotAdmin)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// AdminReloadWeightsCommand returns the reload weights command definition and handler
func AdminReloadWeightsCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-reload-weights",
		Description:              "[Admin] Reload engagement weight cache",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := genericAdminCommandHandler(
		"🔄 Engagement Weights Reloaded",
		0x3498db,
		"Failed to reload weights",
		func(c *APIClient) (string, error) { return c.AdminReloadWeights() },
	)

	return cmd, handler
}

func genericAdminCommandHandler(title string, color int, errLogMsg string, action func(*APIClient) (string, error)) CommandHandler {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		msg, err := action(client)
		if err != nil {
			errorMsg := fmt.Sprintf("❌ %s: %v", errLogMsg, err)
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &errorMsg,
			}); err != nil {
				slog.Error("Failed to send error response", "error", err)
			}
			return
		}

		embed := createEmbed(title, msg, color, FooterBrandishBotAdmin)
		sendEmbed(s, i, embed)
	}
}
