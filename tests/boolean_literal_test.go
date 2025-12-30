package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// TestBooleanLiterals verifies that true/false/null literals work correctly
func TestBooleanLiterals(t *testing.T) {
	rules := `- metadata:
    id: true-literal
  expression: isActive == true

- metadata:
    id: false-literal
  expression: isDisabled == false

- metadata:
    id: null-literal
  expression: missingField == null

- metadata:
    id: true-comparison
  expression: status == "active" && verified == true
`

	tmpfile, err := os.CreateTemp("", "rule-bool-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(rules)); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	testCases := []struct {
		name      string
		event     map[string]interface{}
		expectIDs []string
	}{
		{
			name:      "true literal matches",
			event:     map[string]interface{}{"isActive": true},
			expectIDs: []string{"true-literal"},
		},
		{
			name:      "false literal matches",
			event:     map[string]interface{}{"isDisabled": false},
			expectIDs: []string{"false-literal"},
		},
		{
			name:      "null literal matches",
			event:     map[string]interface{}{"missingField": nil},
			expectIDs: []string{"null-literal"},
		},
		{
			name:      "complex with true literal",
			event:     map[string]interface{}{"status": "active", "verified": true},
			expectIDs: []string{"true-comparison"},
		},
		{
			name:      "true literal doesn't match false",
			event:     map[string]interface{}{"isActive": false},
			expectIDs: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := engine.NewRuleEngineRepo()
			_, err := repo.RegisterRulesFromFile(tmpfile.Name())
			if err != nil {
				t.Fatalf("Failed to load rules: %v", err)
			}

			ruleEngine, err := engine.NewRuleEngine(repo)
			if err != nil {
				t.Fatalf("Failed to create engine: %v", err)
			}

			matches := ruleEngine.MatchEvent(tc.event)

			matchedIDs := make([]string, 0)
			for _, ruleID := range matches {
				rule := ruleEngine.GetRuleDefinition(uint(ruleID))
				if rule != nil && rule.Metadata != nil {
					if id, ok := rule.Metadata["id"].(string); ok {
						matchedIDs = append(matchedIDs, id)
					}
				}
			}

			if len(matchedIDs) != len(tc.expectIDs) {
				t.Errorf("Expected %d matches %v, got %d matches %v",
					len(tc.expectIDs), tc.expectIDs, len(matchedIDs), matchedIDs)
			}
		})
	}
}
