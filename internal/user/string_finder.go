package user

import (
	"regexp"
	"strings"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// StringFinder is responsible for finding specific strings in messages
type StringFinder struct {
	rules []FinderRule
	mu    sync.RWMutex
}

// FinderRule defines a pattern to look for and the code to return if found
type FinderRule struct {
	Pattern  *regexp.Regexp
	Code     string
	Priority int
}

// NewStringFinder creates a new StringFinder with default rules
func NewStringFinder() *StringFinder {
	sf := &StringFinder{
		rules: make([]FinderRule, 0),
	}
	sf.loadDefaultRules()
	return sf
}

func (sf *StringFinder) loadDefaultRules() {
	// These would ideally come from a config or DB
	sf.addRule("Bapanada", "OBS", 10)
	sf.addRule("going", "TRAP", 5)
}

func (sf *StringFinder) addRule(patternStr, code string, priority int) {
	// Case insensitive, word boundaries
	// escape pattern string just in case, though for simple words it's fine
	escaped := regexp.QuoteMeta(patternStr)
	// \b matches word boundaries
	// (?i) makes it case insensitive
	regex := regexp.MustCompile(`(?i)\b` + escaped + `\b`)
	
	sf.rules = append(sf.rules, FinderRule{
		Pattern:  regex,
		Code:     code,
		Priority: priority,
	})
}

// FindMatches searches the message for known strings and returns the matches
// respecting priority rules (only highest priority matches are returned)
func (sf *StringFinder) FindMatches(message string) []domain.FoundString {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	var matches []struct {
		match    domain.FoundString
		priority int
	}

	trimmedMsg := strings.TrimSpace(message)
	if trimmedMsg == "" {
		return nil
	}

	highestPriority := -1

	for _, rule := range sf.rules {
		// FindString returns "holding the text of the leftmost match in b"
		// We want to see if it matches.
		// If we want multiple matches of the same rule? "Multiple strings can be found per message"
		// The example "Bapanada" -> "Bapanada" (one match).
		// If message is "Bapanada Bapanada", do we return it twice?
		// Requirement says "return to the client which string was found".
		// Usually unique matches per rule per message is enough, or just list distinct found strings.
		// "Multiple strings can be found per message" likely refers to DIFFERENT strings.
		// But let's find all occurrences just in case, or at least one per rule.
		// "Strings need to be exact matches"
		
		found := rule.Pattern.FindString(trimmedMsg)
		if found != "" {
			m := domain.FoundString{
				Code:  rule.Code,
				Value: found, // Return the actual matched string from regex (preserves case if we want, or from input)
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
