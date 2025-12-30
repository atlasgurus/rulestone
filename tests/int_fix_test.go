package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// TestIntToFloat64Fix verifies that int values now work correctly
func TestIntToFloat64Fix(t *testing.T) {
	rules := `- metadata:
    id: age-match
  expression: age == 25
`

	tmpfile, err := os.CreateTemp("", "rule-intfix-*.yaml")
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

	// Test with different numeric types
	testCases := []struct {
		name  string
		event map[string]interface{}
	}{
		{
			name:  "int type",
			event: map[string]interface{}{"age": 25}, // Go int
		},
		{
			name:  "int64 type",
			event: map[string]interface{}{"age": int64(25)}, // Go int64
		},
		{
			name:  "float64 type",
			event: map[string]interface{}{"age": float64(25)}, // Go float64
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

			if len(matches) != 1 {
				t.Errorf("%s: Expected 1 match, got %d", tc.name, len(matches))
			}
		})
	}
}
