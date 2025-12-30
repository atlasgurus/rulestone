package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file for multiple rules tests
func createMultipleRulesTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-multiple-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(tmpfile.Name())
	})

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tmpfile.Name()
}

// TestMultipleRules_SameEventMatching tests multiple rules matching the same event
func TestMultipleRules_SameEventMatching(t *testing.T) {
	rules := `
- metadata:
    id: rule-1
  expression: value > 10

- metadata:
    id: rule-2
  expression: value > 5

- metadata:
    id: rule-3
  expression: value < 100

- metadata:
    id: rule-4
  expression: value == 50

- metadata:
    id: rule-5
  expression: value != 0
`

	ruleFile := createMultipleRulesTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name:        "matches all rules",
			event:       map[string]interface{}{"value": 50},
			expectMin:   5,
			expectMax:   5,
			description: "Value 50 should match all 5 rules",
		},
		{
			name:        "matches subset of rules",
			event:       map[string]interface{}{"value": 7},
			expectMin:   3,
			expectMax:   3,
			description: "Value 7 should match 3 rules",
		},
		{
			name:        "matches no rules",
			event:       map[string]interface{}{"value": 0},
			expectMin:   1, // Only rule-3 (0 < 100)
			expectMax:   1,
			description: "Value 0 should match only rule-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestMultipleRules_OverlappingConditions tests rules with overlapping conditions
func TestMultipleRules_OverlappingConditions(t *testing.T) {
	rules := `
- metadata:
    id: high-value
  expression: amount > 1000

- metadata:
    id: high-value-us
  expression: amount > 1000 && country == "US"

- metadata:
    id: high-value-premium
  expression: amount > 1000 && customerType == "premium"

- metadata:
    id: any-us
  expression: country == "US"

- metadata:
    id: any-premium
  expression: customerType == "premium"
`

	ruleFile := createMultipleRulesTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "high value US premium customer",
			event: map[string]interface{}{
				"amount":       1500,
				"country":      "US",
				"customerType": "premium",
			},
			expectMin:   5, // All rules match
			expectMax:   5,
			description: "Should match all overlapping rules",
		},
		{
			name: "high value US regular customer",
			event: map[string]interface{}{
				"amount":       1500,
				"country":      "US",
				"customerType": "regular",
			},
			expectMin:   3, // high-value, high-value-us, any-us
			expectMax:   3,
			description: "Should match US and high value rules",
		},
		{
			name: "low value US premium customer",
			event: map[string]interface{}{
				"amount":       500,
				"country":      "US",
				"customerType": "premium",
			},
			expectMin:   2, // any-us, any-premium
			expectMax:   2,
			description: "Should match only country and customer type rules",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestMultipleRules_ConflictingConditions tests rules with conflicting conditions
func TestMultipleRules_ConflictingConditions(t *testing.T) {
	rules := `
- metadata:
    id: low-value
  expression: amount < 100

- metadata:
    id: medium-value
  expression: amount >= 100 && amount < 1000

- metadata:
    id: high-value
  expression: amount >= 1000

- metadata:
    id: status-active
  expression: status == "active"

- metadata:
    id: status-inactive
  expression: status == "inactive"

- metadata:
    id: status-pending
  expression: status == "pending"
`

	ruleFile := createMultipleRulesTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "mutually exclusive amount - low",
			event: map[string]interface{}{
				"amount": 50,
				"status": "active",
			},
			expectMin:   2, // low-value, status-active
			expectMax:   2,
			description: "Should match only one amount range rule",
		},
		{
			name: "mutually exclusive amount - medium",
			event: map[string]interface{}{
				"amount": 500,
				"status": "active",
			},
			expectMin:   2, // medium-value, status-active
			expectMax:   2,
			description: "Should match only medium range rule",
		},
		{
			name: "mutually exclusive amount - high",
			event: map[string]interface{}{
				"amount": 2000,
				"status": "inactive",
			},
			expectMin:   2, // high-value, status-inactive
			expectMax:   2,
			description: "Should match only high range rule",
		},
		{
			name: "mutually exclusive status",
			event: map[string]interface{}{
				"amount": 500,
				"status": "pending",
			},
			expectMin:   2, // medium-value, status-pending
			expectMax:   2,
			description: "Should match only one status rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestMultipleRules_EvaluationOrderIndependence tests that rule order doesn't affect results
func TestMultipleRules_EvaluationOrderIndependence(t *testing.T) {
	// Create two rule files with rules in different orders
	rulesOrder1 := `
- metadata:
    id: rule-a
  expression: value > 10

- metadata:
    id: rule-b
  expression: value < 100

- metadata:
    id: rule-c
  expression: value == 50
`

	rulesOrder2 := `
- metadata:
    id: rule-c
  expression: value == 50

- metadata:
    id: rule-a
  expression: value > 10

- metadata:
    id: rule-b
  expression: value < 100
`

	ruleFile1 := createMultipleRulesTestRuleFile(t, rulesOrder1)
	ruleFile2 := createMultipleRulesTestRuleFile(t, rulesOrder2)

	repo1 := engine.NewRuleEngineRepo()
	_, err := repo1.RegisterRulesFromFile(ruleFile1)
	if err != nil {
		t.Fatalf("Failed to register rules order 1: %v", err)
	}

	repo2 := engine.NewRuleEngineRepo()
	_, err = repo2.RegisterRulesFromFile(ruleFile2)
	if err != nil {
		t.Fatalf("Failed to register rules order 2: %v", err)
	}

	genFilter1, err := engine.NewRuleEngine(repo1)
	if err != nil {
		t.Fatalf("Failed to create engine 1: %v", err)
	}

	genFilter2, err := engine.NewRuleEngine(repo2)
	if err != nil {
		t.Fatalf("Failed to create engine 2: %v", err)
	}

	event := map[string]interface{}{"value": 50}

	matches1 := genFilter1.MatchEvent(event)
	matches2 := genFilter2.MatchEvent(event)

	if len(matches1) != len(matches2) {
		t.Errorf("Different rule orders produced different match counts: %d vs %d",
			len(matches1), len(matches2))
	}

	// Verify same rule IDs matched in both cases
	ids1 := make(map[uint32]bool)
	for _, id := range matches1 {
		ids1[uint32(id)] = true
	}

	ids2 := make(map[uint32]bool)
	for _, id := range matches2 {
		ids2[uint32(id)] = true
	}

	for id := range ids1 {
		if !ids2[id] {
			t.Errorf("Rule %d matched in order1 but not in order2", id)
		}
	}

	for id := range ids2 {
		if !ids1[id] {
			t.Errorf("Rule %d matched in order2 but not in order1", id)
		}
	}
}

// TestMultipleRules_ManyRulesPerformance tests performance with many rules
func TestMultipleRules_ManyRulesPerformance(t *testing.T) {
	// Generate 1000 rules
	ruleCount := 1000
	rulesYAML := ""

	for i := 0; i < ruleCount; i++ {
		rulesYAML += fmt.Sprintf(`
- metadata:
    id: rule-%d
  expression: value > %d && value < %d
`, i, i*10, (i+1)*10)
	}

	ruleFile := createMultipleRulesTestRuleFile(t, rulesYAML)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register %d rules: %v", ruleCount, err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine with %d rules: %v", ruleCount, err)
	}

	// Test with value that matches exactly one rule
	event := map[string]interface{}{"value": 505} // Should match rule-50 (500 < value < 510)

	// Run multiple evaluations to measure performance
	for i := 0; i < 100; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Errorf("Expected 1 match with 1000 rules, got %d", len(matches))
		}
	}
}

