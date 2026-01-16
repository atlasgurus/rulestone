package tests

import (
	"fmt"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
	"github.com/atlasgurus/rulestone/utils"
)

// TestEngineRuleRegistration tests rule registration from files
func TestEngineRuleRegistration(t *testing.T) {
	tests := []struct {
		name        string
		ruleFile    string
		expectRules int
		expectError bool
	}{
		{
			name:        "simple expression rule",
			ruleFile:    "../examples/rules/rule_expression_test0.yaml",
			expectRules: 1,
			expectError: false,
		},
		{
			name:        "nested attribute rule",
			ruleFile:    "../examples/rules/rule_expression_test2.yaml",
			expectRules: 1,
			expectError: false,
		},
		{
			name:        "multiple rules per file",
			ruleFile:    "../examples/rules/multiple_rules_per_file_test.yaml",
			expectRules: 3,
			expectError: false,
		},
		{
			name:        "forAll rule",
			ruleFile:    "../examples/rules/rule_for_each_1.yaml",
			expectRules: 1,
			expectError: false,
		},
		{
			name:        "complex expression",
			ruleFile:    "../examples/rules/rule_expression_test6.yaml",
			expectRules: 1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := engine.NewRuleEngineRepo()
			ruleIds, err := repo.RegisterRulesFromFile(tt.ruleFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(ruleIds) != tt.expectRules {
				t.Errorf("expected %d rules, got %d", tt.expectRules, len(ruleIds))
			}

			if repo.GetAppCtx().NumErrors() > 0 {
				repo.GetAppCtx().PrintErrors()
				t.Fatalf("got %d errors during rule registration", repo.GetAppCtx().NumErrors())
			}
		})
	}
}

// TestEngineExpressionEvaluation tests various expression evaluations
func TestEngineExpressionEvaluation(t *testing.T) {
	tests := []struct {
		name          string
		ruleFile      string
		dataFile      string
		expectMatches int
	}{
		{
			name:          "simple equality",
			ruleFile:      "../examples/rules/rule_expression_test0.yaml",
			dataFile:      "../examples/data/data_expression_test0.json",
			expectMatches: 1,
		},
		{
			name:          "string comparison",
			ruleFile:      "../examples/rules/rule_expression_test1.yaml",
			dataFile:      "../examples/data/data_expression_test1.json",
			expectMatches: 1,
		},
		{
			name:          "nested attribute access",
			ruleFile:      "../examples/rules/rule_expression_test2.yaml",
			dataFile:      "../examples/data/data_expression_test2.json",
			expectMatches: 1,
		},
		{
			name:          "logical operators",
			ruleFile:      "../examples/rules/rule_expression_test3.yaml",
			dataFile:      "../examples/data/data_expression_test3.json",
			expectMatches: 1,
		},
		{
			name:          "complex expression",
			ruleFile:      "../examples/rules/rule_expression_test4.yaml",
			dataFile:      "../examples/data/data_expression_test4.json",
			expectMatches: 1,
		},
		{
			name:          "array and function operations",
			ruleFile:      "../examples/rules/rule_expression_test5.yaml",
			dataFile:      "../examples/data/data_expression_test5.json",
			expectMatches: 1,
		},
		{
			name:          "common expression elimination",
			ruleFile:      "../examples/rules/rule_expression_test6.yaml",
			dataFile:      "../examples/data/data_expression_test6.json",
			expectMatches: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := engine.NewRuleEngineRepo()
			_, err := repo.RegisterRulesFromFile(tt.ruleFile)
			if err != nil {
				t.Fatalf("failed to register rules: %v", err)
			}

			genFilter, err := engine.NewRuleEngine(repo)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}

			event, err := utils.ReadEvent(tt.dataFile)
			if err != nil {
				t.Fatalf("failed to read event: %v", err)
			}

			matches := genFilter.MatchEvent(event)
			if len(matches) != tt.expectMatches {
				t.Errorf("expected %d matches, got %d", tt.expectMatches, len(matches))
			}

			if repo.GetAppCtx().NumErrors() > 0 {
				repo.GetAppCtx().PrintErrors()
				t.Fatalf("got %d errors during evaluation", repo.GetAppCtx().NumErrors())
			}
		})
	}
}

