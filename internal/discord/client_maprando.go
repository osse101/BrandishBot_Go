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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

// MapRandoPreset defines a preset mapping
type MapRandoPreset struct {
	File        string `yaml:"file"`
	Description string `yaml:"description"`
	DevOnly     bool   `yaml:"devonly"`
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
	devOnly      map[string]bool   // map of presetName -> is dev only
	presetNames  []string

	sem         chan struct{} // limits concurrent randomize requests
	queueCount  atomic.Int32  // tracks active/queued processes for feedback
	lastRequest sync.Map      // tracks user rate limits (userID -> time.Time)
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
		devOnly:      make(map[string]bool),
		presetNames:  make([]string, 0),
		sem:          make(chan struct{}, 2), // allow 2 concurrent MapRando generations
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
		c.devOnly[name] = preset.DevOnly
		c.presetNames = append(c.presetNames, name)
		slog.Debug("Loaded MapRando preset", "name", name, "file", preset.File, "devonly", preset.DevOnly)
	}

	slog.Info("Successfully loaded MapRando presets", "count", len(c.presetNames))
	return nil
}

func (c *MapRandoClient) PresetNames() []string {
	return c.presetNames
}

// CheckCooldown checks if a user is within the 30-second rate limit.
// Returns the remaining duration and true if they are on cooldown.
// Otherwise, records the current time and returns 0, false.
func (c *MapRandoClient) CheckCooldown(userID string) (time.Duration, bool) {
	now := time.Now()
	const cooldown = 30 * time.Second

	if val, ok := c.lastRequest.Load(userID); ok {
		lastTime := val.(time.Time)
		if elapsed := now.Sub(lastTime); elapsed < cooldown {
			return cooldown - elapsed, true
		}
	}

	c.lastRequest.Store(userID, now)
	return 0, false
}

// ClearCooldown removes a user's cooldown. Useful if their request failed and they should be allowed to retry immediately.
func (c *MapRandoClient) ClearCooldown(userID string) {
	c.lastRequest.Delete(userID)
}

func (c *MapRandoClient) PresetDescription(name string) string {
	return c.descriptions[name]
}

func (c *MapRandoClient) getBaseURL(name string) string {
	if c.devOnly[name] {
		// If baseURL is maprando.com (with or without http/https), switch to dev.maprando.com
		if strings.HasSuffix(c.baseURL, "maprando.com") && !strings.Contains(c.baseURL, "dev.maprando.com") {
			return strings.Replace(c.baseURL, "maprando.com", "dev.maprando.com", 1)
		}
	}
	return c.baseURL
}

func (c *MapRandoClient) SeedURL(seedName string, presetName string) string {
	return fmt.Sprintf("%s/seed/%s", c.getBaseURL(presetName), seedName)
}

// Randomize requests a new seed using the provided preset name. Returns the full seed URL.
// It uses a semaphore to limit concurrent backend requests.
func (c *MapRandoClient) Randomize(presetName string, onQueued func(position int)) (string, error) {
	settingsJSON, ok := c.presets[presetName]
	if !ok {
		return "", fmt.Errorf("unknown preset: %s", presetName)
	}
	baseURL := c.getBaseURL(presetName)
	return c.RandomizeCustom(settingsJSON, baseURL, onQueued)
}

