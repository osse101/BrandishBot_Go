package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// AdminCacheStatsCommand returns the cache stats command definition and handler
func AdminCacheStatsCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	cmd := &discordgo.ApplicationCommand{
		Name:                     "admin-cache-stats",
		Description:              "[Admin] View user cache statistics",
		DefaultMemberPermissions: &[]int64{discordgo.PermissionAdministrator}[0],
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		if !deferResponse(s, i) {
			return
		}

		stats, err := client.AdminGetCacheStats()
		if err != nil {
			errorMsg := fmt.Sprintf("❌ Failed to get cache stats: %v", err)
			if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &errorMsg,
			}); err != nil {
				slog.Error("Failed to send error response", "error", err)
			}
			return
		}

		hitRate := 0.0
		total := stats.Hits + stats.Misses
		if total > 0 {
			hitRate = float64(stats.Hits) / float64(total) * 100
		}

		description := fmt.Sprintf(
			"**Cache Hit Rate:** %.1f%%\\n"+
				"**Hits:** %d\\n"+
				"**Misses:** %d\\n"+
				"**Evictions:** %d\\n"+
				"**Current Size:** %d entries",
			hitRate, stats.Hits, stats.Misses, stats.Evictions, stats.Size,
		)

		embed := &discordgo.MessageEmbed{
			Title:       "📊 User Cache Statistics",
			Description: description,
			Color:       0x3498db, // Blue
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
