package tests_test

import (
	"testing"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/stretchr/testify/require"
)

// TestCountFunction tests count() array aggregation
func TestCountFunction(t *testing.T) {
	t.Run("count matching elements", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: count("items", "item", item.active == true) == 3`

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
				map[string]interface{}{"active": true},
				map[string]interface{}{"active": false},
				map[string]interface{}{"active": true},
				map[string]interface{}{"active": true},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "3 active items")
	})

	t.Run("count with complex condition", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: count("orders", "order", order.status == "pending" && order.total > 100) >= 2`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"orders": []interface{}{
				map[string]interface{}{"status": "pending", "total": 150},
				map[string]interface{}{"status": "pending", "total": 50},
				map[string]interface{}{"status": "complete", "total": 200},
				map[string]interface{}{"status": "pending", "total": 120},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "2 pending orders > 100")
	})

	t.Run("count returns 0 for no matches", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: count("items", "item", item.price > 1000) == 0`

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
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "no items > 1000")
	})

	t.Run("count undefined for missing array", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: count("items", "item", item.active) == undefined`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"other": "data",
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "missing array → undefined")
	})
}

func TestMinOfFunction(t *testing.T) {
	t.Run("minOf finds minimum", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: minOf("items", "item", item.price) == 10`

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
				map[string]interface{}{"price": 30},
				map[string]interface{}{"price": 10},
				map[string]interface{}{"price": 20},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "min price = 10")
	})

	t.Run("minOf with conditional", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: minOf("items", "item", if(item.active, item.price, 999999)) == 20`

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
				map[string]interface{}{"price": 30, "active": true},
				map[string]interface{}{"price": 10, "active": false},
				map[string]interface{}{"price": 20, "active": true},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "min active price = 20")
	})

	t.Run("minOf empty array returns undefined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: minOf("items", "item", item.value) == undefined`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"items": []interface{}{},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "empty array → undefined")
	})
}

func TestMaxOfFunction(t *testing.T) {
	t.Run("maxOf finds maximum", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: maxOf("items", "item", item.price) == 30`

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
				map[string]interface{}{"price": 30},
				map[string]interface{}{"price": 20},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "max price = 30")
	})

	t.Run("maxOf with expression", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: maxOf("items", "item", item.price * item.quantity) == 60`

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
		require.Contains(t, matches, condition.RuleIdType(0), "max = 60")
	})
}

func TestAllAnyAliases(t *testing.T) {
	t.Run("all() as alias for forAll", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: all("items", "item", item.valid == true)`

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
				map[string]interface{}{"valid": true},
				map[string]interface{}{"valid": true},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "all valid")
	})

	t.Run("any() as alias for forSome", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: any("items", "item", item.shipped == true)`

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
				map[string]interface{}{"shipped": false},
				map[string]interface{}{"shipped": true},
				map[string]interface{}{"shipped": false},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "any shipped")
	})

	t.Run("forAll still works", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: forAll("items", "item", item.valid == true)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)
	})

	t.Run("forSome still works", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: forSome("items", "item", item.active)`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)
	})
}

func TestArrayAggregationsWithUndefined(t *testing.T) {
	t.Run("count skips undefined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: count("items", "item", item.price > 15) == 2`

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
		require.Contains(t, matches, condition.RuleIdType(0), "2 items > 15 (undefined excluded)")
	})

	t.Run("minOf skips undefined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: minOf("items", "item", item.price) == 10`

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
		require.Contains(t, matches, condition.RuleIdType(0), "min = 10 (undefined skipped)")
	})

	t.Run("maxOf skips undefined", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: maxOf("items", "item", item.rating) == 4.5`

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
				map[string]interface{}{"name": "no rating"},
				map[string]interface{}{"rating": 4.5},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "max = 4.5 (undefined skipped)")
	})
}

func TestRealWorldUseCase(t *testing.T) {
	t.Run("e-commerce cart total", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: sum("cart_items", "item", item.price * item.quantity) > 100`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"cart_items": []interface{}{
				map[string]interface{}{"price": 20, "quantity": 3},
				map[string]interface{}{"price": 25, "quantity": 2},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "60 + 50 = 110 > 100")
	})

	t.Run("minimum stock level", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: minOf("products", "p", p.stock) < 10`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		event := map[string]interface{}{
			"products": []interface{}{
				map[string]interface{}{"stock": 5},
				map[string]interface{}{"stock": 20},
				map[string]interface{}{"stock": 15},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "min stock = 5 < 10")
	})

	t.Run("count pending orders", func(t *testing.T) {
		repo := engine.NewRuleEngineRepo()
		rule := `- metadata: {id: 1}
  expression: count("orders", "order", order.status == "pending") >= 5`

		result, err := repo.LoadRulesFromString(rule,
			engine.WithValidate(true),
			engine.WithFileFormat("yaml"),
		)
		require.NoError(t, err)
		require.True(t, result.ValidationOK)

		ruleEngine, err := engine.NewRuleEngine(repo)
		require.NoError(t, err)

		// Create 7 orders, 5 pending
		event := map[string]interface{}{
			"orders": []interface{}{
				map[string]interface{}{"status": "pending"},
				map[string]interface{}{"status": "complete"},
				map[string]interface{}{"status": "pending"},
				map[string]interface{}{"status": "pending"},
				map[string]interface{}{"status": "complete"},
				map[string]interface{}{"status": "pending"},
				map[string]interface{}{"status": "pending"},
			},
		}
		matches := ruleEngine.MatchEvent(event)
		require.Contains(t, matches, condition.RuleIdType(0), "5 pending orders")
	})
}
