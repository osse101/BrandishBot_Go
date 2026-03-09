package stringfinder

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Finder is responsible for finding specific strings in messages
type Finder struct {
	// optimizedRegex matches any of the rules' patterns
	optimizedRegex *regexp.Regexp
	// ruleMap maps lowercased matched strings to their corresponding rules
	ruleMap map[string][]Rule
	mu      sync.RWMutex
}

// Rule defines a pattern to look for and the code to return if found
type Rule struct {
	PatternStr string `json:"pattern"`
	Code       string `json:"code"`
	Priority   int    `json:"priority"`
}

// New creates a new Finder with rules loaded from the given path
func New(configPath string) *Finder {
	sf := &Finder{
		ruleMap: make(map[string][]Rule),
	}

	if configPath != "" {
		if err := sf.LoadRules(configPath); err != nil {
			// Fallback to default rules if loading fails
			sf.loadDefaultRules()
		}
	} else {
		sf.loadDefaultRules()
	}

	sf.Compile()
	return sf
}

// LoadRules loads rules from a JSON file
func (sf *Finder) LoadRules(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var rules []Rule
	if err := json.NewDecoder(file).Decode(&rules); err != nil {
		return err
	}

	sf.mu.Lock()
	defer sf.mu.Unlock()

	for _, rule := range rules {
		sf.addRuleUnsafe(rule.PatternStr, rule.Code, rule.Priority)
	}

	return nil
}

func (sf *Finder) loadDefaultRules() {
	// These are now fallbacks if config is missing
	sf.AddRule("Bapanada", "OBS", 10)
	sf.AddRule("gary", "OBS", 10)
	sf.AddRule("shedinja", "OBS", 10)
}

// AddRule adds a new rule and recompiles the regex
func (sf *Finder) AddRule(patternStr, code string, priority int) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.addRuleUnsafe(patternStr, code, priority)
	sf.compileUnsafe()
}

// RemoveRule removes a rule by pattern and recompiles
func (sf *Finder) RemoveRule(patternStr string) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	lowerPattern := strings.ToLower(patternStr)
	delete(sf.ruleMap, lowerPattern)
	sf.compileUnsafe()
}

func (sf *Finder) addRuleUnsafe(patternStr, code string, priority int) {
	lowerPattern := strings.ToLower(patternStr)
	rule := Rule{
		PatternStr: patternStr,
		Code:       code,
		Priority:   priority,
	}

	existingRules := sf.ruleMap[lowerPattern]
	for _, r := range existingRules {
		if r.Code == code && r.Priority == priority {
			return // Already exists
		}
	}

	sf.ruleMap[lowerPattern] = append(sf.ruleMap[lowerPattern], rule)
}

// Compile builds the optimized regex from the added rules safely
func (sf *Finder) Compile() {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.compileUnsafe()
}

func (sf *Finder) compileUnsafe() {
	if len(sf.ruleMap) == 0 {
		sf.optimizedRegex = nil
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

	// Regex flags: (?i) case-insensitive, \b word boundaries, (...) alternation group.
	regexStr := `(?i)\b(` + strings.Join(escapedPatterns, "|") + `)\b`
	compiledRegex := regexp.MustCompile(regexStr)

	sf.optimizedRegex = compiledRegex
}

// FindMatches searches the message for known strings and returns the matches
// respecting priority rules (only highest priority matches are returned)
func (sf *Finder) FindMatches(message string) []domain.FoundString {
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
