package slots

import (
	"fmt"
)

// spinReels generates three random symbols using weighted distribution
func (s *service) spinReels() (string, string, string) {
	return s.selectWeightedSymbol(), s.selectWeightedSymbol(), s.selectWeightedSymbol()
}

// selectWeightedSymbol performs weighted random selection of a symbol
func (s *service) selectWeightedSymbol() string {
	totalWeight := 1000 // Sum of all weights

	roll := s.rng(totalWeight)

	cumulative := 0
	for _, symbol := range []string{SymbolLemon, SymbolCherry, SymbolBell, SymbolBar, SymbolSeven, SymbolDiamond, SymbolStar} {
		cumulative += SymbolWeights[symbol]
		if roll < cumulative {
			return symbol
		}
	}

	// Fallback (should never happen)
	return SymbolLemon
}

// calculatePayout determines the payout amount, multiplier, and trigger type
func (s *service) calculatePayout(reel1, reel2, reel3 string, betAmount int) (payoutAmount int, multiplier float64, triggerType string) {
	// Check for 3 matching symbols
	if reel1 == reel2 && reel2 == reel3 {
		multiplier = PayoutMultipliers[reel1]
		payoutAmount = int(float64(betAmount) * multiplier)
		triggerType = s.determineWinType(multiplier)
		return
	}

	// Check for 2 matching symbols (consolation prize)
	if reel1 == reel2 || reel2 == reel3 || reel1 == reel3 {
		multiplier = TwoMatchMultiplier
		payoutAmount = int(float64(betAmount) * multiplier)
		triggerType = TriggerNormal
		return
	}

	// No match - total loss
	return 0, 0.0, TriggerNormal
}

// determineWinType classifies the win based on multiplier
func (s *service) determineWinType(multiplier float64) string {
	switch {
	case multiplier >= 100.0:
		return TriggerMegaJackpot
	case multiplier >= JackpotThreshold:
		return TriggerJackpot
	case multiplier >= BigWinThreshold:
		return TriggerBigWin
	default:
		return TriggerNormal
	}
}

// formatMessage creates a user-facing message for the result
func (s *service) formatMessage(reel1, reel2, reel3 string, betAmount, payoutAmount int, triggerType string) string {
	if payoutAmount == 0 {
		return fmt.Sprintf("Better luck next time! You lost %d money.", betAmount)
	}

	netWin := payoutAmount - betAmount

	switch triggerType {
	case TriggerMegaJackpot:
		return fmt.Sprintf("🌟 MEGA JACKPOT! 🌟 You won %d money (net +%d)!", payoutAmount, netWin)
	case TriggerJackpot:
		return fmt.Sprintf("💎 JACKPOT! 💎 You won %d money (net +%d)!", payoutAmount, netWin)
	case TriggerBigWin:
		return fmt.Sprintf("🎉 BIG WIN! You won %d money (net +%d)!", payoutAmount, netWin)
	default:
		if netWin > 0 {
			return fmt.Sprintf("You won %d money (net +%d)!", payoutAmount, netWin)
		}
		if netWin == 0 {
			return fmt.Sprintf("You broke even! %d money returned.", payoutAmount)
		}
		if (reel1 == reel2 || reel2 == reel3 || reel1 == reel3) && (reel1 != reel2 || reel2 != reel3) {
			// Consolation prize (2 symbols match, but not 3)
			return fmt.Sprintf("Consolation! You got %d back. (net %d)", payoutAmount, netWin)
		}
		return fmt.Sprintf("No luck! You won %d money (net %d).", payoutAmount, netWin)
	}
}
