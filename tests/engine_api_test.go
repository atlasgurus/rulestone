package tests

import (
	"github.com/atlasgurus/rulestone/engine"
	"github.com/atlasgurus/rulestone/types"
	"github.com/atlasgurus/rulestone/utils"
	"testing"
)

func TestFilterApiExpression0(t *testing.T) {
	ctx := types.NewAppContext()

	repo := engine.NewRuleEngineRepo(ctx)
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_expression_test0.json"); err != nil {
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
	repo := engine.NewRuleEngineRepo(ctx)
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_expression_test1.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_expression_test1.json"); err != nil {
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
	repo := engine.NewRuleEngineRepo(ctx)
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_expression_test2.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_expression_test2.json"); err != nil {
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
	repo := engine.NewRuleEngineRepo(ctx)
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_expression_test3.yaml")
	if err != nil {
		t.Fatalf("failed NewRuleEngineRepo: %v", err)
		return
	}
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_expression_test3.json"); err != nil {
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
	repo := engine.NewRuleEngineRepo(ctx)
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_expression_test4.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_expression_test4.json"); err != nil {
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
	repo := engine.NewRuleEngineRepo(ctx)
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_expression_test5.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}
	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_expression_test5.json"); err != nil {
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
	repo := engine.NewRuleEngineRepo(ctx)
	_, err := repo.RegisterRuleFromFile("../examples/rules/rule_expression_test6.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRuleFromFile: %v", err)
		return
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_expression_test6.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		numCatEvals := genFilter.Metrics.NumCatEvals
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
		if genFilter.Metrics.NumCatEvals-numCatEvals != 4 {
			t.Fatalf("failed common category elimination optimization")
		}
	}

	if ctx.NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", ctx.NumErrors())
		ctx.PrintErrors()
	}
}
