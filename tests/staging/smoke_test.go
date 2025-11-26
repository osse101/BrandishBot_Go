//go:build staging

package staging

import (
	"encoding/json"
	"net/http"
	"testing"
)

type ProgressionTreeResponse struct {
	Nodes []struct {
		NodeKey    string `json:"node_key"`
		IsUnlocked bool   `json:"is_unlocked"`
	} `json:"nodes"`
}

func TestProgressionTree(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/progression/tree", nil)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var tree ProgressionTreeResponse
	if err := json.Unmarshal(body, &tree); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(tree.Nodes) == 0 {
		t.Error("Expected at least one node in progression tree")
	}

	// Verify root node exists (progression_system)
	foundRoot := false
	for _, node := range tree.Nodes {
		if node.NodeKey == "progression_system" {
			foundRoot = true
			break
		}
	}

	if !foundRoot {
		t.Error("Expected to find 'progression_system' node in tree")
	}
}
