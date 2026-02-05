package progression

import (
	"context"
)

// TestHelper provides utilities for test setup
type TestHelper struct {
	service Service
}

// NewTestHelper creates a test helper with the given service
func NewTestHelper(service Service) *TestHelper {
	return &TestHelper{service: service}
}

// UnlockAllFeatures unlocks all gated features for testing
// This should be called in test setup to enable all features
func (h *TestHelper) UnlockAllFeatures(ctx context.Context) error {
	features := []string{
		FeatureEconomy,
		FeatureEconomy,
		FeatureUpgrade,
		FeatureDisassemble,
		FeatureEconomy,
		FeatureSearch,
	}

	for _, feature := range features {
		if err := h.service.AdminUnlock(ctx, feature, 1); err != nil {
			return err
		}
	}

	return nil
}

// UnlockFeature unlocks a specific feature for testing
func (h *TestHelper) UnlockFeature(ctx context.Context, featureKey string) error {
	return h.service.AdminUnlock(ctx, featureKey, 1)
}

// UnlockItem unlocks a specific item for testing
func (h *TestHelper) UnlockItem(ctx context.Context, itemKey string) error {
	return h.service.AdminUnlock(ctx, itemKey, 1)
}
