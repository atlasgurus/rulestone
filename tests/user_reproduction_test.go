package tests_test

import (
	"testing"
	"time"

	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRulestoneTimePanic reproduces the panic when rulestone processes time.Time types
// This is a minimal unit test without e2e machinery
func TestRulestoneTimePanic(t *testing.T) {
	// Create a simple rulestone engine with a basic rule
	repo := engine.NewRuleEngineRepo()

	// Simple rule: total_accounts_on_device >= 1
	yamlRule := `- metadata:
    id: 1
    name: Test Rule
  expression: total_accounts_on_device >= 1
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithRunTests(false),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)
	require.Empty(t, result.Errors)

	// Create engine
	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("Baseline - Safe types work fine", func(t *testing.T) {
		event := map[string]interface{}{
			"total_accounts_on_device": 5,
			"browser_name":             "Chrome",
			"is_new_device":            true,
			"score":                    100.5,
		}

		// Should not panic
		matchedIDs := ruleEngine.MatchEvent(event)
		assert.NotNil(t, matchedIDs)
		t.Log("✓ Safe types (int, string, bool, float) work fine")
	})

	t.Run("FIXED - time.Time value no longer causes panic", func(t *testing.T) {
		now := time.Now()
		event := map[string]interface{}{
			"total_accounts_on_device": 5,
			"device_last_seen":         now, // ← time.Time VALUE
		}

		// This should NOT panic anymore
		matchedIDs := ruleEngine.MatchEvent(event)
		assert.NotNil(t, matchedIDs)
		t.Log("✓ time.Time value works fine now")
	})

	t.Run("FIXED - *time.Time pointer no longer causes panic", func(t *testing.T) {
		now := time.Now()
		event := map[string]interface{}{
			"total_accounts_on_device": 5,
			"device_last_seen":         &now, // ← *time.Time POINTER
		}

		// This should NOT panic anymore
		matchedIDs := ruleEngine.MatchEvent(event)
		assert.NotNil(t, matchedIDs)
		t.Log("✓ *time.Time pointer works fine now")
	})

	t.Run("Workaround - Derived float64 field works", func(t *testing.T) {
		now := time.Now()
		daysSince := time.Since(now).Hours() / 24.0

		event := map[string]interface{}{
			"total_accounts_on_device":    5,
			"days_since_device_last_seen": daysSince, // ← Derived float64 works fine
		}

		// Should not panic
		matchedIDs := ruleEngine.MatchEvent(event)
		assert.NotNil(t, matchedIDs)
		t.Log("✓ Derived float64 field works fine")
	})

	t.Run("Edge case - nil pointer does not cause panic", func(t *testing.T) {
		var nilTime *time.Time = nil

		event := map[string]interface{}{
			"total_accounts_on_device": 5,
			"device_last_seen":         nilTime, // ← nil pointer
		}

		// Should not panic
		matchedIDs := ruleEngine.MatchEvent(event)
		assert.NotNil(t, matchedIDs)
		t.Log("✓ nil pointer is handled correctly")
	})
}
