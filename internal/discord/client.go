package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
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
			jitter := time.Duration(time.Now().UnixNano()%100) * time.Millisecond
			delay := retryDelay*time.Duration(1<<uint(attempt-1)) + jitter
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

// doRequestAndParse performs a request and parses the JSON response into the target struct
func (c *APIClient) doRequestAndParse(method, path string, body interface{}, target interface{}) error {
	resp, err := c.doRequest(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return fmt.Errorf("API error: %s", errResp.Error)
		}
		return fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// doAction performs a request and expects a standard response with a "message" field
func (c *APIClient) doAction(method, path string, body interface{}) (string, error) {
	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequestAndParse(method, path, body, &resp); err != nil {
		return "", err
	}
	return resp.Message, nil
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

	resp, err := c.doRequest(http.MethodPost, "/api/v1/user/register", req)
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

	resp, err := c.doRequest(http.MethodPost, "/api/v1/user/search", req)
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
func (c *APIClient) GetInventory(platform, platformID, username, filter string) ([]user.InventoryItem, error) {
	return c.getInventoryInternal("/api/v1/user/inventory", platform, platformID, username, filter)
}

func (c *APIClient) getInventoryInternal(path, platform, platformID, username, filter string) ([]user.InventoryItem, error) {
	params := url.Values{}
	params.Set("platform", platform)
	params.Set("username", username)
	if platformID != "" {
		params.Set("platform_id", platformID)
	}
	if filter != "" {
		params.Set("filter", filter)
	}

	fullPath := fmt.Sprintf("%s?%s", path, params.Encode())
	resp, err := c.doRequest(http.MethodGet, fullPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var invResp struct {
		Items []user.InventoryItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&invResp); err != nil {
		return nil, fmt.Errorf("failed to decode inventory: %w", err)
	}

	return invResp.Items, nil
}

// UseItem uses an item from inventory
func (c *APIClient) UseItem(platform, platformID, username, itemName string, quantity int, target string) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"item_name":   itemName,
		"quantity":    quantity,
		"target_user": target, // Optional, can be username or job name
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/user/item/use", req)
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
func (c *APIClient) StartGamble(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"bets":        []map[string]interface{}{{"item_name": itemName, "quantity": quantity}},
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/gamble/start", req)
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

	return gambleResp.GambleID, nil
}

// JoinGamble joins an active gamble
func (c *APIClient) JoinGamble(platform, platformID, username, gambleID string) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
	}

	// Note: gambleID goes in the URL query parameter
	path := fmt.Sprintf("/api/v1/gamble/join?id=%s", gambleID)
	resp, err := c.doRequest(http.MethodPost, path, req)
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

	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/progression/vote", req, &resp); err != nil {
		return "", err
	}
	return resp.Message, nil
}

// AdminUnlockNode force-unlocks a progression node (admin only)
func (c *APIClient) AdminUnlockNode(nodeKey string, level int) (string, error) {
	req := map[string]interface{}{
		"node_key": nodeKey,
		"level":    level,
	}
	return c.doAction(http.MethodPost, "/api/v1/progression/admin/unlock", req)
}

// AdminUnlockAllNodes force-unlocks ALL progression nodes at max level (admin only, DEBUG)
func (c *APIClient) AdminUnlockAllNodes() (string, error) {
	return c.doAction(http.MethodPost, "/api/v1/progression/admin/unlock-all", nil)
}

// AdminRelockNode relocks a progression node (admin only)
func (c *APIClient) AdminRelockNode(nodeKey string, level int) (string, error) {
	req := map[string]interface{}{
		"node_key": nodeKey,
		"level":    level,
	}
	return c.doAction(http.MethodPost, "/api/v1/progression/admin/relock", req)
}

// AdminInstantUnlock force-unlocks the current vote leader (admin only)
func (c *APIClient) AdminInstantUnlock() (string, error) {
	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/progression/admin/instant-unlock", nil, &resp); err != nil {
		return "", err
	}
	return resp.Message, nil
}

