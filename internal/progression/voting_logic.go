package progression

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// RandomIntFunc represents a function that returns a random integer in [0, max).
type RandomIntFunc func(max int) int

// selectRandomNodes selects a random subset of nodes.
// If rng is nil, it uses utils.SecureRandomInt.
func selectRandomNodes(nodes []*domain.ProgressionNode, count int, rng RandomIntFunc) []*domain.ProgressionNode {
	if rng == nil {
		rng = utils.SecureRandomInt
	}

	if len(nodes) <= count {
		return nodes
	}

	// Fisher-Yates shuffle
	shuffled := make([]*domain.ProgressionNode, len(nodes))
	copy(shuffled, nodes)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:count]
}

// findWinningOption determines the winning option based on votes and tie-breaking rules.
// If rng is nil, it uses utils.SecureRandomInt for random tie-breaking.
func findWinningOption(options []domain.ProgressionVotingOption, rng RandomIntFunc) *domain.ProgressionVotingOption {
	if rng == nil {
		rng = utils.SecureRandomInt
	}

	if len(options) == 0 {
		return nil
	}

	// Check if all options have 0 votes
	allZeroVotes := true
	for _, opt := range options {
		if opt.VoteCount > 0 {
			allZeroVotes = false
			break
		}
	}

	// If 0 votes total, pick random option
	if allZeroVotes {
		randomIndex := rng(len(options))
		return &options[randomIndex]
	}

	// Normal tie-breaking with votes
	winner := &options[0]
	for i := 1; i < len(options); i++ {
		opt := &options[i]

		// Higher vote count wins
		if opt.VoteCount > winner.VoteCount {
			winner = opt
			continue
		}

		// Tie-breaker: first to reach highest vote (LastHighestVoteAt)
		if opt.VoteCount == winner.VoteCount {
			if opt.LastHighestVoteAt != nil && winner.LastHighestVoteAt != nil {
				if opt.LastHighestVoteAt.Before(*winner.LastHighestVoteAt) {
					winner = opt
				}
			} else if opt.LastHighestVoteAt != nil {
				winner = opt
			}
		}
	}

	return winner
}