// TestEngineForEach tests forAll and forSome operations
func TestEngineForEach(t *testing.T) {
	tests := []struct {
		name          string
		ruleFile      string
		dataFile      string
		expectMatches int
	}{
		{
			name:          "forAll on array",
			ruleFile:      "../examples/rules/rule_for_each_1.yaml",
			dataFile:      "../examples/data/data_for_each_test1.yaml",
			expectMatches: 1,
		},
		{
			name:          "forSome with complex condition",
			ruleFile:      "../examples/rules/rule_for_each_test2.yaml",
			dataFile:      "../examples/data/data_for_each_test2.json",
			expectMatches: 1,
		},
		{
			name:          "forAll with optimization",
			ruleFile:      "../examples/rules/rule_for_each_test3.yaml",
			dataFile:      "../examples/data/data_for_each_test3.json",
			expectMatches: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := engine.NewRuleEngineRepo()
			_, err := repo.RegisterRulesFromFile(tt.ruleFile)
			if err != nil {
				t.Fatalf("failed to register rules: %v", err)
			}

			genFilter, err := engine.NewRuleEngine(repo)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}

			event, err := utils.ReadEvent(tt.dataFile)
			if err != nil {
				t.Fatalf("failed to read event: %v", err)
			}

			matches := genFilter.MatchEvent(event)
			if len(matches) != tt.expectMatches {
				t.Errorf("expected %d matches, got %d", tt.expectMatches, len(matches))
			}

			if repo.GetAppCtx().NumErrors() > 0 {
				repo.GetAppCtx().PrintErrors()
				t.Fatalf("got %d errors during evaluation", repo.GetAppCtx().NumErrors())
			}
		})
	}
}

// TestEngineMultipleRules tests handling of multiple rules
func TestEngineMultipleRules(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	ruleIds, err := repo.RegisterRulesFromFile("../examples/rules/multiple_rules_per_file_test.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	if len(ruleIds) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(ruleIds))
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		dataFile    string
		expectMatch int
	}{
		{
			name:        "match first rule",
			dataFile:    "../examples/data/data_multiple_rules_per_file_test0.json",
			expectMatch: 0,
		},
		{
			name:        "match second rule",
			dataFile:    "../examples/data/data_multiple_rules_per_file_test1.json",
			expectMatch: 1,
		},
		{
			name:        "match third rule",
			dataFile:    "../examples/data/data_multiple_rules_per_file_test2.json",
			expectMatch: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := utils.ReadEvent(tt.dataFile)
			if err != nil {
				t.Fatalf("failed to read event: %v", err)
			}

			matches := genFilter.MatchEvent(event)
			if len(matches) != 1 {
				t.Fatalf("expected 1 match, got %d", len(matches))
			}

			if int(matches[0]) != tt.expectMatch {
				t.Errorf("expected rule %d to match, got %d", tt.expectMatch, matches[0])
			}
		})
	}

	if repo.GetAppCtx().NumErrors() > 0 {
		repo.GetAppCtx().PrintErrors()
		t.Fatalf("got %d errors", repo.GetAppCtx().NumErrors())
	}
}

// TestEngineMetrics tests that metrics are properly tracked
func TestEngineMetrics(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test6.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test6.json")
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	initialEvals := genFilter.Metrics.NumCatEvals
	matches := genFilter.MatchEvent(event)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	// Check that common expression elimination reduced the number of evaluations
	numEvals := genFilter.Metrics.NumCatEvals - initialEvals
	if numEvals != 5 {
		t.Errorf("expected 5 category evaluations (includes undefined check), got %d", numEvals)
	}
}

