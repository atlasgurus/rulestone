package tests

import (
	"github.com/atlasgurus/rulestone/engine"
	"github.com/atlasgurus/rulestone/utils"
	"testing"
)

func TestFilterApiExpression0(t *testing.T) {

	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRulesFromFile: %v", err)
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

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}
func TestFilterApiExpression1(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test1.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRulesFromFile: %v", err)
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

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestFilterApiExpression2(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test2.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRulesFromFile: %v", err)
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

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestFilterApiExpression3(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test3.yaml")
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

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestFilterApiExpression4(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test4.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRulesFromFile: %v", err)
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

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestFilterApiExpression5(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test5.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRulesFromFile: %v", err)
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

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestFilterApiExpression6(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test6.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRulesFromFile: %v", err)
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

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestMultipleRulesPerFile(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	ruleIds, err := repo.RegisterRulesFromFile("../examples/rules/multiple_rules_per_file_test.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRulesFromFile: %v", err)
		return
	}

	if len(ruleIds) != 3 {
		t.Fatalf("failed number of rules %d != 3", len(ruleIds))
		return
	}

	if ruleIds[0] != 0 && ruleIds[1] != 1 && ruleIds[2] != 2 {
		t.Fatalf("failed rule ids %d %d %d", ruleIds[0], ruleIds[1], ruleIds[2])
		return
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_multiple_rules_per_file_test0.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
		if matches[0] != 0 {
			t.Fatalf("failed match %d != 0", matches[0])
		}

	}

	if event, err := utils.ReadEvent("../examples/data/data_multiple_rules_per_file_test1.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
		if matches[0] != 1 {
			t.Fatalf("failed match %d != 0", matches[0])
		}
	}

	if event, err := utils.ReadEvent("../examples/data/data_multiple_rules_per_file_test2.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
		if matches[0] != 2 {
			t.Fatalf("failed match %d != 0", matches[0])
		}
	}

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}

func TestGoKeywordsAndNumericPrefixedFields(t *testing.T) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/go_keywords_numeric_prefixed_test.yaml")
	if err != nil {
		t.Fatalf("failed RegisterRulesFromFile: %v", err)
		return
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed NewRuleEngine: %s", err)
	}

	if event, err := utils.ReadEvent("../examples/data/data_go_keywords_numeric_prefixed_test.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 3 {
			t.Fatalf("failed number of matches %d != 3", len(matches))
		}
	}

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}
