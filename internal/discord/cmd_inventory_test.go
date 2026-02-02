package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/user"
)

func TestInventoryCommand_Empty(t *testing.T) {
	ctx := SetupTestContext(t)
	cmd, handler := InventoryCommand()

	// Mock Register User
	ctx.Mux.HandleFunc("/api/v1/user/register", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, map[string]interface{}{"id": "u", "username": "t"})
	})

	// Mock Empty Inventory
	ctx.Mux.HandleFunc("/api/v1/user/inventory", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, map[string]interface{}{"items": []user.InventoryItem{}})
	})

	// Request
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: cmd.Name,
			},
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "test-user", Username: "Tester"},
			},
		},
	}

	// Capture Response
	var sentEmbed *discordgo.MessageEmbed
	ctx.DiscordMocks.RoundTripFunc = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPatch {
			var body discordgo.WebhookEdit
			json.NewDecoder(req.Body).Decode(&body)
			if body.Embeds != nil && len(*body.Embeds) > 0 {
				sentEmbed = (*body.Embeds)[0]
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("{}")),
		}, nil
	}

	handler(ctx.Session, interaction, ctx.APIClient)

	assert.NotNil(t, sentEmbed)
	if sentEmbed != nil {
		assert.Contains(t, sentEmbed.Title, "Tester's Inventory")
		assert.Contains(t, sentEmbed.Description, "Your inventory is empty")
	}
}

func TestInventoryCommand_WithItems(t *testing.T) {
	ctx := SetupTestContext(t)
	cmd, handler := InventoryCommand()

	ctx.Mux.HandleFunc("/api/v1/user/register", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, map[string]interface{}{"id": "u", "username": "t"})
	})

	ctx.Mux.HandleFunc("/api/v1/user/inventory", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, map[string]interface{}{
			"items": []user.InventoryItem{
				{Name: "sword", Quantity: 1},
				{Name: "potion", Quantity: 5},
			},
		})
	})

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: cmd.Name,
			},
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "test-user", Username: "Tester"},
			},
		},
	}

	var sentEmbed *discordgo.MessageEmbed
	ctx.DiscordMocks.RoundTripFunc = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPatch {
			var body discordgo.WebhookEdit
			json.NewDecoder(req.Body).Decode(&body)
			if body.Embeds != nil && len(*body.Embeds) > 0 {
				sentEmbed = (*body.Embeds)[0]
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("{}")),
		}, nil
	}

	handler(ctx.Session, interaction, ctx.APIClient)

	assert.NotNil(t, sentEmbed)
	if sentEmbed != nil {
		assert.Contains(t, sentEmbed.Description, "**sword** x1")
		assert.Contains(t, sentEmbed.Description, "**potion** x5")
	}
}

func TestInventoryCommand_RegisterError(t *testing.T) {
	ctx := SetupTestContext(t)
	_, handler := InventoryCommand()

	// Mock Register Error
	ctx.Mux.HandleFunc("/api/v1/user/register", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	})

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "u", Username: "t"},
			},
		},
	}

	var sentContent string
	ctx.DiscordMocks.RoundTripFunc = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPatch {
			var body discordgo.WebhookEdit
			json.NewDecoder(req.Body).Decode(&body)
			if body.Content != nil {
				sentContent = *body.Content
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("{}")),
		}, nil
	}

	handler(ctx.Session, interaction, ctx.APIClient)

	assert.Contains(t, sentContent, "❌") // Check for friendly wrapper
}