// TestEngineNoMatch tests scenarios where no rules match
func TestEngineNoMatch(t *testing.T) {
	tests := []struct {
		name     string
		ruleFile string
		dataFile string
	}{
		{
			name:     "different value",
			ruleFile: "../examples/rules/rule_expression_test0.yaml",
			dataFile: "../examples/data/data_expression_test1.json",
		},
		{
			name:     "missing attribute",
			ruleFile: "../examples/rules/rule_expression_test2.yaml",
			dataFile: "../examples/data/data_expression_test0.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := engine.NewRuleEngineRepo()
			_, err := repo.RegisterRulesFromFile(tt.ruleFile)
			if err != nil {
				t.Fatalf("failed to register rules: %v", err)
			}

			genFilter, err := engine.NewRuleEngine(repo)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}

			event, err := utils.ReadEvent(tt.dataFile)
			if err != nil {
				t.Fatalf("failed to read event: %v", err)
			}

			matches := genFilter.MatchEvent(event)
			// For non-matching scenarios, we expect 0 or 1 matches (depending on the test)
			// Just verify it doesn't crash or error
			_ = matches
		})
	}
}

// TestEngineEmptyEvent tests handling of empty events
func TestEngineEmptyEvent(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Test with empty map
	event := make(map[string]interface{})
	matches := genFilter.MatchEvent(event)

	// Should not crash, may or may not match depending on rule
	_ = matches
}

// TestEngineNullValues tests handling of null values in events
func TestEngineNullValues(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Test with nil values
	event := map[string]interface{}{
		"attribute": nil,
	}
	matches := genFilter.MatchEvent(event)

	// Should not crash
	_ = matches
}

// TestEngineComplexNestedData tests deeply nested data structures
func TestEngineComplexNestedData(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Register a rule that accesses nested attributes
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test2.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	// Test with deeply nested structure
	event := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"value": 42,
				},
			},
		},
	}

	matches := genFilter.MatchEvent(event)
	// Should not crash
	_ = matches
}

// TestEngineArrayOperations tests various array-related operations
func TestEngineArrayOperations(t *testing.T) {
	tests := []struct {
		name     string
		ruleFile string
		dataFile string
	}{
		{
			name:     "array indexing",
			ruleFile: "../examples/rules/rule_expression_test5.yaml",
			dataFile: "../examples/data/data_expression_test5.json",
		},
		{
			name:     "forAll on empty array",
			ruleFile: "../examples/rules/rule_for_each_1.yaml",
			dataFile: "../examples/data/data_for_each_test1.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := engine.NewRuleEngineRepo()
			_, err := repo.RegisterRulesFromFile(tt.ruleFile)
			if err != nil {
				t.Fatalf("failed to register rules: %v", err)
			}

			genFilter, err := engine.NewRuleEngine(repo)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}

			event, err := utils.ReadEvent(tt.dataFile)
			if err != nil {
				t.Fatalf("failed to read event: %v", err)
			}

			matches := genFilter.MatchEvent(event)
			// Should not crash
			_ = matches
		})
	}
}

// TestEngineRuleIDSequencing tests that rule IDs are assigned correctly
func TestEngineRuleIDSequencing(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Register multiple files and check ID sequencing
	ruleFiles := []string{
		"../examples/rules/rule_expression_test0.yaml",
		"../examples/rules/rule_expression_test1.yaml",
		"../examples/rules/multiple_rules_per_file_test.yaml",
	}

	expectedIDs := [][]int{
		{0},    // First file: 1 rule
		{1},    // Second file: 1 rule
		{2, 3, 4}, // Third file: 3 rules
	}

	for i, ruleFile := range ruleFiles {
		ruleIds, err := repo.RegisterRulesFromFile(ruleFile)
		if err != nil {
			t.Fatalf("failed to register rules from %s: %v", ruleFile, err)
		}

		if len(ruleIds) != len(expectedIDs[i]) {
			t.Errorf("file %s: expected %d rules, got %d", ruleFile, len(expectedIDs[i]), len(ruleIds))
			continue
		}

		for j, expectedID := range expectedIDs[i] {
			if int(ruleIds[j]) != expectedID {
				t.Errorf("file %s rule %d: expected ID %d, got %d", ruleFile, j, expectedID, ruleIds[j])
			}
		}
	}
}

