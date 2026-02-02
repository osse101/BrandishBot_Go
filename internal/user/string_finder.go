package user

import (
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// StringFinder is responsible for finding specific strings in messages
type StringFinder struct {
	// optimizedRegex matches any of the rules' patterns
	optimizedRegex *regexp.Regexp
	// ruleMap maps lowercased matched strings to their corresponding rules
	ruleMap map[string][]FinderRule
	mu      sync.RWMutex
}

// FinderRule defines a pattern to look for and the code to return if found
type FinderRule struct {
	PatternStr string
	Code       string
	Priority   int
}

// NewStringFinder creates a new StringFinder with default rules
func NewStringFinder() *StringFinder {
	sf := &StringFinder{
		ruleMap: make(map[string][]FinderRule),
	}
	sf.loadDefaultRules()
	sf.compile()
	return sf
}

func (sf *StringFinder) loadDefaultRules() {
	// These would ideally come from a config or DB
	sf.addRule("Bapanada", "OBS", 10)
	sf.addRule("gary", "OBS", 10)
	sf.addRule("shedinja", "OBS", 10)
}

func (sf *StringFinder) addRule(patternStr, code string, priority int) {
	lowerPattern := strings.ToLower(patternStr)
	rule := FinderRule{
		PatternStr: patternStr,
		Code:       code,
		Priority:   priority,
	}

	sf.ruleMap[lowerPattern] = append(sf.ruleMap[lowerPattern], rule)
}

// compile builds the optimized regex from the added rules
// Note: This method acquires a write lock to safely update the regex
func (sf *StringFinder) compile() {
	if len(sf.ruleMap) == 0 {
		return
	}

	// efficient matching: sort patterns by length descending to handle overlapping prefixes correctly
	// e.g. "superman" before "super"
	patterns := make([]string, 0, len(sf.ruleMap))
	for p := range sf.ruleMap {
		patterns = append(patterns, p)
	}
	sort.Slice(patterns, func(i, j int) bool {
		return len(patterns[i]) > len(patterns[j])
	})

	escapedPatterns := make([]string, 0, len(patterns))
	for _, p := range patterns {
		escapedPatterns = append(escapedPatterns, regexp.QuoteMeta(p))
	}

	// (?i) case insensitive
	// \b word boundaries
	// (p1|p2|...) alternation
	regexStr := `(?i)\b(` + strings.Join(escapedPatterns, "|") + `)\b`
	compiledRegex := regexp.MustCompile(regexStr)

	// Acquire write lock to safely update the regex
	sf.mu.Lock()
	sf.optimizedRegex = compiledRegex
	sf.mu.Unlock()
}

// FindMatches searches the message for known strings and returns the matches
// respecting priority rules (only highest priority matches are returned)
func (sf *StringFinder) FindMatches(message string) []domain.FoundString {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	trimmedMsg := strings.TrimSpace(message)
	if trimmedMsg == "" || sf.optimizedRegex == nil {
		return nil
	}

	var matches []struct {
		match    domain.FoundString
		priority int
	}
	highestPriority := -1

	// Find all non-overlapping matches
	foundStrings := sf.optimizedRegex.FindAllString(trimmedMsg, -1)

	for _, found := range foundStrings {
		// Look up rules for this string (case insensitive)
		lowerFound := strings.ToLower(found)
		rules, exists := sf.ruleMap[lowerFound]
		if !exists {
			continue
		}

		for _, rule := range rules {
			m := domain.FoundString{
				Code:  rule.Code,
				Value: found, // Return the actual matched string from text
			}

			matches = append(matches, struct {
				match    domain.FoundString
				priority int
			}{m, rule.Priority})

			if rule.Priority > highestPriority {
				highestPriority = rule.Priority
			}
		}
	}

	// Filter by highest priority
	var result []domain.FoundString
	for _, m := range matches {
		if m.priority == highestPriority {
			result = append(result, m.match)
		}
	}

	return result
}
