package info

import (
	"fmt"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

const (
	platformStreamerbot = "streamerbot"
)

type Formatter struct{}

func NewFormatter() *Formatter {
	return &Formatter{}
}

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

func (f *Formatter) FormatFeature(feature *Feature, platform string) string {
	return f.getContentByPlatform(feature.Discord.Description, feature.Streamerbot.Description, platform)
}

func (f *Formatter) FormatTopic(topic *Topic, platform string) string {
	return f.getContentByPlatform(topic.Discord.Description, topic.Streamerbot.Description, platform)
}

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
