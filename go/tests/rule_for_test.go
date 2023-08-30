package tests

import (
	"github.com/rulestone/Utils"
	"github.com/rulestone/api"
	"github.com/rulestone/engine"
	"github.com/rulestone/types"
	"strings"
	"testing"
)

func TestFilterForError0(t *testing.T) {
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
    "rules": [
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
		t.Fatalf("failed parsing JSON: %v", err)
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

func TestFilterForError1(t *testing.T) {
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
    "rules": [
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
		t.Fatalf("failed parsing JSON: %v", err)
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

func TestFilterForError2(t *testing.T) {
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
    "rules": [
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
		t.Fatalf("failed parsing JSON: %v", err)
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

func TestFilterForError3(t *testing.T) {
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
    "rules": [
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
		t.Fatalf("failed parsing JSON: %v", err)
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

func TestFilterFor1(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_for_each_1.json", ctx)
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %v", err)
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_each_1.json"); err != nil {
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

func TestFilterFor2(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_for_each_2.yaml", ctx)
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %s", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %s", err)
	}
	if event, err := utils.ReadEvent("../examples/data/data_for_each_2.json"); err != nil {
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

func TestFilterFor3(t *testing.T) {
	ctx := types.NewAppContext()
	fapi := api.NewRuleApi(ctx)

	repo := engine.NewRuleEngineRepo(ctx)
	rule1, err := utils.ReadRuleFromFile("../examples/rules/rule_for_each_3.yaml", ctx)
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	fd1, err := fapi.RuleToRuleDefinition(rule1)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %s", err)
		return
	}
	repo.Register(fd1)
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}
	if event, err := utils.ReadEvent("../examples/data/data_for_each_3.json"); err != nil {
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
