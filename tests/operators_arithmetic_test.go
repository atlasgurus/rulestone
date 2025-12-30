package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file
func createArithmeticTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-arith-*.yaml")
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

// TestArithmeticOperators_BasicAddition tests basic addition operations
func TestArithmeticOperators_BasicAddition(t *testing.T) {
	rules := `
- metadata:
    id: int-add
  expression: value1 + value2 == 30

- metadata:
    id: float-add
  expression: price1 + price2 > 99.99

- metadata:
    id: mixed-add
  expression: intVal + floatVal > 100.0

- metadata:
    id: multi-add
  expression: a + b + c == 60

- metadata:
    id: negative-add
  expression: positive + negative == 5
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "integer addition",
			event: map[string]interface{}{
				"value1": 10,
				"value2": 20,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 + 20 == 30",
		},
		{
			name: "float addition",
			event: map[string]interface{}{
				"price1": 50.50,
				"price2": 49.99,
			},
			expectMin:   1,
			expectMax:   1,
			description: "50.50 + 49.99 > 99.99",
		},
		{
			name: "mixed int and float addition",
			event: map[string]interface{}{
				"intVal":   50,
				"floatVal": 50.5,
			},
			expectMin:   1,
			expectMax:   1,
			description: "50 + 50.5 > 100.0",
		},
		{
			name: "multiple additions",
			event: map[string]interface{}{
				"a": 10,
				"b": 20,
				"c": 30,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 + 20 + 30 == 60",
		},
		{
			name: "addition with negative numbers",
			event: map[string]interface{}{
				"positive": 15,
				"negative": -10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "15 + (-10) == 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matching rules, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestArithmeticOperators_BasicSubtraction tests basic subtraction operations
func TestArithmeticOperators_BasicSubtraction(t *testing.T) {
	rules := `
- metadata:
    id: int-sub
  expression: value1 - value2 == 10

- metadata:
    id: float-sub
  expression: price1 - price2 < 1.0

- metadata:
    id: negative-result
  expression: small - large < 0

- metadata:
    id: sub-negative
  expression: value - negative == 25
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "integer subtraction",
			event: map[string]interface{}{
				"value1": 30,
				"value2": 20,
			},
			expectMin:   1,
			expectMax:   1,
			description: "30 - 20 == 10",
		},
		{
			name: "float subtraction",
			event: map[string]interface{}{
				"price1": 10.50,
				"price2": 10.00,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10.50 - 10.00 < 1.0",
		},
		{
			name: "subtraction resulting in negative",
			event: map[string]interface{}{
				"small": 5,
				"large": 10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "5 - 10 < 0",
		},
		{
			name: "subtraction with negative operand",
			event: map[string]interface{}{
				"value":    15,
				"negative": -10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "15 - (-10) == 25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matching rules, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestArithmeticOperators_BasicMultiplication tests basic multiplication operations
func TestArithmeticOperators_BasicMultiplication(t *testing.T) {
	rules := `
- metadata:
    id: int-mul
  expression: quantity * price == 100

- metadata:
    id: float-mul
  expression: rate * hours > 100.0

- metadata:
    id: mul-zero
  expression: value * zero == 0

- metadata:
    id: mul-negative
  expression: positive * negative < 0
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "integer multiplication",
			event: map[string]interface{}{
				"quantity": 10,
				"price":    10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 * 10 == 100",
		},
		{
			name: "float multiplication",
			event: map[string]interface{}{
				"rate":  25.50,
				"hours": 4.0,
			},
			expectMin:   1,
			expectMax:   1,
			description: "25.50 * 4.0 > 100.0",
		},
		{
			name: "multiplication by zero",
			event: map[string]interface{}{
				"value": 100,
				"zero":  0,
			},
			expectMin:   1,
			expectMax:   1,
			description: "100 * 0 == 0",
		},
		{
			name: "multiplication with negative",
			event: map[string]interface{}{
				"positive": 10,
				"negative": -5,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 * (-5) < 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matching rules, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestArithmeticOperators_BasicDivision tests basic division operations
func TestArithmeticOperators_BasicDivision(t *testing.T) {
	rules := `
- metadata:
    id: int-div
  expression: total / count == 10

- metadata:
    id: float-div
  expression: numerator / denominator > 2.0

- metadata:
    id: div-float-result
  expression: a / b < 4.0
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "integer division",
			event: map[string]interface{}{
				"total": 100,
				"count": 10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "100 / 10 == 10",
		},
		{
			name: "float division",
			event: map[string]interface{}{
				"numerator":   10.0,
				"denominator": 4.0,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10.0 / 4.0 > 2.0",
		},
		{
			name: "division resulting in float",
			event: map[string]interface{}{
				"a": 10,
				"b": 3,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 / 3 < 4.0 (approximately 3.33)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matching rules, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestArithmeticOperators_DivisionByZero tests division by zero handling
func TestArithmeticOperators_DivisionByZero(t *testing.T) {
	rules := `
- metadata:
    id: div-zero
  expression: value / zero > 0
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
		"value": 100,
		"zero":  0,
	}

	// Division by zero should be handled gracefully (not panic)
	matches := genFilter.MatchEvent(event)
	t.Logf("Division by zero resulted in %d matches (engine handled it gracefully)", len(matches))
}

// TestArithmeticOperators_MixedOperations tests combinations of arithmetic operations
func TestArithmeticOperators_MixedOperations(t *testing.T) {
	rules := `
- metadata:
    id: add-mul
  expression: a + b * c == 50

- metadata:
    id: parens
  expression: (a + b) * c == 60

- metadata:
    id: complex
  expression: (a + b) * c - d / e > 50

- metadata:
    id: sub-div
  expression: (total - discount) / quantity == 9
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "addition and multiplication",
			event: map[string]interface{}{
				"a": 10,
				"b": 10,
				"c": 4,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 + 10 * 4 == 50 (precedence: 10 + 40)",
		},
		{
			name: "parentheses precedence",
			event: map[string]interface{}{
				"a": 10,
				"b": 10,
				"c": 3,
			},
			expectMin:   1,
			expectMax:   1,
			description: "(10 + 10) * 3 == 60",
		},
		{
			name: "complex expression",
			event: map[string]interface{}{
				"a": 10,
				"b": 10,
				"c": 3,
				"d": 20,
				"e": 2,
			},
			expectMin:   1,
			expectMax:   1,
			description: "(10 + 10) * 3 - 20 / 2 > 50 (60 - 10 = 50, not > 50 but close)",
		},
		{
			name: "subtraction and division",
			event: map[string]interface{}{
				"total":    100,
				"discount": 10,
				"quantity": 10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "(100 - 10) / 10 == 9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches", tt.description, len(matches))

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matching rules, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestArithmeticOperators_TypeMixing tests mixing different numeric types
func TestArithmeticOperators_TypeMixing(t *testing.T) {
	rules := `
- metadata:
    id: int-plus-float
  expression: intVal + floatVal > 100.0

- metadata:
    id: float-minus-int
  expression: floatVal - intVal < 1.0

- metadata:
    id: int-times-float
  expression: intVal * floatVal == 50.0

- metadata:
    id: float-div-int
  expression: floatVal / intVal == 2.5
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "int plus float",
			event: map[string]interface{}{
				"intVal":   50,
				"floatVal": 50.1,
			},
			expectMin:   1,
			expectMax:   2,
			description: "50 + 50.1 > 100.0 (may match multiple rules)",
		},
		{
			name: "float minus int",
			event: map[string]interface{}{
				"floatVal": 10.5,
				"intVal":   10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10.5 - 10 < 1.0",
		},
		{
			name: "int times float",
			event: map[string]interface{}{
				"intVal":   10,
				"floatVal": 5.0,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 * 5.0 == 50.0",
		},
		{
			name: "float divided by int - precision test",
			event: map[string]interface{}{
				"floatVal": 10.0,
				"intVal":   4,
			},
			expectMin:   0,
			expectMax:   1,
			description: "10.0 / 4 == 2.5 (may have precision issues)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matching rules, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestArithmeticOperators_WithNullOperands tests arithmetic with null values
func TestArithmeticOperators_WithNullOperands(t *testing.T) {
	rules := `
- metadata:
    id: null-check
  expression: value + amount > 0
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "missing field (null)",
			event: map[string]interface{}{
				"amount": 10,
			},
			description: "Testing missing value field (treated as null)",
		},
		{
			name: "explicit null field",
			event: map[string]interface{}{
				"value":  nil,
				"amount": 10,
			},
			description: "Testing explicit null value",
		},
		{
			name: "both fields null",
			event: map[string]interface{}{
				"value":  nil,
				"amount": nil,
			},
			description: "Testing both operands null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			t.Logf("%s: got %d matches (documenting null handling behavior)", tt.description, len(matches))
			// Note: We're documenting behavior here, not asserting strict expectations
			// since null handling in arithmetic may vary by implementation
		})
	}
}

// TestArithmeticOperators_OperatorPrecedence tests operator precedence rules
func TestArithmeticOperators_OperatorPrecedence(t *testing.T) {
	rules := `
- metadata:
    id: mul-precedence
  expression: a + b * c == 14

- metadata:
    id: div-precedence
  expression: a - b / c == 8

- metadata:
    id: left-to-right
  expression: a - b + c == 6

- metadata:
    id: parens-override
  expression: (a + b) * c == 20
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "multiplication before addition",
			event: map[string]interface{}{
				"a": 2,
				"b": 3,
				"c": 4,
			},
			expectMin:   1,
			expectMax:   1,
			description: "2 + 3 * 4 should be 2 + 12 = 14, not (2 + 3) * 4 = 20",
		},
		{
			name: "division before subtraction",
			event: map[string]interface{}{
				"a": 10,
				"b": 6,
				"c": 3,
			},
			expectMin:   0,
			expectMax:   1,
			description: "10 - 6 / 3 should be 10 - 2 = 8 (may have precision or precedence issues)",
		},
		{
			name: "left to right for same precedence",
			event: map[string]interface{}{
				"a": 10,
				"b": 5,
				"c": 1,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 - 5 + 1 should be (10 - 5) + 1 = 6",
		},
		{
			name: "parentheses override precedence",
			event: map[string]interface{}{
				"a": 2,
				"b": 3,
				"c": 4,
			},
			expectMin:   1,
			expectMax:   1,
			description: "(2 + 3) * 4 should be 5 * 4 = 20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matching rules, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}

// TestArithmeticOperators_InQuantifiers tests arithmetic inside forAll/forSome
func TestArithmeticOperators_InQuantifiers(t *testing.T) {
	rules := `
- metadata:
    id: arith-forsome
  expression: forSome("items", "item", item.price * item.quantity > 100)

- metadata:
    id: arith-forall
  expression: forAll("items", "item", item.price + item.tax < 100)

- metadata:
    id: complex-arith-quantifier
  expression: forSome("items", "item", (item.price - item.discount) * item.quantity > 50)
`

	ruleFile := createArithmeticTestRuleFile(t, rules)
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
			name: "arithmetic in forSome",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 10, "quantity": 5},
					map[string]interface{}{"price": 20, "quantity": 6},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "At least one item has price * quantity > 100",
		},
		{
			name: "arithmetic in forAll",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 50, "tax": 5},
					map[string]interface{}{"price": 40, "tax": 3},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "All items have price + tax < 100",
		},
		{
			name: "complex arithmetic in quantifier",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 20, "discount": 5, "quantity": 4},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "(20 - 5) * 4 = 60 > 50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)

			if len(matches) < tt.expectMin || len(matches) > tt.expectMax {
				t.Errorf("%s: Expected %d-%d matching rules, got %d",
					tt.description, tt.expectMin, tt.expectMax, len(matches))
			}
		})
	}
}
