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

// FormatFeature formats a feature for the specified platform
func (f *Formatter) FormatFeature(feature *Feature, platform string) string {
	switch strings.ToLower(platform) {
	case domain.PlatformDiscord:
		return feature.Discord.Description
	case domain.PlatformTwitch, platformStreamerbot:
		return feature.Streamerbot.Description
	default:
		// Fallback to Discord format
		return feature.Discord.Description
	}
}

// FormatTopic formats a topic for the specified platform
func (f *Formatter) FormatTopic(topic *Topic, platform string) string {
	switch strings.ToLower(platform) {
	case domain.PlatformDiscord:
		return topic.Discord.Description
	case domain.PlatformTwitch, platformStreamerbot:
		return topic.Streamerbot.Description
	default:
		return topic.Discord.Description
	}
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
