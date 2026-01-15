package tests_test

import (
	"testing"
	"time"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestMatchEvent_TimeOperandCategories tests that time.Time fields in events trigger rule matches
func TestMatchEvent_TimeOperandCategories(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Rule that references a time field
	yamlRule := `- metadata:
    id: 1
    name: Recent Login Check
  expression: last_login != null
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("time.Time value triggers match", func(t *testing.T) {
		event := map[string]interface{}{
			"last_login": time.Now(),
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("*time.Time pointer triggers match", func(t *testing.T) {
		now := time.Now()
		event := map[string]interface{}{
			"last_login": &now,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("nil *time.Time does not match", func(t *testing.T) {
		var nilTime *time.Time = nil
		event := map[string]interface{}{
			"last_login": nilTime,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("zero time.Time triggers match", func(t *testing.T) {
		// Zero time is still a valid time value, not null
		var zeroTime time.Time
		event := map[string]interface{}{
			"last_login": zeroTime,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})
}

// TestMatchEvent_TimeWithOtherTypes tests mixed type events including time.Time
func TestMatchEvent_TimeWithOtherTypes(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Mixed type event
	yamlRule := `- metadata:
    id: 1
    name: Mixed Type Check
  expression: user_count > 5 && last_login != null && is_active == true
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
			"last_login":  time.Now(),
			"is_active":   true,
			"device_name": "iPhone 12",
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("mixed types with nil *time.Time", func(t *testing.T) {
		var nilTime *time.Time = nil
		event := map[string]interface{}{
			"user_count": 10,
			"last_login": nilTime,
			"is_active":  true,
		}

		// Should not match because last_login is null
		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("mixed types with zero time.Time", func(t *testing.T) {
		var zeroTime time.Time
		event := map[string]interface{}{
			"user_count": 10,
			"last_login": zeroTime,
			"is_active":  true,
		}

		// Should match because zero time != null
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})
}
