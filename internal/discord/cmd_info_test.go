package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/info"
)

func createTestLoader(t *testing.T) (*info.Loader, func()) {
	tmpDir, err := os.MkdirTemp("", "info_test")
	assert.NoError(t, err)

	// Create overview feature
	overviewYaml := `
name: overview
title: BrandishBot Overview
discord:
  description: "Welcome to BrandishBot!"
streamerbot:
  description: "Welcome"
`
	err = os.WriteFile(filepath.Join(tmpDir, "overview.yaml"), []byte(overviewYaml), 0644)
	assert.NoError(t, err)

	// Create economy feature
	economyYaml := `
name: economy
title: Economy System
discord:
  description: "Money stuff"
streamerbot:
  description: "Money"
`
	err = os.WriteFile(filepath.Join(tmpDir, "economy.yaml"), []byte(economyYaml), 0644)
	assert.NoError(t, err)

	loader := info.NewLoader(tmpDir)
	return loader, func() { os.RemoveAll(tmpDir) }
}

func TestInfoCommand_Overview(t *testing.T) {
	ctx := SetupTestContext(t)
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	cmd, handler := InfoCommand(loader)

	// Request
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: cmd.Name,
				// No options = overview
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

	handler(ctx.Session, interaction, ctx.APIClient)

	// Verify
	assert.NotNil(t, sentEmbed)
	if sentEmbed != nil {
		// New behavior: returns list of features
		assert.Contains(t, sentEmbed.Description, "Available:")
		assert.Contains(t, sentEmbed.Description, "economy")
		assert.Contains(t, sentEmbed.Description, "overview")
		assert.Contains(t, sentEmbed.Title, "Overview")
	}
}

func TestInfoCommand_SpecificFeature(t *testing.T) {
	ctx := SetupTestContext(t)
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	cmd, handler := InfoCommand(loader)

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: cmd.Name,
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "feature", Value: "economy", Type: discordgo.ApplicationCommandOptionString},
				},
			},
			Member: &discordgo.Member{User: &discordgo.User{ID: "u", Username: "t"}},
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
		assert.Contains(t, sentEmbed.Title, "Economy")
		assert.Contains(t, sentEmbed.Description, "Money stuff")
	}
}

func TestInfoCommand_NotFound(t *testing.T) {
	ctx := SetupTestContext(t)
	loader, cleanup := createTestLoader(t)
	defer cleanup()

	cmd, handler := InfoCommand(loader)

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: cmd.Name,
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "feature", Value: "missing", Type: discordgo.ApplicationCommandOptionString},
				},
			},
			Member: &discordgo.Member{User: &discordgo.User{ID: "u", Username: "t"}},
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

	// The friendly error message (from respondFriendlyError -> formatFriendlyError)
	// typically prefixes "❌ " for unknown errors or returns specific messages.
	// Since "Info not found" is not a specific mapped error in formatFriendlyError,
	// it will be returned as "❌ Info not found..."
	assert.Contains(t, sentContent, "Info not found")
}
