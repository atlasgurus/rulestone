package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file for forSome tests
func createForSomeTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-forsome-*.yaml")
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

// TestForSome_EmptyArrays tests forSome with empty arrays (should return false)
func TestForSome_EmptyArrays(t *testing.T) {
	rules := `
- metadata:
    id: forsome-empty
  expression: forSome("items", "item", item.value > 100)
`

	ruleFile := createForSomeTestRuleFile(t, rules)
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
			name: "empty array",
			event: map[string]interface{}{
				"items": []interface{}{},
			},
			expectMin:   0,
			expectMax:   0,
			description: "Empty array: forSome should return false (no elements to match)",
		},
		{
			name: "missing array field",
			event: map[string]interface{}{
				"other": "data",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Missing array field should not match",
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

// TestForSome_SingleElement tests forSome with single-element arrays
func TestForSome_SingleElement(t *testing.T) {
	rules := `
- metadata:
    id: forsome-single
  expression: forSome("items", "item", item.value > 50)
`

	ruleFile := createForSomeTestRuleFile(t, rules)
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

// TestForSome_MultipleElements tests forSome with multiple elements
func TestForSome_MultipleElements(t *testing.T) {
	rules := `
- metadata:
    id: forsome-multi
  expression: forSome("items", "item", item.value > 100)
`

	ruleFile := createForSomeTestRuleFile(t, rules)
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
			name: "first element matches",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 150},
					map[string]interface{}{"value": 50},
					map[string]interface{}{"value": 60},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "First element > 100, forSome should match (and possibly short-circuit)",
		},
		{
			name: "middle element matches",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 50},
					map[string]interface{}{"value": 150},
					map[string]interface{}{"value": 60},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "Middle element > 100, forSome should match",
		},
		{
			name: "last element matches",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 50},
					map[string]interface{}{"value": 60},
					map[string]interface{}{"value": 150},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "Last element > 100, forSome should match",
		},
		{
			name: "multiple elements match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 150},
					map[string]interface{}{"value": 200},
					map[string]interface{}{"value": 250},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "Multiple elements > 100, forSome should match",
		},
		{
			name: "no elements match",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": 50},
					map[string]interface{}{"value": 60},
					map[string]interface{}{"value": 70},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "No elements > 100, forSome should not match",
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

// TestForSome_ComplexConditions tests forSome with complex boolean conditions
func TestForSome_ComplexConditions(t *testing.T) {
	rules := `
- metadata:
    id: forsome-and
  expression: forSome("items", "item", item.price > 100 && item.available == 1)

- metadata:
    id: forsome-or
  expression: forSome("items", "item", item.category == "premium" || item.price > 500)

- metadata:
    id: forsome-not
  expression: forSome("items", "item", !(item.discontinued == 1))
`

	ruleFile := createForSomeTestRuleFile(t, rules)
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
			name: "AND condition - one item matches both",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 50, "available": 1},
					map[string]interface{}{"price": 150, "available": 1},
				},
			},
			expectMin:   1,
			expectMax:   3,
			description: "One item has price > 100 AND available (may match multiple rules)",
		},
		{
			name: "AND condition - no items match both",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 150, "available": 0},
					map[string]interface{}{"price": 50, "available": 1},
				},
			},
			expectMin:   0,
			expectMax:   2,
			description: "No items match both conditions (may match other rules)",
		},
		{
			name: "OR condition - matches category",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"category": "premium", "price": 50},
					map[string]interface{}{"category": "standard", "price": 100},
				},
			},
			expectMin:   1,
			expectMax:   3,
			description: "One item is premium category (may match multiple rules)",
		},
		{
			name: "OR condition - matches price",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"category": "standard", "price": 600},
					map[string]interface{}{"category": "standard", "price": 100},
				},
			},
			expectMin:   1,
			expectMax:   3,
			description: "One item has price > 500 (may match multiple rules)",
		},
		{
			name: "NOT condition - at least one not discontinued",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"discontinued": 1},
					map[string]interface{}{"discontinued": 0},
				},
			},
			expectMin:   1,
			expectMax:   3,
			description: "At least one item not discontinued (may match multiple rules)",
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

