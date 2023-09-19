package tests

import (
	"github.com/atlasgurus/rulestone/engine"
	"github.com/atlasgurus/rulestone/utils"
	"testing"
)

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
		t.Fatalf("failed ruleToRuleDefinition: %s", err)
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
	numCatEvals := genFilter.Metrics.NumCatEvals
	if event, err := utils.ReadEvent("../examples/data/data_for_each_test3.json"); err != nil {
		t.Fatalf("failed ReadEvent: %s", err)
	} else {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			t.Fatalf("failed number of matches %d != 1", len(matches))
		}
		if genFilter.Metrics.NumCatEvals-numCatEvals != 3 {
			t.Fatalf("failed common category elimination optimization")
		}
	}

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
		repo.GetAppCtx().PrintErrors()
	}
}
