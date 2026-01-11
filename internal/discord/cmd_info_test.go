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
)

func TestInfoCommand_Overview(t *testing.T) {
	ctx := SetupTestContext(t)
	cmd, handler := InfoCommand()

	// Setup Test Info Dir
	tmpDir, err := os.MkdirTemp("", "info_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create overview.txt
	err = os.WriteFile(filepath.Join(tmpDir, "overview.txt"), []byte("Welcome to BrandishBot!"), 0644)
	assert.NoError(t, err)

	// Override InfoDir
	oldInfoDir := InfoDir
	InfoDir = tmpDir
	defer func() { InfoDir = oldInfoDir }()

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
		assert.Contains(t, sentEmbed.Description, "Welcome to BrandishBot!")
		assert.Contains(t, sentEmbed.Title, "Overview")
	}
}

func TestInfoCommand_SpecificFeature(t *testing.T) {
	ctx := SetupTestContext(t)
	cmd, handler := InfoCommand()

	tmpDir, _ := os.MkdirTemp("", "info_test")
	defer os.RemoveAll(tmpDir)
	os.WriteFile(filepath.Join(tmpDir, "economy.txt"), []byte("Money stuff"), 0644)

	oldInfoDir := InfoDir
	InfoDir = tmpDir
	defer func() { InfoDir = oldInfoDir }()

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

func TestInfoCommand_FileNotFound(t *testing.T) {
	ctx := SetupTestContext(t)
	cmd, handler := InfoCommand()

	tmpDir, _ := os.MkdirTemp("", "info_test")
	defer os.RemoveAll(tmpDir)
	// No files created

	oldInfoDir := InfoDir
	InfoDir = tmpDir
	defer func() { InfoDir = oldInfoDir }()

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

	assert.Contains(t, sentContent, "Error loading information")
}
