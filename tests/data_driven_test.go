package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// TestDataDrivenRules discovers and runs all rule files in tests/data/ directory
func TestDataDrivenRules(t *testing.T) {
	// Find all YAML files in tests/data/ directory
	dataDir := "data"
	files, err := filepath.Glob(filepath.Join(dataDir, "*.yaml"))
	if err != nil {
		t.Fatalf("Failed to glob test files: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No YAML test files found in tests/data/")
	}

	// Run each file as a subtest
	for _, file := range files {
		// Extract filename without extension for test name
		baseName := filepath.Base(file)
		testName := strings.TrimSuffix(baseName, filepath.Ext(baseName))

		t.Run(testName, func(t *testing.T) {
			runDataDrivenTestFile(t, file)
		})
	}
}

// runDataDrivenTestFile loads a rule file and executes all its tests
func runDataDrivenTestFile(t *testing.T, filePath string) {
	t.Helper()

	// Create a fresh repository for this file
	repo := engine.NewRuleEngineRepo()

	// Load rules with validation and testing enabled
	result, err := repo.LoadRulesFromFile(filePath,
		engine.WithValidate(true),
		engine.WithRunTests(true),
		engine.WithFileFormat("yaml"),
	)

	if err != nil {
		t.Fatalf("Failed to load rules from %s: %v", filePath, err)
	}

	// Check validation
	if !result.ValidationOK {
		t.Errorf("Validation failed for %s", filePath)
		for _, verr := range result.Errors {
			t.Errorf("  Validation error: %v", verr)
		}
		return
	}

	// Get test summary
	summary := result.GetTestSummary()
	t.Logf("Test summary for %s: %s", filePath, summary.FormatTestSummary())

	// Report any failed tests
	failedTests := result.GetFailedTests()
	if len(failedTests) > 0 {
		t.Errorf("Found %d failed tests in %s:", len(failedTests), filePath)
		for _, ft := range failedTests {
			t.Errorf("  %s", ft.FormatTestResult())
		}
	}

	// Verify we have rules and tests
	if len(result.RuleIDs) == 0 {
		t.Errorf("No rules loaded from %s", filePath)
	}

	if len(result.TestResults) == 0 {
		t.Logf("Warning: No tests found in %s", filePath)
	}
}
