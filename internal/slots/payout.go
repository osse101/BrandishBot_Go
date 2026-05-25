package slots

import (
	"fmt"
)

func (s *service) calculatePayout(result ResultType, betAmount int) (amount int, mult float64, trigger string) {
	mult = ResultPayouts[result]
	amount = int(float64(betAmount) * mult)
	trigger = s.determineWinType(mult)
	return
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

func (s *service) formatMessage(result ResultType, betAmount, amount int, trigger string) string {
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
		if result == ResultLemonTwoMatch {
			return fmt.Sprintf("Consolation! You got %d back. (net %d)", amount, netWin)
		}
		return fmt.Sprintf("No luck! You won %d money (net %d).", amount, netWin)
	}
}
