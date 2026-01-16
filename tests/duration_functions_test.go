package tests_test

import (
	"testing"
	"time"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestDurationFunctions tests the days, hours, minutes, and seconds duration functions
func TestDurationFunctions(t *testing.T) {
	t.Run("days function with time subtraction", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: created_at > now() - days(5)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Test with recent timestamp (3 days ago)
		threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour)
		event := map[string]interface{}{
			"created_at": threeDaysAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "3 days ago should be within 5 days")

		// Test with old timestamp (7 days ago)
		sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
		event2 := map[string]interface{}{
			"created_at": sevenDaysAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "7 days ago should not be within 5 days")
	})

	t.Run("hours function with time subtraction", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: session_start > now() - hours(2)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Recent session (1 hour ago)
		oneHourAgo := time.Now().Add(-1 * time.Hour)
		event := map[string]interface{}{
			"session_start": oneHourAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "1 hour ago should be within 2 hours")

		// Old session (3 hours ago)
		threeHoursAgo := time.Now().Add(-3 * time.Hour)
		event2 := map[string]interface{}{
			"session_start": threeHoursAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "3 hours ago should not be within 2 hours")
	})

	t.Run("minutes function with time subtraction", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: event_time >= now() - minutes(30)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Recent event (15 minutes ago)
		fifteenMinutesAgo := time.Now().Add(-15 * time.Minute)
		event := map[string]interface{}{
			"event_time": fifteenMinutesAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "15 minutes ago should be within 30 minutes")

		// Old event (45 minutes ago)
		fortyFiveMinutesAgo := time.Now().Add(-45 * time.Minute)
		event2 := map[string]interface{}{
			"event_time": fortyFiveMinutesAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "45 minutes ago should not be within 30 minutes")
	})

	t.Run("seconds function with time subtraction", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: timestamp < now() - seconds(10)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Old timestamp (30 seconds ago)
		thirtySecondsAgo := time.Now().Add(-30 * time.Second)
		event := map[string]interface{}{
			"timestamp": thirtySecondsAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "30 seconds ago is older than 10 seconds")

		// Recent timestamp (5 seconds ago)
		fiveSecondsAgo := time.Now().Add(-5 * time.Second)
		event2 := map[string]interface{}{
			"timestamp": fiveSecondsAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "5 seconds ago should not be older than 10 seconds")
	})

	t.Run("compound duration - days and hours", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: deadline <= now() + days(1) + hours(12)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Deadline in 1 day (within 1.5 days)
		oneDayFromNow := time.Now().Add(24 * time.Hour)
		event := map[string]interface{}{
			"deadline": oneDayFromNow,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "1 day from now is within 1.5 days")

		// Deadline in 2 days (beyond 1.5 days)
		twoDaysFromNow := time.Now().Add(48 * time.Hour)
		event2 := map[string]interface{}{
			"deadline": twoDaysFromNow,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "2 days from now exceeds 1.5 days")
	})

	t.Run("duration with time difference", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: (now() - last_login) < days(30)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Login 15 days ago
		fifteenDaysAgo := time.Now().Add(-15 * 24 * time.Hour)
		event := map[string]interface{}{
			"last_login": fifteenDaysAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "15 days ago is within 30 days")

		// Login 45 days ago
		fortyFiveDaysAgo := time.Now().Add(-45 * 24 * time.Hour)
		event2 := map[string]interface{}{
			"last_login": fortyFiveDaysAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "45 days ago exceeds 30 days")
	})

	t.Run("fractional days", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: event_time > now() - days(0.5)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// 6 hours ago (0.25 days)
		sixHoursAgo := time.Now().Add(-6 * time.Hour)
		event := map[string]interface{}{
			"event_time": sixHoursAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "6 hours ago is within 0.5 days (12 hours)")

		// 18 hours ago (0.75 days)
		eighteenHoursAgo := time.Now().Add(-18 * time.Hour)
		event2 := map[string]interface{}{
			"event_time": eighteenHoursAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "18 hours ago exceeds 0.5 days")
	})
}

func TestDurationFunctionValidation(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	t.Run("days requires one argument", func(t *testing.T) {
		rule := `- metadata: {id: 1}
  expression: days()`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.False(t, result.ValidationOK, "Should fail validation")
		require.NotEmpty(t, result.Errors)
	})


	t.Run("hours requires numeric argument", func(t *testing.T) {
		rule := `- metadata: {id: 1}
  expression: timestamp > now() - hours("not a number")`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.False(t, result.ValidationOK, "Should fail validation - non-numeric argument")
	})
}

