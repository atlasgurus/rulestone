package tests

import (
	"github.com/atlasgurus/rulestone/cateng"
	c "github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
	"testing"
)

func TestFilter0(t *testing.T) {
	rule1 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(8)),
		))
	rule2 := c.NewRule(
		2,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(8)),
			c.NewOrCond(c.NewCategoryCond(200)),
			c.NewOrCond(c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(190)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule1, rule2})
	catFilter := cateng.NewCategoryEngine(repo, nil)
	matches := catFilter.MatchEvent([]types.Category{8, 5, 190})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}
}

func TestFilter1(t *testing.T) {
	cond := c.CategoryArraysToCondition([][]types.Category{{1}, {2}})

	repo := c.NewRuleRepo([]*c.Rule{{RuleId: 1, Cond: cond}})
	catFilter := cateng.NewCategoryEngine(repo, nil)
	matches := catFilter.MatchEvent([]types.Category{1, 2})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}
}

func TestFilter2(t *testing.T) {
	rule := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(3)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule})
	catFilter := cateng.NewCategoryEngine(repo, nil)
	matches := catFilter.MatchEvent([]types.Category{1, 2})
	if len(matches) != 0 {
		t.Fatalf("failed number of matches %d != 0", len(matches))
	}
}

func TestFilter3(t *testing.T) {
	rule := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(3), c.NewCategoryCond(2)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule})
	catFilter := cateng.NewCategoryEngine(repo, nil)
	matches := catFilter.MatchEvent([]types.Category{1, 2})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}
}

func TestFilter4(t *testing.T) {
	rule1 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
		))
	rule2 := c.NewRule(
		2,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(3), c.NewCategoryCond(2)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule1, rule2})
	catFilter := cateng.NewCategoryEngine(repo, nil)
	matches := catFilter.MatchEvent([]types.Category{1, 2})
	if len(matches) != 2 {
		t.Fatalf("failed number of matches %d != 2", len(matches))
	}
}

func TestFilter5(t *testing.T) {
	rule1 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
		))
	rule2 := c.NewRule(
		2,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewNotCond(c.NewOrCond(c.NewCategoryCond(4))),
			c.NewOrCond(c.NewCategoryCond(3), c.NewCategoryCond(2)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule1, rule2})
	catFilter := cateng.NewCategoryEngine(repo, nil)
	matches := catFilter.MatchEvent([]types.Category{1, 2})
	if len(matches) != 2 {
		t.Fatalf("failed number of matches %d != 2", len(matches))
	}
}

func TestFilterOpt0(t *testing.T) {
	rule1 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(c.NewCategoryCond(3)),
		))
	rule2 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(c.NewCategoryCond(4)),
		))
	rule3 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(c.NewCategoryCond(5)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule1, rule2, rule3})
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      true,
	})
	matches := catFilter.MatchEvent([]types.Category{1, 2, 3})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	if catFilter.Metrics.NumMaskArrayLookups != 7 {
		t.Fatalf("failed optimization verification")
	}

	if catFilter.Metrics.NumBitMaskMatches != 1 {
		t.Fatalf("failed optimization verification")
	}
	catFilter.PrintMetrics()
}

func TestFilterOpt1(t *testing.T) {
	rule1 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(c.NewCategoryCond(3)),
		))
	rule2 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(c.NewCategoryCond(4)),
		))
	rule3 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(c.NewCategoryCond(5)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule1, rule2, rule3})
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  2,
		AndOptimizationFreqThreshold: 2,
		Verbose:                      true,
	})
	matches := catFilter.MatchEvent([]types.Category{1, 2, 3})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	if catFilter.Metrics.NumMaskArrayLookups != 6 {
		t.Fatalf("failed optimization verification")
	}

	if catFilter.Metrics.NumBitMaskMatches != 2 {
		t.Fatalf("failed optimization verification")
	}
	catFilter.PrintMetrics()
}

func TestFilterOpt2(t *testing.T) {
	rule1 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(c.NewCategoryCond(3)),
			c.NewOrCond(c.NewCategoryCond(4)),
			c.NewOrCond(c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(6)),
		))
	rule2 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(c.NewCategoryCond(3)),
			c.NewOrCond(c.NewCategoryCond(4)),
			c.NewOrCond(c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(7)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule1, rule2})
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      true,
	})

	if catFilter.FilterTables.BuilderMetrics.AndOrSetsInlined != 3 {
		t.Fatalf("failed AndOrSetsInlined")
	}

	matches := catFilter.MatchEvent([]types.Category{1, 2, 3, 4, 5, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	catFilter.PrintMetrics()
}

func TestFilterOpt3(t *testing.T) {
	rule1 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(
				c.NewCategoryCond(3),
				c.NewCategoryCond(4),
				c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(6)),
		))
	rule2 := c.NewRule(
		2,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(
				c.NewCategoryCond(3),
				c.NewCategoryCond(4),
				c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(7)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule1, rule2})
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      true,
	})

	if catFilter.FilterTables.BuilderMetrics.AndOrSetsInlined != 1 {
		t.Fatalf("failed AndOrSetsInlined")
	}

	if catFilter.FilterTables.BuilderMetrics.OrSetsInlined != 2 {
		t.Fatalf("failed OrSetsInlined")
	}

	matches := catFilter.MatchEvent([]types.Category{1, 2, 3, 4, 5, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	matches = catFilter.MatchEvent([]types.Category{1, 2, 3, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	matches = catFilter.MatchEvent([]types.Category{1, 2, 4, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	matches = catFilter.MatchEvent([]types.Category{1, 2, 5, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	catFilter.PrintMetrics()
}

func TestFilterOpt4(t *testing.T) {
	rule1 := c.NewRule(
		1,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(
				c.NewCategoryCond(3),
				c.NewCategoryCond(4),
				c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(6)),
		))
	rule2 := c.NewRule(
		2,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(
				c.NewCategoryCond(3),
				c.NewCategoryCond(4),
				c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(7)),
		))
	rule3 := c.NewRule(
		3,
		c.NewAndCond(
			c.NewOrCond(c.NewCategoryCond(1)),
			c.NewOrCond(c.NewCategoryCond(2)),
			c.NewOrCond(
				c.NewCategoryCond(3),
				c.NewCategoryCond(5)),
			c.NewOrCond(c.NewCategoryCond(8)),
		))
	repo := c.NewRuleRepo([]*c.Rule{rule1, rule2, rule3})
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      true,
	})

	matches := catFilter.MatchEvent([]types.Category{1, 2, 3, 4, 5, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	matches = catFilter.MatchEvent([]types.Category{1, 2, 3, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	matches = catFilter.MatchEvent([]types.Category{1, 2, 4, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	matches = catFilter.MatchEvent([]types.Category{1, 2, 5, 6})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	matches = catFilter.MatchEvent([]types.Category{1, 2, 5, 8})
	if len(matches) != 1 {
		t.Fatalf("failed number of matches %d != 1", len(matches))
	}

	catFilter.PrintMetrics()
}
