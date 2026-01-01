package tests

import (
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// TestTestReporting demonstrates test result formatting and summary features
func TestTestReporting(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	result, err := repo.LoadRulesFromFile("data/comprehensive_tests.yaml", engine.LoadOptions{
		Validate:   true,
		RunTests:   true,
		FileFormat: "yaml",
	})

	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	if !result.ValidationOK {
		t.Errorf("Validation failed")
		for _, err := range result.Errors {
			t.Errorf("  Error: %v", err)
		}
	}

	// Test summary functionality
	summary := result.GetTestSummary()
	t.Logf("Test Summary: %s", summary.FormatTestSummary())

	if summary.Total == 0 {
		t.Errorf("Expected tests to run, but got 0")
	}

	// Log any failed tests
	failedTests := result.GetFailedTests()
	if len(failedTests) > 0 {
		t.Errorf("Found %d failed tests:", len(failedTests))
		for _, ft := range failedTests {
			t.Errorf("  %s", ft.FormatTestResult())
		}
	}

	// Verify all tests passed
	if summary.Failed > 0 || summary.Errors > 0 {
		t.Errorf("Expected all tests to pass, but %d failed and %d had errors", summary.Failed, summary.Errors)
	}

	// Verify we have the expected number of rules
	if len(result.RuleIDs) != 7 {
		t.Errorf("Expected 7 rules, got %d", len(result.RuleIDs))
	}
}

// TestTestReportingFailures demonstrates failure reporting with intentionally failing tests
func TestTestReportingFailures(t *testing.T) {
	// Create a rule with a test that will fail
	invalidRules := `
- metadata:
    id: test-failure-demo
  expression: a == 10
  tests:
    - name: should pass
      event:
        a: 10
      expect: true
    - name: should fail
      event:
        a: 10
      expect: false
    - name: should pass again
      event:
        a: 5
      expect: false
`

	repo := engine.NewRuleEngineRepo()
	result, err := repo.LoadRulesFromString(invalidRules, engine.LoadOptions{
		Validate:   true,
		RunTests:   true,
		FileFormat: "yaml",
	})

	if err != nil {
		t.Fatalf("Failed to load rules: %v", err)
	}

	summary := result.GetTestSummary()
	t.Logf("Test Summary: %s", summary.FormatTestSummary())

	// We expect 3 tests: 2 pass, 1 fail
	if summary.Total != 3 {
		t.Errorf("Expected 3 tests, got %d", summary.Total)
	}

	if summary.Passed != 2 {
		t.Errorf("Expected 2 passing tests, got %d", summary.Passed)
	}

	if summary.Failed != 1 {
		t.Errorf("Expected 1 failing test, got %d", summary.Failed)
	}

	// Get and log failed tests
	failedTests := result.GetFailedTests()
	if len(failedTests) != 1 {
		t.Errorf("Expected 1 failed test, got %d", len(failedTests))
	} else {
		t.Logf("Failed test: %s", failedTests[0].FormatTestResult())
	}
}

// TestFormattingWithError demonstrates error formatting
func TestFormattingWithError(t *testing.T) {
	// Create a rule that will cause an engine creation error
	invalidRules := `
- metadata:
    id: engine-error-demo
  expression: regexpMatch(x, "test")
  tests:
    - name: test case
      event:
        x: variable
      expect: true
`

	repo := engine.NewRuleEngineRepo()
	result, err := repo.LoadRulesFromString(invalidRules, engine.LoadOptions{
		Validate:   true,
		RunTests:   true,
		FileFormat: "yaml",
	})

	// This might fail during validation or test execution
	if err != nil {
		t.Logf("Expected error during load: %v", err)
		return
	}

	summary := result.GetTestSummary()
	t.Logf("Test Summary: %s", summary.FormatTestSummary())

	// If we got test results, check for errors
	if len(result.TestResults) > 0 {
		for _, tr := range result.TestResults {
			t.Logf("Test result: %s", tr.FormatTestResult())
		}
	}
}