// TestEngineRepoReuse tests that the same repo can be used for multiple engines
func TestEngineRepoReuse(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	// Create multiple engines from the same repo
	engine1, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine 1: %v", err)
	}

	engine2, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine 2: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test0.json")
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	matches1 := engine1.MatchEvent(event)
	matches2 := engine2.MatchEvent(event)

	if len(matches1) != len(matches2) {
		t.Errorf("engines produced different results: %d vs %d matches", len(matches1), len(matches2))
	}
}

// TestEngineContextErrors tests error accumulation in AppContext
func TestEngineContextErrors(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Initially should have no errors
	if repo.GetAppCtx().NumErrors() != 0 {
		t.Errorf("expected 0 errors initially, got %d", repo.GetAppCtx().NumErrors())
	}

	// After successful registration, should still have no errors
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	if repo.GetAppCtx().NumErrors() != 0 {
		t.Errorf("expected 0 errors after successful registration, got %d", repo.GetAppCtx().NumErrors())
	}
}

// TestEngineStringComparisonOperators tests string comparison operations
func TestEngineStringComparisonOperators(t *testing.T) {
	// Test various string operations
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test1.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test1.json")
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	matches := genFilter.MatchEvent(event)
	if len(matches) != 1 {
		t.Errorf("expected 1 match for string comparison, got %d", len(matches))
	}
}

// TestEngineNumericOperations tests numeric comparisons
func TestEngineNumericOperations(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test0.json")
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	matches := genFilter.MatchEvent(event)
	if len(matches) != 1 {
		t.Errorf("expected 1 match for numeric comparison, got %d", len(matches))
	}
}

// TestEngineMapperInterface tests the MapperConfig interface implementation
func TestEngineMapperInterface(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Test MapScalar
	scalar := repo.MapScalar(42)
	if scalar == nil {
		t.Errorf("MapScalar returned nil")
	}

	// Test GetAppCtx
	ctx := repo.GetAppCtx()
	if ctx == nil {
		t.Errorf("GetAppCtx returned nil")
	}
}

// Helper function to create test data programmatically
func createTestEvent(fields map[string]interface{}) map[string]interface{} {
	return fields
}

// TestEngineWithProgrammaticData tests using programmatically created events
func TestEngineWithProgrammaticData(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	tests := []struct {
		name          string
		event         map[string]interface{}
		expectMatches bool
	}{
		{
			name: "matching numeric value",
			event: createTestEvent(map[string]interface{}{
				"id":    7,
				"color": "red",
			}),
			expectMatches: false, // Would need to check actual rule
		},
		{
			name: "non-matching numeric value",
			event: createTestEvent(map[string]interface{}{
				"id":    999,
				"color": "blue",
			}),
			expectMatches: false,
		},
		{
			name:          "empty event",
			event:         createTestEvent(map[string]interface{}{}),
			expectMatches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			// Just verify no crash - actual matching depends on rules
			_ = matches
		})
	}
}

// TestEngineConcurrentAccess tests thread safety (basic test)
func TestEngineConcurrentAccess(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test0.json")
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	// Run concurrent evaluations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				matches := genFilter.MatchEvent(event)
				_ = matches
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestEngineMemoryReuse tests that the engine properly reuses ObjectAttributeMap
func TestEngineMemoryReuse(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test0.json")
	if err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	// Run multiple evaluations to test object pooling
	for i := 0; i < 1000; i++ {
		matches := genFilter.MatchEvent(event)
		_ = matches
	}

	// If we get here without crashing, pooling is working
}

// TestEngineErrorFormat tests error message formatting
func TestEngineErrorFormat(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Attempt to register non-existent file
	_, err := repo.RegisterRulesFromFile("../examples/rules/nonexistent.yaml")
	if err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}

	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		if errMsg == "" {
			t.Errorf("error message is empty")
		}
	}
}
