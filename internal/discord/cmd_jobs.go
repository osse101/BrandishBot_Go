package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// JobProgressCommand returns the job progress command definition and handler
func JobProgressCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:        "jobs",
		Description: "View your job levels and XP progress",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "User to check (optional, defaults to self)",
				Required:    false,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		var targetUser *discordgo.User
		options := i.ApplicationCommandData().Options
		if len(options) > 0 {
			targetUser = options[0].UserValue(s)
		} else {
			targetUser = i.Interaction.Member.User
		}

		// Ensure user is registered/known
		if !ensureUserRegistered(s, i, client, targetUser, true) {
			return
		}

		// Get user jobs
		jobsData, err := client.GetUserJobs(domain.PlatformDiscord, targetUser.ID)
		if err != nil {
			slog.Error("Failed to get user jobs", "error", err)
			respondFriendlyError(s, i, err.Error())
			return
		}

		// Extract jobs list from response
		jobsList := jobsData.Jobs
		if len(jobsList) == 0 {
			embed := createEmbed("📋 Job Progress", fmt.Sprintf("%s has no job progress yet.", targetUser.Username), 0x95A5A6, "")
			sendEmbed(s, i, embed)
			return
		}

		primaryJob := jobsData.PrimaryJob

		// Build embed with job progress
		var fields []*discordgo.MessageEmbedField
		for _, job := range jobsList {
			jobKey := job.JobKey
			// Let's Capitalize JobKey for display
			displayName := cases.Title(language.English).String(strings.ReplaceAll(jobKey, "_", " "))

			level := float64(job.Level)
			currentXP := float64(job.XP)
			xpToNext := float64(job.XPForNext)

			// Calculate progress percentage
			progressPct := 0.0
			if xpToNext > 0 {
				progressPct = (currentXP / xpToNext) * 100
			}

			// Add star emoji for primary job
			nameDisplay := displayName
			if jobKey == primaryJob {
				nameDisplay = "⭐ " + displayName
			}

			// Create progress bar
			progressBar := createProgressBar(progressPct)

			fieldValue := fmt.Sprintf("Level %d\n%s `%.0f%%`\nXP: %.0f / %.0f",
				int(level),
				progressBar,
				progressPct,
				currentXP,
				xpToNext,
			)

			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   nameDisplay,
				Value:  fieldValue,
				Inline: true,
			})
		}

		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("📋 %s's Job Progress", targetUser.Username),
			Description: "⭐ indicates your primary job",
			Fields:      fields,
			Color:       0x3498DB, // Blue
			Footer: &discordgo.MessageEmbedFooter{
				Text: "BrandishBot Jobs",
			},
		}

		sendEmbed(s, i, embed)
	}

	return cmd, handler
}