// AdminResetProgression resets the entire progression tree (admin only)
func (c *APIClient) AdminResetProgression(resetBy, reason string, preserveUser bool) (string, error) {
	req := map[string]interface{}{
		"reset_by":                  resetBy,
		"reason":                    reason,
		"preserve_user_progression": preserveUser,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/progression/admin/reset", req)
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

	var resetResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&resetResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return resetResp.Message, nil
}

// AdminReloadWeights invalidates the engagement weight cache (admin only)
func (c *APIClient) AdminReloadWeights() (string, error) {
	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequestAndParse(http.MethodPost, "/api/admin/progression/reload-weights", nil, &resp); err != nil {
		return "", err
	}
	return resp.Message, nil
}

// AdminGetCacheStats retrieves user cache statistics (admin only)
func (c *APIClient) AdminGetCacheStats() (*user.CacheStats, error) {
	var stats user.CacheStats
	if err := c.doRequestAndParse(http.MethodGet, "/api/v1/admin/cache/stats", nil, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// AdminStartVoting starts a new voting session (admin only)
func (c *APIClient) AdminStartVoting() (string, error) {
	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/progression/admin/start-voting", nil, &resp); err != nil {
		return "", err
	}
	return resp.Message, nil
}

// AdminEndVoting forces the current voting session to end (admin only)
func (c *APIClient) AdminEndVoting() (string, error) {
	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/progression/admin/end-voting", nil, &resp); err != nil {
		return "", err
	}
	return resp.Message, nil
}

// GetProgressionTree retrieves the full progression tree
func (c *APIClient) GetProgressionTree() ([]*domain.ProgressionTreeNode, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/v1/progression/tree", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var treeResp struct {
		Nodes []*domain.ProgressionTreeNode `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return treeResp.Nodes, nil
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
	return c.doAction(http.MethodPost, "/api/v1/user/item/buy", req)
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
	return c.doAction(http.MethodPost, "/api/v1/user/item/sell", req)
}

// GetSellPrices retrieves current sell prices
func (c *APIClient) GetSellPrices() (string, error) {
	return c.getPricesInternal("/api/v1/prices")
}

func (c *APIClient) getPricesInternal(endpoint string) (string, error) {
	resp, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var pricesResp struct {
		Message string        `json:"message"`
		Items   []domain.Item `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pricesResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(pricesResp.Items) == 0 {
		return "No items available.", nil
	}

	var sb strings.Builder
	for _, item := range pricesResp.Items {
		fmt.Fprintf(&sb, "**%s**: %d coins\n", item.InternalName, item.BaseValue)
	}
	return sb.String(), nil
}

// GetBuyPrices retrieves current buy prices
func (c *APIClient) GetBuyPrices() (string, error) {
	return c.getPricesInternal("/api/v1/prices/buy")
}

// AddItemByUsername adds an item by username (no platformID required)
func (c *APIClient) AddItemByUsername(platform, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":  platform,
		"username":  username,
		"item_name": itemName,
		"quantity":  quantity,
	}

	var resp struct {
		Message string `json:"message"`
	}
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/user/item/add", req, &resp); err != nil {
		return "", err
	}
	return resp.Message, nil
}

// RemoveItemByUsername removes an item by username (no platformID required)
func (c *APIClient) RemoveItemByUsername(platform, username, itemName string, quantity int) (int, error) {
	req := map[string]interface{}{
		"platform":  platform,
		"username":  username,
		"item_name": itemName,
		"quantity":  quantity,
	}

	var resp struct {
		Removed int `json:"removed"`
	}
	if err := c.doRequestAndParse(http.MethodPost, "/api/v1/user/item/remove", req, &resp); err != nil {
		return 0, err
	}
	return resp.Removed, nil
}

// GiveItemByUsername transfers an item by usernames (no platformIDs required)
func (c *APIClient) GiveItemByUsername(fromPlatform, fromUsername, toPlatform, toUsername, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"from_platform": fromPlatform,
		"from_username": fromUsername,
		"to_platform":   toPlatform,
		"to_username":   toUsername,
		"item_name":     itemName,
		"quantity":      quantity,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/user/item/give", req)
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

	var result struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Message, nil
}

// GiveItem transfers an item between users user
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

	resp, err := c.doRequest(http.MethodPost, "/api/v1/user/item/give", req)
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
func (c *APIClient) UpgradeItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"item":        itemName,
		"quantity":    quantity,
	}
	return c.doAction(http.MethodPost, "/api/v1/user/item/upgrade", req)
}

// DisassembleItem breaks down an item for materials
func (c *APIClient) DisassembleItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"item":        itemName,
		"quantity":    quantity,
	}
	return c.doAction(http.MethodPost, "/api/v1/user/item/disassemble", req)
}

