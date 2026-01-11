package user

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestStringFinder_FindMatches(t *testing.T) {
	// Create a fresh StringFinder for testing
	// Instead of relying on NewStringFinder and modifying private fields,
	// we reconstruct it or use a helper if possible.
	// Since we changed internal structure, we need to adapt the test setup.

	// Create blank finder
	sf := &StringFinder{
		ruleMap: make(map[string][]FinderRule),
	}

	// Add test rules (using real production rules)
	sf.addRule("Bapanada", "OBS", 10)
	sf.addRule("gary", "OBS", 10)
	sf.addRule("shedinja", "OBS", 10)

	// Compile regex
	sf.compile()

	tests := []struct {
		name    string
		message string
		want    []domain.FoundString
	}{
		{
			name:    "Single match - Bapanada",
			message: "Bapanada",
			want: []domain.FoundString{
				{Code: "OBS", Value: "Bapanada"},
			},
		},
		{
			name:    "Single match - gary",
			message: "gary is here",
			want: []domain.FoundString{
				{Code: "OBS", Value: "gary"},
			},
		},
		{
			name:    "Single match - shedinja",
			message: "I got a shedinja!",
			want: []domain.FoundString{
				{Code: "OBS", Value: "shedinja"},
			},
		},
		{
			name:    "Multiple matches - same priority",
			message: "Bapanada and gary together",
			want: []domain.FoundString{
				{Code: "OBS", Value: "Bapanada"},
				{Code: "OBS", Value: "gary"},
			},
		},
		{
			name:    "All three matches",
			message: "Bapanada gary shedinja",
			want: []domain.FoundString{
				{Code: "OBS", Value: "Bapanada"},
				{Code: "OBS", Value: "gary"},
				{Code: "OBS", Value: "shedinja"},
			},
		},
		{
			name:    "Case insensitive match - lowercase",
			message: "bapanada",
			want: []domain.FoundString{
				{Code: "OBS", Value: "bapanada"},
			},
		},
		{
			name:    "Case insensitive match - uppercase",
			message: "BAPANADA",
			want: []domain.FoundString{
				{Code: "OBS", Value: "BAPANADA"},
			},
		},
		{
			name:    "Case insensitive - mixed case",
			message: "BaPaNaDa",
			want: []domain.FoundString{
				{Code: "OBS", Value: "BaPaNaDa"},
			},
		},
		{
			name:    "Match with possessive - apostrophe is word boundary",
			message: "gary's friend is legendary",
			want: []domain.FoundString{
				{Code: "OBS", Value: "gary"}, // Apostrophe acts as word boundary, matches gary
			},
		},
		{
			name:    "No match - empty string",
			message: "",
			want:    nil,
		},
		{
			name:    "No match - completely unrelated",
			message: "hello world",
			want:    nil,
		},
		{
			name:    "Word boundaries - not embedded in other words",
			message: "notgary garynot",
			want:    nil, // Should not match because gary is embedded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sf.FindMatches(tt.message)

			// Sort results for consistent comparison
			sort.Slice(got, func(i, j int) bool {
				return got[i].Code < got[j].Code || (got[i].Code == got[j].Code && got[i].Value < got[j].Value)
			})
			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Code < tt.want[j].Code || (tt.want[i].Code == tt.want[j].Code && tt.want[i].Value < tt.want[j].Value)
			})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringFinder.FindMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStringFinder_EdgeCases tests edge cases per TEST_GUIDANCE.md
func TestStringFinder_EdgeCases(t *testing.T) {
	sf := &StringFinder{
		ruleMap: make(map[string][]FinderRule),
	}
	sf.addRule("Bapanada", "OBS", 10)
	sf.addRule("gary", "OBS", 10)
	sf.compile()

	tests := []struct {
		name        string
		message     string
		wantMatches int
		wantCode    string
	}{
		{
			name:        "Punctuation after match",
			message:     "Bapanada!",
			wantMatches: 1,
			wantCode:    "OBS",
		},
		{
			name:        "Match at start of sentence",
			message:     "Bapanada is cool",
			wantMatches: 1,
			wantCode:    "OBS",
		},
		{
			name:        "Match at end of sentence",
			message:     "I love Bapanada",
			wantMatches: 1,
			wantCode:    "OBS",
		},
		{
			name:        "Multiple spaces around match",
			message:     "hello   Bapanada   world",
			wantMatches: 1,
			wantCode:    "OBS",
		},
		{
			name:        "Match with comma",
			message:     "Bapanada, gary, wow",
			wantMatches: 2,
			wantCode:    "OBS",
		},
		{
			name:        "Match in question",
			message:     "Is Bapanada here?",
			wantMatches: 1,
			wantCode:    "OBS",
		},
		{
			name:        "Match with exclamation",
			message:     "Wow! gary!",
			wantMatches: 1,
			wantCode:    "OBS",
		},
		{
			name:        "Newline in message",
			message:     "Hello\nBapanada\nWorld",
			wantMatches: 1,
			wantCode:    "OBS",
		},
		{
			name:        "Tab in message",
			message:     "Hello\tBapanada\tWorld",
			wantMatches: 1,
			wantCode:    "OBS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sf.FindMatches(tt.message)
			if len(got) != tt.wantMatches {
				t.Errorf("StringFinder.FindMatches() got %d matches, want %d", len(got), tt.wantMatches)
			}
			if len(got) > 0 && got[0].Code != tt.wantCode {
				t.Errorf("StringFinder.FindMatches() code = %s, want %s", got[0].Code, tt.wantCode)
			}
		})
	}
}

