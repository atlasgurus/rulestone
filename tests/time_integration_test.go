package tests_test

import (
	"testing"
	"time"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestTimeIntegration demonstrates that time.Time support works end-to-end
// This test verifies that the panic issue is fixed and time values can be processed
func TestTimeIntegration(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Test 1: Simple null check on time field (should not panic)
	yamlRule1 := `- metadata:
    id: 1
    name: Time Field Null Check
  expression: login_time != null
`

	result, err := repo.LoadRulesFromString(yamlRule1,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("time.Time value does not panic", func(t *testing.T) {
		event := map[string]interface{}{
			"login_time": time.Now(),
		}

		// This should not panic - that was the main bug
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("*time.Time pointer does not panic", func(t *testing.T) {
		now := time.Now()
		event := map[string]interface{}{
			"login_time": &now,
		}

		// This should not panic
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("nil *time.Time is treated as null", func(t *testing.T) {
		var nilTime *time.Time = nil
		event := map[string]interface{}{
			"login_time": nilTime,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("zero time.Time is not null", func(t *testing.T) {
		var zeroTime time.Time
		event := map[string]interface{}{
			"login_time": zeroTime,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})
}

// TestTimeWithMixedTypes verifies time fields work alongside other types
func TestTimeWithMixedTypes(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	yamlRule := `- metadata:
    id: 1
    name: Mixed Types
  expression: user_count > 5 && login_time != null && is_active == true
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("mixed types including time.Time", func(t *testing.T) {
		event := map[string]interface{}{
			"user_count":  10,
			"login_time":  time.Now(),
			"is_active":   true,
			"device_name": "iPhone",
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("mixed types with nil time fails", func(t *testing.T) {
		var nilTime *time.Time = nil
		event := map[string]interface{}{
			"user_count": 10,
			"login_time": nilTime,
			"is_active":  true,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})
}

// TestMultipleTimeFields verifies multiple time fields in the same event
func TestMultipleTimeFields(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	yamlRule := `- metadata:
    id: 1
    name: Multiple Time Fields
  expression: created_at != null && updated_at != null
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("multiple time fields", func(t *testing.T) {
		now := time.Now()
		event := map[string]interface{}{
			"created_at": now.Add(-24 * time.Hour),
			"updated_at": now,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("mixed time.Time and *time.Time", func(t *testing.T) {
		now := time.Now()
		created := now.Add(-24 * time.Hour)
		event := map[string]interface{}{
			"created_at": created, // time.Time
			"updated_at": &now,    // *time.Time
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})
}
