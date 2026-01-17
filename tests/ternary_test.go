package tests_test

import (
	"testing"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestTernaryOperator tests the if() ternary/conditional function
func TestTernaryOperator(t *testing.T) {
	t.Run("simple conditional - true branch", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(age >= 18, "adult", "minor") == "adult"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"age": 20,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "age >= 18 should return 'adult'")
	})

	t.Run("simple conditional - false branch", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(age >= 18, "adult", "minor") == "minor"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"age": 15,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "age < 18 should return 'minor'")
	})

	t.Run("conditional with numeric return values", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(premium, discount * 2, discount) == 20`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Premium user gets double discount
		event := map[string]interface{}{
			"premium":  true,
			"discount": 10,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "premium user gets discount * 2")

		// Non-premium user gets regular discount
		event2 := map[string]interface{}{
			"premium":  false,
			"discount": 20,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.Contains(t, matches2, condition.RuleIdType(0), "non-premium user gets regular discount")
	})

	t.Run("nested ternary", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(score >= 90, "A", if(score >= 80, "B", "C")) == "B"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Score of 85 should get B
		event := map[string]interface{}{
			"score": 85,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "score 85 should get B")

		// Score of 95 should get A
		rule2 := `- metadata: {id: 2}
  expression: if(score >= 90, "A", if(score >= 80, "B", "C")) == "A"`
		result2, err := repo.LoadRulesFromString(rule2,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result2.ValidationOK)

		ruleEngine2, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event2 := map[string]interface{}{
			"score": 95,
		}
		matches2 := ruleEngine2.MatchEvent(event2)
		require.Contains(t, matches2, condition.RuleIdType(1), "score 95 should get A")
	})

	t.Run("ternary with undefined condition", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(missing_field > 10, "yes", "no") == "yes"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Missing field makes condition undefined, so if() returns undefined
		event := map[string]interface{}{
			"other_field": "value",
		}
		matches := ruleEngine.MatchEvent(event)
		require.NotContains(t, matches, condition.RuleIdType(0), "undefined condition should return undefined, not match")
	})

	t.Run("ternary with null condition", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(status == "active", "enabled", "disabled") == "enabled"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Null status
		event := map[string]interface{}{
			"status": nil,
		}
		matches := ruleEngine.MatchEvent(event)
		require.NotContains(t, matches, condition.RuleIdType(0), "null status should evaluate to false, take false branch")
	})

	t.Run("ternary with boolean expressions", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(age >= 18 && verified == true, true, false) == true`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Both conditions true
		event := map[string]interface{}{
			"age":      20,
			"verified": true,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "both conditions true")

		// One condition false
		event2 := map[string]interface{}{
			"age":      20,
			"verified": false,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.NotContains(t, matches2, condition.RuleIdType(0), "verified is false")
	})

	t.Run("ternary in arithmetic expression", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(premium, 100, 50) + bonus == 120`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Premium: 100 + 20 = 120
		event := map[string]interface{}{
			"premium": true,
			"bonus":   20,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "premium: 100 + 20 = 120")

		// Non-premium: 50 + 70 = 120
		event2 := map[string]interface{}{
			"premium": false,
			"bonus":   70,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.Contains(t, matches2, condition.RuleIdType(0), "non-premium: 50 + 70 = 120")
	})

	t.Run("ternary with comparison operators", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: price < if(vip, 100, 200)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// VIP: price 80 < 100
		event := map[string]interface{}{
			"vip":   true,
			"price": 80,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "vip: 80 < 100")

		// Non-VIP: price 150 < 200
		event2 := map[string]interface{}{
			"vip":   false,
			"price": 150,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.Contains(t, matches2, condition.RuleIdType(0), "non-vip: 150 < 200")

		// VIP: price 120 < 100 (false)
		event3 := map[string]interface{}{
			"vip":   true,
			"price": 120,
		}
		matches3 := ruleEngine.MatchEvent(event3)
		require.NotContains(t, matches3, condition.RuleIdType(0), "vip: 120 not < 100")
	})

	t.Run("ternary with field access in branches", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(use_alt, alt_value, main_value) > 100`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Use alt value
		event := map[string]interface{}{
			"use_alt":    true,
			"alt_value":  150,
			"main_value": 50,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "use alt value 150 > 100")

		// Use main value
		event2 := map[string]interface{}{
			"use_alt":    false,
			"alt_value":  50,
			"main_value": 200,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.Contains(t, matches2, condition.RuleIdType(0), "use main value 200 > 100")
	})
}

func TestTernaryOperatorValidation(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	t.Run("if requires three arguments", func(t *testing.T) {
		rule := `- metadata: {id: 1}
  expression: if(age >= 18, "adult")`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.False(t, result.ValidationOK, "Should fail validation - missing third argument")
		require.NotEmpty(t, result.Errors)
	})

	t.Run("if with too many arguments", func(t *testing.T) {
		rule := `- metadata: {id: 1}
  expression: if(age >= 18, "adult", "minor", "extra")`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.False(t, result.ValidationOK, "Should fail validation - too many arguments")
		require.NotEmpty(t, result.Errors)
	})
}

func TestTernaryOperatorEdgeCases(t *testing.T) {
	t.Run("ternary with zero values", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(count > 0, "has items", "empty") == "empty"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Zero count
		event := map[string]interface{}{
			"count": 0,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "count 0 should take false branch")
	})

	t.Run("ternary with empty string", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(name == "", "unnamed", name) == "unnamed"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Empty name
		event := map[string]interface{}{
			"name": "",
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "empty name should take true branch")
	})

	t.Run("ternary with false literal", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(false, "never", "always") == "always"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "false literal should take false branch")
	})

	t.Run("ternary with true literal", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(true, "always", "never") == "always"`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "true literal should take true branch")
	})
}

func TestTernaryOperatorTypeConsistency(t *testing.T) {
	t.Run("different types in branches", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		// This is valid - branches can have different types
		rule := `- metadata: {id: 1}
  expression: if(use_number, 100, "text") == 100`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"use_number": true,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "should return number 100")
	})

	t.Run("boolean result used in comparison", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(premium, true, false) == true && verified == true`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"premium":  true,
			"verified": true,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "both conditions true")
	})
}
