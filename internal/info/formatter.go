package info

import (
	"fmt"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

const (
	platformStreamerbot = "streamerbot"
)

// Formatter provides platform-specific formatting for info content
type Formatter struct{}

// NewFormatter creates a new formatter
func NewFormatter() *Formatter {
	return &Formatter{}
}

// getContentByPlatform returns the platform-specific content
func (f *Formatter) getContentByPlatform(discordContent, streamerbotContent, platform string) string {
	switch strings.ToLower(platform) {
	case domain.PlatformDiscord:
		return discordContent
	case domain.PlatformTwitch, platformStreamerbot:
		return streamerbotContent
	default:
		// Fallback to Discord format
		return discordContent
	}
}

// FormatFeature formats a feature for the specified platform
func (f *Formatter) FormatFeature(feature *Feature, platform string) string {
	return f.getContentByPlatform(feature.Discord.Description, feature.Streamerbot.Description, platform)
}

// FormatTopic formats a topic for the specified platform
func (f *Formatter) FormatTopic(topic *Topic, platform string) string {
	return f.getContentByPlatform(topic.Discord.Description, topic.Streamerbot.Description, platform)
}

// FormatFeatureList formats a list of available features for the platform
func (f *Formatter) FormatFeatureList(features map[string]*Feature, platform string, gistLink string) string {
	featureNames := make([]string, 0, len(features))
	for name := range features {
		featureNames = append(featureNames, name)
	}

	switch strings.ToLower(platform) {
	case domain.PlatformDiscord:
		return fmt.Sprintf("**BrandishBot Features**\nAvailable: %s\n\nFull documentation: %s",
			strings.Join(featureNames, ", "), gistLink)
	case domain.PlatformTwitch, platformStreamerbot:
		return fmt.Sprintf("BrandishBot Features: %s %s",
			strings.Join(featureNames, ", "), gistLink)
	default:
		return fmt.Sprintf("Available features: %s", strings.Join(featureNames, ", "))
	}
}
