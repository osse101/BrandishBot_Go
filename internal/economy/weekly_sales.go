package economy

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// loadWeeklySales loads the weekly sales configuration from file
func (s *service) loadWeeklySales() error {
	data, err := os.ReadFile(config.ConfigPathWeeklySales)
	if err != nil {
		return fmt.Errorf("failed to read weekly sales config: %w", err)
	}

	var cfg domain.WeeklySaleConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse weekly sales config: %w", err)
	}

	s.weeklySalesMu.Lock()
	s.weeklySales = cfg.SalesSchedule
	s.weeklySalesMu.Unlock()

	return nil
}

// getCurrentWeeklySale returns the current week's sale (based on week offset)
func (s *service) getCurrentWeeklySale() *domain.WeeklySale {
	s.weeklySalesMu.RLock()
	defer s.weeklySalesMu.RUnlock()

	if len(s.weeklySales) == 0 {
		return nil
	}

	// Calculate which week we're in (0-3 for 4-week rotation)
	_, weekNum := s.now().ISOWeek()
	weekOffset := (weekNum - 1) % 4 // 0, 1, 2, 3

	// Find the sale for this week's offset
	for _, sale := range s.weeklySales {
		if sale.WeekOffset == weekOffset {
			return &sale
		}
	}

	return nil
}
