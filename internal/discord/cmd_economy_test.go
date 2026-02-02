package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestPricesCommand_Buy(t *testing.T) {
	ctx := SetupTestContext(t)
	cmd, handler := PricesCommand()

	// Mock Backend Response
	ctx.Mux.HandleFunc("/api/v1/prices/buy", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		WriteJSON(w, map[string]interface{}{
			"items": []domain.Item{
				{InternalName: "health_potion", BaseValue: 50},
				{InternalName: "iron_sword", BaseValue: 200},
			},
		})
	})

	// Setup Request
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

	// Capture Discord Interaction Edit
	var sentEmbed *discordgo.MessageEmbed
	ctx.DiscordMocks.RoundTripFunc = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPatch {
			// This is likely the ResponseEdit call
			// Parse body to verify embed
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

	// Execute Handler
	handler(ctx.Session, interaction, ctx.APIClient)

	// Verify
	assert.NotNil(t, sentEmbed, "Should send an embed response")
	if sentEmbed != nil {
		assert.Contains(t, sentEmbed.Title, "Buy Prices")
		assert.Contains(t, sentEmbed.Description, "health_potion")
		assert.Contains(t, sentEmbed.Description, "50 coins")
		assert.Contains(t, sentEmbed.Description, "iron_sword")
	}
}

func TestPricesCommand_Sell(t *testing.T) {
	ctx := SetupTestContext(t)
	cmd, handler := SellPricesCommand()

	// Mock Backend Response
	ctx.Mux.HandleFunc("/api/v1/prices", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		WriteJSON(w, map[string]interface{}{
			"items": []domain.Item{
				{InternalName: "gold_nugget", BaseValue: 100},
			},
		})
	})

	// Setup Request
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

	// Capture response
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

	// Execute
	handler(ctx.Session, interaction, ctx.APIClient)

	// Verify
	assert.NotNil(t, sentEmbed)
	if sentEmbed != nil {
		assert.Contains(t, sentEmbed.Title, "Sell Prices")
		assert.Contains(t, sentEmbed.Description, "gold_nugget")
		assert.Contains(t, sentEmbed.Description, "100 coins")
	}
}
