package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/stretchr/testify/assert"
)

func TestProfileCommand_Success(t *testing.T) {
	ctx := SetupTestContext(t)
	cmd, handler := ProfileCommand()

	// Mock Register
	ctx.Mux.HandleFunc("/api/v1/user/register", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, map[string]interface{}{"internal_id": "u-123", "username": "Tester"})
	})

	// Mock Inventory (for item count)
	ctx.Mux.HandleFunc("/api/v1/user/inventory", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, map[string]interface{}{
			"items": []user.UserInventoryItem{
				{Name: "item1", Quantity: 1},
				{Name: "item2", Quantity: 2},
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
				User: &discordgo.User{ID: "123", Username: "Tester", Avatar: "abc"},
			},
		},
	}

	var sentEmbed *discordgo.MessageEmbed
	ctx.DiscordMocks.RoundTripFunc = func(req *http.Request) (*http.Response, error) {
		if req.Method == "PATCH" {
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
		t.Logf("Embed Title: %s", sentEmbed.Title)
		for i, f := range sentEmbed.Fields {
			t.Logf("Field %d: Name='%s' Value='%s'", i, f.Name, f.Value)
		}

		assert.Contains(t, sentEmbed.Title, "Tester's Profile")
		// Check Fields
		foundID := false
		foundItems := false
		for _, field := range sentEmbed.Fields {
			if field.Name == "User ID" && field.Value == "u-123" {
				foundID = true
			}
			if field.Name == "Items" && field.Value == "2" {
				foundItems = true
			}
		}
		assert.True(t, foundID, "Profile should contain User ID")
		assert.True(t, foundItems, "Profile should contain Item count")
	}
}

func TestProfileCommand_RegisterError(t *testing.T) {
	ctx := SetupTestContext(t)
	_, handler := ProfileCommand()

	// Mock Error
	ctx.Mux.HandleFunc("/api/v1/user/register", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusInternalServerError)
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
		if req.Method == "PATCH" {
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

	assert.Contains(t, sentContent, "❌") // Checks for friendly error wrapper
}
