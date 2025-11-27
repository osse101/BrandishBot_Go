package economy

func calculateAffordableQuantity(requestedQuantity, price, balance int) (int, int) {
	if price <= 0 {
		return requestedQuantity, 0
	}
	maxAffordable := balance / price
	if maxAffordable == 0 {
		return 0, 0
	}
	actual := requestedQuantity
	if actual > maxAffordable {
		actual = maxAffordable
	}
	cost := actual * price
	return actual, cost
}
