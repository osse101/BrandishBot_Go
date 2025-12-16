package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/user"
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
		"known_platform":    domain.PlatformDiscord,
		"known_platform_id": discordID,
		"new_platform":      domain.PlatformDiscord,
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

// Search performs a search action
func (c *APIClient) Search(platform, platformID, username string) (string, error) {
	req := map[string]string{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/search", req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to read error message
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("API error: %s", errResp.Error)
		}
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var searchResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return searchResp.Message, nil
}

// GetInventory retrieves user inventory
func (c *APIClient) GetInventory(platform, platformID, username string) ([]user.UserInventoryItem, error) {
	params := url.Values{}
	params.Set("platform", platform)
	params.Set("username", username)
	params.Set("platform_id", platformID)

	path := fmt.Sprintf("/user/inventory?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var invResp struct {
		Items []user.UserInventoryItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&invResp); err != nil {
		return nil, fmt.Errorf("failed to decode inventory: %w", err)
	}

	return invResp.Items, nil
}

// UseItem uses an item from inventory
func (c *APIClient) UseItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":      platform,
		"platform_id":   platformID,
		"username":      username,
		"item_name":     itemName,
		"quantity":      quantity,
		"target_user":   "", // Optional, empty for non-targeted items
	}

	resp, err := c.doRequest(http.MethodPost, "/user/item/use", req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("API error: %s", errResp.Error)
		}
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var useResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&useResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return useResp.Message, nil
}

// StartGamble starts a new gamble
func (c *APIClient) StartGamble(platform, platformID, username string, wager int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"wager":       wager,
	}

	resp, err := c.doRequest(http.MethodPost, "/gamble/start", req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("API error: %s", errResp.Error)
		}
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var gambleResp struct {
		Message  string `json:"message"`
		GambleID string `json:"gamble_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gambleResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return gambleResp.Message, nil
}

// JoinGamble joins an active gamble
func (c *APIClient) JoinGamble(platform, platformID, username, gambleID string) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"gamble_id":   gambleID,
	}

	resp, err := c.doRequest(http.MethodPost, "/gamble/join", req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("API error: %s", errResp.Error)
		}
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var joinResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&joinResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return joinResp.Message, nil
}

// VoteForNode votes for a progression node unlock
func (c *APIClient) VoteForNode(platform, platformID, username, nodeKey string) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"node_key":    nodeKey,
	}

	resp, err := c.doRequest(http.MethodPost, "/progression/vote", req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("API error: %s", errResp.Error)
		}
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var voteResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&voteResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return voteResp.Message, nil
}

// AdminUnlockNode force-unlocks a progression node (admin only)
func (c *APIClient) AdminUnlockNode(nodeKey string, level int) (string, error) {
	req := map[string]interface{}{
		"node_key": nodeKey,
		"level":    level,
	}

	resp, err := c.doRequest(http.MethodPost, "/progression/admin/unlock", req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return "", fmt.Errorf("API error: %s", errResp.Error)
		}
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var unlockResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&unlockResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return unlockResp.Message, nil
}
