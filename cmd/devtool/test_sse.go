package main

import (
	"fmt"
	"net/http"
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

	body := map[string]interface{}{
		"type":    eventType,
		"payload": payload,
	}

	path := "/api/v1/admin/sse/broadcast"
	resp, err := makeAPIRequest("POST", path, body)
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