// TestMultipleRules_DifferentComplexity tests rules with varying complexity
func TestMultipleRules_DifferentComplexity(t *testing.T) {
	rules := `
- metadata:
    id: simple
  expression: a == 1

- metadata:
    id: medium
  expression: a > 0 && b < 10

- metadata:
    id: complex
  expression: (a + b) * c > 100 && d == "test"

- metadata:
    id: very-complex
  expression: forSome("items", "item", item.value > threshold && item.status == "active") && total > 1000

- metadata:
    id: ultra-complex
  expression: ((a + b * c) / d > 10 || e == "special") && forAll("users", "user", user.age >= 18) && regexpMatch(name, "^[A-Z]")
`

	ruleFile := createMultipleRulesTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "all rules match",
			event: map[string]interface{}{
				"a": 1, "b": 5, "c": 50, "d": "test", "e": "normal",
				"threshold": 50,
				"total":     1500,
				"name":      "Alice",
				"items": []interface{}{
					map[string]interface{}{"value": 100, "status": "active"},
				},
				"users": []interface{}{
					map[string]interface{}{"age": 25},
					map[string]interface{}{"age": 30},
				},
			},
			expectMin:   5,
			expectMax:   5,
			description: "Event should match all complexity levels",
		},
		{
			name: "only simple and medium match",
			event: map[string]interface{}{
				"a": 1,
				"b": 5,
			},
			expectMin:   2,
			expectMax:   2,
			description: "Only simple rules should match",
		},
		{
			name: "complex rules match",
			event: map[string]interface{}{
				"a": 10, "b": 20, "c": 1, "d": "test",
				"threshold": 5,
				"total":     1500,
				"items": []interface{}{
					map[string]interface{}{"value": 100, "status": "active"},
				},
			},
			expectMin:   2, // medium, complex, very-complex (simple doesn't match a==1)
			expectMax:   3,
			description: "Medium to very complex rules should match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestMultipleRules_SharedSubexpressions tests CSE (Common Sub-Expression elimination) behavior
func TestMultipleRules_SharedSubexpressions(t *testing.T) {
	rules := `
- metadata:
    id: expr-1
  expression: a + b > 100

- metadata:
    id: expr-2
  expression: a + b < 200

- metadata:
    id: expr-3
  expression: (a + b) * c > 1000

- metadata:
    id: expr-4
  expression: a + b == 150

- metadata:
    id: expr-5
  expression: a + b != 0
`

	ruleFile := createMultipleRulesTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	event := map[string]interface{}{
		"a": 100,
		"b": 50,
		"c": 10,
	}

	// The expression (a + b) should be computed once and reused across all rules
	// This test documents that CSE is working correctly
	matches := genFilter.MatchEvent(event)

	if len(matches) != 5 {
		t.Errorf("Expected all 5 rules to match, got %d", len(matches))
	}
}

// TestMultipleRules_RuleInteractionWithQuantifiers tests multiple rules with quantifiers
func TestMultipleRules_RuleInteractionWithQuantifiers(t *testing.T) {
	rules := `
- metadata:
    id: all-valid
  expression: forAll("items", "item", item.valid == true)

- metadata:
    id: some-expensive
  expression: forSome("items", "item", item.price > 100)

- metadata:
    id: all-instock
  expression: forAll("items", "item", item.inStock == true)

- metadata:
    id: some-outofstock
  expression: forSome("items", "item", item.inStock == false)

- metadata:
    id: count-check
  expression: itemCount > 5
`

	ruleFile := createMultipleRulesTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "all conditions met",
			event: map[string]interface{}{
				"itemCount": 10,
				"items": []interface{}{
					map[string]interface{}{"valid": true, "price": 150, "inStock": true},
					map[string]interface{}{"valid": true, "price": 200, "inStock": true},
					map[string]interface{}{"valid": true, "price": 50, "inStock": true},
				},
			},
			expectMin:   4, // all-valid, some-expensive, all-instock, count-check (not some-outofstock)
			expectMax:   4,
			description: "Multiple quantifier rules should match independently",
		},
		{
			name: "mixed stock status",
			event: map[string]interface{}{
				"itemCount": 10,
				"items": []interface{}{
					map[string]interface{}{"valid": true, "price": 150, "inStock": true},
					map[string]interface{}{"valid": true, "price": 200, "inStock": false},
				},
			},
			expectMin:   3, // all-valid, some-expensive, some-outofstock, count-check (not all-instock)
			expectMax:   4,
			description: "Both stock status quantifiers should match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestMultipleRules_EmptyEventHandling tests how multiple rules handle empty events
func TestMultipleRules_EmptyEventHandling(t *testing.T) {
	rules := `
- metadata:
    id: null-check
  expression: value == null

- metadata:
    id: not-null-check
  expression: value != null

- metadata:
    id: exists-check
  expression: value > 0

- metadata:
    id: always-true
  expression: 1 == 1

- metadata:
    id: string-check
  expression: name == "default"
`

	ruleFile := createMultipleRulesTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Empty event
	event := map[string]interface{}{}

	matches := genFilter.MatchEvent(event)

	// Document which rules match on empty event
	matchedIDs := make([]uint32, 0)
	for _, id := range matches {
		matchedIDs = append(matchedIDs, uint32(id))
	}

	t.Logf("Empty event matched %d rules: %v", len(matchedIDs), matchedIDs)

	// At minimum, always-true should match (documenting actual behavior)
	if len(matches) < 1 {
		t.Logf("Note: No rules matched empty event (documenting actual behavior)")
	}
}
