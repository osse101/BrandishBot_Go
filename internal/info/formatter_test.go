package info_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/info"
)

func TestFormatter_FormatFeature(t *testing.T) {
	formatter := info.NewFormatter()

	feature := &info.Feature{
		Discord: info.PlatformContent{
			Description: "Discord feature description",
		},
		Streamerbot: info.PlatformContent{
			Description: "Streamerbot feature description",
		},
	}

	tests := []struct {
		name     string
		platform string
		expected string
	}{
		{
			name:     "Discord platform",
			platform: domain.PlatformDiscord,
			expected: "Discord feature description",
		},
		{
			name:     "Twitch platform",
			platform: domain.PlatformTwitch,
			expected: "Streamerbot feature description",
		},
		{
			name:     "Streamerbot platform",
			platform: "streamerbot",
			expected: "Streamerbot feature description",
		},
		{
			name:     "Unknown platform fallback",
			platform: "unknown",
			expected: "Discord feature description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatFeature(feature, tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatter_FormatTopic(t *testing.T) {
	formatter := info.NewFormatter()

	topic := &info.Topic{
		Discord: info.PlatformContent{
			Description: "Discord topic description",
		},
		Streamerbot: info.PlatformContent{
			Description: "Streamerbot topic description",
		},
	}

	tests := []struct {
		name     string
		platform string
		expected string
	}{
		{
			name:     "Discord platform",
			platform: domain.PlatformDiscord,
			expected: "Discord topic description",
		},
		{
			name:     "Twitch platform",
			platform: domain.PlatformTwitch,
			expected: "Streamerbot topic description",
		},
		{
			name:     "Streamerbot platform",
			platform: "streamerbot",
			expected: "Streamerbot topic description",
		},
		{
			name:     "Unknown platform fallback",
			platform: "unknown",
			expected: "Discord topic description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatTopic(topic, tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatter_FormatFeatureList(t *testing.T) {
	formatter := info.NewFormatter()

	features := map[string]*info.Feature{
		"farming":  {},
		"crafting": {},
	}
	gistLink := "https://gist.github.com/example"

	tests := []struct {
		name     string
		platform string
		expected []string // Check if all strings are contained
	}{
		{
			name:     "Discord platform",
			platform: domain.PlatformDiscord,
			expected: []string{"**BrandishBot Features**\nAvailable: ", "farming", "crafting", "\n\nFull documentation: " + gistLink},
		},
		{
			name:     "Twitch platform",
			platform: domain.PlatformTwitch,
			expected: []string{"BrandishBot Features: ", "farming", "crafting", " " + gistLink},
		},
		{
			name:     "Streamerbot platform",
			platform: "streamerbot",
			expected: []string{"BrandishBot Features: ", "farming", "crafting", " " + gistLink},
		},
		{
			name:     "Unknown platform fallback",
			platform: "unknown",
			expected: []string{"Available features: ", "farming", "crafting"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatFeatureList(features, tt.platform, gistLink)
			for _, expectedStr := range tt.expected {
				assert.Contains(t, result, expectedStr)
			}
		})
	}
}
