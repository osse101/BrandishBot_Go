package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type TestLootboxCommand struct{}

func (c *TestLootboxCommand) Name() string {
	return "test-lootbox"
}

func (c *TestLootboxCommand) Description() string {
	return "Test lootbox1 injection and usage"
}

func (c *TestLootboxCommand) Run(args []string) error {
	PrintHeader("Testing Lootbox1 Repro")

	baseURL := os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	username := "debug_user"
	platform := "twitch"
	platformID := "debug_123"

	// 1. Register User (via HandleMessage)
	PrintInfo("Registering user...")
	msgPayload := map[string]interface{}{
		"username":    username,
		"platform":    platform,
		"platform_id": platformID,
	}
	if err := c.postJSON(baseURL+"/message/handle", msgPayload); err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	// 2. Give Lootbox1
	PrintInfo("Giving lootbox1...")
	addPayload := map[string]interface{}{
		"username": username,
		"platform": platform,
		"itemName": "lootbox1",
		"quantity": 1,
	}
	if err := c.postJSON(baseURL+"/user/item/add", addPayload); err != nil {
		return fmt.Errorf("failed to give lootbox1: %w", err)
	}

	// 3. Use Lootbox1
	PrintInfo("Using lootbox1...")
	usePayload := map[string]interface{}{
		"username": username,
		"platform": platform,
		"itemName": "lootbox1",
		"quantity": 1,
	}
	respStr, err := c.postJSONStr(baseURL+"/user/item/use", usePayload)
	if err != nil {
		PrintError("Error using item: %v", err)
		return fmt.Errorf("failed to use lootbox1: %w", err)
	}
	PrintInfo("Response: %s", respStr)

	// 4. Check Inventory
	PrintInfo("Checking inventory...")
	inv, err := c.getJSON(fmt.Sprintf("%s/user/inventory?username=%s", baseURL, username))
	if err != nil {
		return fmt.Errorf("failed to check inventory: %w", err)
	}

	foundLootbox0 := false
	if invList, ok := inv.([]interface{}); ok {
		for _, itemIf := range invList {
			if item, ok := itemIf.(map[string]interface{}); ok {
				if name, ok := item["name"].(string); ok && name == "lootbox0" {
					foundLootbox0 = true
					break
				}
			}
		}
	}

	if foundLootbox0 {
		PrintSuccess("SUCCESS: Found lootbox0 in inventory.")
	} else {
		PrintError("FAILURE: lootbox0 not found.")
		return fmt.Errorf("lootbox0 not found in inventory")
	}

	return nil
}

func (c *TestLootboxCommand) postJSON(url string, payload interface{}) error {
	_, err := c.postJSONStr(url, payload)
	return err
}

func (c *TestLootboxCommand) postJSONStr(url string, payload interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

func (c *TestLootboxCommand) getJSON(url string) (interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
