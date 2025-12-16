// GetBuyablePrices retrieves all buyable items with their prices
func (r *UserRepository) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	query := `
		SELECT DISTINCT i.item_id, i.item_name, i.item_description, i.base_value
		FROM items i
		INNER JOIN item_type_assignments ita ON i.item_id = ita.item_id
		INNER JOIN item_types it ON ita.item_type_id = it.item_type_id
		WHERE it.type_name = 'buyable'
		ORDER BY i.item_name
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query buyable items: %w", err)
	}
	defer rows.Close()

	var items []domain.Item
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.BaseValue); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating buyable items: %w", err)
	}

	return items, nil
}
