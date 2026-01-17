package tests_test

import (
	"testing"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestAbsFunction tests the abs() absolute value function
func TestAbsFunction(t *testing.T) {
	t.Run("abs of positive number", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: abs(balance) > 1000`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"balance": 1500,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "abs(1500) = 1500 > 1000")
	})

	t.Run("abs of negative number", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: abs(balance) > 1000`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"balance": -1500,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "abs(-1500) = 1500 > 1000")
	})

	t.Run("abs of zero", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: abs(value) == 0`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": 0,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "abs(0) = 0")
	})

	t.Run("abs with undefined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: abs(missing_field) > 10`

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
		require.NotContains(t, matches, condition.RuleIdType(0), "abs(undefined) should return undefined")
	})

	t.Run("abs with null", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: abs(value) > 10`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": nil,
		}
		matches := ruleEngine.MatchEvent(event)
		require.NotContains(t, matches, condition.RuleIdType(0), "abs(null) should return undefined")
	})
}

// TestCeilFunction tests the ceil() ceiling function
func TestCeilFunction(t *testing.T) {
	t.Run("ceil of positive float", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: ceil(price * 1.08) <= budget`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// 100 * 1.08 = 108.0, ceil = 108
		event := map[string]interface{}{
			"price":  100,
			"budget": 108,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "ceil(108.0) = 108 <= 108")

		// 99.5 * 1.08 = 107.46, ceil = 108
		event2 := map[string]interface{}{
			"price":  99.5,
			"budget": 108,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.Contains(t, matches2, condition.RuleIdType(0), "ceil(107.46) = 108 <= 108")
	})

	t.Run("ceil of negative float", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: ceil(value) == -2`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// ceil(-2.5) = -2
		event := map[string]interface{}{
			"value": -2.5,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "ceil(-2.5) = -2")
	})

	t.Run("ceil of whole number", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: ceil(value) == 5`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": 5,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "ceil(5) = 5")
	})
}

// TestFloorFunction tests the floor() floor function
func TestFloorFunction(t *testing.T) {
	t.Run("floor of positive float", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: floor(rating) >= 4`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"rating": 4.8,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "floor(4.8) = 4 >= 4")
	})

	t.Run("floor of negative float", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: floor(value) == -3`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// floor(-2.5) = -3
		event := map[string]interface{}{
			"value": -2.5,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "floor(-2.5) = -3")
	})

	t.Run("floor of whole number", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: floor(value) == 7`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": 7,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "floor(7) = 7")
	})
}

// TestRoundFunction tests the round() rounding function
func TestRoundFunction(t *testing.T) {
	t.Run("round to whole number", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: round(score) >= 95`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// round(94.5) = 95 (rounds to nearest even for .5)
		event := map[string]interface{}{
			"score": 94.6,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "round(94.6) = 95 >= 95")
	})

	t.Run("round with decimal places", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: round(score, 2) >= 95.50`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"score": 95.499,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "round(95.499, 2) = 95.50 >= 95.50")
	})

	t.Run("round with 1 decimal place", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: round(value, 1) == 3.1`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": 3.14159,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "round(3.14159, 1) = 3.1")
	})

	t.Run("round negative number", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: round(value) == -3`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": -2.6,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "round(-2.6) = -3")
	})
}

// TestMinFunction tests the min() minimum function
func TestMinFunction(t *testing.T) {
	t.Run("min of two numbers", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: min(price1, price2) < 100`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"price1": 150,
			"price2": 80,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "min(150, 80) = 80 < 100")
	})

	t.Run("min of three numbers", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: min(price1, price2, price3) < 100`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"price1": 150,
			"price2": 80,
			"price3": 200,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "min(150, 80, 200) = 80 < 100")
	})

	t.Run("min with negative numbers", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: min(a, b, c) == -10`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"a": 5,
			"b": -10,
			"c": 3,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "min(5, -10, 3) = -10")
	})

	t.Run("min with undefined values skips them", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: min(a, b, c) == 5`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// b is missing (undefined)
		event := map[string]interface{}{
			"a": 5,
			"c": 10,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "min(5, undefined, 10) = 5")
	})

	t.Run("min with all undefined returns undefined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: min(a, b) < 100`

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
		require.NotContains(t, matches, condition.RuleIdType(0), "min(undefined, undefined) should return undefined")
	})
}

// TestMaxFunction tests the max() maximum function
func TestMaxFunction(t *testing.T) {
	t.Run("max of two numbers", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: max(age, min_age) >= 21`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"age":     25,
			"min_age": 18,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "max(25, 18) = 25 >= 21")
	})

	t.Run("max of three numbers", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: max(a, b, c) == 200`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"a": 150,
			"b": 80,
			"c": 200,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "max(150, 80, 200) = 200")
	})

	t.Run("max with negative numbers", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: max(a, b, c) == 3`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"a": -5,
			"b": -10,
			"c": 3,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "max(-5, -10, 3) = 3")
	})

	t.Run("max with undefined values skips them", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: max(a, b, c) == 10`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// b is missing (undefined)
		event := map[string]interface{}{
			"a": 5,
			"c": 10,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "max(5, undefined, 10) = 10")
	})
}

