package domain

// IsValidFilterType checks if a filter string is valid (empty string is valid = no filter)
func IsValidFilterType(filter string) bool {
	if filter == "" {
		return true
	}
	return filter == FilterTypeUpgrade || filter == FilterTypeSellable || filter == FilterTypeConsumable
}
