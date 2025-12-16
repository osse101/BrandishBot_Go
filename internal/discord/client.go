package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
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

// doRequest performs an HTTP request with retry logic
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
	
	// Retry configuration
	maxRetries := 3
	retryDelay := 500 * time.Millisecond
	
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			jitter := time.Duration(time.Now().UnixNano() % 100) * time.Millisecond
			delay := retryDelay * time.Duration(1<<uint(attempt-1)) + jitter
			time.Sleep(delay)
			slog.Info("Retrying API request", "attempt", attempt, "path", path, "delay", delay)
		}
		
		req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if c.APIKey != "" {
			req.Header.Set("X-API-Key", c.APIKey)
		}

		resp, err := c.Client.Do(req)
		if err != nil {
			lastErr = err
			slog.Warn("API request failed", "error", err, "attempt", attempt)
			continue
		}
		
		// Success or non-retryable error
		if resp.StatusCode < 500 {
			return resp, nil
		}
		
		// Server error - retry
		resp.Body.Close()
		lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
		slog.Warn("Server error, will retry", "status", resp.StatusCode, "attempt", attempt)
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
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

// BuyItem purchases an item from the shop
func (c *APIClient) BuyItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"item_name":   itemName,
		"quantity":    quantity,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/item/buy", req)
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

	var buyResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&buyResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return buyResp.Message, nil
}

// SellItem sells an item from inventory
func (c *APIClient) SellItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"item_name":   itemName,
		"quantity":    quantity,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/item/sell", req)
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

	var sellResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sellResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return sellResp.Message, nil
}

// GetPrices retrieves current market prices
func (c *APIClient) GetPrices() (string, error) {
	resp, err := c.doRequest(http.MethodGet, "/prices", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var pricesResp struct {
		Message string `json:"message"`
		Prices  string `json:"prices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pricesResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if pricesResp.Prices != "" {
		return pricesResp.Prices, nil
	}
	return pricesResp.Message, nil
}

// GiveItem transfers an item to another user
func (c *APIClient) GiveItem(fromPlatform, fromPlatformID, toPlatform, toPlatformID, toUsername, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"from_platform":    fromPlatform,
		"from_platform_id": fromPlatformID,
		"to_platform":      toPlatform,
		"to_platform_id":   toPlatformID,
		"to_username":      toUsername,
		"item_name":        itemName,
		"quantity":         quantity,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/item/give", req)
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

	var giveResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&giveResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return giveResp.Message, nil
}

// UpgradeItem crafts an item upgrade
func (c *APIClient) UpgradeItem(platform, platformID, username string, recipeID int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"recipe_id":   recipeID,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/item/upgrade", req)
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

	var upgradeResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&upgradeResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return upgradeResp.Message, nil
}

// DisassembleItem breaks down an item for materials
func (c *APIClient) DisassembleItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"item_name":   itemName,
		"quantity":    quantity,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/item/disassemble", req)
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

	var disassembleResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&disassembleResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return disassembleResp.Message, nil
}

// GetRecipes retrieves all crafting recipes
func (c *APIClient) GetRecipes() (string, error) {
	resp, err := c.doRequest(http.MethodGet, "/recipes", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var recipesResp struct {
		Message string `json:"message"`
		Recipes string `json:"recipes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&recipesResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if recipesResp.Recipes != "" {
		return recipesResp.Recipes, nil
	}
	return recipesResp.Message, nil
}

// GetLeaderboard retrieves leaderboard rankings
func (c *APIClient) GetLeaderboard(metric string, limit int) (string, error) {
	params := url.Values{}
	params.Set("metric", metric)
	params.Set("limit", fmt.Sprintf("%d", limit))

	path := fmt.Sprintf("/stats/leaderboard?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var leaderboardResp struct {
		Message string `json:"message"`
		Data    string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&leaderboardResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if leaderboardResp.Data != "" {
		return leaderboardResp.Data, nil
	}
	return leaderboardResp.Message, nil
}

// GetUserStats retrieves stats for a specific user
func (c *APIClient) GetUserStats(platform, platformID string) (string, error) {
	params := url.Values{}
	params.Set("platform", platform)
	params.Set("platform_id", platformID)

	path := fmt.Sprintf("/stats/user?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var statsResp struct {
		Message string `json:"message"`
		Stats   string `json:"stats"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statsResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if statsResp.Stats != "" {
		return statsResp.Stats, nil
	}
	return statsResp.Message, nil
}

// AddItem adds items to a user's inventory (admin only)
func (c *APIClient) AddItem(platform, platformID, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"item_name":   itemName,
		"quantity":    quantity,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/item/add", req)
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

	var addResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return addResp.Message, nil
}

// RemoveItem removes items from a user's inventory (admin only)
func (c *APIClient) RemoveItem(platform, platformID, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"item_name":   itemName,
		"quantity":    quantity,
	}

	resp, err := c.doRequest(http.MethodPost, "/user/item/remove", req)
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

	var removeResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&removeResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return removeResp.Message, nil
}
