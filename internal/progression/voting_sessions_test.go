package progression

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
			assert.NotNil(t, winner, "winner should not be nil")

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
	iterations := 100

	for i := 0; i < iterations; i++ {
		winner := findWinningOption(options)
		results[winner.NodeID]++
	}

	// Verify all options were selected at least once (probabilistic test)
	// With 100 iterations and 4 options, each should appear ~25 times
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

	tests := []struct {
		name          string
		count         int
		expectedCount int
	}{
		{
			name:          "select 4 from 5",
			count:         4,
			expectedCount: 4,
		},
		{
			name:          "select more than available",
			count:         10,
			expectedCount: 5,
		},
		{
			name:          "select exact count",
			count:         5,
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selected := selectRandomNodes(nodes, tt.count)
			assert.Equal(t, tt.expectedCount, len(selected), "unexpected selection count")

			// Verify no duplicates
			seen := make(map[int]bool)
			for _, node := range selected {
				assert.False(t, seen[node.ID], "duplicate node selected")
				seen[node.ID] = true
			}
		})
	}
}
