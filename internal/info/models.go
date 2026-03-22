package info

type PlatformContent struct {
	Description string `yaml:"description" json:"description"`
}

type Topic struct {
	Command string `yaml:"command,omitempty" json:"command,omitempty"`

	Discord     PlatformContent `yaml:"discord" json:"discord"`
	Streamerbot PlatformContent `yaml:"streamerbot" json:"streamerbot"`
}

type Feature struct {
	Name  string `yaml:"name" json:"name"`
	Title string `yaml:"title" json:"title"`
	Icon  string `yaml:"icon,omitempty" json:"icon,omitempty"`
	Color string `yaml:"color,omitempty" json:"color,omitempty"`

	Discord     PlatformContent `yaml:"discord" json:"discord"`
	Streamerbot PlatformContent `yaml:"streamerbot" json:"streamerbot"`

	Topics map[string]Topic `yaml:"topics,omitempty" json:"topics,omitempty"`
}
