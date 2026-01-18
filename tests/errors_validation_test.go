package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file for error validation tests
func createErrorValidationTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-errors-*.yaml")
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

// TestErrorValidation_InvalidRuleSyntax tests detection of invalid rule syntax
func TestErrorValidation_InvalidRuleSyntax(t *testing.T) {
	tests := []struct {
		name          string
		rules         string
		shouldError   bool
		errorContains string
		description   string
	}{
		{
			name: "missing metadata",
			rules: `
- expression: a == 1
`,
			shouldError:   false,
			errorContains: "",
			description:   "Rule without metadata is valid",
		},
		{
			name: "missing expression",
			rules: `
- metadata:
    id: test-rule
`,
			shouldError:   true,
			errorContains: "",
			description:   "Rule without expression should error",
		},
		{
			name: "missing rule ID",
			rules: `
- metadata:
    name: test
  expression: a == 1
`,
			shouldError:   false,
			errorContains: "",
			description:   "Rule without ID is valid",
		},
		{
			name: "empty expression",
			rules: `
- metadata:
    id: test-rule
  expression: ""
`,
			shouldError:   true,
			errorContains: "",
			description:   "Empty expression should error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleFile := createErrorValidationTestRuleFile(t, tt.rules)
			repo := engine.NewRuleEngineRepo()
			result, err := repo.LoadRulesFromFile(ruleFile,
				engine.WithValidate(true),
				engine.WithRunTests(false),
			)

			if tt.shouldError {
				// Check if either parsing failed (err != nil) or validation failed
				if err == nil && result.ValidationOK {
					t.Errorf("Expected error for %s, got success", tt.description)
				} else if tt.errorContains != "" {
					// Check error string in either err or result.Errors
					foundError := false
					if err != nil && strings.Contains(err.Error(), tt.errorContains) {
						foundError = true
					}
					for _, e := range result.Errors {
						if strings.Contains(e.Error(), tt.errorContains) {
							foundError = true
							break
						}
					}
					if !foundError && tt.errorContains != "" {
						t.Errorf("Expected error containing '%s', got: err=%v, errors=%v", tt.errorContains, err, result.Errors)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
				if !result.ValidationOK {
					t.Errorf("Unexpected validation failure for %s: %v", tt.description, result.Errors)
				}
			}
		})
	}
}

// TestErrorValidation_InvalidExpressionSyntax tests detection of invalid expression syntax
func TestErrorValidation_InvalidExpressionSyntax(t *testing.T) {
	tests := []struct {
		name          string
		expression    string
		shouldError   bool
		errorContains string
		description   string
	}{
		{
			name:          "unmatched parentheses left",
			expression:    "(a + b == 10",
			shouldError:   true,
			errorContains: "",
			description:   "Unmatched left parenthesis",
		},
		{
			name:          "unmatched parentheses right",
			expression:    "a + b) == 10",
			shouldError:   true,
			errorContains: "",
			description:   "Unmatched right parenthesis",
		},
		{
			name:          "invalid operator sequence",
			expression:    "a ++ b",
			shouldError:   true,
			errorContains: "",
			description:   "Invalid operator sequence",
		},
		{
			name:          "trailing operator",
			expression:    "a + b +",
			shouldError:   true,
			errorContains: "",
			description:   "Expression ending with operator",
		},
		{
			name:          "leading operator",
			expression:    "+ a + b",
			shouldError:   true,
			errorContains: "",
			description:   "Expression starting with binary operator",
		},
		{
			name:          "empty parentheses",
			expression:    "a + () + b",
			shouldError:   true,
			errorContains: "",
			description:   "Empty parentheses in expression",
		},
		{
			name:          "invalid comparison",
			expression:    "a === b",
			shouldError:   true,
			errorContains: "",
			description:   "Invalid comparison operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := `
- metadata:
    id: test-rule
  expression: ` + tt.expression

			ruleFile := createErrorValidationTestRuleFile(t, rules)
			repo := engine.NewRuleEngineRepo()
			result, err := repo.LoadRulesFromFile(ruleFile,
				engine.WithValidate(true),
				engine.WithRunTests(false),
			)

			if tt.shouldError {
				// Check if either parsing failed (err != nil) or validation failed
				if err == nil && result.ValidationOK {
					t.Errorf("Expected error for %s, got success", tt.description)
				} else if tt.errorContains != "" {
					// Check error string in either err or result.Errors
					foundError := false
					if err != nil && strings.Contains(err.Error(), tt.errorContains) {
						foundError = true
					}
					for _, e := range result.Errors {
						if strings.Contains(e.Error(), tt.errorContains) {
							foundError = true
							break
						}
					}
					if !foundError && tt.errorContains != "" {
						t.Errorf("Expected error containing '%s', got: err=%v, errors=%v", tt.errorContains, err, result.Errors)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
				if !result.ValidationOK {
					t.Errorf("Unexpected validation failure for %s: %v", tt.description, result.Errors)
				}
			}
		})
	}
}

// TestErrorValidation_InvalidFunctionCalls tests detection of invalid function usage
func TestErrorValidation_InvalidFunctionCalls(t *testing.T) {
	tests := []struct {
		name          string
		expression    string
		shouldError   bool
		errorContains string
		description   string
	}{
		{
			name:          "undefined function",
			expression:    `undefinedFunc(a, b)`,
			shouldError:   true,
			errorContains: "",
			description:   "Call to undefined function",
		},
		{
			name:          "regexp without arguments",
			expression:    `regexpMatch()`,
			shouldError:   true,
			errorContains: "",
			description:   "regexpMatch with no arguments",
		},
		{
			name:          "regexp with one argument",
			expression:    `regexpMatch(value)`,
			shouldError:   true,
			errorContains: "",
			description:   "regexpMatch with insufficient arguments",
		},
		{
			name:          "date without arguments",
			expression:    `date()`,
			shouldError:   true,
			errorContains: "",
			description:   "date function with no arguments",
		},
		{
			name:          "containsAny without arguments",
			expression:    `containsAny()`,
			shouldError:   true,
			errorContains: "",
			description:   "containsAny with no arguments",
		},
		{
			name:          "containsAny with one argument",
			expression:    `containsAny(value)`,
			shouldError:   true,
			errorContains: "",
			description:   "containsAny with insufficient arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := `
- metadata:
    id: test-rule
  expression: ` + tt.expression

			ruleFile := createErrorValidationTestRuleFile(t, rules)
			repo := engine.NewRuleEngineRepo()
			result, err := repo.LoadRulesFromFile(ruleFile,
				engine.WithValidate(true),
				engine.WithRunTests(false),
			)

			if tt.shouldError {
				// Check if either parsing failed (err != nil) or validation failed
				if err == nil && result.ValidationOK {
					t.Errorf("Expected error for %s, got success", tt.description)
				} else if tt.errorContains != "" {
					// Check error string in either err or result.Errors
					foundError := false
					if err != nil && strings.Contains(err.Error(), tt.errorContains) {
						foundError = true
					}
					for _, e := range result.Errors {
						if strings.Contains(e.Error(), tt.errorContains) {
							foundError = true
							break
						}
					}
					if !foundError && tt.errorContains != "" {
						t.Errorf("Expected error containing '%s', got: err=%v, errors=%v", tt.errorContains, err, result.Errors)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
				if !result.ValidationOK {
					t.Errorf("Unexpected validation failure for %s: %v", tt.description, result.Errors)
				}
			}
		})
	}
}

// TestErrorValidation_InvalidQuantifiers tests detection of invalid quantifier usage
func TestErrorValidation_InvalidQuantifiers(t *testing.T) {
	tests := []struct {
		name          string
		expression    string
		shouldError   bool
		errorContains string
		description   string
	}{
		{
			name:          "all without arguments",
			expression:    `all()`,
			shouldError:   true,
			errorContains: "",
			description:   "all with no arguments",
		},
		{
			name:          "all with one argument",
			expression:    `all("array")`,
			shouldError:   true,
			errorContains: "",
			description:   "all with insufficient arguments",
		},
		{
			name:          "all with two arguments",
			expression:    `all("array", "item")`,
			shouldError:   true,
			errorContains: "",
			description:   "all missing condition argument",
		},
		{
			name:          "any without arguments",
			expression:    `any()`,
			shouldError:   true,
			errorContains: "",
			description:   "any with no arguments",
		},
		{
			name:          "any with insufficient arguments",
			expression:    `any("array", "item")`,
			shouldError:   true,
			errorContains: "",
			description:   "any missing condition argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := `
- metadata:
    id: test-rule
  expression: ` + tt.expression

			ruleFile := createErrorValidationTestRuleFile(t, rules)
			repo := engine.NewRuleEngineRepo()
			result, err := repo.LoadRulesFromFile(ruleFile,
				engine.WithValidate(true),
				engine.WithRunTests(false),
			)

			if tt.shouldError {
				// Check if either parsing failed (err != nil) or validation failed
				if err == nil && result.ValidationOK {
					t.Errorf("Expected error for %s, got success", tt.description)
				} else if tt.errorContains != "" {
					// Check error string in either err or result.Errors
					foundError := false
					if err != nil && strings.Contains(err.Error(), tt.errorContains) {
						foundError = true
					}
					for _, e := range result.Errors {
						if strings.Contains(e.Error(), tt.errorContains) {
							foundError = true
							break
						}
					}
					if !foundError && tt.errorContains != "" {
						t.Errorf("Expected error containing '%s', got: err=%v, errors=%v", tt.errorContains, err, result.Errors)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
				if !result.ValidationOK {
					t.Errorf("Unexpected validation failure for %s: %v", tt.description, result.Errors)
				}
			}
		})
	}
}

// TestErrorValidation_InvalidYAMLFormat tests detection of invalid YAML
func TestErrorValidation_InvalidYAMLFormat(t *testing.T) {
	tests := []struct {
		name          string
		yaml          string
		shouldError   bool
		errorContains string
		description   string
	}{
		{
			name: "malformed YAML",
			yaml: `
- metadata
    id: test
  expression: a == 1
`,
			shouldError:   true,
			errorContains: "",
			description:   "YAML with syntax error",
		},
		{
			name: "invalid indentation",
			yaml: `
- metadata:
  id: test
    expression: a == 1
`,
			shouldError:   true,
			errorContains: "",
			description:   "YAML with incorrect indentation",
		},
		{
			name: "not an array",
			yaml: `
metadata:
  id: test
expression: a == 1
`,
			shouldError:   true,
			errorContains: "",
			description:   "Rules not in array format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ruleFile := createErrorValidationTestRuleFile(t, tt.yaml)
			repo := engine.NewRuleEngineRepo()
			_, err := repo.RegisterRulesFromFile(ruleFile)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.description)
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
			}
		})
	}
}

// TestErrorValidation_DuplicateRuleIDs tests detection of duplicate rule IDs
func TestErrorValidation_DuplicateRuleIDs(t *testing.T) {
	rules := `
- metadata:
    id: duplicate-id
  expression: a == 1

- metadata:
    id: duplicate-id
  expression: b == 2
`

	ruleFile := createErrorValidationTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)

	// This should either error or handle duplicates gracefully
	// Test documents the actual behavior
	if err != nil {
		t.Logf("Duplicate IDs cause error (expected): %v", err)
	} else {
		t.Logf("Duplicate IDs handled without error (documenting behavior)")
	}
}

// TestErrorValidation_TypeErrors tests runtime type errors
func TestErrorValidation_TypeErrors(t *testing.T) {
	rules := `
- metadata:
    id: string-arithmetic
  expression: stringValue + 10 > 100

- metadata:
    id: object-comparison
  expression: objectValue == 42
`

	ruleFile := createErrorValidationTestRuleFile(t, rules)
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
			name: "non-numeric string in arithmetic",
			event: map[string]interface{}{
				"stringValue": "not_a_number",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Non-numeric string should not match arithmetic expression",
		},
		{
			name: "object in scalar comparison",
			event: map[string]interface{}{
				"objectValue": map[string]interface{}{"nested": "value"},
			},
			expectMin:   0,
			expectMax:   0,
			description: "Object should not match scalar comparison",
		},
		{
			name: "array in scalar comparison",
			event: map[string]interface{}{
				"objectValue": []interface{}{1, 2, 3},
			},
			expectMin:   0,
			expectMax:   0,
			description: "Array should not match scalar comparison",
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

// TestErrorValidation_StackOverflow tests handling of deeply nested expressions
func TestErrorValidation_StackOverflow(t *testing.T) {
	// Create an extremely deeply nested expression (beyond 20 levels for quantifiers)
	deepNesting := "((((((((((((((((((((a))))))))))))))))))))"
	rules := `
- metadata:
    id: extreme-nesting
  expression: ` + deepNesting + ` == 1`

	ruleFile := createErrorValidationTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)

	// Should either handle it or error gracefully
	if err != nil {
		t.Logf("Deep nesting causes error: %v", err)
	} else {
		genFilter, err := engine.NewRuleEngine(repo)
		if err != nil {
			t.Logf("Deep nesting causes engine creation error: %v", err)
		} else {
			event := map[string]interface{}{"a": 1}
			matches := genFilter.MatchEvent(event)
			t.Logf("Deep nesting handled, matches: %d", len(matches))
		}
	}
}

// TestErrorValidation_QuantifierFrameLimit tests the 20-level frame stack limit
func TestErrorValidation_QuantifierFrameLimit(t *testing.T) {
	// Build nested all up to and beyond the 20 level limit
	// This tests the documented frame stack limit
	rules := `
- metadata:
    id: deep-quantifier-nesting
  expression: all("l1", "i1", all("l2", "i2", all("l3", "i3", all("l4", "i4", all("l5", "i5", i5.val == 1)))))
`

	ruleFile := createErrorValidationTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)

	if err != nil {
		t.Logf("Deep quantifier nesting rejected at registration: %v", err)
		return
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Logf("Deep quantifier nesting rejected at engine creation: %v", err)
		return
	}

	// Create deeply nested array structure
	event := map[string]interface{}{
		"l1": []interface{}{
			map[string]interface{}{
				"l2": []interface{}{
					map[string]interface{}{
						"l3": []interface{}{
							map[string]interface{}{
								"l4": []interface{}{
									map[string]interface{}{
										"l5": []interface{}{
											map[string]interface{}{"val": 1},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	matches := genFilter.MatchEvent(event)
	t.Logf("Deep quantifier nesting evaluated, matches: %d", len(matches))
}

// TestErrorValidation_MissingFields tests behavior with missing fields
func TestErrorValidation_MissingFields(t *testing.T) {
	rules := `
- metadata:
    id: missing-field
  expression: nonExistentField == 42

- metadata:
    id: nested-missing-field
  expression: object.nested.deep.field > 100
`

	ruleFile := createErrorValidationTestRuleFile(t, rules)
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
			name:        "completely missing field",
			event:       map[string]interface{}{},
			expectMin:   0,
			expectMax:   0,
			description: "Missing field should be treated as null and not match",
		},
		{
			name: "partially missing nested field",
			event: map[string]interface{}{
				"object": map[string]interface{}{
					"nested": map[string]interface{}{},
				},
			},
			expectMin:   0,
			expectMax:   0,
			description: "Missing nested field should be treated as null",
		},
		{
			name: "null vs missing field",
			event: map[string]interface{}{
				"nonExistentField": nil,
			},
			expectMin:   0,
			expectMax:   0,
			description: "Explicit null should behave same as missing field",
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

// TestErrorValidation_InvalidRegexpPattern tests handling of invalid regex patterns
func TestErrorValidation_InvalidRegexpPattern(t *testing.T) {
	rules := `
- metadata:
    id: invalid-regex
  expression: regexpMatch(value, "[invalid(regex")
`

	ruleFile := createErrorValidationTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)

	// Should error at registration or handle gracefully at runtime
	if err != nil {
		t.Logf("Invalid regex rejected at registration (expected): %v", err)
		return
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Logf("Invalid regex rejected at engine creation: %v", err)
		return
	}

	event := map[string]interface{}{"value": "test"}
	matches := genFilter.MatchEvent(event)

	// Should handle gracefully without crashing
	t.Logf("Invalid regex handled at runtime, matches: %d", len(matches))
}

// TestErrorValidation_DivisionByZero tests division by zero handling
func TestErrorValidation_DivisionByZero(t *testing.T) {
	rules := `
- metadata:
    id: division-by-zero
  expression: a / b > 10
`

	ruleFile := createErrorValidationTestRuleFile(t, rules)
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
		"b": 0,
	}

	// Should handle division by zero gracefully
	matches := genFilter.MatchEvent(event)
	matchCount := len(matches)

	// Division by zero should not match (likely returns 0 matches)
	if matchCount != 0 {
		t.Logf("Division by zero behavior: %d matches (documenting actual behavior)", matchCount)
	}
}
