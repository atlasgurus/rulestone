package tests

import (
	"github.com/rulestone/Utils"
	"github.com/rulestone/api"
	"github.com/rulestone/engine"
	"github.com/rulestone/types"
	"strings"
	"testing"
)

func TestFilterApiError0(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	rule1, err := fapi.ReadRule(strings.NewReader(`
{
  "metadata": {
    "created": "2023-03-29",
    "priority": 10
  },
  "condition": {
    "operation": "AND",
    "args": [
      {
        "operation": "AND",
        "field": "[0].name",
        "operator": "=",
        "value": "Frank"
      },
      {
        "field": "[0].child.age",
        "operator": ">",
        "value": 5,
        "comment": "all children must be less than 5 years old"
      }
    ]
  }
}
`), "json")
	if err != nil {
		t.Fatalf("Error parsing JSON: %v", err)
		return
	}

	_, err = fapi.RuleToRuleDefinition(rule1)
	if err == nil {
		t.Fatalf("Error expected")
		return
	}

	if ctx.NumErrors() == 0 {
		t.Fatalf("failed: expected > 0 errors ")
	}
}

func TestFilterApiError1(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	rule1, err := fapi.ReadRule(strings.NewReader(`
{
  "metadata": {
    "created": "2023-03-29",
    "priority": 10
  },
  "condition": {
    "operation": "AND",
    "args": [
      {
        "field": "[0].name",
        "operator": "="
      },
      {
        "field": "[0].child.age",
        "operator": ">",
        "value": 5,
        "comment": "all children must be less than 5 years old"
      }
    ]
  }
}
`), "json")
	if err != nil {
		t.Fatalf("Error parsing JSON: %v", err)
		return
	}

	_, err = fapi.RuleToRuleDefinition(rule1)
	if err == nil {
		t.Fatalf("Error expected")
		return
	}

	if ctx.NumErrors() == 0 {
		t.Fatalf("failed: expected > 0 errors ")
	}
	ctx.PrintErrors()
}

func TestFilterApiError2(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	rule1, err := fapi.ReadRule(strings.NewReader(`
{
  "metadata": {
    "created": "2023-03-29",
    "priority": 10
  },
  "condition": {
    "operation": "AND",
    "args": [
      {
        "field1": "[0].name",
        "operator": "*",
        "field2": "[1].name"
      },
      {
        "field": "[0].child.age",
        "operator": ">",
        "value": 5,
        "comment": "all children must be less than 5 years old"
      }
    ]
  }
}
`), "json")
	if err != nil {
		t.Fatalf("Error parsing JSON: %v", err)
		return
	}

	_, err = fapi.RuleToRuleDefinition(rule1)
	if err == nil {
		t.Fatalf("Error expected")
		return
	}

	if ctx.NumErrors() == 0 {
		t.Fatalf("failed: expected > 0 errors ")
	}
	ctx.PrintErrors()
}

func TestFilterApiError3(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	rule1, err := fapi.ReadRule(strings.NewReader(`
{
  "metadata": {
    "created": "2023-03-29",
    "priority": 10
  },
  "condition": {
    "operation": "AND",
    "args": [
      {
        "field1": "[0].name",
        "operator": "=",
        "field2": "[1].name"
      },
      {
        "field": "[0].child.age",
        "operator": ">",
        "value": {"foo":10},
        "comment": "all children must be less than 5 years old"
      }
    ]
  }
}
`), "json")
	if err != nil {
		t.Fatalf("Error parsing JSON: %v", err)
		return
	}

	_, err = fapi.RuleToRuleDefinition(rule1)
	if err == nil {
		t.Fatalf("Error expected")
		return
	}

	if ctx.NumErrors() == 0 {
		t.Fatalf("failed: expected > 0 errors ")
	}
	ctx.PrintErrors()
}

func TestFilterApi0(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_test1.json", ctx)
	if err != nil {
		t.Fatalf("failed ReadRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("faield RuleToRuleDefinition: %v", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}
	if event, err := utils.ReadEvent("../examples/data/data0.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}

func TestFilterApiExpression0(t *testing.T) {
	// Disable the test.  NOT is not supported yet.
	return

	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_expression_test0.json", ctx)
	if err != nil {
		t.Fatalf("failed ReadRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %v", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_rule_expression_test0.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}
func TestFilterApiExpression1(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_expression_test1.json", ctx)
	if err != nil {
		t.Fatalf("failed ReadRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %v", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_rule_expression_test1.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}

func TestFilterApiExpression2(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_expression_test2.json", ctx)
	if err != nil {
		t.Fatalf("failed ReadRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %v", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_rule_expression_test2.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}

func TestFilterApiExpression3(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_expression_test3.json", ctx)
	if err != nil {
		t.Fatalf("failed NewRuleEngineRepo: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %v", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_rule_expression_test3.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}

func TestFilterApiExpression4(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_expression_test4.json", ctx)
	if err != nil {
		t.Fatalf("failed ReadRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %v", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_rule_expression_test4.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}

func TestFilterApiExpression5(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_expression_test5.json", ctx)
	if err != nil {
		t.Fatalf("failed ReadRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %v", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_rule_expression_test5.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}

func TestFilterApiExpression6(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_expression_test6.yaml", ctx)
	if err != nil {
		t.Fatalf("failed ReadRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %v", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_rule_expression_test6.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		numCatEvals := genFilter.Metrics.NumCatEvals
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
		if genFilter.Metrics.NumCatEvals-numCatEvals != 3 {
			t.Fatalf("failed common category elimination optimization")
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}