func TestDurationFunctionEdgeCases(t *testing.T) {
	t.Run("zero duration", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: event_time > now() - days(0)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Future time
		future := time.Now().Add(1 * time.Hour)
		event := map[string]interface{}{
			"event_time": future,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "Future time is after now")

		// Past time
		past := time.Now().Add(-1 * time.Hour)
		event2 := map[string]interface{}{
			"event_time": past,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "Past time is before now")
	})

	t.Run("negative duration", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: event_time < now() - days(-1)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		if !result.ValidationOK {
			t.Logf("Validation errors: %v", result.Errors)
		}
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// now() - days(-1) = now() + 1 day (future)
		// event_time < future
		past := time.Now().Add(-6 * time.Hour)
		event := map[string]interface{}{
			"event_time": past,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "Past time is before future")
	})

	t.Run("large duration values", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: created_at <= now() - days(365)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// 2 years ago
		twoYearsAgo := time.Now().Add(-2 * 365 * 24 * time.Hour)
		event := map[string]interface{}{
			"created_at": twoYearsAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "2 years ago is older than 1 year")

		// 6 months ago
		sixMonthsAgo := time.Now().Add(-180 * 24 * time.Hour)
		event2 := map[string]interface{}{
			"created_at": sixMonthsAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "6 months ago is within 1 year")
	})
}

func TestDurationFunctionCombinations(t *testing.T) {
	t.Run("combined durations in addition", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: deadline <= now() + days(1) + hours(12)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Deadline in 1 day (within 1.5 days)
		oneDayFromNow := time.Now().Add(24 * time.Hour)
		event := map[string]interface{}{
			"deadline": oneDayFromNow,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "1 day from now is within 1.5 days")

		// Deadline in 2 days (beyond 1.5 days)
		twoDaysFromNow := time.Now().Add(48 * time.Hour)
		event2 := map[string]interface{}{
			"deadline": twoDaysFromNow,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "2 days from now exceeds 1.5 days")
	})

	t.Run("hours and minutes combined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: event_time > now() - hours(2) - minutes(30)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// 2 hours ago (within 2.5 hours)
		twoHoursAgo := time.Now().Add(-2 * time.Hour)
		event := map[string]interface{}{
			"event_time": twoHoursAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "2 hours ago is within 2.5 hours")

		// 3 hours ago (beyond 2.5 hours)
		threeHoursAgo := time.Now().Add(-3 * time.Hour)
		event2 := map[string]interface{}{
			"event_time": threeHoursAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "3 hours ago exceeds 2.5 hours")
	})

	t.Run("all duration types combined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: event_time >= now() - days(1) - hours(6) - minutes(30) - seconds(45)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Total: 1 day + 6 hours + 30 minutes + 45 seconds = ~30.5 hours
		twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
		event := map[string]interface{}{
			"event_time": twentyFourHoursAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "24 hours ago is within 30.5 hours")

		// 32 hours ago (beyond limit)
		thirtyTwoHoursAgo := time.Now().Add(-32 * time.Hour)
		event2 := map[string]interface{}{
			"event_time": thirtyTwoHoursAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "32 hours ago exceeds 30.5 hours")
	})
}

func TestDurationWithTimeDifference(t *testing.T) {
	t.Run("time difference less than duration", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: (now() - last_activity) < hours(24)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Active 12 hours ago
		twelveHoursAgo := time.Now().Add(-12 * time.Hour)
		event := map[string]interface{}{
			"last_activity": twelveHoursAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "12 hours is less than 24 hours")

		// Active 30 hours ago
		thirtyHoursAgo := time.Now().Add(-30 * time.Hour)
		event2 := map[string]interface{}{
			"last_activity": thirtyHoursAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "30 hours exceeds 24 hours")
	})

	t.Run("time difference greater than duration", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: (now() - created_at) > days(7)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Created 10 days ago
		tenDaysAgo := time.Now().Add(-10 * 24 * time.Hour)
		event := map[string]interface{}{
			"created_at": tenDaysAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "10 days is greater than 7 days")

		// Created 5 days ago
		fiveDaysAgo := time.Now().Add(-5 * 24 * time.Hour)
		event2 := map[string]interface{}{
			"created_at": fiveDaysAgo,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "5 days is less than 7 days")
	})
}

func TestDurationWithMissingFields(t *testing.T) {
	t.Run("duration with missing time field", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: created_at > now() - days(5)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Missing created_at field
		event := map[string]interface{}{
			"other_field": "data",
		}
		matches := ruleEngine.MatchEvent(event)
		require.NotContains(t, matches, condition.RuleIdType(0), "Missing field should not match (undefined > time → undefined)")
	})

	t.Run("duration with null time field", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: created_at > now() - days(5)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Null created_at field
		event := map[string]interface{}{
			"created_at": nil,
		}
		matches := ruleEngine.MatchEvent(event)
		require.NotContains(t, matches, condition.RuleIdType(0), "Null field should not match (null > time → false)")
	})
}

func TestDurationFunctionTypes(t *testing.T) {
	t.Run("days with integer", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: (now() - event_time) < days(7)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)
	})

	t.Run("hours with float", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: session_time > now() - hours(2.5)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)
	})

	t.Run("minutes with expression constant", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: event_time >= now() - minutes(30 + 15)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// 40 minutes ago (within 45 minutes)
		fortyMinutesAgo := time.Now().Add(-40 * time.Minute)
		event := map[string]interface{}{
			"event_time": fortyMinutesAgo,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "40 minutes ago is within 45 minutes")
	})
}
