package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// APIClient handles communication with the BrandishBot Core API
type APIClient struct {
	BaseURL string
	Client  *http.Client
	APIKey  string
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, apiKey string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		APIKey: apiKey,
	}
}

// doRequest performs an HTTP request
func (c *APIClient) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	return c.Client.Do(req)
}

// RegisterUser registers or retrieves a user
func (c *APIClient) RegisterUser(username, discordID string) (*domain.User, error) {
	req := map[string]string{
		"username":          username,
		"known_platform":    "discord",
		"known_platform_id": discordID,
		"new_platform":      "discord",
		"new_platform_id":   discordID,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/register", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var user domain.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	return &user, nil
}

// GetUserStats retrieves stats for a user
func (c *APIClient) GetUserStats(userID string) (*domain.StatsSummary, error) {
	params := url.Values{}
	params.Set("user_id", userID)
	params.Set("period", "all_time")

	path := fmt.Sprintf("/stats/user?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var stats domain.StatsSummary
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return &stats, nil
}
