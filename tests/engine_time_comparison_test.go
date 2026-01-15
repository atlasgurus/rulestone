package tests_test

import (
	"testing"
	"time"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestRuleEngine_TimeComparisons tests time comparison operations in rules
func TestRuleEngine_TimeComparisons(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Reference time: 2024-01-15 12:00:00 UTC
	refTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	before := refTime.Add(-1 * time.Hour)
	after := refTime.Add(1 * time.Hour)

	yamlRules := `
- metadata:
    id: 1
    name: After Reference Time
  expression: event_time > "2024-01-15T12:00:00Z"

- metadata:
    id: 2
    name: Before Reference Time
  expression: event_time < "2024-01-15T12:00:00Z"

- metadata:
    id: 3
    name: Equal Reference Time
  expression: event_time == "2024-01-15T12:00:00Z"

- metadata:
    id: 4
    name: Not Equal Reference Time
  expression: event_time != "2024-01-15T12:00:00Z"
`

	result, err := repo.LoadRulesFromString(yamlRules,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("time.Time after reference", func(t *testing.T) {
		event := map[string]interface{}{
			"event_time": after,
		}
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
		require.NotContains(t, matchedIDs, condition.RuleIdType(1))
		require.NotContains(t, matchedIDs, condition.RuleIdType(2))
		require.Contains(t, matchedIDs, condition.RuleIdType(3))
	})

	t.Run("time.Time before reference", func(t *testing.T) {
		event := map[string]interface{}{
			"event_time": before,
		}
		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
		require.Contains(t, matchedIDs, condition.RuleIdType(1))
		require.NotContains(t, matchedIDs, condition.RuleIdType(2))
		require.Contains(t, matchedIDs, condition.RuleIdType(3))
	})

	t.Run("time.Time equal reference", func(t *testing.T) {
		event := map[string]interface{}{
			"event_time": refTime,
		}
		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
		require.NotContains(t, matchedIDs, condition.RuleIdType(1))
		require.Contains(t, matchedIDs, condition.RuleIdType(2))
		require.NotContains(t, matchedIDs, condition.RuleIdType(3))
	})

	t.Run("*time.Time pointer comparisons", func(t *testing.T) {
		afterPtr := after
		event := map[string]interface{}{
			"event_time": &afterPtr,
		}
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("time.Time with nanosecond precision", func(t *testing.T) {
		// Time with nanoseconds
		nanoTime := time.Date(2024, 1, 15, 12, 0, 0, 123456789, time.UTC)
		event := map[string]interface{}{
			"event_time": nanoTime,
		}
		matchedIDs := ruleEngine.MatchEvent(event)
		// Should still be after refTime (which has 0 nanoseconds)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("time.Time greater than or equal", func(t *testing.T) {
		repo2 := engine.NewRuleEngineRepo()
		yamlRule := `- metadata:
    id: 1
    name: Greater or Equal
  expression: event_time >= "2024-01-15T12:00:00Z"
`
		result, err := repo2.LoadRulesFromString(yamlRule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine2, err := engine.NewRuleEngine(repo2)
		require.NoError(t, err)

		// Test equal time
		event1 := map[string]interface{}{"event_time": refTime}
		matchedIDs1 := ruleEngine2.MatchEvent(event1)
		require.Contains(t, matchedIDs1, condition.RuleIdType(0))

		// Test greater time
		event2 := map[string]interface{}{"event_time": after}
		matchedIDs2 := ruleEngine2.MatchEvent(event2)
		require.Contains(t, matchedIDs2, condition.RuleIdType(0))

		// Test lesser time
		event3 := map[string]interface{}{"event_time": before}
		matchedIDs3 := ruleEngine2.MatchEvent(event3)
		require.NotContains(t, matchedIDs3, condition.RuleIdType(0))
	})
}

// TestRuleEngine_TimezoneHandling tests timezone-aware time comparisons
func TestRuleEngine_TimezoneHandling(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Same instant in different timezones
	utcTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	estLoc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	estTime := utcTime.In(estLoc) // 07:00:00 EST (same instant)

	yamlRule := `- metadata:
    id: 1
    name: Same Instant Check
  expression: event_time == "2024-01-15T12:00:00Z"
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("UTC time matches", func(t *testing.T) {
		event := map[string]interface{}{"event_time": utcTime}
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("EST time matches same instant", func(t *testing.T) {
		event := map[string]interface{}{"event_time": estTime}
		matchedIDs := ruleEngine.MatchEvent(event)
		// Should match because time.Time.Equal() compares instants
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("Different instant in EST does not match", func(t *testing.T) {
		// Different instant: 2024-01-15 12:00:00 EST (which is 17:00:00 UTC)
		differentEstTime := time.Date(2024, 1, 15, 12, 0, 0, 0, estLoc)
		event := map[string]interface{}{"event_time": differentEstTime}
		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})
}

// TestRuleEngine_TimeArithmetic tests time arithmetic in rules
func TestRuleEngine_TimeArithmetic(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	yamlRule := `- metadata:
    id: 1
    name: Recent Event (within 1 day in seconds)
  expression: (now() - event_time) < 86400000000000
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("recent time matches", func(t *testing.T) {
		recentTime := time.Now().Add(-1 * time.Hour)
		event := map[string]interface{}{
			"event_time": recentTime,
		}
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("old time does not match", func(t *testing.T) {
		oldTime := time.Now().Add(-2 * 24 * time.Hour) // 2 days ago
		event := map[string]interface{}{
			"event_time": oldTime,
		}
		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})
}
