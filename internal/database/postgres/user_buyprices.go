package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetBuyablePrices retrieves all buyable items with their prices
func (r *UserRepository) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	rows, err := r.q.GetBuyablePrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query buyable items: %w", err)
	}

	var items []domain.Item
	for _, row := range rows {
		items = append(items, domain.Item{
			ID:           int(row.ItemID),
			InternalName: row.InternalName,
			Description:  row.ItemDescription.String,
			BaseValue:    int(row.BaseValue.Int32),
		})
	}

	return items, nil
}
