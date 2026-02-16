package progression

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestFindWinningOption_WithVotes(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-5 * time.Minute)

	tests := []struct {
		name     string
		options  []domain.ProgressionVotingOption
		expected int // expected winner's node ID
	}{
		{
			name: "clear winner by vote count",
			options: []domain.ProgressionVotingOption{
				{ID: 1, NodeID: 101, VoteCount: 5},
				{ID: 2, NodeID: 102, VoteCount: 10},
				{ID: 3, NodeID: 103, VoteCount: 3},
			},
			expected: 102,
		},
		{
			name: "tie broken by LastHighestVoteAt",
			options: []domain.ProgressionVotingOption{
				{ID: 1, NodeID: 101, VoteCount: 5, LastHighestVoteAt: &now},
				{ID: 2, NodeID: 102, VoteCount: 5, LastHighestVoteAt: &earlier}, // wins (earlier)
				{ID: 3, NodeID: 103, VoteCount: 3},
			},
			expected: 102,
		},
		{
			name: "tie broken by LastHighestVoteAt (nil vs non-nil)",
			options: []domain.ProgressionVotingOption{
				{ID: 1, NodeID: 101, VoteCount: 5, LastHighestVoteAt: nil},
				{ID: 2, NodeID: 102, VoteCount: 5, LastHighestVoteAt: &now}, // wins (non-nil)
			},
			expected: 102,
		},
		{
			name: "tie broken by LastHighestVoteAt (non-nil vs nil)",
			options: []domain.ProgressionVotingOption{
				{ID: 1, NodeID: 101, VoteCount: 5, LastHighestVoteAt: &now}, // wins (non-nil)
				{ID: 2, NodeID: 102, VoteCount: 5, LastHighestVoteAt: nil},
			},
			expected: 101,
		},
		{
			name: "tie with both LastHighestVoteAt nil",
			options: []domain.ProgressionVotingOption{
				{ID: 1, NodeID: 101, VoteCount: 5, LastHighestVoteAt: nil},
				{ID: 2, NodeID: 102, VoteCount: 5, LastHighestVoteAt: nil},
			},
			expected: 101, // First one wins (implementation detail)
		},
		{
			name: "all zero votes - random selection",
			options: []domain.ProgressionVotingOption{
				{ID: 1, NodeID: 101, VoteCount: 0},
				{ID: 2, NodeID: 102, VoteCount: 0},
				{ID: 3, NodeID: 103, VoteCount: 0},
				{ID: 4, NodeID: 104, VoteCount: 0},
			},
			expected: -1, // any is valid
		},
		{
			name: "single option",
			options: []domain.ProgressionVotingOption{
				{ID: 1, NodeID: 101, VoteCount: 0},
			},
			expected: 101,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winner := findWinningOption(tt.options)
			require.NotNil(t, winner, "winner should not be nil")

			if tt.expected == -1 {
				// For random selection, just verify it's one of the options
				found := false
				for _, opt := range tt.options {
					if winner.NodeID == opt.NodeID {
						found = true
						break
					}
				}
				assert.True(t, found, "winner should be one of the options")
			} else {
				assert.Equal(t, tt.expected, winner.NodeID, "unexpected winner")
			}
		})
	}
}

func TestFindWinningOption_EmptyOptions(t *testing.T) {
	winner := findWinningOption([]domain.ProgressionVotingOption{})
	assert.Nil(t, winner, "winner should be nil for empty options")
}

func TestFindWinningOption_ZeroVotesRandomness(t *testing.T) {
	// Test that 0-vote selection is actually random by running multiple times
	options := []domain.ProgressionVotingOption{
		{ID: 1, NodeID: 101, VoteCount: 0},
		{ID: 2, NodeID: 102, VoteCount: 0},
		{ID: 3, NodeID: 103, VoteCount: 0},
		{ID: 4, NodeID: 104, VoteCount: 0},
	}

	results := make(map[int]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		winner := findWinningOption(options)
		require.NotNil(t, winner, "winner should not be nil")
		results[winner.NodeID]++
	}

	// Verify all options were selected at least once (probabilistic test)
	// With 1000 iterations and 4 options, each should appear ~250 times.
	// We'll just check that at least 2 different options were selected
	assert.GreaterOrEqual(t, len(results), 2, "random selection should pick different options")
}

func TestSelectRandomNodes(t *testing.T) {
	nodes := []*domain.ProgressionNode{
		{ID: 1, NodeKey: "node1"},
		{ID: 2, NodeKey: "node2"},
		{ID: 3, NodeKey: "node3"},
		{ID: 4, NodeKey: "node4"},
		{ID: 5, NodeKey: "node5"},
	}

	t.Run("select 4 from 5", func(t *testing.T) {
		selected := selectRandomNodes(nodes, 4)
		assert.Equal(t, 4, len(selected), "unexpected selection count")

		// Verify no duplicates
		seen := make(map[int]bool)
		for _, node := range selected {
			assert.False(t, seen[node.ID], "duplicate node selected")
			seen[node.ID] = true
		}
	})

	t.Run("select more than available", func(t *testing.T) {
		selected := selectRandomNodes(nodes, 10)
		assert.Equal(t, 5, len(selected), "unexpected selection count")
		assert.Equal(t, nodes, selected, "should return original slice if count >= len")
		if len(nodes) > 0 {
			assert.Same(t, &nodes[0], &selected[0], "should return original underlying array")
		}
	})

	t.Run("select exact count", func(t *testing.T) {
		selected := selectRandomNodes(nodes, 5)
		assert.Equal(t, 5, len(selected), "unexpected selection count")
		assert.Equal(t, nodes, selected, "should return original slice if count == len")
		if len(nodes) > 0 {
			assert.Same(t, &nodes[0], &selected[0], "should return original underlying array")
		}
	})

	t.Run("select 0", func(t *testing.T) {
		selected := selectRandomNodes(nodes, 0)
		assert.Empty(t, selected, "should return empty slice")
		assert.NotNil(t, selected, "should return empty slice, not nil")
	})

	t.Run("nil input", func(t *testing.T) {
		selected := selectRandomNodes(nil, 5)
		assert.Empty(t, selected, "should return empty/nil slice")
	})

	t.Run("preserves original slice order", func(t *testing.T) {
		originalOrder := make([]int, len(nodes))
		for i, n := range nodes {
			originalOrder[i] = n.ID
		}

		_ = selectRandomNodes(nodes, 3)

		for i, n := range nodes {
			assert.Equal(t, originalOrder[i], n.ID, "original slice should not be modified")
		}
	})

	t.Run("returns copy when count < len", func(t *testing.T) {
		selected := selectRandomNodes(nodes, 3)
		if len(nodes) > 0 && len(selected) > 0 {
			assert.NotSame(t, &nodes[0], &selected[0], "should return a new underlying array")
		}

		// Verify modifying result slice doesn't modify original slice
		originalFirstID := nodes[0].ID
		selected[0] = &domain.ProgressionNode{ID: 999}
		assert.Equal(t, originalFirstID, nodes[0].ID, "modifying result slice should not affect original slice")
	})
}
