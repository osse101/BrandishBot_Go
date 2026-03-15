package slots

import (
	"fmt"
)

func (s *service) spinReels() (string, string, string) {
	return s.selectWeightedSymbol(), s.selectWeightedSymbol(), s.selectWeightedSymbol()
}

func (s *service) selectWeightedSymbol() string {
	totalWeight := 1000

	roll := s.rng(totalWeight)

	cumulative := 0
	for _, symbol := range []string{SymbolLemon, SymbolCherry, SymbolBell, SymbolBar, SymbolSeven, SymbolDiamond, SymbolStar} {
		cumulative += SymbolWeights[symbol]
		if roll < cumulative {
			return symbol
		}
	}

	return SymbolLemon
}

func (s *service) calculatePayout(reel1, reel2, reel3 string, betAmount int) (amount int, mult float64, trigger string) {
	if reel1 == reel2 && reel2 == reel3 {
		mult = PayoutMultipliers[reel1]
		amount = int(float64(betAmount) * mult)
		trigger = s.determineWinType(mult)
		return
	}

	if reel1 == reel2 || reel2 == reel3 || reel1 == reel3 {
		mult = TwoMatchMultiplier
		amount = int(float64(betAmount) * mult)
		trigger = TriggerNormal
		return
	}

	return 0, 0.0, TriggerNormal
}

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

func (s *service) formatMessage(reel1, reel2, reel3 string, betAmount, amount int, trigger string) string {
	if amount == 0 {
		return fmt.Sprintf("Better luck next time! You lost %d money.", betAmount)
	}

	netWin := amount - betAmount

	switch trigger {
	case TriggerMegaJackpot:
		return fmt.Sprintf("🌟 MEGA JACKPOT! 🌟 You won %d money (net +%d)!", amount, netWin)
	case TriggerJackpot:
		return fmt.Sprintf("💎 JACKPOT! 💎 You won %d money (net +%d)!", amount, netWin)
	case TriggerBigWin:
		return fmt.Sprintf("🎉 BIG WIN! You won %d money (net +%d)!", amount, netWin)
	default:
		if netWin > 0 {
			return fmt.Sprintf("You won %d money (net +%d)!", amount, netWin)
		}
		if netWin == 0 {
			return fmt.Sprintf("You broke even! %d money returned.", amount)
		}
		if (reel1 == reel2 || reel2 == reel3 || reel1 == reel3) && (reel1 != reel2 || reel2 != reel3) {
			return fmt.Sprintf("Consolation! You got %d back. (net %d)", amount, netWin)
		}
		return fmt.Sprintf("No luck! You won %d money (net %d).", amount, netWin)
	}
}
