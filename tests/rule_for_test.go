package tests

import (
	"github.com/atlasgurus/rulestone/api"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/atlasgurus/rulestone/types"
	"github.com/atlasgurus/rulestone/utils"
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
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_for_each_1.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_for_each_test1.yaml"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestFilterFor2(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_for_each_test2.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed RuleToRuleDefinition: %s", err)
	}
	if event, err := utils.ReadEvent("../examples/data/data_for_each_test2.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestFilterFor3(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_for_each_test3.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}
	if event, err := utils.ReadEvent("../examples/data/data_for_each_test3.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}
