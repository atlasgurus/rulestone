package tests

import (
	"github.com/atlasgurus/rulestone/api"
	"github.com/atlasgurus/rulestone/types"
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
