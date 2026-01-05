package tests

import (
	"strings"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// TestLoadRules_WithValidation tests loading rules with validation enabled
func TestLoadRules_WithValidation(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	result, err := repo.LoadRulesFromFile("data/simple_rules_with_tests.yaml",
		engine.WithValidate(true),
		engine.WithRunTests(true),
		engine.WithFileFormat("yaml"),
	)

	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	if !result.ValidationOK {
		t.Errorf("Validation failed")
		for _, err := range result.Errors {
			t.Errorf("  Error: %v", err)
		}
	}

	if len(result.RuleIDs) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(result.RuleIDs))
	}

	// Check test results
	expectedTests := 8 // 2 + 3 + 3 tests
	if len(result.TestResults) != expectedTests {
		t.Errorf("Expected %d test results, got %d", expectedTests, len(result.TestResults))
	}

	// All tests should pass
	for _, tr := range result.TestResults {
		if !tr.Passed {
			t.Errorf("Test failed: %s - %s (expected: %v, actual: %v)",
				tr.RuleID, tr.TestName, tr.Expected, tr.Actual)
		}
	}
}

// TestLoadRules_WithoutValidation tests loading rules without validation
func TestLoadRules_WithoutValidation(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	result, err := repo.LoadRulesFromFile("data/simple_rules_with_tests.yaml",
		engine.WithValidate(false),
		engine.WithRunTests(false),
		engine.WithFileFormat("yaml"),
	)

	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	if len(result.RuleIDs) != 3 {
		t.Errorf("Expected 3 rules, got %d", len(result.RuleIDs))
	}

	// No tests should have run
	if len(result.TestResults) != 0 {
		t.Errorf("Expected 0 test results, got %d", len(result.TestResults))
	}
}

// TestLoadRules_InvalidRule tests handling of invalid rules
func TestLoadRules_InvalidRule(t *testing.T) {
	// Create a rule with unmatched parentheses
	invalidRules := `
- metadata:
    id: invalid-rule
  expression: (a + b == 10
`

	repo := engine.NewRuleEngineRepo()
	result, err := repo.LoadRules(strings.NewReader(invalidRules),
		engine.WithValidate(true),
		engine.WithRunTests(false),
		engine.WithFileFormat("yaml"),
	)

	// Either parsing fails immediately (err != nil) or validation catches it
	if err != nil {
		t.Logf("Parsing failed as expected: %v", err)
		return
	}

	if result.ValidationOK {
		t.Errorf("Expected validation to fail for invalid rule")
	}

	if len(result.Errors) == 0 {
		t.Errorf("Expected validation errors")
	}
}