// Recipe represents a recipe returned by the API
type Recipe struct {
	ItemName string `json:"item_name"`
	ItemID   int    `json:"item_id"`
}

// GetRecipes retrieves all crafting recipes
func (c *APIClient) GetRecipes() ([]Recipe, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/v1/recipes", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var recipesResp struct {
		Recipes []Recipe `json:"recipes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&recipesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return recipesResp.Recipes, nil
}

// GetUnlockedRecipes retrieves unlocked recipes for a user
func (c *APIClient) GetUnlockedRecipes(platform, platformID, username string) ([]Recipe, error) {
	params := url.Values{}
	params.Set("platform", platform)
	params.Set("platform_id", platformID)
	params.Set("user", username)

	path := fmt.Sprintf("/api/v1/recipes?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var recipesResp struct {
		Recipes []Recipe `json:"recipes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&recipesResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return recipesResp.Recipes, nil
}

// AdminAddContribution adds contribution points (admin only)
func (c *APIClient) AdminAddContribution(amount int) (string, error) {
	req := map[string]interface{}{
		"amount": amount,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/progression/admin/contribution", req)
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

	var contribResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&contribResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return contribResp.Message, nil
}

// GetUserTimeout retrieves timeout status for a user
func (c *APIClient) GetUserTimeout(username string) (bool, float64, error) {
	params := url.Values{}
	params.Set("username", username)

	path := fmt.Sprintf("/api/v1/user/timeout?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return false, 0, fmt.Errorf("API error: %s", errResp.Error)
		}
		return false, 0, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var timeoutResp struct {
		IsTimedOut       bool    `json:"is_timed_out"`
		RemainingSeconds float64 `json:"remaining_seconds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&timeoutResp); err != nil {
		return false, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return timeoutResp.IsTimedOut, timeoutResp.RemainingSeconds, nil
}

// GetLeaderboard retrieves leaderboard rankings
func (c *APIClient) GetLeaderboard(metric string, limit int) (string, error) {
	params := url.Values{}
	params.Set("metric", metric)
	params.Set("limit", fmt.Sprintf("%d", limit))

	path := fmt.Sprintf("/api/v1/stats/leaderboard?%s", params.Encode())
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

	path := fmt.Sprintf("/api/v1/stats/user?%s", params.Encode())
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

// GetInventoryByUsername retrieves user inventory by username
func (c *APIClient) GetInventoryByUsername(platform, username, filter string) ([]user.InventoryItem, error) {
	return c.getInventoryInternal("/api/v1/user/inventory-by-username", platform, "", username, filter)
}

// XPAwardResult represents the result of awarding XP
type XPAwardResult struct {
	LeveledUp bool `json:"leveled_up"`
	NewLevel  int  `json:"new_level"`
	NewXP     int  `json:"new_xp"`
}

// AdminAwardXP awards job XP to a user via platform and username (admin only)
func (c *APIClient) AdminAwardXP(platform, username, jobKey string, amount int) (*XPAwardResult, error) {
	req := map[string]interface{}{
		"platform": platform,
		"username": username,
		"job_key":  jobKey,
		"amount":   amount,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/admin/job/award-xp", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var awardResp struct {
		Success bool `json:"success"`
		Result  struct {
			LeveledUp bool `json:"leveled_up"`
			NewLevel  int  `json:"new_level"`
			NewXP     int  `json:"new_xp"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&awardResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &XPAwardResult{
		LeveledUp: awardResp.Result.LeveledUp,
		NewLevel:  awardResp.Result.NewLevel,
		NewXP:     awardResp.Result.NewXP,
	}, nil
}

// GetUnlockProgress returns current unlock progress
func (c *APIClient) GetUnlockProgress() (*map[string]interface{}, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/v1/progression/unlock-progress", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var progress map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&progress); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &progress, nil
}

// GetUserEngagement returns user's engagement breakdown
func (c *APIClient) GetUserEngagement(platform, platformID string) (*domain.ContributionBreakdown, error) {
	params := url.Values{}
	params.Set("platform", platform)
	params.Set("platform_id", platformID)

	path := fmt.Sprintf("/api/v1/progression/engagement?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var breakdown domain.ContributionBreakdown
	if err := json.NewDecoder(resp.Body).Decode(&breakdown); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &breakdown, nil
}

// GetContributionLeaderboard returns top contributors
func (c *APIClient) GetContributionLeaderboard(limit int) (string, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))

	path := fmt.Sprintf("/api/v1/progression/leaderboard?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
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

	var entries []domain.ContributionLeaderboardEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(entries) == 0 {
		return "No contributions yet.", nil
	}

	var sb strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&sb, "**%d.** <@%s>: %d points\n", entry.Rank, entry.UserID, entry.Contribution)
	}
	return sb.String(), nil
}

