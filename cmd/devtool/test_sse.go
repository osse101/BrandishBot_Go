package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type TestSSECommand struct{}

func (c *TestSSECommand) Name() string {
	return "test-sse"
}

func (c *TestSSECommand) Description() string {
	return "Trigger a test SSE event for the Discord bot"
}

func (c *TestSSECommand) Run(args []string) error {
	PrintHeader("Testing SSE Events...")

	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}
	apiKey := os.Getenv("API_KEY")

	eventType := "job.level_up"
	if len(args) > 0 {
		eventType = args[0]
	}

	payload := map[string]interface{}{
		"user_id":   "test_user",
		"job_key":   "warrior",
		"old_level": 5,
		"new_level": 6,
		"source":    "devtool",
		"is_test":   true,
	}

	// For specific event types, we can provide better mock data
	switch eventType {
	case "progression.voting_started":
		payload = map[string]interface{}{
			"session_id": 123,
			"options": []map[string]interface{}{
				{"node_key": "feature_test", "display_name": "Test Feature"},
				{"node_key": "feature_awesome", "display_name": "Awesome Feature"},
			},
		}
	case "progression.cycle_completed":
		payload = map[string]interface{}{
			"unlocked_node": map[string]interface{}{
				"node_key":     "feature_test",
				"display_name": "Test Feature",
			},
		}
	case "gamble.completed":
		payload = map[string]interface{}{
			"gamble_id":         "test-gamble",
			"winner_id":         "winner_user",
			"total_value":       1000,
			"participant_count": 5,
		}
	}

	data, err := json.Marshal(map[string]interface{}{
		"type":    eventType,
		"payload": payload,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/admin/sse/broadcast", apiURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	PrintSuccess("Successfully broadcasted test event: %s", eventType)
	PrintInfo("Check the Discord notification channel to see if the message arrived.")
	return nil
}
