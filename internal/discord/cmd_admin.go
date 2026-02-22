package discord

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// AddItemCommand returns the add item command definition and handler (admin only)
func AddItemCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "add-item",
		Description: "[ADMIN] Add items to a user's inventory",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to add item to",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to add",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to add",
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
		targetUser := options[0].UserValue(s)
		itemName := options[1].StringValue()
		quantity := int(options[2].IntValue())

		// Ensure target user exists
		_, err := client.RegisterUser(targetUser.Username, targetUser.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		msg, err := client.AddItemByUsername(domain.PlatformDiscord, targetUser.Username, itemName, quantity)
		if err != nil {
			slog.Error("Failed to add item", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to add item: %v", err))
			return
		}

		embed := createEmbed("✅ Items Added", fmt.Sprintf("Added %d x %s to %s\n\n%s", quantity, itemName, targetUser.Username, msg), 0x2ecc71, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// RemoveItemCommand returns the remove item command definition and handler (admin only)
func RemoveItemCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "remove-item",
		Description: "[ADMIN] Remove items from a user's inventory",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to remove item from",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item",
				Description: "Item name to remove",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "Quantity to remove",
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
		targetUser := options[0].UserValue(s)
		itemName := options[1].StringValue()
		quantity := int(options[2].IntValue())

		// Ensure target user exists
		_, err := client.RegisterUser(targetUser.Username, targetUser.ID)
		if err != nil {
			slog.Error("Failed to register user", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		removed, err := client.RemoveItemByUsername(domain.PlatformDiscord, targetUser.Username, itemName, quantity)
		if err != nil {
			slog.Error("Failed to remove item", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to remove item: %v", err))
			return
		}

		// Build description with partial removal warning if applicable
		description := fmt.Sprintf("Removed %d x %s from %s\n\n**Items removed:** %d",
			quantity, itemName, targetUser.Username, removed)

		if removed < quantity {
			description += fmt.Sprintf("\n\n⚠️ **Partial Removal**: Only %d items were available (requested %d)",
				removed, quantity)
		}

		embed := createEmbed("🗑️ Items Removed", description, 0xe74c3c, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// AdminAwardXPCommand returns the award XP command definition and handler (admin only)
func AdminAwardXPCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-award-xp",
		Description: "[ADMIN] Award job XP to a user",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "platform",
				Description: "Platform (discord, twitch, youtube)",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Discord", Value: domain.PlatformDiscord},
					{Name: "Twitch", Value: domain.PlatformTwitch},
					{Name: "YouTube", Value: domain.PlatformYoutube},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "username",
				Description: "Username on the specified platform",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "job",
				Description:  "Job to award XP to",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Amount of XP to award (1-10000)",
				Required:    true,
				MinValue:    floatPtr(1.0),
				MaxValue:    10000.0,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		platform := options[0].StringValue()
		username := options[1].StringValue()
		jobKey := options[2].StringValue()
		amount := int(options[3].IntValue())

		// Call API to award XP
		result, err := client.AdminAwardXP(platform, username, jobKey, amount)
		if err != nil {
			slog.Error("Failed to award XP", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to award XP: %v", err))
			return
		}

		// Build response message
		description := fmt.Sprintf("Awarded **%d XP** to **%s** (@%s) for job **%s**",
			amount, platform, username, jobKey)

		if result.LeveledUp {
			description += fmt.Sprintf("\n\n🎉 **Level Up!** %s → %d",
				jobKey, result.NewLevel)
		}

		embed := &discordgo.MessageEmbed{
			Title:       "✅ XP Awarded",
			Description: description,
			Color:       0x3498db, // Blue
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Platform",
					Value:  platform,
					Inline: true,
				},
				{
					Name:   "Username",
					Value:  username,
					Inline: true,
				},
				{
					Name:   "Job",
					Value:  jobKey,
					Inline: true,
				},
				{
					Name:   "XP Awarded",
					Value:  fmt.Sprintf("%d", amount),
					Inline: true,
				},
				{
					Name:   "New Level",
					Value:  fmt.Sprintf("%d", result.NewLevel),
					Inline: true,
				},
				{
					Name:   "Total XP",
					Value:  fmt.Sprintf("%d", result.NewXP),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Admin Action by %s", i.Member.User.Username),
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

func floatPtr(v float64) *float64 {
	return &v
}

// UserLookupCommand returns the user lookup command definition and handler
func UserLookupCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-user",
		Description: "[ADMIN] Lookup detailed user information",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "username",
				Description: "Username to lookup",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "platform",
				Description: "Platform (default: discord)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Discord", Value: domain.PlatformDiscord},
					{Name: "Twitch", Value: domain.PlatformTwitch},
					{Name: "YouTube", Value: domain.PlatformYoutube},
				},
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		options := getOptions(i)
		username := options[0].StringValue()
		platform := domain.PlatformDiscord
		if len(options) > 1 {
			platform = options[1].StringValue()
		}

		result, err := client.AdminUserLookup(platform, username)
		if err != nil {
			slog.Error("Failed to lookup user", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to lookup user: %v", err))
			return
		}

		description := fmt.Sprintf("**ID:** %s\n**Platform ID:** %s\n**Created:** %s",
			result.ID, result.PlatformID, result.CreatedAt)

		embed := createEmbed("👤 User Lookup: "+result.Username, description, 0x3498db, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// RecentUsersCommand returns the recent users command definition and handler
func RecentUsersCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-users-recent",
		Description: "[ADMIN] List recently active users",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "limit",
				Description: "Number of users to show (default: 10)",
				Required:    false,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		limit := 10
		options := getOptions(i)
		if len(options) > 0 {
			limit = int(options[0].IntValue())
		}

		users, err := client.AdminGetRecentUsers(limit)
		if err != nil {
			slog.Error("Failed to get recent users", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to get recent users: %v", err))
			return
		}

		var sb strings.Builder
		for _, u := range users {
			fmt.Fprintf(&sb, "• **%s** (%s) - %s\n", u.Username, u.ID, u.UpdatedAt.Format("15:04:05"))
		}

		if sb.Len() == 0 {
			sb.WriteString("No recent users found.")
		}

		embed := createEmbed("🕒 Recent Users", sb.String(), 0x3498db, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// ActiveChattersCommand returns the active chatters command definition and handler
func ActiveChattersCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-users-active",
		Description: "[ADMIN] List active chatters",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "minutes",
				Description: "Time window in minutes (default: 10)",
				Required:    false,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		minutes := 10
		options := getOptions(i)
		if len(options) > 0 {
			minutes = int(options[0].IntValue())
		}

		users, err := client.AdminGetActiveChatters(minutes)
		if err != nil {
			slog.Error("Failed to get active chatters", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to get active chatters: %v", err))
			return
		}

		var sb strings.Builder
		// Handling []domain.User response
		for _, u := range users {
			fmt.Fprintf(&sb, "• %s\n", u.Username)
		}

		if sb.Len() == 0 {
			sb.WriteString("No active chatters found.")
		}

		embed := createEmbed(fmt.Sprintf("💬 Active Chatters (%d min)", minutes), sb.String(), 0x3498db, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// EventsCommand returns the events command definition and handler
func EventsCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-events",
		Description: "[ADMIN] View recent system events",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "limit",
				Description: "Number of events to show (default: 10)",
				Required:    false,
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		limit := 10
		options := getOptions(i)
		if len(options) > 0 {
			limit = int(options[0].IntValue())
		}

		events, err := client.AdminGetEvents(limit)
		if err != nil {
			slog.Error("Failed to get events", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to get events: %v", err))
			return
		}

		var sb strings.Builder
		for _, e := range events {
			// formatting log line roughly
			if len(e) > 100 {
				e = e[:97] + "..."
			}
			fmt.Fprintf(&sb, "`%s`\n", e)
		}

		if sb.Len() == 0 {
			sb.WriteString("No events found.")
		}

		embed := createEmbed("📜 System Events", sb.String(), 0x3498db, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// DailyResetCommand returns the daily reset command definition and handler
func DailyResetCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-reset-daily",
		Description:              "[ADMIN] Trigger manual daily reset",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		msg, err := client.AdminManualDailyReset()
		if err != nil {
			slog.Error("Failed to trigger daily reset", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to triggered reset: %v", err))
			return
		}

		embed := createEmbed("🔄 Daily Reset Triggered", fmt.Sprintf("Response: %s", msg), 0xe67e22, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// ResetStatusCommand returns the reset status command definition and handler
func ResetStatusCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-reset-status",
		Description:              "[ADMIN] Check daily reset status",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		status, err := client.AdminGetResetStatus()
		if err != nil {
			slog.Error("Failed to get reset status", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to get status: %v", err))
			return
		}

		description := fmt.Sprintf("**Last Reset:** %s\n**Next Reset:** %s\n**Records Affected:** %d",
			status.LastResetTime.Format(time.RFC1123),
			status.NextResetTime.Format(time.RFC1123),
			status.RecordsAffected)

		embed := createEmbed("⏱️ Reset Status", description, 0x3498db, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// MetricsCommand returns the metrics command definition and handler
func MetricsCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-metrics",
		Description:              "[ADMIN] View system metrics",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		metrics, err := client.AdminGetMetrics()
		if err != nil {
			slog.Error("Failed to get metrics", "error", err)
			respondError(s, i, fmt.Sprintf("Failed to get metrics: %v", err))
			return
		}

		var sb strings.Builder
		for k, v := range metrics {
			fmt.Fprintf(&sb, "**%s**: %v\n", k, v)
		}

		embed := createEmbed("📊 System Metrics", sb.String(), 0x3498db, FooterAdminAction)
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// SimulationCommand returns the simulation command definition and handler
func SimulationCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "admin-simulation",
		Description: "[ADMIN] Manage simulations",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "capabilities",
				Description: "List simulation capabilities",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "scenarios",
				Description: "List available scenarios",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "run",
				Description: "Run a scenario",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "scenario_id",
						Description: "ID of the scenario to run",
						Required:    true,
					},
				},
			},
		},
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		subcmd := i.ApplicationCommandData().Options[0].Name

		switch subcmd {
		case "capabilities":
			caps, err := client.AdminGetScenarioCapabilities()
			if err != nil {
				respondError(s, i, fmt.Sprintf("Error: %v", err))
				return
			}
			var sb strings.Builder
			for _, c := range caps {
				fmt.Fprintf(&sb, "**%s**: %s (Params: %v)\n", c.ID, c.Description, c.Params)
			}
			sendEmbed(s, i, createEmbed("🧠 Capabilities", sb.String(), 0x9b59b6, FooterAdminAction))

		case "scenarios":
			scenarios, err := client.AdminGetScenarios()
			if err != nil {
				respondError(s, i, fmt.Sprintf("Error: %v", err))
				return
			}
			var sb strings.Builder
			for _, sc := range scenarios {
				fmt.Fprintf(&sb, "**%s**: %s\n", sc.ID, sc.Description)
			}
			sendEmbed(s, i, createEmbed("🎬 Scenarios", sb.String(), 0x9b59b6, FooterAdminAction))

		case "run":
			scenarioID := i.ApplicationCommandData().Options[0].Options[0].StringValue()
			result, err := client.AdminRunScenario(scenarioID, nil)
			if err != nil {
				respondError(s, i, fmt.Sprintf("Error: %v", err))
				return
			}

			desc := fmt.Sprintf("**Status**: %s\n\n**Logs**:\n```\n%s\n```", result.Status, result.Logs)
			if len(desc) > 2000 {
				desc = desc[:1997] + "..."
			}

			sendEmbed(s, i, createEmbed("▶️ Simulation Run: "+scenarioID, desc, 0x2ecc71, FooterAdminAction))
		}
	}

	return cmd, handler
}
