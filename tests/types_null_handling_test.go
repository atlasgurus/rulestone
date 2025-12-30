package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file for null tests
func createNullTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-null-*.yaml")
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

// TestNullHandling_InComparisons tests null behavior in comparison operators
func TestNullHandling_InComparisons(t *testing.T) {
	rules := `
- metadata:
    id: null-equality
  expression: value == 0

- metadata:
    id: null-inequality
  expression: value != 0

- metadata:
    id: null-greater
  expression: value > 0

- metadata:
    id: null-less
  expression: value < 0
`

	ruleFile := createNullTestRuleFile(t, rules)
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
			name: "missing field treated as null",
			event: map[string]interface{}{
				"other": "data",
			},
			description: "Missing 'value' field should be treated as null",
		},
		{
			name: "explicit null field",
			event: map[string]interface{}{
				"value": nil,
			},
			description: "Explicit null value",
		},
		{
			name: "zero value vs null",
			event: map[string]interface{}{
				"value": 0,
			},
			description: "Zero should be different from null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches (documenting null comparison behavior)", tt.description, len(matches))
		})
	}
}

// TestNullHandling_InLogicalOperations tests null in AND/OR/NOT
func TestNullHandling_InLogicalOperations(t *testing.T) {
	rules := `
- metadata:
    id: null-and-true
  expression: value > 0 && other == "yes"

- metadata:
    id: null-or-true
  expression: value > 0 || other == "yes"

- metadata:
    id: null-and-false
  expression: value > 0 && other == "no"

- metadata:
    id: null-or-false
  expression: value > 0 || other == "no"
`

	ruleFile := createNullTestRuleFile(t, rules)
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
			name: "null AND true",
			event: map[string]interface{}{
				"other": "yes",
			},
			description: "Null value in AND with true condition",
		},
		{
			name: "null OR true",
			event: map[string]interface{}{
				"other": "yes",
			},
			description: "Null value in OR with true condition (OR should short-circuit)",
		},
		{
			name: "null AND false",
			event: map[string]interface{}{
				"other": "no",
			},
			description: "Null value in AND with false condition (AND should short-circuit)",
		},
		{
			name: "null OR false",
			event: map[string]interface{}{
				"other": "no",
			},
			description: "Null value in OR with false condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches", tt.description, len(matches))
		})
	}
}

// TestNullHandling_InArithmetic tests null in arithmetic operations
func TestNullHandling_InArithmetic(t *testing.T) {
	rules := `
- metadata:
    id: null-addition
  expression: value1 + value2 > 0

- metadata:
    id: null-multiplication
  expression: value1 * value2 > 0

- metadata:
    id: null-with-constant
  expression: value + 10 > 5
`

	ruleFile := createNullTestRuleFile(t, rules)
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
			name: "one null operand in addition",
			event: map[string]interface{}{
				"value2": 10,
			},
			description: "null + 10",
		},
		{
			name: "both null operands in addition",
			event: map[string]interface{}{},
			description: "null + null",
		},
		{
			name: "null in multiplication",
			event: map[string]interface{}{
				"value2": 5,
			},
			description: "null * 5",
		},
		{
			name: "null plus constant",
			event: map[string]interface{}{},
			description: "null + 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches (documenting null arithmetic behavior)", tt.description, len(matches))
		})
	}
}

// TestNullHandling_InQuantifiers tests null in forAll/forSome
func TestNullHandling_InQuantifiers(t *testing.T) {
	rules := `
- metadata:
    id: forsome-null-array
  expression: forSome("items", "item", item.value > 0)

- metadata:
    id: forall-null-array
  expression: forAll("items", "item", item.value < 100)

- metadata:
    id: forsome-with-null-elements
  expression: forSome("items", "item", item.value > 10)

- metadata:
    id: forall-with-null-elements
  expression: forAll("items", "item", item.value != 0)
`

	ruleFile := createNullTestRuleFile(t, rules)
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
			name: "null array field",
			event: map[string]interface{}{
				"other": "data",
			},
			description: "Missing items array (treated as null)",
		},
		{
			name: "explicit null array",
			event: map[string]interface{}{
				"items": nil,
			},
			description: "Explicit null items array",
		},
		{
			name: "empty array",
			event: map[string]interface{}{
				"items": []interface{}{},
			},
			description: "Empty array (forSome should be false, forAll should be true - vacuous truth)",
		},
		{
			name: "array with null elements",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"value": nil},
					map[string]interface{}{"value": 20},
				},
			},
			description: "Array containing elements with null values",
		},
		{
			name: "array with missing fields",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"other": "field"},
					map[string]interface{}{"value": 15},
				},
			},
			description: "Array elements with missing value field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches", tt.description, len(matches))
		})
	}
}

