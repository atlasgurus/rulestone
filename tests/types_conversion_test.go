package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file for type conversion tests
func createTypeConversionTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-typeconv-*.yaml")
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

// TestTypeConversion_StringToNumber tests string to number conversions
func TestTypeConversion_StringToNumber(t *testing.T) {
	rules := `
- metadata:
    id: string-to-int-equal
  expression: stringNum == 42

- metadata:
    id: string-to-int-compare
  expression: stringNum > 40

- metadata:
    id: string-to-float-equal
  expression: stringFloat == 3.14

- metadata:
    id: string-to-float-compare
  expression: stringFloat < 4.0

- metadata:
    id: string-numeric-add
  expression: stringNum + 8 == 50
`

	ruleFile := createTypeConversionTestRuleFile(t, rules)
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
			name: "string number equals integer",
			event: map[string]interface{}{
				"stringNum": "42",
			},
			expectMin:   2, // string-to-int-equal, string-to-int-compare
			expectMax:   2,
			description: "String '42' should convert and compare with integer 42",
		},
		{
			name: "string float equals float",
			event: map[string]interface{}{
				"stringFloat": "3.14",
			},
			expectMin:   2, // string-to-float-equal, string-to-float-compare
			expectMax:   2,
			description: "String '3.14' should convert and compare with float",
		},
		{
			name: "string number in arithmetic",
			event: map[string]interface{}{
				"stringNum": "42",
			},
			expectMin:   3, // Including arithmetic rule
			expectMax:   3,
			description: "String '42' should work in arithmetic expressions",
		},
		{
			name: "invalid string number",
			event: map[string]interface{}{
				"stringNum": "not_a_number",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Invalid numeric string should not match",
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

// TestTypeConversion_NumberToString tests number to string conversions
func TestTypeConversion_NumberToString(t *testing.T) {
	rules := `
- metadata:
    id: int-equals-string
  expression: age == "25"

- metadata:
    id: float-equals-string
  expression: price == "99.99"

- metadata:
    id: int-contains-check
  expression: status contains "200"
`

	ruleFile := createTypeConversionTestRuleFile(t, rules)
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
			name: "integer equals string",
			event: map[string]interface{}{
				"age": 25,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Integer 25 should equal string '25'",
		},
		{
			name: "float equals string",
			event: map[string]interface{}{
				"price": 99.99,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Float 99.99 should equal string '99.99'",
		},
		{
			name: "integer mismatch with string",
			event: map[string]interface{}{
				"age": 30,
			},
			expectMin:   0,
			expectMax:   0,
			description: "Integer 30 should not equal string '25'",
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

// TestTypeConversion_BooleanToNumeric tests boolean to numeric conversions
func TestTypeConversion_BooleanToNumeric(t *testing.T) {
	rules := `
- metadata:
    id: true-equals-one
  expression: isActive == 1

- metadata:
    id: false-equals-zero
  expression: isDisabled == 0

- metadata:
    id: bool-in-arithmetic
  expression: boolValue + 5 == 6

- metadata:
    id: bool-comparison
  expression: flag > 0
`

	ruleFile := createTypeConversionTestRuleFile(t, rules)
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
			name: "true converts to 1",
			event: map[string]interface{}{
				"isActive": true,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Boolean true should equal numeric 1",
		},
		{
			name: "false converts to 0",
			event: map[string]interface{}{
				"isDisabled": false,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Boolean false should equal numeric 0",
		},
		{
			name: "boolean in arithmetic (true)",
			event: map[string]interface{}{
				"boolValue": true,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Boolean true (1) + 5 should equal 6",
		},
		{
			name: "boolean in comparison",
			event: map[string]interface{}{
				"flag": true,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Boolean true (1) should be > 0",
		},
		{
			name: "false in comparison",
			event: map[string]interface{}{
				"flag": false,
			},
			expectMin:   0,
			expectMax:   0,
			description: "Boolean false (0) should not be > 0",
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

// TestTypeConversion_ImplicitConversions tests implicit type conversions in operators
func TestTypeConversion_ImplicitConversions(t *testing.T) {
	rules := `
- metadata:
    id: int-float-comparison
  expression: intVal < floatVal

- metadata:
    id: mixed-arithmetic
  expression: intVal + floatVal > 100.0

- metadata:
    id: string-int-equality
  expression: stringValue == intValue

- metadata:
    id: bool-numeric-mixed
  expression: boolVal + intVal == 11
`

	ruleFile := createTypeConversionTestRuleFile(t, rules)
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
			name: "int and float comparison",
			event: map[string]interface{}{
				"intVal":   50,
				"floatVal": 75.5,
			},
			expectMin:   2, // int-float-comparison, mixed-arithmetic
			expectMax:   2,
			description: "Int 50 < float 75.5, and 50 + 75.5 > 100",
		},
		{
			name: "string and int equality",
			event: map[string]interface{}{
				"stringValue": "42",
				"intValue":    42,
			},
			expectMin:   1,
			expectMax:   1,
			description: "String '42' should equal int 42",
		},
		{
			name: "boolean and int arithmetic",
			event: map[string]interface{}{
				"boolVal": true,
				"intVal":  10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Boolean true (1) + 10 should equal 11",
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

// TestTypeConversion_TypeReconciliationMatrix tests comprehensive type reconciliation
func TestTypeConversion_TypeReconciliationMatrix(t *testing.T) {
	rules := `
- metadata:
    id: int-int
  expression: a == 10

- metadata:
    id: int-float
  expression: a == 10.0

- metadata:
    id: float-float
  expression: b == 3.14

- metadata:
    id: string-string
  expression: c == "hello"

- metadata:
    id: bool-bool
  expression: d == true

- metadata:
    id: int-string
  expression: a == "10"

- metadata:
    id: float-string
  expression: b == "3.14"

- metadata:
    id: bool-int
  expression: d == 1
`

	ruleFile := createTypeConversionTestRuleFile(t, rules)
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
			name: "integer reconciliation",
			event: map[string]interface{}{
				"a": 10,
			},
			expectMin:   3, // int-int, int-float, int-string
			expectMax:   3,
			description: "Integer 10 should match int, float, and string comparisons",
		},
		{
			name: "float reconciliation",
			event: map[string]interface{}{
				"b": 3.14,
			},
			expectMin:   2, // float-float, float-string
			expectMax:   2,
			description: "Float 3.14 should match float and string comparisons",
		},
		{
			name: "string reconciliation",
			event: map[string]interface{}{
				"c": "hello",
			},
			expectMin:   1, // string-string
			expectMax:   1,
			description: "String 'hello' should match string comparison only",
		},
		{
			name: "boolean reconciliation",
			event: map[string]interface{}{
				"d": true,
			},
			expectMin:   2, // bool-bool, bool-int
			expectMax:   2,
			description: "Boolean true should match bool and int (1) comparisons",
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

// TestTypeConversion_PrecisionLoss tests float to int precision handling
func TestTypeConversion_PrecisionLoss(t *testing.T) {
	rules := `
- metadata:
    id: float-to-int-truncate
  expression: floatValue == 42

- metadata:
    id: float-to-int-round
  expression: roundValue == 10

- metadata:
    id: precise-float
  expression: preciseValue == 42.7
`

	ruleFile := createTypeConversionTestRuleFile(t, rules)
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
			name: "float with decimals vs int",
			event: map[string]interface{}{
				"floatValue": 42.7,
			},
			expectMin:   0,
			expectMax:   0,
			description: "42.7 should not equal 42 (no truncation)",
		},
		{
			name: "exact float vs int",
			event: map[string]interface{}{
				"floatValue": 42.0,
			},
			expectMin:   1,
			expectMax:   1,
			description: "42.0 should equal 42",
		},
		{
			name: "precise float match",
			event: map[string]interface{}{
				"preciseValue": 42.7,
			},
			expectMin:   1,
			expectMax:   1,
			description: "42.7 should equal 42.7 exactly",
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

// TestTypeConversion_InvalidConversions tests error handling for invalid conversions
func TestTypeConversion_InvalidConversions(t *testing.T) {
	rules := `
- metadata:
    id: invalid-string-to-number
  expression: badNumber > 10

- metadata:
    id: invalid-comparison
  expression: value == 42
`

	ruleFile := createTypeConversionTestRuleFile(t, rules)
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
			name: "non-numeric string in comparison",
			event: map[string]interface{}{
				"badNumber": "not_a_number",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Invalid numeric string should not match numeric comparison",
		},
		{
			name: "empty string in comparison",
			event: map[string]interface{}{
				"badNumber": "",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Empty string should not match numeric comparison",
		},
		{
			name: "complex object in comparison",
			event: map[string]interface{}{
				"value": map[string]interface{}{"nested": "value"},
			},
			expectMin:   0,
			expectMax:   0,
			description: "Complex object should not match simple value comparison",
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

// TestTypeConversion_MixedOperations tests type conversion in complex mixed operations
func TestTypeConversion_MixedOperations(t *testing.T) {
	rules := `
- metadata:
    id: mixed-arithmetic-comparison
  expression: (intVal + floatVal) / 2 > threshold

- metadata:
    id: bool-string-logic
  expression: (isActive == "1" || isDisabled == 0) && status == "active"

- metadata:
    id: complex-mixed
  expression: stringNum + intVal * 2 == 100
`

	ruleFile := createTypeConversionTestRuleFile(t, rules)
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
			name: "arithmetic with mixed types",
			event: map[string]interface{}{
				"intVal":    50,
				"floatVal":  70.0,
				"threshold": 50.0,
			},
			expectMin:   1,
			expectMax:   1,
			description: "(50 + 70.0) / 2 = 60.0 > 50.0",
		},
		{
			name: "boolean with string and numeric",
			event: map[string]interface{}{
				"isActive":   true,
				"isDisabled": false,
				"status":     "active",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Boolean conversions in complex logical expression",
		},
		{
			name: "string number in arithmetic",
			event: map[string]interface{}{
				"stringNum": "20",
				"intVal":    40,
			},
			expectMin:   1,
			expectMax:   1,
			description: "String '20' + (40 * 2) = 100",
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
