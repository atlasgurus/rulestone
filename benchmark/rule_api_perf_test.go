package benchmark

import (
	"github.com/atlasgurus/rulestone/engine"
	"github.com/atlasgurus/rulestone/utils"
	"io/ioutil"
	"path"
	"testing"
)

func TestFilterApiPerf1(t *testing.T) {
	repo := engine.NewRuleEngineRepo()

	// Load rule files from a directory
	ruleFiles, err := ioutil.ReadDir("../examples/rules/gen.configs.rulestone")
	if err != nil {
		t.Fatalf("Error reading directory: %v", err)
		return
	}

	for _, ruleFile := range ruleFiles {
		_, err := repo.RegisterRulesFromFile(path.Join("../examples/rules/gen.configs.rulestone", ruleFile.Name()))
		if err != nil {
			t.Fatalf("Error opening file: %v", err)
			return
		}
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Error creating RuleEngine: %s", err)
	}

	numMatches := 0
	numEvents := 0
	err = utils.ReadEvents("../examples/data/rule_benchmark_data.jsonl", func(event interface{}) error {
		matches := genFilter.MatchEvent(event)
		numEvents++
		numMatches += len(matches)
		return nil
	})

	t.Logf("Number of matches: %d/%d", numMatches, numEvents)

	if repo.GetAppCtx().NumErrors() > 0 {
		t.Fatalf("failed due to %d errors", repo.GetAppCtx().NumErrors())
	}
}
