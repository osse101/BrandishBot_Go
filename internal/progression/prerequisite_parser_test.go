package progression

import (
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePrerequisite_Static(t *testing.T) {
	isDynamic, dynamic, staticKey, err := ParsePrerequisite("item_money")
	assert.NoError(t, err)
	assert.False(t, isDynamic)
	assert.Nil(t, dynamic)
	assert.Equal(t, "item_money", staticKey)
}

func TestParsePrerequisite_NodesUnlockedBelowTier(t *testing.T) {
	isDynamic, dynamic, staticKey, err := ParsePrerequisite("-nodes_unlocked_below_tier:2:5")
	assert.NoError(t, err)
	assert.True(t, isDynamic)
	assert.Empty(t, staticKey)
	assert.NotNil(t, dynamic)
	assert.Equal(t, "nodes_unlocked_below_tier", dynamic.Type)
	assert.Equal(t, 2, dynamic.Tier)
	assert.Equal(t, 5, dynamic.Count)
}

func TestParsePrerequisite_TotalNodesUnlocked(t *testing.T) {
	isDynamic, dynamic, staticKey, err := ParsePrerequisite("-total_nodes_unlocked:10")
	assert.NoError(t, err)
	assert.True(t, isDynamic)
	assert.Empty(t, staticKey)
	assert.NotNil(t, dynamic)
	assert.Equal(t, "total_nodes_unlocked", dynamic.Type)
	assert.Equal(t, 10, dynamic.Count)
}

func TestParsePrerequisite_InvalidNodesUnlockedBelowTier_WrongParams(t *testing.T) {
	_, _, _, err := ParsePrerequisite("-nodes_unlocked_below_tier:2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid syntax")
}

func TestParsePrerequisite_InvalidTotalNodesUnlocked_WrongParams(t *testing.T) {
	_, _, _, err := ParsePrerequisite("-total_nodes_unlocked:10:extra")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid syntax")
}

func TestParsePrerequisite_InvalidNodesUnlockedBelowTier_NonIntTier(t *testing.T) {
	_, _, _, err := ParsePrerequisite("-nodes_unlocked_below_tier:abc:5")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tier")
}

func TestParsePrerequisite_InvalidNodesUnlockedBelowTier_NonIntCount(t *testing.T) {
	_, _, _, err := ParsePrerequisite("-nodes_unlocked_below_tier:2:xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid count")
}

func TestParsePrerequisite_InvalidTotalNodesUnlocked_NonIntCount(t *testing.T) {
	_, _, _, err := ParsePrerequisite("-total_nodes_unlocked:xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid count")
}

func TestParsePrerequisite_UnknownDynamicType(t *testing.T) {
	_, _, _, err := ParsePrerequisite("-unknown_type:5")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dynamic prerequisite type")
}

func TestValidateDynamicPrerequisite_Valid_NodesUnlockedBelowTier(t *testing.T) {
	prereq := &domain.DynamicPrerequisite{
		Type:  "nodes_unlocked_below_tier",
		Tier:  2,
		Count: 5,
	}
	err := ValidateDynamicPrerequisite(prereq)
	assert.NoError(t, err)
}

func TestValidateDynamicPrerequisite_Valid_TotalNodesUnlocked(t *testing.T) {
	prereq := &domain.DynamicPrerequisite{
		Type:  "total_nodes_unlocked",
		Count: 10,
	}
	err := ValidateDynamicPrerequisite(prereq)
	assert.NoError(t, err)
}

func TestValidateDynamicPrerequisite_InvalidCount_Zero(t *testing.T) {
	prereq := &domain.DynamicPrerequisite{
		Type:  "total_nodes_unlocked",
		Count: 0,
	}
	err := ValidateDynamicPrerequisite(prereq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count must be > 0")
}

func TestValidateDynamicPrerequisite_InvalidCount_Negative(t *testing.T) {
	prereq := &domain.DynamicPrerequisite{
		Type:  "total_nodes_unlocked",
		Count: -5,
	}
	err := ValidateDynamicPrerequisite(prereq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count must be > 0")
}

func TestValidateDynamicPrerequisite_InvalidTier_Negative(t *testing.T) {
	prereq := &domain.DynamicPrerequisite{
		Type:  "nodes_unlocked_below_tier",
		Tier:  -1,
		Count: 5,
	}
	err := ValidateDynamicPrerequisite(prereq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tier")
}

func TestValidateDynamicPrerequisite_ValidTier_LargeValue(t *testing.T) {
	// Large tier values are valid (no maximum tier enforced)
	prereq := &domain.DynamicPrerequisite{
		Type:  "nodes_unlocked_below_tier",
		Tier:  10,
		Count: 5,
	}
	err := ValidateDynamicPrerequisite(prereq)
	assert.NoError(t, err)
}

func TestValidateDynamicPrerequisite_Nil(t *testing.T) {
	err := ValidateDynamicPrerequisite(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prerequisite is nil")
}

func TestParsePrerequisite_MultipleFormats(t *testing.T) {
	tests := []struct {
		name       string
		prereq     string
		isDynamic  bool
		wantType   string
		wantTier   int
		wantCount  int
		wantStatic string
	}{
		{
			name:       "static key",
			prereq:     "progression_system",
			isDynamic:  false,
			wantStatic: "progression_system",
		},
		{
			name:      "dynamic tier 0",
			prereq:    "-nodes_unlocked_below_tier:0:1",
			isDynamic: true,
			wantType:  "nodes_unlocked_below_tier",
			wantTier:  0,
			wantCount: 1,
		},
		{
			name:      "dynamic tier 4",
			prereq:    "-nodes_unlocked_below_tier:4:20",
			isDynamic: true,
			wantType:  "nodes_unlocked_below_tier",
			wantTier:  4,
			wantCount: 20,
		},
		{
			name:      "total unlock large",
			prereq:    "-total_nodes_unlocked:100",
			isDynamic: true,
			wantType:  "total_nodes_unlocked",
			wantCount: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDynamic, dynamic, staticKey, err := ParsePrerequisite(tt.prereq)
			require.NoError(t, err)
			assert.Equal(t, tt.isDynamic, isDynamic)

			if tt.isDynamic {
				require.NotNil(t, dynamic)
				assert.Equal(t, tt.wantType, dynamic.Type)
				assert.Equal(t, tt.wantCount, dynamic.Count)
				if tt.wantType == "nodes_unlocked_below_tier" {
					assert.Equal(t, tt.wantTier, dynamic.Tier)
				}
			} else {
				assert.Equal(t, tt.wantStatic, staticKey)
			}
		})
	}
}
