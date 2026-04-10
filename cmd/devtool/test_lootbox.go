package main

import (
	"fmt"
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
	if err := postAPIJSON("/message/handle", msgPayload, nil); err != nil {
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
	if err := postAPIJSON("/user/item/add", addPayload, nil); err != nil {
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
	respStr, err := postAPIJSONStr("/user/item/use", usePayload)
	if err != nil {
		PrintError("Error using item: %v", err)
		return fmt.Errorf("failed to use lootbox1: %w", err)
	}
	PrintInfo("Response: %s", respStr)

	// 4. Check Inventory
	PrintInfo("Checking inventory...")
	var inv interface{}
	err = getAPIJSON(fmt.Sprintf("/user/inventory?username=%s", username), &inv)
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
