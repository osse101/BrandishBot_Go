package user

import (
	"reflect"
	"testing"
	"sort"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestStringFinder_FindMatches(t *testing.T) {
	sf := NewStringFinder()
	
	// Override rules for testing predictability
	sf.rules = nil
	sf.addRule("Bapanada", "OBS", 10)
	sf.addRule("going", "TRAP", 5)
	sf.addRule("hello", "GREET", 10)

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
			
			// Sort results for consistent comparison if order doesn't strictly matter for equality check 
			// (though implementation currently returns in rule order, let's just use ElementsMatch equivalent logic or sort)
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
