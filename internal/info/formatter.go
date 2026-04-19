package info

import (
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