// TestNullHandling_InStringOperations tests null in string operations
func TestNullHandling_InStringOperations(t *testing.T) {
	rules := `
- metadata:
    id: string-equality
  expression: name == "test"

- metadata:
    id: string-containsany
  expression: containsAny(name, "test")

- metadata:
    id: regexp-match
  expression: regexpMatch("^[a-z]+@[a-z]+\\.[a-z]+$", email)
`

	ruleFile := createNullTestRuleFile(t, rules)
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
			name: "null in string equality",
			event: map[string]interface{}{
				"other": "field",
			},
			description: "Missing name field (null) compared to string",
		},
		{
			name: "null in containsany",
			event: map[string]interface{}{},
			description: "Null value in containsAny function",
		},
		{
			name: "null in regexp",
			event: map[string]interface{}{},
			description: "Null value in regexpMatch function",
		},
		{
			name: "empty string vs null",
			event: map[string]interface{}{
				"name":  "",
				"email": "",
			},
			description: "Empty string should be different from null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches", tt.description, len(matches))
		})
	}
}

// TestNullHandling_NestedFields tests null in nested object access
func TestNullHandling_NestedFields(t *testing.T) {
	rules := `
- metadata:
    id: nested-field-access
  expression: user.profile.name == "test"

- metadata:
    id: deep-nested-access
  expression: data.level1.level2.level3.value > 0

- metadata:
    id: nested-with-fallback
  expression: user.profile.age > 18 || user.guest == 1
`

	ruleFile := createNullTestRuleFile(t, rules)
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
			name: "missing parent object",
			event: map[string]interface{}{
				"other": "data",
			},
			description: "user.profile.name when user is missing",
		},
		{
			name: "missing intermediate object",
			event: map[string]interface{}{
				"user": map[string]interface{}{
					"id": 123,
				},
			},
			description: "user.profile.name when profile is missing",
		},
		{
			name: "missing leaf field",
			event: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"email": "test@example.com",
					},
				},
			},
			description: "user.profile.name when name is missing",
		},
		{
			name: "explicit null in path",
			event: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": nil,
				},
			},
			description: "user.profile.name when profile is explicit null",
		},
		{
			name: "deep nesting all missing",
			event: map[string]interface{}{},
			description: "data.level1.level2.level3.value when all missing",
		},
		{
			name: "OR with null on left",
			event: map[string]interface{}{
				"user": map[string]interface{}{
					"guest": 1,
				},
			},
			description: "Null age but guest=1 (OR should match)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches", tt.description, len(matches))
		})
	}
}

// TestNullHandling_MixedScenarios tests complex null scenarios
func TestNullHandling_MixedScenarios(t *testing.T) {
	rules := `
- metadata:
    id: complex-null-check
  expression: (value1 > 0 && value2 < 100) || value3 == "fallback"

- metadata:
    id: arithmetic-with-null-check
  expression: (price * quantity) > 100 && discount < 20

- metadata:
    id: quantifier-with-null-context
  expression: forSome("orders", "order", order.amount > total * 0.5)
`

	ruleFile := createNullTestRuleFile(t, rules)
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
			name: "partial data with fallback",
			event: map[string]interface{}{
				"value3": "fallback",
			},
			description: "Null value1 and value2, but fallback matches",
		},
		{
			name: "arithmetic with one null",
			event: map[string]interface{}{
				"price":    50,
				"discount": 10,
			},
			description: "price * null > 100 (quantity missing)",
		},
		{
			name: "quantifier with null context",
			event: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{"amount": 100},
				},
			},
			description: "forSome with null 'total' in condition",
		},
		{
			name: "all fields null",
			event: map[string]interface{}{},
			description: "All fields missing (all null)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches", tt.description, len(matches))
		})
	}
}

// TestNullHandling_TypeCoercion tests null with type conversions
func TestNullHandling_TypeCoercion(t *testing.T) {
	rules := `
- metadata:
    id: null-as-number
  expression: value + 0 == 0

- metadata:
    id: null-as-string
  expression: containsAny(text, "null")

- metadata:
    id: null-as-boolean
  expression: flag == 0
`

	ruleFile := createNullTestRuleFile(t, rules)
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
			name: "null coerced to number",
			event: map[string]interface{}{
				"other": "field",
			},
			description: "Does null + 0 == 0?",
		},
		{
			name: "null coerced to string",
			event: map[string]interface{}{
				"other": "field",
			},
			description: "Does null contain 'null' as string?",
		},
		{
			name: "null coerced to boolean",
			event: map[string]interface{}{
				"other": "field",
			},
			description: "Does null == 0 (false)?",
		},
		{
			name: "actual zero",
			event: map[string]interface{}{
				"value": 0,
				"text":  "contains null text",
				"flag":  0,
			},
			description: "Real zero values vs null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches (documenting type coercion behavior)", tt.description, len(matches))
		})
	}
}
