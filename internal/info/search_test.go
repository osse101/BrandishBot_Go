package info_test

import (
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/info"
	"github.com/stretchr/testify/assert"
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

			assert.Equal(t, tt.expectFound, found, "Expected found result to match")

			if found {
				assert.Equal(t, tt.expectFeature, featureName, "Expected feature to match")
			}

			if found {
				assert.NotNil(t, topic, "Expected topic data, got nil")
			}
		})
	}
}
