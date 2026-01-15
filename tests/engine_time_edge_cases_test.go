package tests_test

import (
	"testing"
	"time"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestMatchEvent_TimeEdgeCases tests edge cases for time.Time handling
func TestMatchEvent_TimeEdgeCases(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	yamlRule := `- metadata:
    id: 1
    name: Time Field Check
  expression: event_time != null
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("zero time.Time (default value)", func(t *testing.T) {
		var zeroTime time.Time
		event := map[string]interface{}{
			"event_time": zeroTime,
		}

		// Zero time is a valid time, not null
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("nil *time.Time pointer", func(t *testing.T) {
		var nilTime *time.Time = nil
		event := map[string]interface{}{
			"event_time": nilTime,
		}

		// Nil pointer should be treated as null
		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("Unix epoch", func(t *testing.T) {
		epoch := time.Unix(0, 0)
		event := map[string]interface{}{
			"event_time": epoch,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("far future time", func(t *testing.T) {
		farFuture := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
		event := map[string]interface{}{
			"event_time": farFuture,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("far past time", func(t *testing.T) {
		farPast := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
		event := map[string]interface{}{
			"event_time": farPast,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("nanosecond precision", func(t *testing.T) {
		nanoTime := time.Date(2024, 1, 15, 12, 0, 0, 123456789, time.UTC)
		event := map[string]interface{}{
			"event_time": nanoTime,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("time at exact midnight", func(t *testing.T) {
		midnight := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		event := map[string]interface{}{
			"event_time": midnight,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("time at exact noon", func(t *testing.T) {
		noon := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		event := map[string]interface{}{
			"event_time": noon,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("leap second boundary", func(t *testing.T) {
		// Test time near leap second boundary
		leapTime := time.Date(2016, 12, 31, 23, 59, 59, 999999999, time.UTC)
		event := map[string]interface{}{
			"event_time": leapTime,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("february 29 leap year", func(t *testing.T) {
		leapDay := time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC)
		event := map[string]interface{}{
			"event_time": leapDay,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("daylight saving time transition", func(t *testing.T) {
		// DST transition time in US (Spring forward: 2024-03-10 02:00:00 EST -> 03:00:00 EDT)
		loc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)
		dstTime := time.Date(2024, 3, 10, 2, 30, 0, 0, loc)
		event := map[string]interface{}{
			"event_time": dstTime,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})
}

// TestMatchEvent_TimeIsZeroCheck tests time.IsZero() behavior with comparisons
func TestMatchEvent_TimeIsZeroCheck(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Test that zero time behaves correctly in comparisons
	yamlRule := `- metadata:
    id: 1
    name: Non-null time check
  expression: event_time != null
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("non-zero time matches", func(t *testing.T) {
		now := time.Now()
		event := map[string]interface{}{
			"event_time": now,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("zero time matches (zero time is not null)", func(t *testing.T) {
		var zeroTime time.Time
		event := map[string]interface{}{
			"event_time": zeroTime,
		}

		// Zero time is a valid time value, not null
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("nil *time.Time does not match", func(t *testing.T) {
		var nilTime *time.Time = nil
		event := map[string]interface{}{
			"event_time": nilTime,
		}

		// Nil pointer is treated as null
		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("Unix epoch matches", func(t *testing.T) {
		epoch := time.Unix(0, 0)
		event := map[string]interface{}{
			"event_time": epoch,
		}

		// Unix epoch is a valid time, not null
		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})
}

// TestMatchEvent_MultipleTimeFields tests events with multiple time fields
func TestMatchEvent_MultipleTimeFields(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	yamlRule := `- metadata:
    id: 1
    name: Multiple Time Fields
  expression: created_at < updated_at && updated_at < accessed_at
`

	result, err := repo.LoadRulesFromString(yamlRule,
		engine.WithValidate(true),
		engine.WithFileFormat("yaml"),
	)
	require.NoError(t, err)
	require.True(t, result.ValidationOK)

	ruleEngine, err := engine.NewRuleEngine(repo)
	require.NoError(t, err)

	t.Run("ordered time fields match", func(t *testing.T) {
		now := time.Now()
		event := map[string]interface{}{
			"created_at":  now.Add(-2 * time.Hour),
			"updated_at":  now.Add(-1 * time.Hour),
			"accessed_at": now,
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("unordered time fields do not match", func(t *testing.T) {
		now := time.Now()
		event := map[string]interface{}{
			"created_at":  now,
			"updated_at":  now.Add(-1 * time.Hour),
			"accessed_at": now.Add(-2 * time.Hour),
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.NotContains(t, matchedIDs, condition.RuleIdType(0))
	})

	t.Run("mixed time.Time and *time.Time", func(t *testing.T) {
		now := time.Now()
		updated := now.Add(-1 * time.Hour)
		event := map[string]interface{}{
			"created_at":  now.Add(-2 * time.Hour), // time.Time
			"updated_at":  &updated,                 // *time.Time
			"accessed_at": now,                      // time.Time
		}

		matchedIDs := ruleEngine.MatchEvent(event)
		require.Contains(t, matchedIDs, condition.RuleIdType(0))
	})
}
