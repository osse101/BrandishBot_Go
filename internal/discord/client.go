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
func (c *APIClient) GetInventory(platform, platformID, username, filter string) ([]user.UserInventoryItem, error) {
	params := url.Values{}
	params.Set("platform", platform)
	params.Set("username", username)
	params.Set("platform_id", platformID)
	if filter != "" {
		params.Set("filter", filter)
	}

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
func (c *APIClient) StartGamble(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"bets":        []map[string]interface{}{{"item_name": itemName, "quantity": quantity}},
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
func (c *APIClient) JoinGamble(platform, platformID, username, gambleID, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"bets":        []map[string]interface{}{{"item_name": itemName, "quantity": quantity}},
	}

	// Note: gambleID goes in the URL query parameter
	path := fmt.Sprintf("/gamble/join?id=%s", gambleID)
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

// AdminRelockNode relocks a progression node (admin only)
func (c *APIClient) AdminRelockNode(nodeKey string, level int) (string, error) {
	req := map[string]interface{}{
		"node_key": nodeKey,
		"level":    level,
	}

	resp, err := c.doRequest(http.MethodPost, "/progression/admin/relock", req)
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

	var relockResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&relockResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return relockResp.Message, nil
}

// AdminInstantUnlock force-unlocks the current vote leader (admin only)
func (c *APIClient) AdminInstantUnlock() (string, error) {
	resp, err := c.doRequest(http.MethodPost, "/progression/admin/instant-unlock", nil)
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

	var instantResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&instantResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return instantResp.Message, nil
}

// AdminResetProgression resets the entire progression tree (admin only)
func (c *APIClient) AdminResetProgression(resetBy, reason string, preserveUser bool) (string, error) {
	req := map[string]interface{}{
		"reset_by":                  resetBy,
		"reason":                    reason,
		"preserve_user_progression": preserveUser,
	}

	resp, err := c.doRequest(http.MethodPost, "/progression/admin/reset", req)
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

// AdminStartVoting starts a new voting session (admin only)
func (c *APIClient) AdminStartVoting() (string, error) {
	resp, err := c.doRequest(http.MethodPost, "/progression/admin/start-voting", nil)
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

	var startResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&startResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return startResp.Message, nil
}

// AdminEndVoting forces the current voting session to end (admin only)
func (c *APIClient) AdminEndVoting() (string, error) {
	resp, err := c.doRequest(http.MethodPost, "/progression/admin/end-voting", nil)
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

	var endResp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&endResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return endResp.Message, nil
}

// GetProgressionTree retrieves the full progression tree
func (c *APIClient) GetProgressionTree() ([]*domain.ProgressionTreeNode, error) {
	resp, err := c.doRequest(http.MethodGet, "/progression/tree", nil)
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

// GetSellPrices retrieves current sell prices
func (c *APIClient) GetSellPrices() (string, error) {
	resp, err := c.doRequest(http.MethodGet, "/prices", nil)
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
		return "No sellable items available.", nil
	}

	var sb strings.Builder
	for _, item := range pricesResp.Items {
		fmt.Fprintf(&sb, "**%s**: %d coins\n", item.InternalName, item.BaseValue)
	}
	return sb.String(), nil
}

// GetBuyPrices retrieves current buy prices
func (c *APIClient) GetBuyPrices() (string, error) {
	resp, err := c.doRequest(http.MethodGet, "/prices/buy", nil)
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
		return "No buyable items available.", nil
	}

	var sb strings.Builder
	for _, item := range pricesResp.Items {
		fmt.Fprintf(&sb, "**%s**: %d coins\n", item.InternalName, item.BaseValue)
	}
	return sb.String(), nil
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
func (c *APIClient) UpgradeItem(platform, platformID, username, itemName string, quantity int) (string, error) {
	req := map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"username":    username,
		"item":        itemName,
		"quantity":    quantity,
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

// Recipe represents a recipe returned by the API
type Recipe struct {
	ItemName string `json:"item_name"`
	ItemID   int    `json:"item_id"`
}

// GetRecipes retrieves all crafting recipes
func (c *APIClient) GetRecipes() ([]Recipe, error) {
	resp, err := c.doRequest(http.MethodGet, "/recipes", nil)
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

	path := fmt.Sprintf("/recipes?%s", params.Encode())
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

// GetJobBonus retrieves the active bonus for a job
func (c *APIClient) GetJobBonus(userID, jobKey, bonusType string) (int, error) {
	params := url.Values{}
	params.Set("user_id", userID)
	params.Set("job_key", jobKey)
	params.Set("bonus_type", bonusType)

	path := fmt.Sprintf("/jobs/bonus?%s", params.Encode())
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return 0, fmt.Errorf("API error: %s", errResp.Error)
		}
		return 0, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	var bonusResp struct {
		BonusVal int `json:"bonus_val"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&bonusResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return bonusResp.BonusVal, nil
}

// AdminAddContribution adds contribution points (admin only)
func (c *APIClient) AdminAddContribution(amount int) (string, error) {
	req := map[string]interface{}{
		"amount": amount,
	}

	resp, err := c.doRequest(http.MethodPost, "/progression/admin/contribution", req)
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

	path := fmt.Sprintf("/user/timeout?%s", params.Encode())
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

// GetUnlockProgress returns current unlock progress
func (c *APIClient) GetUnlockProgress() (*map[string]interface{}, error) {
	resp, err := c.doRequest(http.MethodGet, "/progression/unlock-progress", nil)
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
func (c *APIClient) GetUserEngagement(userID string) (*domain.ContributionBreakdown, error) {
	params := url.Values{}
	params.Set("user_id", userID)

	path := fmt.Sprintf("/progression/engagement?%s", params.Encode())
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

	path := fmt.Sprintf("/progression/leaderboard?%s", params.Encode())
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
	resp, err := c.doRequest(http.MethodGet, "/progression/session", nil)
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

	resp, err := c.doRequest(http.MethodPost, "/message/handle", req)
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
