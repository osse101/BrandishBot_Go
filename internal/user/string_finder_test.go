package user

import (
	"reflect"
	"sort"
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

	// Add test rules
	sf.addRule("Bapanada", "OBS", 10)
	sf.addRule("going", "TRAP", 5)
	sf.addRule("hello", "GREET", 10)

	// Compile regex
	sf.compile()

	tests := []struct {
		name    string
		message string
		want    []domain.FoundString
	}{
		{
			name:    "Single match high priority",
			message: "Bapanada",
			want: []domain.FoundString{
				{Code: "OBS", Value: "Bapanada"},
			},
		},
		{
			name:    "Single match low priority",
			message: "Where is it going?",
			want: []domain.FoundString{
				{Code: "TRAP", Value: "going"},
			},
		},
		{
			name:    "Mixed priority - only high returned",
			message: "Bapanada is going somewhere",
			want: []domain.FoundString{
				{Code: "OBS", Value: "Bapanada"},
			},
		},
		{
			name:    "Multiple high priority matches",
			message: "Bapanada say hello",
			want: []domain.FoundString{
				{Code: "OBS", Value: "Bapanada"},
				{Code: "GREET", Value: "hello"},
			},
		},
		{
			name:    "Case insensitive match",
			message: "bapanada",
			want: []domain.FoundString{
				{Code: "OBS", Value: "bapanada"},
			},
		},
		{
			name:    "No match - partial word",
			message: "ongoing",
			want:    nil,
		},
		{
			name:    "No match - empty string",
			message: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sf.FindMatches(tt.message)

			// Sort results for consistent comparison
			sort.Slice(got, func(i, j int) bool {
				return got[i].Code < got[j].Code
			})
			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Code < tt.want[j].Code
			})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringFinder.FindMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