// TestStringFinder_BoundaryConditions tests boundary conditions per TEST_GUIDANCE.md
func TestStringFinder_BoundaryConditions(t *testing.T) {
	sf := &StringFinder{
		ruleMap: make(map[string][]FinderRule),
	}
	sf.addRule("Bapanada", "OBS", 10)
	sf.compile()

	tests := []struct {
		name        string
		message     string
		wantMatches int
	}{
		{
			name:        "Empty message",
			message:     "",
			wantMatches: 0,
		},
		{
			name:        "Whitespace only",
			message:     "   \t\n   ",
			wantMatches: 0,
		},
		{
			name:        "Single character - no match",
			message:     "a",
			wantMatches: 0,
		},
		{
			name:        "Exact match only",
			message:     "Bapanada",
			wantMatches: 1,
		},
		{
			name:        "Very long message with match at start",
			message:     "Bapanada " + strings.Repeat("word ", 1000),
			wantMatches: 1,
		},
		{
			name:        "Very long message with match at end",
			message:     strings.Repeat("word ", 1000) + "Bapanada",
			wantMatches: 1,
		},
		{
			name:        "Very long message with match in middle",
			message:     strings.Repeat("word ", 500) + "Bapanada " + strings.Repeat("word ", 500),
			wantMatches: 1,
		},
		{
			name:        "Multiple repeated matches",
			message:     "Bapanada Bapanada Bapanada",
			wantMatches: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sf.FindMatches(tt.message)
			if len(got) != tt.wantMatches {
				t.Errorf("StringFinder.FindMatches() got %d matches, want %d", len(got), tt.wantMatches)
			}
		})
	}
}

// TestStringFinder_EmptyRules tests behavior when no rules are configured
func TestStringFinder_EmptyRules(t *testing.T) {
	sf := &StringFinder{
		ruleMap: make(map[string][]FinderRule),
	}
	// Don't add any rules, don't compile

	got := sf.FindMatches("Bapanada gary shedinja")
	if got != nil {
		t.Errorf("StringFinder.FindMatches() with no rules should return nil, got %v", got)
	}
}

// TestStringFinder_PriorityFiltering tests that only highest priority matches are returned
func TestStringFinder_PriorityFiltering(t *testing.T) {
	sf := &StringFinder{
		ruleMap: make(map[string][]FinderRule),
	}
	sf.addRule("high", "HIGH", 10)
	sf.addRule("low", "LOW", 5)
	sf.compile()

	tests := []struct {
		name     string
		message  string
		wantCode string
		wantLen  int
	}{
		{
			name:     "Only high priority returned",
			message:  "high and low words",
			wantCode: "HIGH",
			wantLen:  1,
		},
		{
			name:     "Low priority when alone",
			message:  "only low word",
			wantCode: "LOW",
			wantLen:  1,
		},
		{
			name:     "Multiple high priority",
			message:  "high high high",
			wantCode: "HIGH",
			wantLen:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sf.FindMatches(tt.message)
			if len(got) != tt.wantLen {
				t.Errorf("StringFinder.FindMatches() len = %d, want %d", len(got), tt.wantLen)
			}
			if len(got) > 0 && got[0].Code != tt.wantCode {
				t.Errorf("StringFinder.FindMatches() code = %s, want %s", got[0].Code, tt.wantCode)
			}
		})
	}
}
