package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MapRandoPreset defines a preset mapping
type MapRandoPreset struct {
	File        string `yaml:"file"`
	Description string `yaml:"description"`
}

// MapRandoConfig configures presets
type MapRandoConfig struct {
	PresetsDir string                    `yaml:"presets_dir"`
	Presets    map[string]MapRandoPreset `yaml:"presets"`
}

// MapRandoClient interacts with the MapRandomizer API
type MapRandoClient struct {
	baseURL      string
	spoilerToken string
	httpClient   *http.Client
	presets      map[string]string // map of presetName -> json string
	descriptions map[string]string // map of presetName -> description
	presetNames  []string
}

// NewMapRandoClient creates a new MapRandomizer integration client
func NewMapRandoClient(baseURL, spoilerToken string) *MapRandoClient {
	// The http.Client follows redirects natively by default, which is what we want
	return &MapRandoClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		spoilerToken: spoilerToken,
		httpClient: &http.Client{
			// Setting a custom CheckRedirect ensures we follow redirects for POST
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		},
		presets:      make(map[string]string),
		descriptions: make(map[string]string),
		presetNames:  make([]string, 0),
	}
}

// LoadConfig loads the presets from yaml and reads the corresponding JSON files
func (c *MapRandoClient) LoadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// It's ok if it doesn't exist, we just won't have presets
			return nil
		}
		return fmt.Errorf("failed to read maprando config: %w", err)
	}

	var cfg MapRandoConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse maprando config: %w", err)
	}

	slog.Info("Loading MapRando presets", "dir", cfg.PresetsDir, "count", len(cfg.Presets))

	c.presetNames = make([]string, 0, len(cfg.Presets))

	for name, preset := range cfg.Presets {
		jsonPath := filepath.Join(cfg.PresetsDir, preset.File)
		jsonData, err := os.ReadFile(jsonPath)
		if err != nil {
			return fmt.Errorf("failed to read preset file %s for preset '%s': %w", jsonPath, name, err)
		}

		c.presets[name] = string(jsonData)
		c.descriptions[name] = preset.Description
		c.presetNames = append(c.presetNames, name)
		slog.Debug("Loaded MapRando preset", "name", name, "file", preset.File)
	}

	slog.Info("Successfully loaded MapRando presets", "count", len(c.presetNames))
	return nil
}

func (c *MapRandoClient) PresetNames() []string {
	return c.presetNames
}

func (c *MapRandoClient) PresetDescription(name string) string {
	return c.descriptions[name]
}

func (c *MapRandoClient) SeedURL(seedName string) string {
	return fmt.Sprintf("%s/seed/%s", c.baseURL, seedName)
}

// Randomize requests a new seed using the provided preset name. Returns the full seed URL.
func (c *MapRandoClient) Randomize(presetName string) (string, error) {
	settingsJSON, ok := c.presets[presetName]
	if !ok {
		return "", fmt.Errorf("unknown preset: %s", presetName)
	}

	// Prepare multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add settings field
	fw, err := w.CreateFormField("settings")
	if err != nil {
		return "", fmt.Errorf("failed to create settings field: %w", err)
	}
	_, err = io.Copy(fw, strings.NewReader(settingsJSON))
	if err != nil {
		return "", fmt.Errorf("failed to write settings json: %w", err)
	}

	// Add spoiler_token field
	fw, err = w.CreateFormField("spoiler_token")
	if err != nil {
		return "", fmt.Errorf("failed to create spoiler_token field: %w", err)
	}
	_, err = io.Copy(fw, strings.NewReader(c.spoilerToken))
	if err != nil {
		return "", fmt.Errorf("failed to write spoiler token: %w", err)
	}

	w.Close()

	// Make request
	req, err := http.NewRequest("POST", c.baseURL+"/randomize", &b)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("randomize failed with status: %s", resp.Status)
	}

	// Read {"seed_url": "/seed/{seed_name}"}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	seedURL, ok := result["seed_url"]
	if !ok {
		return "", fmt.Errorf("response missing seed_url")
	}

	// Clean trailing slash for consistency
	seedURL = strings.TrimRight(seedURL, "/")

	return c.baseURL + seedURL, nil
}

// Unlock unlocks a generated seed using the global spoiler token
func (c *MapRandoClient) Unlock(seedName string) error {
	data := url.Values{}
	data.Set("spoiler_token", c.spoilerToken)

	reqURL := fmt.Sprintf("%s/seed/%s/unlock", c.baseURL, seedName)
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create unlock request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("unlock request failed: %w", err)
	}
	defer resp.Body.Close()

	// The web server replies with 302 to the seed page on success, and Go's client follows it.
	// So returning 200 or 302 on success is valid.
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
		return nil
	}

	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("token mismatched - unlock forbidden")
	} else if resp.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("seed is already unlocked")
	}

	return fmt.Errorf("unlock failed with status: %s", resp.Status)
}