// GetVotingSession returns current voting session
func (c *APIClient) GetVotingSession() (*domain.ProgressionVotingSession, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/v1/progression/session", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	// Handles "no active session" message wrapper if needed, but endpoint returns direct object or "session": null
	// Checking for the map wrapper first just in case
	var raw map[string]interface{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := json.NewDecoder(io.NopCloser(bytes.NewBuffer(bodyBytes))).Decode(&raw); err == nil {
		if _, ok := raw["message"]; ok {
			// e.g. "No active voting session"
			return nil, nil
		}
	}

	var session domain.ProgressionVotingSession
	if err := json.NewDecoder(io.NopCloser(bytes.NewBuffer(bodyBytes))).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}

	return &session, nil
}

// HandleMessage sends a chat message to the server for processing
func (c *APIClient) HandleMessage(platform, platformID, username, message string) (*domain.MessageResult, error) {
	req := map[string]string{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"message":     message,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/message/handle", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result domain.MessageResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetAllJobs retrieves all available jobs
func (c *APIClient) GetAllJobs() ([]domain.Job, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/v1/jobs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var jobsResp struct {
		Jobs []domain.Job `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jobsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return jobsResp.Jobs, nil
}

// GetUserJobs retrieves job progress for a user
func (c *APIClient) GetUserJobs(platform, platformID string) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("platform", platform)
	params.Set("platform_id", platformID)

	userID := fmt.Sprintf("%s:%s", platform, platformID)
	params.Set("user_id", userID)

	path := fmt.Sprintf("/api/v1/jobs/user?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// AwardJobXP awards XP (Standard/Bot method)
func (c *APIClient) AwardJobXP(userID, jobKey string, amount int, source string) (*domain.XPAwardResult, error) {
	req := map[string]interface{}{
		"user_id":   userID,
		"job_key":   jobKey,
		"xp_amount": amount,
		"source":    source,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/jobs/award-xp", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result domain.XPAwardResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetSystemStats retrieves system-wide statistics
func (c *APIClient) GetSystemStats() (string, error) {
	resp, err := c.doRequest(http.MethodGet, "/api/v1/stats/system", nil)
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

// RecordEvent records a generic user event
func (c *APIClient) RecordEvent(platform, platformID, eventType string, metadata map[string]interface{}) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"event_type":  eventType,
		"metadata":    metadata,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/stats/event", req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Message, nil
}

// ReloadAliases reloads item aliases (admin only)
func (c *APIClient) ReloadAliases() error {
	resp, err := c.doRequest(http.MethodPost, "/api/v1/admin/reload-aliases", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status: %d", resp.StatusCode)
	}
	return nil
}

// Test endpoint
func (c *APIClient) Test(platform, platformID, username string) (string, error) {
	req := map[string]string{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/test", req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var result struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Message, nil
}

// Harvest collects accumulated rewards for a user
func (c *APIClient) Harvest(platform, platformID, username string) (*domain.HarvestResponse, error) {
	req := map[string]string{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
	}

	resp, err := c.doRequest(http.MethodPost, "/api/v1/harvest", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to read error message
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var harvestResp domain.HarvestResponse
	if err := json.NewDecoder(resp.Body).Decode(&harvestResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &harvestResp, nil
}

// AdminClearTimeout clears a user's timeout (admin only)
func (c *APIClient) AdminClearTimeout(platform, username string) (string, error) {
	req := map[string]string{
		"platform": platform,
		"username": username,
	}
	return c.doAction(http.MethodPost, "/api/v1/admin/timeout/clear", req)
}

// SetUserTimeout applies or extends a timeout for a user
func (c *APIClient) SetUserTimeout(platform, username string, durationSeconds int, reason string) (string, error) {
	req := map[string]interface{}{
		"platform":         platform,
		"username":         username,
		"duration_seconds": durationSeconds,
		"reason":           reason,
	}
	return c.doAction(http.MethodPut, "/api/v1/user/timeout", req)
}
