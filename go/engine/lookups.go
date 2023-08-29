package engine

import (
	"github.com/cloudflare/ahocorasick"
	"github.com/rulestone/condition"
)

func contains(text string, patterns []string) []string {
	machine := ahocorasick.NewStringMatcher(patterns)
	hits := machine.Match([]byte(text))

	// Extracting the matched patterns from the hits.
	matchedPatterns := make([]string, len(hits))
	for i, hit := range hits {
		matchedPatterns[i] = patterns[hit]
	}

	return matchedPatterns
}

type StringMatcher struct {
	machine    *ahocorasick.Matcher
	patterns   []string
	categories [][]condition.Operand
	matchMap   map[string]int
}

func NewStringMatcher() *StringMatcher {
	return &StringMatcher{matchMap: make(map[string]int)}
}

func (sm *StringMatcher) AddPattern(pattern string, category condition.Operand) {
	if index, ok := sm.matchMap[pattern]; !ok {
		sm.matchMap[pattern] = len(sm.patterns)
		sm.patterns = append(sm.patterns, pattern)
		sm.categories = append(sm.categories, []condition.Operand{category})
	} else {
		sm.categories[index] = append(sm.categories[index], category)
	}
}

func (sm *StringMatcher) Build() {
	if sm.machine != nil {
		panic("StringMatcher already built")
	}
	sm.machine = ahocorasick.NewStringMatcher(sm.patterns)
}

func (sm *StringMatcher) Match(text string) []condition.Operand {
	if sm.machine == nil {
		panic("StringMatcher not built")
	}
	hits := sm.machine.Match([]byte(text))

	matchedCategories := make([]condition.Operand, 0)
	for _, hit := range hits {
		matchedCategories = append(matchedCategories, sm.categories[hit]...)
	}

	return matchedCategories
}
