package tests

import (
	"github.com/atlasgurus/rulestone/cateng"
	c "github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
	"testing"
)

func TestBenchmarkIssue1(t *testing.T) {
	ruleSlice := [][][]types.Category{
		{{1}, {2}},
		{{1}, {2}, {3}},
	}

	repo := c.AndOrTablesToRuleRepo(ruleSlice)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      true,
	})

	matches := catFilter.MatchEvent([]types.Category{1, 2})
	if len(matches) != 1 {
		t.Errorf("failed number of matches %d != 1", len(matches))
	}

	catFilter.PrintMetrics()
}

func TestBenchmarkIssue2(t *testing.T) {
	ruleSlice := [][][]types.Category{
		{{4}, {5}, {1, 200, 104, 106, 139, 89, 122, 156, 158, 190}, {6}},
		{{5}, {6}},
		{{4}, {32, 34, 195, 164, 138, 14, 86, 27, 91, 157}, {30}, {3}, {2}},
		{{4}, {5}, {3}, {32, 9, 142, 16, 148, 88, 89, 188, 30, 191}, {2}},
		{{4}, {3, 156, 5}, {6}},
		{{4}, {5}, {2}, {33, 11, 116, 86, 152, 88, 26, 158, 191}},
		{{4}, {2}, {95}, {33}, {3, 67, 71, 80, 20, 182, 87, 94, 127}},
		{{5}, {6}, {77}, {2, 3, 165, 41, 106, 45, 152, 184, 91, 59}},
		{{4}, {2}, {6}, {3}},
		{{4}, {5}, {2, 3, 6}},
		{{4}, {2}},
		{{4}, {3}, {34, 137, 108, 44, 141, 79, 147, 60, 92}, {42}},
		{{4}, {42}, {3}},
		{{2}, {129, 163, 5, 38, 6, 75, 12, 107, 46, 112, 90, 95}, {3}},
		{{4}, {5}, {3}, {64, 2, 197, 134, 138, 42, 178, 21, 24, 154, 95}},
		{{59}, {32, 5, 168, 9, 169, 143, 144, 179, 23, 24, 155}, {6}, {77}},
		{{4}, {162, 67, 3, 111, 147, 52, 184, 157, 29}, {6}, {156}},
		{{4}, {5}, {2}, {192, 98, 99, 68, 72, 138, 183, 185, 154, 95}, {33}},
		{{5}, {133, 40, 13, 141, 77, 17, 148, 52, 25, 187}, {6}},
	}

	repo := c.AndOrTablesToRuleRepo(ruleSlice)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      true,
	})

	matches := catFilter.MatchEvent([]types.Category{42, 3, 4})
	if len(matches) != 1 {
		t.Errorf("failed number of matches %d != 1", len(matches))
	}

	catFilter.PrintMetrics()
}

func TestBenchmarkIssue3(t *testing.T) {
	ruleSlice := [][][]types.Category{
		{{1, 2, 3}, {4}},
		{{1, 2, 3}, {5}},
	}

	repo := c.AndOrTablesToRuleRepo(ruleSlice)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      true,
	})

	matches := catFilter.MatchEvent([]types.Category{1, 2, 3, 4})
	if len(matches) != 1 {
		t.Errorf("failed number of matches %d != 1", len(matches))
	}

	catFilter.PrintMetrics()
}

func TestBenchmarkIssue4(t *testing.T) {
	ruleSlice := [][][]types.Category{
		{{1, 2, 3, 4}},
		{{1, 2, 3, 5}},
	}

	repo := c.AndOrTablesToRuleRepo(ruleSlice)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      true,
	})

	matches := catFilter.MatchEvent([]types.Category{42, 7, 4})
	if len(matches) != 1 {
		t.Errorf("failed number of matches %d != 1", len(matches))
	}

	catFilter.PrintMetrics()
}

func TestBenchmarkIssue5(t *testing.T) {
	ruleSlice := [][][]types.Category{
		{{1}, {2}, {3}},
		{{1}, {2}, {4}},
	}

	repo := c.AndOrTablesToRuleRepo(ruleSlice)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      true,
	})

	matches := catFilter.MatchEvent([]types.Category{1, 2, 4})
	if len(matches) != 1 {
		t.Errorf("failed number of matches %d != 1", len(matches))
	}

	catFilter.PrintMetrics()
}
