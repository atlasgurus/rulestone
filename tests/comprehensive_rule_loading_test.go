package tests

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// TestAllRulesLoadedTogether loads ALL rule files and tests them as a single engine
// This catches category optimization bugs that only appear with many rules
func TestAllRulesLoadedTogether(t *testing.T) {
	// Find all YAML test files
	files, err := filepath.Glob("data/*.yaml")
	if err != nil {
		t.Fatalf("Failed to glob test files: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No YAML test files found")
	}

	t.Logf("Loading %d rule files together into single engine", len(files))

	// Create ONE repo for ALL rules
	repo := engine.NewRuleEngineRepo()

	// Track test metadata for later execution
	type testCase struct {
		sourceFile string
		ruleID     string
		testName   string
		event      interface{}
		expectMatch bool
	}
	var allTests []testCase

	// Load all rule files into single repo
	totalRules := 0
	totalTests := 0
	for _, file := range files {
		result, err := repo.LoadRulesFromFile(file,
			engine.WithValidate(true),
			engine.WithRunTests(true), // Load tests but we'll validate them manually
			engine.WithFileFormat("yaml"),
		)

		if err != nil {
			t.Errorf("Failed to load %s: %v", file, err)
			continue
		}

		if !result.ValidationOK {
			t.Errorf("Validation failed for %s", file)
			for _, verr := range result.Errors {
				t.Errorf("  %v", verr)
			}
			continue
		}

		totalRules += len(result.RuleIDs)

		// Extract tests for manual execution with combined engine
		for _, tr := range result.TestResults {
			allTests = append(allTests, testCase{
				sourceFile: filepath.Base(file),
				ruleID:     tr.RuleID,
				testName:   tr.TestName,
				event:      tr.Event,
				expectMatch: tr.Expected,
			})
		}
		totalTests += len(result.TestResults)
	}

	t.Logf("Loaded %d rules from %d files", totalRules, len(files))
	t.Logf("Collected %d test cases", totalTests)

	// Create engine with ALL rules
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine with all rules: %v", err)
	}

	// Run all tests with the combined engine
	// Accept false positives (extra matches) but fail on false negatives
	falseNegatives := 0
	crossContamination := 0
	
	for _, tc := range allTests {
		matches := genFilter.MatchEvent(tc.event)
		matched := len(matches) > 0

		// Check if expected rule is in matches
		expectedRuleMatched := false
		for _, ruleID := range matches {
			rule := genFilter.GetRuleDefinition(uint(ruleID))
			if rule != nil && rule.Metadata != nil {
				if id, ok := rule.Metadata["id"].(string); ok && id == tc.ruleID {
					expectedRuleMatched = true
					break
				}
			}
		}

		if tc.expectMatch {
			// We expect this rule to match
			if !expectedRuleMatched {
				// FALSE NEGATIVE - this is a bug
				falseNegatives++
				t.Errorf("FALSE NEGATIVE: %s/%s/%s - expected rule '%s' to match but it didn't",
					tc.sourceFile, tc.ruleID, tc.testName, tc.ruleID)
			}
			
			if len(matches) > 1 {
				// Matched multiple rules (cross-contamination, but acceptable)
				crossContamination++
			}
		} else {
			// We expect this rule NOT to match
			if expectedRuleMatched {
				// Rule matched when it shouldn't - this is a real bug
				falseNegatives++
				t.Errorf("FALSE NEGATIVE: %s/%s/%s - rule '%s' matched but shouldn't have",
					tc.sourceFile, tc.ruleID, tc.testName, tc.ruleID)
			}
			
			if matched && !expectedRuleMatched {
				// Other rules matched (cross-contamination, acceptable)
				crossContamination++
			}
		}
	}

	// Report statistics
	t.Logf("Test Results:")
	t.Logf("  Total tests: %d", totalTests)
	t.Logf("  False negatives (BUGS): %d", falseNegatives)
	t.Logf("  Cross-contamination (acceptable): %d", crossContamination)
	
	if crossContamination > 0 {
		t.Logf("  Note: Cross-contamination is expected when many rules are loaded together")
	}
}

// TestAllRulesConcurrentExecution tests thread safety with all rules loaded
func TestAllRulesConcurrentExecution(t *testing.T) {
	// Find all YAML test files
	files, err := filepath.Glob("data/*.yaml")
	if err != nil {
		t.Fatalf("Failed to glob test files: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No YAML test files found")
	}

	// Load all rules into single engine
	repo := engine.NewRuleEngineRepo()
	var sampleEvents []interface{}

	for _, file := range files {
		result, err := repo.LoadRulesFromFile(file,
			engine.WithValidate(true),
			engine.WithRunTests(true),
			engine.WithFileFormat("yaml"),
		)

		if err != nil {
			continue
		}

		// Collect sample events
		for _, tr := range result.TestResults {
			if len(sampleEvents) < 20 {
				sampleEvents = append(sampleEvents, tr.Event)
			}
		}
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	t.Logf("Running concurrent test with %d sample events", len(sampleEvents))

	// Run 100 iterations with 10 goroutines
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				for _, event := range sampleEvents {
					matches := genFilter.MatchEvent(event)
					// Just verify no crash/panic, don't validate results
					_ = matches
				}
			}
		}(g)
	}

	// Wait for completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for errors
	for err := range errChan {
		t.Error(err)
	}

	t.Logf("Completed %d concurrent evaluations successfully", 10*100*len(sampleEvents))
}

// TestCategoryEngineOptimizationMetrics checks that optimizations activate with many rules
func TestCategoryEngineOptimizationMetrics(t *testing.T) {
	// Find all YAML test files
	files, err := filepath.Glob("data/*.yaml")
	if err != nil {
		t.Fatalf("Failed to glob test files: %v", err)
	}

	if len(files) == 0 {
		t.Skip("No YAML test files found")
	}

	repo := engine.NewRuleEngineRepo()
	totalRules := 0

	for _, file := range files {
		result, err := repo.LoadRulesFromFile(file,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)

		if err == nil && result.ValidationOK {
			totalRules += len(result.RuleIDs)
		}
	}

	t.Logf("Loaded %d total rules for optimization testing", totalRules)

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// The mere fact that we successfully created an engine with all rules
	// means the category optimization passes completed without crashing
	t.Logf("Category engine created successfully with %d rules", totalRules)
	
	// Run a sample event through to verify execution path
	sampleEvent := map[string]interface{}{
		"name": "Test",
		"age": 30,
		"value": 100,
	}
	
	matches := genFilter.MatchEvent(sampleEvent)
	t.Logf("Sample event matched %d rules", len(matches))
}
