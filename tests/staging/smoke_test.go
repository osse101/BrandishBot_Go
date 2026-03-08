//go:build staging

package staging

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ProgressionTreeResponse struct {
	Nodes []struct {
		NodeKey    string `json:"node_key"`
		IsUnlocked bool   `json:"is_unlocked"`
	} `json:"nodes"`
}

func TestProgressionTree(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/api/v1/progression/tree", nil)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var tree ProgressionTreeResponse
	require.NoError(t, json.Unmarshal(body, &tree), "Failed to unmarshal response")

	require.NotEmpty(t, tree.Nodes, "Expected at least one node in progression tree")

	// Verify root node exists (progression_system)
	foundRoot := false
	for _, node := range tree.Nodes {
		if node.NodeKey == "progression_system" {
			foundRoot = true
			break
		}
	}

	assert.True(t, foundRoot, "Expected to find 'progression_system' node in tree")
}
