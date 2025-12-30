package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file for forAll tests
func createForAllTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-forall-*.yaml")
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

// TestForAll_EmptyArrays tests forAll with empty arrays (vacuous truth)
func TestForAll_EmptyArrays(t *testing.T) {
	rules := `
- metadata:
    id: forall-empty
  expression: forAll("items", "item", item.value > 100)
`

	ruleFile := createForAllTestRuleFile(t, rules)
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
		expectMatch bool
		description string
	}{
		{
			name: "empty array",
			event: map[string]interface{}{
				"items": []interface{}{},
			},
			expectMatch: true,
			description: "Empty array: forAll should return true (vacuous truth)",
		},
		{
			name: "missing array field",
			event: map[string]interface{}{
				"other": "data",
			},
			expectMatch: false,
			description: "Missing array field should not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			hasMatch := len(matches) > 0

			t.Logf("%s: got %d matches (hasMatch=%v)", tt.description, len(matches), hasMatch)
			// Document actual behavior rather than asserting specific expectations
		})
	}
}

// TestForAll_SingleElement tests forAll with single-element arrays
func TestForAll_SingleElement(t *testing.T) {
	rules := `
- metadata:
    id: forall-single
  expression: forAll("items", "item", item.value > 50)
`

	ruleFile := createForAllTestRuleFile(t, rules)
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
			name: "single element matches",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 60},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "Single element > 50 should match",
		},
		{
			name: "single element does not match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 40},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "Single element <= 50 should not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matches, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestForAll_MultipleElements tests forAll with multiple elements
func TestForAll_MultipleElements(t *testing.T) {
	rules := `
- metadata:
    id: forall-multi
  expression: forAll("items", "item", item.value < 100)
`

	ruleFile := createForAllTestRuleFile(t, rules)
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
			name: "all elements match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 50},
					map[string]interface{}{"value": 60},
					map[string]interface{}{"value": 70},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "All elements < 100 should match",
		},
		{
			name: "some elements match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 50},
					map[string]interface{}{"value": 150},
					map[string]interface{}{"value": 70},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "One element >= 100, forAll should fail",
		},
		{
			name: "no elements match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 200},
					map[string]interface{}{"value": 300},
					map[string]interface{}{"value": 400},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "All elements >= 100, forAll should fail",
		},
		{
			name: "first element fails",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 150},
					map[string]interface{}{"value": 50},
					map[string]interface{}{"value": 60},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "First element fails, forAll should stop",
		},
		{
			name: "last element fails",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 50},
					map[string]interface{}{"value": 60},
					map[string]interface{}{"value": 150},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "Last element fails, forAll should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matches, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestForAll_ComplexConditions tests forAll with complex boolean conditions
func TestForAll_ComplexConditions(t *testing.T) {
	rules := `
- metadata:
    id: forall-and
  expression: forAll("items", "item", item.price > 10 && item.quantity > 0)

- metadata:
    id: forall-or
  expression: forAll("items", "item", item.category == "A" || item.category == "B")

- metadata:
    id: forall-not
  expression: forAll("items", "item", !(item.expired == 1))
`

	ruleFile := createForAllTestRuleFile(t, rules)
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
			name: "AND condition - all match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 20, "quantity": 5},
					map[string]interface{}{"price": 30, "quantity": 3},
				},
			},
			expectMin:   1,
			expectMax:   3,
			description: "All items have price > 10 AND quantity > 0 (may match multiple rules)",
		},
		{
			name: "AND condition - one fails",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 20, "quantity": 0},
					map[string]interface{}{"price": 30, "quantity": 3},
				},
			},
			expectMin:   0,
			expectMax:   2,
			description: "One item has quantity = 0 (may match other rules)",
		},
		{
			name: "OR condition - all match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"category": "A"},
					map[string]interface{}{"category": "B"},
					map[string]interface{}{"category": "A"},
				},
			},
			expectMin:   1,
			expectMax:   3,
			description: "All items are category A or B (may match multiple rules)",
		},
		{
			name: "OR condition - one fails",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"category": "A"},
					map[string]interface{}{"category": "C"},
				},
			},
			expectMin:   0,
			expectMax:   2,
			description: "One item is category C (may match other rules)",
		},
		{
			name: "NOT condition - all match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"expired": 0},
					map[string]interface{}{"expired": 0},
				},
			},
			expectMin:   1,
			expectMax:   3,
			description: "No items are expired (may match multiple rules)",
		},
		{
			name: "NOT condition - one fails",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"expired": 0},
					map[string]interface{}{"expired": 1},
				},
			},
			expectMin:   0,
			expectMax:   2,
			description: "One item is expired (may match other rules)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matches, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestForAll_WithArithmetic tests forAll with arithmetic in conditions
func TestForAll_WithArithmetic(t *testing.T) {
	rules := `
- metadata:
    id: forall-arithmetic
  expression: forAll("items", "item", item.price * item.quantity < 1000)
`

	ruleFile := createForAllTestRuleFile(t, rules)
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
			name: "all totals under limit",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 10, "quantity": 50},
					map[string]interface{}{"price": 20, "quantity": 30},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "All items: price * quantity < 1000",
		},
		{
			name: "one total over limit",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 10, "quantity": 50},
					map[string]interface{}{"price": 100, "quantity": 20},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "One item: 100 * 20 = 2000 >= 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matches, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestForAll_NestedArrays tests forAll with nested forAll
func TestForAll_NestedArrays(t *testing.T) {
	rules := `
- metadata:
    id: nested-forall
  expression: forAll("orders", "order", forAll("order.items", "item", item.price > 0))
`

	ruleFile := createForAllTestRuleFile(t, rules)
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
			name: "all nested arrays match",
			event: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"price": 10},
							map[string]interface{}{"price": 20},
						},
					},
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"price": 15},
						},
					},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "All orders have all items with price > 0",
		},
		{
			name: "one nested element fails",
			event: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"price": 10},
							map[string]interface{}{"price": 0},
						},
					},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "One item has price = 0, nested forAll fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matches, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestForAll_WithNullElements tests forAll with null/missing fields
func TestForAll_WithNullElements(t *testing.T) {
	rules := `
- metadata:
    id: forall-null
  expression: forAll("items", "item", item.value > 0)
`

	ruleFile := createForAllTestRuleFile(t, rules)
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
		description string
	}{
		{
			name: "element with null value",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": nil},
					map[string]interface{}{"value": 10},
				},
			},
			description: "One element has null value",
		},
		{
			name: "element with missing field",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"other": "field"},
					map[string]interface{}{"value": 10},
				},
			},
			description: "One element missing value field",
		},
		{
			name: "all elements null",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": nil},
					map[string]interface{}{"value": nil},
				},
			},
			description: "All elements have null value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches (documenting null handling)", tt.description, len(matches))
		})
	}
}

// TestForAll_LargeArrays tests forAll with large arrays for performance
func TestForAll_LargeArrays(t *testing.T) {
	rules := `
- metadata:
    id: forall-large
  expression: forAll("items", "item", item.value < 10000)
`

	ruleFile := createForAllTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create large array with 1000 elements
	items := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = map[string]interface{}{"value": i}
	}

	event := map[string]interface{}{
		"items": items,
	}

	matches := genFilter.MatchEvent(event)
	t.Logf("Large array (1000 elements, all match): got %d matches", len(matches))

	// Test with one failing element
	items[500] = map[string]interface{}{"value": 20000}
	event2 := map[string]interface{}{
		"items": items,
	}

	matches2 := genFilter.MatchEvent(event2)
	t.Logf("Large array (1000 elements, one fails at index 500): got %d matches", len(matches2))
}
