package info

// PlatformContent represents content for a specific platform
type PlatformContent struct {
	Description string `yaml:"description" json:"description"`
}

// Topic represents a sub-topic within a feature
type Topic struct {
	// Optional command syntax associated with this topic
	Command string `yaml:"command,omitempty" json:"command,omitempty"`

	// Platform-specific content
	Discord     PlatformContent `yaml:"discord" json:"discord"`
	Streamerbot PlatformContent `yaml:"streamerbot" json:"streamerbot"`
}

// Feature represents a complete feature with platform-specific content
type Feature struct {
	// Metadata
	Name  string `yaml:"name" json:"name"`
	Title string `yaml:"title" json:"title"`
	Icon  string `yaml:"icon,omitempty" json:"icon,omitempty"`
	Color string `yaml:"color,omitempty" json:"color,omitempty"`

	// Platform-specific feature-level descriptions
	Discord     PlatformContent `yaml:"discord" json:"discord"`
	Streamerbot PlatformContent `yaml:"streamerbot" json:"streamerbot"`

	// Sub-topics (e.g., farming -> harvest, farmer, compost)
	Topics map[string]Topic `yaml:"topics,omitempty" json:"topics,omitempty"`
}
