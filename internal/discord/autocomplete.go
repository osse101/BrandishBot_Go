package discord

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
)

// HandleAutocomplete routes autocomplete interactions to the appropriate handler
func HandleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
	data := i.ApplicationCommandData()

	switch data.Name {
	case "upgrade":
		handleRecipeAutocomplete(s, i, client)
	case "job-bonus":
		handleJobAutocomplete(s, i)
	case "use":
		handleItemAutocomplete(s, i, client, true, nil)
	case "buy":
		handleItemAutocomplete(s, i, client, false, nil)
	case "sell", "give":
		handleItemAutocomplete(s, i, client, true, nil)
	case "disassemble":
		handleItemAutocomplete(s, i, client, true, nil)
	case "gamble-start", "gamble-join":
		handleGambleItemAutocomplete(s, i, client)
	default:
		slog.Warn("Unhandled autocomplete command", "command", data.Name)
	}
}

// handleRecipeAutocomplete provides autocomplete for crafting recipes
func handleRecipeAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
	user := i.Member.User
	if user == nil {
		user = i.User
	}

	// Defensive check
	if user == nil {
		return
	}

	data := i.ApplicationCommandData()
	var focusedValue string
	for _, opt := range data.Options {
		if opt.Focused {
			focusedValue = strings.ToLower(opt.StringValue())
			break
		}
	}

	// Get unlocked recipes for this user
	recipes, err := client.GetUnlockedRecipes(domain.PlatformDiscord, user.ID, user.Username)
	if err != nil {
		slog.Error("Failed to get recipes for autocomplete", "error", err)
		// Fallback to all recipes if getting unlocked fails? Or just return error choice
		// Let's try to get all recipes as fallback if unlocked fails?
		// No, usually if backend fails we probably can't get all either.
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, r := range recipes {
		if focusedValue == "" || strings.Contains(strings.ToLower(r.ItemName), focusedValue) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  r.ItemName,
				Value: r.ItemName, // Pass name as value, matching new UpgradeCommand logic
			})
		}
		if len(choices) >= 25 {
			break
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

// handleJobAutocomplete provides autocomplete for job keys
func handleJobAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	var focusedValue string
	for _, opt := range data.Options {
		if opt.Focused {
			focusedValue = strings.ToLower(opt.StringValue())
			break
		}
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, jobInfo := range job.AllJobs {
		if focusedValue == "" || strings.Contains(strings.ToLower(jobInfo.DisplayName), focusedValue) || strings.Contains(strings.ToLower(jobInfo.Key), focusedValue) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  jobInfo.DisplayName,
				Value: jobInfo.Key,
			})
		}
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}

// handleItemAutocomplete provides autocomplete suggestions for item names
// onlyOwned: if true, only shows items from user's inventory
// filterFunc: optional custom filter function
// handleItemAutocomplete provides autocomplete suggestions for item names
// onlyOwned: if true, only shows items from user's inventory
// filterFunc: optional custom filter function
func handleItemAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient, onlyOwned bool, filterFunc func(string) bool) {
	user := getInteractionUser(i)

	if user == nil {
		slog.Error("Failed to get user from autocomplete interaction")
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Error: Unable to identify user", Value: "error"},
				},
			},
		})
		return
	}

	focusedValue := getFocusedOptionValue(i.ApplicationCommandData().Options)

	var choices []*discordgo.ApplicationCommandOptionChoice
	if onlyOwned {
		choices = getOwnedItemChoices(client, user, focusedValue, filterFunc)
	} else {
		choices = getBuyableItemChoices(focusedValue)
	}

	if len(choices) == 0 {
		choices = getNoItemsChoices(onlyOwned)
	}

	respondAutocomplete(s, i, choices)
}

func getFocusedOptionValue(options []*discordgo.ApplicationCommandInteractionDataOption) string {
	for _, opt := range options {
		if opt.Focused {
			return strings.ToLower(opt.StringValue())
		}
	}
	return ""
}

func getOwnedItemChoices(client *APIClient, user *discordgo.User, focusedValue string, filterFunc func(string) bool) []*discordgo.ApplicationCommandOptionChoice {
	inventory, err := client.GetInventory(domain.PlatformDiscord, user.ID, user.Username, "")
	if err != nil {
		slog.Error("Failed to get inventory for autocomplete", "error", err, "user", user.Username)
		return getCommonItemChoices(focusedValue)
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, item := range inventory {
		itemNameLower := strings.ToLower(item.PublicName)
		if focusedValue == "" || strings.Contains(itemNameLower, focusedValue) {
			if filterFunc != nil && !filterFunc(item.PublicName) {
				continue
			}
			displayName := fmt.Sprintf("%s (x%d)", item.PublicName, item.Quantity)
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  displayName,
				Value: item.PublicName,
			})
		}
		if len(choices) >= 25 {
			break
		}
	}
	return choices
}

func getNoItemsChoices(onlyOwned bool) []*discordgo.ApplicationCommandOptionChoice {
	if onlyOwned {
		return []*discordgo.ApplicationCommandOptionChoice{
			{Name: "No items found (try /search to find items)", Value: "none"},
		}
	}
	return []*discordgo.ApplicationCommandOptionChoice{
		{Name: "No matching items", Value: "none"},
	}
}

func respondAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, choices []*discordgo.ApplicationCommandOptionChoice) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		slog.Error("Failed to respond to autocomplete", "error", err)
	}
}

// handleGambleItemAutocomplete provides autocomplete for gamble commands (lootboxes only)
func handleGambleItemAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, client *APIClient) {
	// Filter to only show lootbox items
	lootboxFilter := func(itemName string) bool {
		// Use prefix check for precision - avoids matching non-lootbox items like "toolbox"
		return strings.HasPrefix(itemName, "lootbox") ||
			itemName == "junkbox" ||
			itemName == "goldbox"
	}

	handleItemAutocomplete(s, i, client, true, lootboxFilter)
}

// getCommonItemChoices returns a fallback list of common items
func getCommonItemChoices(filter string) []*discordgo.ApplicationCommandOptionChoice {
	commonItems := []struct {
		Name  string
		Value string
	}{
		{"Money", "money"},
		{"Junkbox (Tier 0)", "junkbox"},
		{"Lootbox (Tier 1)", "lootbox"},
		{"Goldbox (Tier 2)", "goldbox"},
		{"Missile", "missile"},
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, item := range commonItems {
		if filter == "" || strings.Contains(strings.ToLower(item.Name), filter) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  item.Name,
				Value: item.Value,
			})
		}
	}

	return choices
}

// getBuyableItemChoices returns items available for purchase
func getBuyableItemChoices(filter string) []*discordgo.ApplicationCommandOptionChoice {
	// These should match what's actually buyable in your shop
	buyableItems := []struct {
		Name  string
		Value string
	}{
		{"Junkbox (Tier 0) - Cheapest", "junkbox"},
		{"Lootbox (Tier 1) - Common", "lootbox"},
		{"Goldbox (Tier 2) - Rare", "goldbox"},
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, item := range buyableItems {
		if filter == "" || strings.Contains(strings.ToLower(item.Name), filter) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  item.Name,
				Value: item.Value,
			})
		}
	}

	return choices
}