// TestPowFunction tests the pow() power/exponentiation function
func TestPowFunction(t *testing.T) {
	t.Run("pow with integer exponent", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: pow(base, 2) == 100`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"base": 10,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "pow(10, 2) = 100")
	})

	t.Run("pow with fractional exponent", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: pow(value, 0.5) == 5`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// pow(25, 0.5) = sqrt(25) = 5
		event := map[string]interface{}{
			"value": 25,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "pow(25, 0.5) = 5")
	})

	t.Run("pow with zero exponent", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: pow(value, 0) == 1`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": 999,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "pow(999, 0) = 1")
	})

	t.Run("pow with negative exponent", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: pow(base, -1) == 0.1`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// pow(10, -1) = 1/10 = 0.1
		event := map[string]interface{}{
			"base": 10,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "pow(10, -1) = 0.1")
	})

	t.Run("pow with negative base", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: pow(base, 2) == 9`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"base": -3,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "pow(-3, 2) = 9")
	})
}

// TestMathFunctionCombinations tests combining math functions
func TestMathFunctionCombinations(t *testing.T) {
	t.Run("nested math functions", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: ceil(abs(value)) == 3`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": -2.3,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "ceil(abs(-2.3)) = ceil(2.3) = 3")
	})

	t.Run("math functions with arithmetic", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: floor(price * 1.1) + ceil(tax) <= budget`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// floor(100 * 1.1) + ceil(5.2) = floor(110) + ceil(5.2) = 110 + 6 = 116
		event := map[string]interface{}{
			"price":  100,
			"tax":    5.2,
			"budget": 116,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "floor(110) + ceil(5.2) = 116 <= 116")
	})

	t.Run("min and max together", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: max(min(a, b), min(c, d)) == 15`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// max(min(10, 20), min(15, 25)) = max(10, 15) = 15
		event := map[string]interface{}{
			"a": 10,
			"b": 20,
			"c": 15,
			"d": 25,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "max(min(10, 20), min(15, 25)) = 15")
	})

	t.Run("pow with round", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: round(pow(value, 2), 1) == 6.3`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// pow(2.5, 2) = 6.25, round to 1 decimal = 6.3 (rounds half away from zero)
		event := map[string]interface{}{
			"value": 2.5,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "round(pow(2.5, 2), 1) = round(6.25, 1) = 6.3")
	})
}

// TestMathFunctionValidation tests validation of math functions
func TestMathFunctionValidation(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	t.Run("abs requires one argument", func(t *testing.T) {
		rule := `- metadata: {id: 1}
  expression: abs()`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.False(t, result.ValidationOK, "Should fail validation")
		require.NotEmpty(t, result.Errors)
	})

	t.Run("min requires at least two arguments", func(t *testing.T) {
		rule := `- metadata: {id: 1}
  expression: min(value)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.False(t, result.ValidationOK, "Should fail validation")
		require.NotEmpty(t, result.Errors)
	})

	t.Run("pow requires two arguments", func(t *testing.T) {
		rule := `- metadata: {id: 1}
  expression: pow(base)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.False(t, result.ValidationOK, "Should fail validation")
		require.NotEmpty(t, result.Errors)
	})
}

// TestMathFunctionEdgeCases tests edge cases for math functions
func TestMathFunctionEdgeCases(t *testing.T) {
	t.Run("operations with float precision", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: round(value, 2) == 0.33`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// 1/3 = 0.333..., round to 2 decimals = 0.33
		event := map[string]interface{}{
			"value": 0.3333333,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "round(0.3333333, 2) = 0.33")
	})

	t.Run("min/max with all same values", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: min(a, b, c) == 5 && max(a, b, c) == 5`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"a": 5,
			"b": 5,
			"c": 5,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "all values are 5")
	})

	t.Run("pow with very small result", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: pow(value, -10) < 1`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"value": 2,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "pow(2, -10) is very small")
	})
}

// TestMathFunctionsWithTernary tests math functions combined with ternary operator
func TestMathFunctionsWithTernary(t *testing.T) {
	t.Run("ternary selecting between math operations", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: if(use_ceil, ceil(value), floor(value)) == 4`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Use ceil: ceil(3.5) = 4
		event := map[string]interface{}{
			"use_ceil": true,
			"value":    3.5,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "ceil(3.5) = 4")

		// Use floor: floor(4.5) = 4
		event2 := map[string]interface{}{
			"use_ceil": false,
			"value":    4.5,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.Contains(t, matches2, condition.RuleIdType(0), "floor(4.5) = 4")
	})

	t.Run("math function on ternary result", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: abs(if(positive, value, -value)) > 10`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// positive: abs(15) = 15 > 10
		event := map[string]interface{}{
			"positive": true,
			"value":    15,
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "abs(15) = 15 > 10")

		// not positive: abs(-15) = 15 > 10
		event2 := map[string]interface{}{
			"positive": false,
			"value":    15,
		}
		matches2 := ruleEngine.MatchEvent(event2)
		require.Contains(t, matches2, condition.RuleIdType(0), "abs(-15) = 15 > 10")
	})
}