// RandomizeWithOverrides merges a base preset or JSON file with dynamic overrides, then randomizes.
func (c *MapRandoClient) RandomizeWithOverrides(presetName string, presetFileURL string, overrides map[string]string, onQueued func(position int)) (string, error) {
	var settingsJSON string

	// Base URL logic - default to main server if no predefined preset is selected
	baseURL := c.baseURL
	if presetName != "" {
		settingsJSON = c.presets[presetName]
		if settingsJSON == "" {
			return "", fmt.Errorf("unknown preset: %s", presetName)
		}
		baseURL = c.getBaseURL(presetName)
	}

	// If file is provided, download it and strictly use it as the base
	if presetFileURL != "" {
		resp, err := c.httpClient.Get(presetFileURL)
		if err != nil {
			return "", fmt.Errorf("failed to download preset_file: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download preset_file: status %d", resp.StatusCode)
		}

		// Read file up to 512KB (MapRando presets can be over 140KB)
		body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		if err != nil {
			return "", fmt.Errorf("failed to read preset_file body: %w", err)
		}
		settingsJSON = string(body)
	}

	// Apply overrides if any
	if len(overrides) > 0 {
		var data map[string]any
		if err := json.Unmarshal([]byte(settingsJSON), &data); err != nil {
			return "", fmt.Errorf("failed to parse base preset JSON: %w", err)
		}

		// We must overwrite preset name with "Custom" to preserve our overrides.
		data["name"] = "Custom"

		for key, valStr := range overrides {
			if err := c.applyDotNotationOverride(data, key, valStr); err != nil {
				return "", fmt.Errorf("failed to apply override '%s': %w", key, err)
			}
		}

		modifiedJSON, err := json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("failed to encode modified JSON: %w", err)
		}
		settingsJSON = string(modifiedJSON)
	}

	return c.RandomizeCustom(settingsJSON, baseURL, onQueued)
}

func (c *MapRandoClient) applyDotNotationOverride(data map[string]any, dotKey string, valStr string) error {
	keys := strings.Split(dotKey, ".")
	var current any = data

	// Parse value cleanly
	var parsedVal any = valStr
	if valStr == "null" {
		parsedVal = nil
	} else if b, err := strconv.ParseBool(valStr); err == nil {
		parsedVal = b
	} else if i, err := strconv.ParseInt(valStr, 10, 64); err == nil {
		parsedVal = i
	} else if f, err := strconv.ParseFloat(valStr, 64); err == nil {
		parsedVal = f
	}

	for i, part := range keys {
		isLeaf := (i == len(keys)-1)

		if m, ok := current.(map[string]any); ok {
			if isLeaf {
				m[part] = parsedVal
				return nil
			}
			next, exists := m[part]
			if !exists {
				m[part] = make(map[string]any)
				next = m[part]
			}
			current = next
		} else if a, ok := current.([]any); ok {
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(a) {
				return fmt.Errorf("invalid array index '%s' at '%s'", part, strings.Join(keys[:i], "."))
			}
			if isLeaf {
				a[idx] = parsedVal
				return nil
			}
			current = a[idx]
		} else {
			return fmt.Errorf("cannot traverse key '%s' on non-object at '%s'", part, strings.Join(keys[:i], "."))
		}
	}
	return nil
}

// RandomizeCustom requests a new seed using a custom JSON payload and a specific baseURL.
func (c *MapRandoClient) RandomizeCustom(settingsJSON string, baseURL string, onQueued func(position int)) (string, error) {
	qSize := c.queueCount.Add(1)
	defer c.queueCount.Add(-1)

	// Try to acquire semaphore without blocking first
	select {
	case c.sem <- struct{}{}:
		// acquired immediately
	default:
		// Queue is full, we are waiting
		if onQueued != nil {
			pos := int(qSize) - cap(c.sem)
			if pos < 1 {
				pos = 1
			}
			onQueued(pos)
		}
		c.sem <- struct{}{} // wait for real
	}
	defer func() { <-c.sem }()

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
	req, err := http.NewRequest("POST", baseURL+"/randomize", &b)
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
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("randomize failed with status %d: %s", resp.StatusCode, string(respBody))
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

	return baseURL + seedURL, nil
}

// Unlock unlocks a generated seed using the global spoiler token
func (c *MapRandoClient) Unlock(seedName string, presetName string) error {
	data := url.Values{}
	data.Set("spoiler_token", c.spoilerToken)

	var baseURL string
	if presetName != "" {
		baseURL = c.getBaseURL(presetName)
	} else {
		baseURL = c.baseURL
	}

	return c.unlockAt(baseURL, seedName)
}

func (c *MapRandoClient) unlockAt(baseURL string, seedName string) error {
	data := url.Values{}
	data.Set("spoiler_token", c.spoilerToken)

	reqURL := fmt.Sprintf("%s/seed/%s/unlock", baseURL, seedName)
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