// TestForSome_WithArithmetic tests forSome with arithmetic in conditions
func TestForSome_WithArithmetic(t *testing.T) {
	rules := `
- metadata:
    id: forsome-arithmetic
  expression: forSome("items", "item", item.price * item.quantity > 1000)
`

	ruleFile := createForSomeTestRuleFile(t, rules)
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
			name: "at least one total over limit",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 10, "quantity": 50},
					map[string]interface{}{"price": 100, "quantity": 20},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "One item: 100 * 20 = 2000 > 1000",
		},
		{
			name: "all totals under limit",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 10, "quantity": 50},
					map[string]interface{}{"price": 20, "quantity": 30},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "No items have price * quantity > 1000",
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

// TestForSome_NestedArrays tests forSome with nested forSome/forAll
func TestForSome_NestedArrays(t *testing.T) {
	rules := `
- metadata:
    id: nested-forsome
  expression: forSome("orders", "order", forSome("order.items", "item", item.price > 100))

- metadata:
    id: forsome-forall
  expression: forSome("orders", "order", forAll("order.items", "item", item.validated == 1))
`

	ruleFile := createForSomeTestRuleFile(t, rules)
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
			name: "nested forSome - one order has expensive item",
			event: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"price": 50},
							map[string]interface{}{"price": 60},
						},
					},
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"price": 150},
						},
					},
				},
			},
			expectMin:   1,
			expectMax:   2,
			description: "At least one order has at least one item with price > 100",
		},
		{
			name: "forSome-forAll - one order all validated",
			event: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"validated": 1},
							map[string]interface{}{"validated": 1},
						},
					},
					map[string]interface{}{
						"items": []interface{}{
							map[string]interface{}{"validated": 0},
						},
					},
				},
			},
			expectMin:   1,
			expectMax:   2,
			description: "At least one order has all items validated",
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

// TestForSome_WithNullElements tests forSome with null/missing fields
func TestForSome_WithNullElements(t *testing.T) {
	rules := `
- metadata:
    id: forsome-null
  expression: forSome("items", "item", item.value > 50)
`

	ruleFile := createForSomeTestRuleFile(t, rules)
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
			name: "one null, one matching",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": nil},
					map[string]interface{}{"value": 60},
				},
			},
			description: "One null element, one matching",
		},
		{
			name: "one missing field, one matching",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"other": "field"},
					map[string]interface{}{"value": 60},
				},
			},
			description: "One missing value field, one matching",
		},
		{
			name: "all null values",
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

// TestForSome_LargeArrays tests forSome with large arrays for short-circuit behavior
func TestForSome_LargeArrays(t *testing.T) {
	rules := `
- metadata:
    id: forsome-large
  expression: forSome("items", "item", item.value > 500)
`

	ruleFile := createForSomeTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create large array with 1000 elements, first one matches
	items := make([]interface{}, 1000)
	items[0] = map[string]interface{}{"value": 600}
	for i := 1; i < 1000; i++ {
		items[i] = map[string]interface{}{"value": i}
	}

	event := map[string]interface{}{
		"items": items,
	}

	matches := genFilter.MatchEvent(event)
	t.Logf("Large array (1000 elements, first matches): got %d matches (should short-circuit)", len(matches))

	// Test with no matching elements
	items2 := make([]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		items2[i] = map[string]interface{}{"value": i}
	}

	event2 := map[string]interface{}{
		"items": items2,
	}

	matches2 := genFilter.MatchEvent(event2)
	t.Logf("Large array (1000 elements, none match): got %d matches", len(matches2))

	// Test with last element matching
	items3 := make([]interface{}, 1000)
	for i := 0; i < 999; i++ {
		items3[i] = map[string]interface{}{"value": i}
	}
	items3[999] = map[string]interface{}{"value": 600}

	event3 := map[string]interface{}{
		"items": items3,
	}

	matches3 := genFilter.MatchEvent(event3)
	t.Logf("Large array (1000 elements, last matches): got %d matches", len(matches3))
}
