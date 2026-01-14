package progression

import (
	"context"
	"testing"
)

// TestUnlockAllFeatures tests the UnlockAllFeatures helper
func TestUnlockAllFeatures(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	helper := NewTestHelper(service)

	// Unlock all features
	err := helper.UnlockAllFeatures(ctx)
	if err != nil {
		t.Fatalf("UnlockAllFeatures failed: %v", err)
	}

	// Verify features are unlocked
	features := []string{
		FeatureBuy,
		FeatureSell,
		FeatureUpgrade,
		FeatureDisassemble,
	}

	for _, feature := range features {
		unlocked, err := service.IsFeatureUnlocked(ctx, feature)
		if err != nil {
			t.Fatalf("IsFeatureUnlocked failed for %s: %v", feature, err)
		}
		if !unlocked {
			t.Errorf("Feature %s should be unlocked", feature)
		}
	}
}

// TestUnlockFeature tests unlocking individual features
func TestUnlockFeature(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	helper := NewTestHelper(service)

	// Unlock buy feature
	err := helper.UnlockFeature(ctx, FeatureBuy)
	if err != nil {
		t.Fatalf("UnlockFeature failed: %v", err)
	}

	// Verify it's unlocked
	unlocked, err := service.IsFeatureUnlocked(ctx, FeatureBuy)
	if err != nil {
		t.Fatalf("IsFeatureUnlocked failed: %v", err)
	}
	if !unlocked {
		t.Error("Buy feature should be unlocked")
	}
}
