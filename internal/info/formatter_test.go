package info

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestNewFormatter(t *testing.T) {
	f := NewFormatter()
	assert.NotNil(t, f, "Formatter should not be nil")
}

func TestFormatter_FormatFeature(t *testing.T) {
	f := NewFormatter()
	feature := &Feature{
		Discord: PlatformContent{
			Description: "Discord feature description",
		},
		Streamerbot: PlatformContent{
			Description: "Streamerbot feature description",
		},
	}

	tests := []struct {
		name     string
		platform string
		expected string
	}{
		{
			name:     "Discord Platform",
			platform: domain.PlatformDiscord,
			expected: "Discord feature description",
		},
		{
			name:     "Twitch Platform",
			platform: domain.PlatformTwitch,
			expected: "Streamerbot feature description",
		},
		{
			name:     "Streamerbot Platform",
			platform: platformStreamerbot,
			expected: "Streamerbot feature description",
		},
		{
			name:     "Unknown Platform defaults to Discord",
			platform: "unknown",
			expected: "Discord feature description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatFeature(feature, tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatter_FormatTopic(t *testing.T) {
	f := NewFormatter()
	topic := &Topic{
		Discord: PlatformContent{
			Description: "Discord topic description",
		},
		Streamerbot: PlatformContent{
			Description: "Streamerbot topic description",
		},
	}

	tests := []struct {
		name     string
		platform string
		expected string
	}{
		{
			name:     "Discord Platform",
			platform: domain.PlatformDiscord,
			expected: "Discord topic description",
		},
		{
			name:     "Twitch Platform",
			platform: domain.PlatformTwitch,
			expected: "Streamerbot topic description",
		},
		{
			name:     "Streamerbot Platform",
			platform: platformStreamerbot,
			expected: "Streamerbot topic description",
		},
		{
			name:     "Unknown Platform defaults to Discord",
			platform: "unknown",
			expected: "Discord topic description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatTopic(topic, tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatter_FormatFeatureList(t *testing.T) {
	f := NewFormatter()
	features := map[string]*Feature{
		"feature1": {},
		"feature2": {},
	}
	gistLink := "https://gist.github.com/test"

	tests := []struct {
		name     string
		platform string
		checkFunc func(*testing.T, string)
	}{
		{
			name:     "Discord Platform",
			platform: domain.PlatformDiscord,
			checkFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "**BrandishBot Features**")
				assert.Contains(t, result, "feature1")
				assert.Contains(t, result, "feature2")
				assert.Contains(t, result, gistLink)
			},
		},
		{
			name:     "Twitch Platform",
			platform: domain.PlatformTwitch,
			checkFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "BrandishBot Features:")
				assert.Contains(t, result, "feature1")
				assert.Contains(t, result, "feature2")
				assert.Contains(t, result, gistLink)
				assert.NotContains(t, result, "**")
			},
		},
		{
			name:     "Unknown Platform defaults",
			platform: "unknown",
			checkFunc: func(t *testing.T, result string) {
				assert.Contains(t, result, "Available features:")
				assert.Contains(t, result, "feature1")
				assert.Contains(t, result, "feature2")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FormatFeatureList(features, tt.platform, gistLink)
			tt.checkFunc(t, result)
		})
	}
}
