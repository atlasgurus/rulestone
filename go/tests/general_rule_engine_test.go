package tests

import (
	"github.com/rulestone/Utils"
	"github.com/rulestone/api"
	c "github.com/rulestone/condition"
	"github.com/rulestone/engine"
	"github.com/rulestone/types"
	"testing"
)

func TestGeneralFilter0(t *testing.T) {
	ctx := types.NewAppContext()

	cond1 :=
		c.NewAndCond(
			c.NewCompareCond(c.CompareEqualOp, c.NewAttributeOperand("child.age"), c.NewFloatOperand(10)),
			c.NewCompareCond(c.CompareEqualOp, c.NewAttributeOperand("name"), c.NewStringOperand("Frank")),
		)
	cond2 :=
		c.NewAndCond(
			c.NewCompareCond(c.CompareEqualOp, c.NewAttributeOperand("name"), c.NewStringOperand("Alice")),
			c.NewCompareCond(c.CompareEqualOp, c.NewAttributeOperand("age"), c.NewFloatOperand(30)),
			c.NewCompareCond(c.CompareEqualOp, c.NewAttributeOperand("gender"), c.NewStringOperand("female")),
			c.NewCompareCond(c.CompareEqualOp, c.NewAttributeOperand("children[1].name"), c.NewStringOperand("David")),
		)
	repo := engine.NewRuleEngineRepo(ctx)
	ruleDef1 := &api.RuleDefinition{Condition: cond1}
	repo.Register(ruleDef1)
	ruleDef2 := &api.RuleDefinition{Condition: cond2}
	repo.Register(ruleDef2)
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

	if event, err := utils.ReadEvent("../examples/data/data1.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
	}

	ruleDefFromEngine1 := genFilter.GetRuleDefinition(0)
	ruleDefFromEngine2 := genFilter.GetRuleDefinition(1)

	if ruleDefFromEngine1 == nil || ruleDefFromEngine2 == nil || ruleDefFromEngine1 != ruleDef1 || ruleDefFromEngine2 != ruleDef2 {
		t.Fatalf("failed: rule engine must return correct rule definition by rule id")
	}

	ruleDefNonExisting1 := genFilter.GetRuleDefinition(1001)
	ruleDefNonExisting2 := genFilter.GetRuleDefinition(-1)

	if ruleDefNonExisting1 != nil || ruleDefNonExisting2 != nil {
		t.Fatalf("failed: rule engine must return nil for non-existing rule id")
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}
