package info_test

import (
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/info"
)

func TestSearchTopic(t *testing.T) {
	loader := info.NewLoader("../../configs/info")
	if err := loader.Load(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	tests := []struct {
		name          string
		topicName     string
		expectFound   bool
		expectFeature string
	}{
		{
			name:          "Find harvest topic under farming",
			topicName:     "harvest",
			expectFound:   true,
			expectFeature: "farming",
		},
		{
			name:          "Find compost topic under farming",
			topicName:     "compost",
			expectFound:   true,
			expectFeature: "farming",
		},
		{
			name:        "Topic does not exist",
			topicName:   "nonexistent",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, featureName, found := loader.SearchTopic(tt.topicName)

			if found != tt.expectFound {
				t.Errorf("Expected found=%v, got %v", tt.expectFound, found)
			}

			if found && featureName != tt.expectFeature {
				t.Errorf("Expected feature=%s, got %s", tt.expectFeature, featureName)
			}

			if found && topic == nil {
				t.Error("Expected topic data, got nil")
			}
		})
	}
}
