package discord

import (
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// SlotsCommand returns the slots minigame command definition and handler
func SlotsCommand() (*discordgo.ApplicationCommand, CommandHandler) {
	minValue := float64(10)
	maxValue := float64(10000)

	cmd := &discordgo.ApplicationCommand{
		Name:        "slots",
		Description: "Spin the slots machine and win money!",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "bet",
				Description: "Amount of money to bet (10-10000)",
				Required:    true,
				MinValue:    &minValue,
				MaxValue:    maxValue,
			},
		},
	}

	handler := func(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
		// Defer response
		if !deferResponse(s, i) {
			return
		}

		// Extract user info
		user := getInteractionUser(i)

		// Get bet amount from options
		options := getOptions(i)
		betAmount := int(options[0].IntValue())

		// Register user
		if !ensureUserRegistered(s, i, client, user, false) {
			return
		}

		// Call API
		result, err := client.SpinSlots("discord", user.ID, user.Username, betAmount)
		if err != nil {
			slog.Error("Failed to spin slots", "error", err, "username", user.Username)
			respondFriendlyError(s, i, err.Error())
			return
		}

		// Build embed
		embed := buildSlotsEmbed(result)

		// Send response
		sendEmbed(s, i, embed)
	}

	return cmd, handler
}

// buildSlotsEmbed creates an embed for slots results
func buildSlotsEmbed(result *domain.SlotsResult) *discordgo.MessageEmbed {
	// Map symbols to emojis
	emojiMap := map[string]string{
		"LEMON":   "🍋",
		"CHERRY":  "🍒",
		"BELL":    "🔔",
		"BAR":     "💰",
		"SEVEN":   "7️⃣",
		"DIAMOND": "💎",
		"STAR":    "⭐",
	}

	// Format reels
	reels := fmt.Sprintf("%s | %s | %s",
		emojiMap[result.Reel1],
		emojiMap[result.Reel2],
		emojiMap[result.Reel3],
	)

	// Determine color based on outcome
	var color int
	switch result.TriggerType {
	case "mega_jackpot":
		color = 0xFFD700 // Gold
	case "jackpot":
		color = 0xFF8C00 // Dark orange
	case "big_win":
		color = 0xFFA500 // Orange
	default:
		if result.IsWin {
			color = 0x00FF00 // Green
		} else {
			color = 0xFF0000 // Red
		}
	}

	// Build fields
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Reels",
			Value:  reels,
			Inline: false,
		},
		{
			Name:   "Bet Amount",
			Value:  fmt.Sprintf("%d money", result.BetAmount),
			Inline: true,
		},
		{
			Name:   "Payout",
			Value:  fmt.Sprintf("%d money", result.PayoutAmount),
			Inline: true,
		},
	}

	// Add multiplier field for wins
	if result.PayoutMultiplier > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Multiplier",
			Value:  fmt.Sprintf("%.2fx", result.PayoutMultiplier),
			Inline: true,
		})
	}

	// Add near-miss indicator
	if result.IsNearMiss && !result.IsWin {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Close!",
			Value:  "You were so close! Try again!",
			Inline: false,
		})
	}

	// Determine title based on trigger type
	var title string
	switch result.TriggerType {
	case "mega_jackpot":
		title = "🌟 MEGA JACKPOT! 🌟"
	case "jackpot":
		title = "💎 JACKPOT! 💎"
	case "big_win":
		title = "🎉 BIG WIN! 🎉"
	default:
		if result.IsWin {
			title = "🎰 Slots - Win! 🎰"
		} else {
			title = "🎰 Slots 🎰"
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: result.Message,
		Color:       color,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Player: %s", result.Username),
		},
	}

	return embed
}
