package user

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringFinder_DynamicRules(t *testing.T) {
	sf := NewStringFinder("") // Empty path loads default rules

	// Test AddRule
	sf.AddRule("TempRule", "TEMP", 100)
	matches := sf.FindMatches("Here is a TempRule")
	require.Len(t, matches, 1)
	assert.Equal(t, "TEMP", matches[0].Code)

	// Test RemoveRule
	sf.RemoveRule("TempRule")
	matches = sf.FindMatches("Here is a TempRule")
	assert.Empty(t, matches)
}

func TestStringFinder_LoadRules(t *testing.T) {
	// Create a temporary config file
	content := `[{"pattern": "TestPattern", "code": "TEST", "priority": 50}]`
	tmpfile, err := os.CreateTemp("", "rules_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name()) // clean up

	_, err = tmpfile.Write([]byte(content))
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	sf := NewStringFinder(tmpfile.Name())
	matches := sf.FindMatches("This is a TestPattern")
	require.Len(t, matches, 1)
	assert.Equal(t, "TEST", matches[0].Code)
}
