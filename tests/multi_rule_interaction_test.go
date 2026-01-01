package tests

import (
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// TestMultiRuleInteraction tests that multiple rules can correctly evaluate the same event
func TestMultiRuleInteraction(t *testing.T) {
	// Create rules that will match different subsets of the same event
	rulesYAML := `
- metadata: {id: "rule-age-gt-25"}
  expression: age > 25
- metadata: {id: "rule-age-lt-50"}
  expression: age < 50
- metadata: {id: "rule-name-john"}
  expression: name == "John"
- metadata: {id: "rule-name-jane"}
  expression: name == "Jane"
- metadata: {id: "rule-complex"}
  expression: age > 20 && name == "John"
`

	repo := engine.NewRuleEngineRepo()
	_, err := repo.LoadRulesFromString(rulesYAML, engine.LoadOptions{Validate: true})
	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name          string
		event         map[string]interface{}
		expectRules   []string // Expected rule IDs that should match
	}{
		{
			name:  "John age 30 - should match rules 0,1,2,4",
			event: map[string]interface{}{"name": "John", "age": 30},
			expectRules: []string{"rule-age-gt-25", "rule-age-lt-50", "rule-name-john", "rule-complex"},
		},
		{
			name:  "Jane age 20 - should match rules 1,3 only",
			event: map[string]interface{}{"name": "Jane", "age": 20},
			expectRules: []string{"rule-age-lt-50", "rule-name-jane"},
		},
		{
			name:  "John age 60 - should match rules 0,2,4",
			event: map[string]interface{}{"name": "John", "age": 60},
			expectRules: []string{"rule-age-gt-25", "rule-name-john", "rule-complex"},
		},
		{
			name:  "Bob age 15 - should match rule 1 only",
			event: map[string]interface{}{"name": "Bob", "age": 15},
			expectRules: []string{"rule-age-lt-50"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matchedRules := genFilter.MatchEvent(tt.event)
			
			// Get rule IDs
			var matchedIDs []string
			for _, ruleID := range matchedRules {
				rule := genFilter.GetRuleDefinition(uint(ruleID))
				if rule != nil && rule.Metadata != nil {
					if id, ok := rule.Metadata["id"].(string); ok {
						matchedIDs = append(matchedIDs, id)
					}
				}
			}

			// Sort for comparison
			sort.Strings(matchedIDs)
			sort.Strings(tt.expectRules)

			if !equalStringSlices(matchedIDs, tt.expectRules) {
				t.Errorf("Rule mismatch:\n  Got:      %v\n  Expected: %v", matchedIDs, tt.expectRules)
			}
		})
	}
}

// TestMultiRuleConcurrentInteraction tests thread safety with multiple rules and events
func TestMultiRuleConcurrentInteraction(t *testing.T) {
	rulesYAML := `
- metadata: {id: "rule-1"}
  expression: value1 > 10
- metadata: {id: "rule-2"}
  expression: value2 < 100
- metadata: {id: "rule-3"}
  expression: value1 > 10 && value2 < 100
- metadata: {id: "rule-4"}
  expression: value3 == "test"
`

	repo := engine.NewRuleEngineRepo()
	_, err := repo.LoadRulesFromString(rulesYAML, engine.LoadOptions{Validate: true})
	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	testCases := []struct {
		event       map[string]interface{}
		expectCount int
	}{
		{map[string]interface{}{"value1": 20, "value2": 50, "value3": "test"}, 4}, // All rules match
		{map[string]interface{}{"value1": 20, "value2": 50}, 3},                   // Rules 1,2,3
		{map[string]interface{}{"value1": 5, "value2": 50}, 1},                    // Rule 2 only
		{map[string]interface{}{"value3": "test"}, 1},                             // Rule 4 only
	}

	// Run 100 iterations concurrently with 20 goroutines
	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				for tcIdx, tc := range testCases {
					matches := genFilter.MatchEvent(tc.event)
					if len(matches) != tc.expectCount {
						errChan <- fmt.Errorf("goroutine %d, iteration %d, test %d: expected %d matches, got %d",
							goroutineID, i, tcIdx, tc.expectCount, len(matches))
						return
					}
				}
			}
		}(g)
	}

	// Wait for all goroutines
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for errors
	for err := range errChan {
		t.Error(err)
	}
}

// TestMultiRuleSharedAttributes tests rules that share attribute access
func TestMultiRuleSharedAttributes(t *testing.T) {
	rulesYAML := `
- metadata: {id: "shared-1"}
  expression: user.age > 25
- metadata: {id: "shared-2"}
  expression: user.age < 50
- metadata: {id: "shared-3"}
  expression: user.name == "Alice"
- metadata: {id: "shared-nested"}
  expression: user.address.city == "NYC"
`

	repo := engine.NewRuleEngineRepo()
	_, err := repo.LoadRulesFromString(rulesYAML, engine.LoadOptions{Validate: true})
	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Test with nested object - all rules access the same "user" object
	event := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "Alice",
			"age":  30,
			"address": map[string]interface{}{
				"city": "NYC",
			},
		},
	}

	// All 4 rules should match
	matches := genFilter.MatchEvent(event)
	if len(matches) != 4 {
		t.Errorf("Expected 4 rules to match, got %d", len(matches))
	}

	// Test concurrently
	for i := 0; i < 100; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 4 {
			t.Errorf("Iteration %d: Expected 4 rules to match, got %d", i, len(matches))
			break
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
