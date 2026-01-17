package tests_test

import (
	"testing"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestFilterFunction tests filter() with length()
func TestFilterFunction(t *testing.T) {
	t.Run("filter with length", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: length(filter("items", "item", item.active == true)) > 2`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"active": true, "price": 10},
				map[string]interface{}{"active": false, "price": 20},
				map[string]interface{}{"active": true, "price": 30},
				map[string]interface{}{"active": true, "price": 40},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "3 active items > 2")
	})

	t.Run("double filter", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: length(filter(filter("items", "item", item.category == "food"), "item", item.price > 10)) == 1`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"category": "food", "price": 5},
				map[string]interface{}{"category": "food", "price": 15},
				map[string]interface{}{"category": "toys", "price": 20},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "1 item is food AND price > 10")
	})
}

func TestMapFunction(t *testing.T) {
	t.Run("map with sum", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: sum(map("items", "item", item.price)) > 50`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"price": 10},
				map[string]interface{}{"price": 20},
				map[string]interface{}{"price": 30},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "sum of prices = 60 > 50")
	})

	t.Run("map with transformation", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: sum(map("items", "item", item.price * 2)) == 120`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"price": 10},
				map[string]interface{}{"price": 20},
				map[string]interface{}{"price": 30},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "sum of doubled = 120")
	})
}

func TestFilterMapComposition(t *testing.T) {
	t.Run("filter then map then sum", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: sum(map(filter("items", "item", item.active == true), "item", item.price)) == 40`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"price": 10, "active": true},
				map[string]interface{}{"price": 20, "active": false},
				map[string]interface{}{"price": 30, "active": true},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "10 + 30 = 40")
	})
}

func TestSumDirect(t *testing.T) {
	t.Run("sum array directly", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: sum("items", "item", item.price) == 60`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"price": 10},
				map[string]interface{}{"price": 20},
				map[string]interface{}{"price": 30},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "direct sum = 60")
	})

	t.Run("sum with expression", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: sum("items", "item", item.price * item.quantity) == 110`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"price": 10, "quantity": 2},
				map[string]interface{}{"price": 20, "quantity": 3},
				map[string]interface{}{"price": 15, "quantity": 2},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "20 + 60 + 30 = 110")
	})
}

func TestAvgFunction(t *testing.T) {
	t.Run("avg direct array", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: avg("items", "item", item.rating) >= 4.0`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"rating": 3.5},
				map[string]interface{}{"rating": 4.0},
				map[string]interface{}{"rating": 4.5},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "avg = 4.0")
	})

	t.Run("avg on filtered iterator", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: avg(filter("users", "user", user.age >= 18), "user", user.score) == 80`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"age": 16, "score": 50},
				map[string]interface{}{"age": 20, "score": 75},
				map[string]interface{}{"age": 25, "score": 85},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "avg of adult scores = 80")
	})
}

func TestIteratorWithUndefined(t *testing.T) {
	t.Run("filter skips undefined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: length(filter("items", "item", item.price > 15)) == 2`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"price": 10},
				map[string]interface{}{"price": 20},
				map[string]interface{}{"name": "no price"},
				map[string]interface{}{"price": 30},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "2 items with price > 15 (undefined excluded)")
	})

	t.Run("map skips undefined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: sum(map("items", "item", item.price)) == 30`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"price": 10},
				map[string]interface{}{"name": "no price"},
				map[string]interface{}{"price": 20},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "sum = 30 (undefined excluded)")
	})
}
