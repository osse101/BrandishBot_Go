package user

import (
	"os"
	"testing"
)

func TestStringFinder_DynamicRules(t *testing.T) {
	sf := NewStringFinder("") // Empty path loads default rules

	// Test AddRule
	sf.AddRule("TempRule", "TEMP", 100)
	matches := sf.FindMatches("Here is a TempRule")
	if len(matches) != 1 || matches[0].Code != "TEMP" {
		t.Errorf("Expected match for TempRule, got %v", matches)
	}

	// Test RemoveRule
	sf.RemoveRule("TempRule")
	matches = sf.FindMatches("Here is a TempRule")
	if len(matches) != 0 {
		t.Errorf("Expected no matches after removing rule, got %v", matches)
	}
}

func TestStringFinder_LoadRules(t *testing.T) {
	// Create a temporary config file
	content := `[{"pattern": "TestPattern", "code": "TEST", "priority": 50}]`
	tmpfile, err := os.CreateTemp("", "rules_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	sf := NewStringFinder(tmpfile.Name())
	matches := sf.FindMatches("This is a TestPattern")
	if len(matches) != 1 || matches[0].Code != "TEST" {
		t.Errorf("Expected match from loaded rule, got %v", matches)
	}
}
