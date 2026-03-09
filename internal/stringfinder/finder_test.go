package stringfinder

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestFinder_FindMatches(t *testing.T) {
	// Create blank finder
	sf := &Finder{
		ruleMap: make(map[string][]Rule),
	}

	// Add test rules
	sf.AddRule("Bapanada", "OBS", 10)
	sf.AddRule("gary", "OBS", 10)
	sf.AddRule("shedinja", "OBS", 10)

	// Compile regex
	sf.Compile()

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

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFinder_EdgeCases(t *testing.T) {
	sf := &Finder{
		ruleMap: make(map[string][]Rule),
	}
	sf.AddRule("Bapanada", "OBS", 10)
	sf.AddRule("gary", "OBS", 10)
	sf.Compile()

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
			assert.Len(t, got, tt.wantMatches)
			if len(got) > 0 {
				assert.Equal(t, tt.wantCode, got[0].Code)
			}
		})
	}
}

func TestFinder_BoundaryConditions(t *testing.T) {
	sf := &Finder{
		ruleMap: make(map[string][]Rule),
	}
	sf.AddRule("Bapanada", "OBS", 10)
	sf.Compile()

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
			assert.Len(t, got, tt.wantMatches)
		})
	}
}

func TestFinder_EmptyRules(t *testing.T) {
	sf := &Finder{
		ruleMap: make(map[string][]Rule),
	}
	// Don't add any rules, don't compile

	got := sf.FindMatches("Bapanada gary shedinja")
	assert.Empty(t, got)
}

func TestFinder_PriorityFiltering(t *testing.T) {
	sf := &Finder{
		ruleMap: make(map[string][]Rule),
	}
	sf.AddRule("high", "HIGH", 10)
	sf.AddRule("low", "LOW", 5)
	sf.Compile()

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
			assert.Len(t, got, tt.wantLen)
			if len(got) > 0 {
				assert.Equal(t, tt.wantCode, got[0].Code)
			}
		})
	}
}
