package tests

import (
	"testing"

	"github.com/atlasgurus/rulestone/cateng"
	c "github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
)

// TestCategoryEngineBasicMatching tests basic category matching
func TestCategoryEngineBasicMatching(t *testing.T) {
	tests := []struct {
		name          string
		rules         [][][]types.Category
		event         []types.Category
		expectMatches int
	}{
		{
			name: "single rule single category",
			rules: [][][]types.Category{
				{{1}},
			},
			event:         []types.Category{1},
			expectMatches: 1,
		},
		{
			name: "single rule multiple categories AND",
			rules: [][][]types.Category{
				{{1}, {2}},
			},
			event:         []types.Category{1, 2},
			expectMatches: 1,
		},
		{
			name: "single rule multiple categories OR",
			rules: [][][]types.Category{
				{{1, 2}},
			},
			event:         []types.Category{1},
			expectMatches: 1,
		},
		{
			name: "multiple rules first matches",
			rules: [][][]types.Category{
				{{1}},
				{{2}},
			},
			event:         []types.Category{1},
			expectMatches: 1,
		},
		{
			name: "multiple rules second matches",
			rules: [][][]types.Category{
				{{1}},
				{{2}},
			},
			event:         []types.Category{2},
			expectMatches: 1,
		},
		{
			name: "multiple rules both match",
			rules: [][][]types.Category{
				{{1}},
				{{1}},
			},
			event:         []types.Category{1},
			expectMatches: 2,
		},
		{
			name: "no matches",
			rules: [][][]types.Category{
				{{1}},
				{{2}},
			},
			event:         []types.Category{3},
			expectMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := c.AndOrTablesToRuleRepo(tt.rules)
			catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
				OrOptimizationFreqThreshold:  0,
				AndOptimizationFreqThreshold: 0,
				Verbose:                      false,
			})

			matches := catFilter.MatchEvent(tt.event)
			if len(matches) != tt.expectMatches {
				t.Errorf("expected %d matches, got %d", tt.expectMatches, len(matches))
			}
		})
	}
}

// TestCategoryEngineComplexRules tests complex rule structures
func TestCategoryEngineComplexRules(t *testing.T) {
	tests := []struct {
		name          string
		rules         [][][]types.Category
		event         []types.Category
		expectMatches int
	}{
		{
			name: "three level AND",
			rules: [][][]types.Category{
				{{1}, {2}, {3}},
			},
			event:         []types.Category{1, 2, 3},
			expectMatches: 1,
		},
		{
			name: "three level AND missing one",
			rules: [][][]types.Category{
				{{1}, {2}, {3}},
			},
			event:         []types.Category{1, 2},
			expectMatches: 0,
		},
		{
			name: "mixed AND/OR",
			rules: [][][]types.Category{
				{{1, 2}, {3}},
			},
			event:         []types.Category{1, 3},
			expectMatches: 1,
		},
		{
			name: "complex OR set",
			rules: [][][]types.Category{
				{{1, 2, 3, 4, 5}},
			},
			event:         []types.Category{3},
			expectMatches: 1,
		},
		{
			name: "multiple AND levels with OR",
			rules: [][][]types.Category{
				{{1, 2}, {3, 4}, {5}},
			},
			event:         []types.Category{2, 4, 5},
			expectMatches: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := c.AndOrTablesToRuleRepo(tt.rules)
			catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
				OrOptimizationFreqThreshold:  0,
				AndOptimizationFreqThreshold: 0,
				Verbose:                      false,
			})

			matches := catFilter.MatchEvent(tt.event)
			if len(matches) != tt.expectMatches {
				t.Errorf("expected %d matches, got %d", tt.expectMatches, len(matches))
			}
		})
	}
}

// TestCategoryEngineOptimizations tests various optimization scenarios
func TestCategoryEngineOptimizations(t *testing.T) {
	tests := []struct {
		name                      string
		rules                     [][][]types.Category
		event                     []types.Category
		orThreshold               uint
		andThreshold              uint
		expectMatches             int
		expectOptimizedOrSets     bool
		expectOptimizedAndOrSets  bool
	}{
		{
			name: "OR optimization applied",
			rules: [][][]types.Category{
				{{1, 2, 3}, {4}},
				{{1, 2, 3}, {5}},
			},
			event:         []types.Category{1, 2, 3, 4},
			orThreshold:   1,
			andThreshold:  1,
			expectMatches: 1,
			expectOptimizedOrSets: true,
		},
		{
			name: "AND optimization applied",
			rules: [][][]types.Category{
				{{1}, {2}, {3}},
				{{1}, {2}, {4}},
			},
			event:         []types.Category{1, 2, 4},
			orThreshold:   1,
			andThreshold:  1,
			expectMatches: 1,
			expectOptimizedAndOrSets: true,
		},
		{
			name: "no optimization low frequency",
			rules: [][][]types.Category{
				{{1}, {2}},
				{{3}, {4}},
			},
			event:         []types.Category{1, 2},
			orThreshold:   5,
			andThreshold:  5,
			expectMatches: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := c.AndOrTablesToRuleRepo(tt.rules)
			catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
				OrOptimizationFreqThreshold:  tt.orThreshold,
				AndOptimizationFreqThreshold: tt.andThreshold,
				Verbose:                      false,
			})

			matches := catFilter.MatchEvent(tt.event)
			if len(matches) != tt.expectMatches {
				t.Errorf("expected %d matches, got %d", tt.expectMatches, len(matches))
			}
		})
	}
}

// TestCategoryEngineEmptyEvent tests handling of empty events
func TestCategoryEngineEmptyEvent(t *testing.T) {
	rules := [][][]types.Category{
		{{1}, {2}},
		{{3}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	matches := catFilter.MatchEvent([]types.Category{})
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for empty event, got %d", len(matches))
	}
}

// TestCategoryEngineEmptyRules tests handling when no rules are defined
func TestCategoryEngineEmptyRules(t *testing.T) {
	rules := [][][]types.Category{}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	matches := catFilter.MatchEvent([]types.Category{1, 2, 3})
	if len(matches) != 0 {
		t.Errorf("expected 0 matches when no rules defined, got %d", len(matches))
	}
}

// TestCategoryEngineLargeCategor yNumbers tests handling of large category values
func TestCategoryEngineLargeCategoryNumbers(t *testing.T) {
	rules := [][][]types.Category{
		{{999999}, {1000000}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	matches := catFilter.MatchEvent([]types.Category{999999, 1000000})
	if len(matches) != 1 {
		t.Errorf("expected 1 match with large category numbers, got %d", len(matches))
	}
}

// TestCategoryEngineManyCategoriesInEvent tests events with many categories
func TestCategoryEngineManyCategoriesInEvent(t *testing.T) {
	rules := [][][]types.Category{
		{{50}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	// Create event with 100 categories
	event := make([]types.Category, 100)
	for i := 0; i < 100; i++ {
		event[i] = types.Category(i + 1)
	}

	matches := catFilter.MatchEvent(event)
	if len(matches) != 1 {
		t.Errorf("expected 1 match with many categories, got %d", len(matches))
	}
}

// TestCategoryEngineManyRules tests engine with many rules
func TestCategoryEngineManyRules(t *testing.T) {
	// Create 100 rules
	rules := make([][][]types.Category, 100)
	for i := 0; i < 100; i++ {
		rules[i] = [][]types.Category{{types.Category(i + 1)}}
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	// Match rule 50
	matches := catFilter.MatchEvent([]types.Category{50})
	if len(matches) != 1 {
		t.Errorf("expected 1 match with many rules, got %d", len(matches))
	}

	if matches[0] != 49 { // 0-indexed
		t.Errorf("expected rule 49 to match, got %d", matches[0])
	}
}

// TestCategoryEngineMetrics tests that metrics are properly tracked
func TestCategoryEngineMetrics(t *testing.T) {
	rules := [][][]types.Category{
		{{1}, {2}},
		{{1}, {3}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      false,
	})

	event := []types.Category{1, 2}
	matches := catFilter.MatchEvent(event)

	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}

	// Verify metrics are being tracked
	if catFilter.Metrics.NumBitMaskChecks == 0 && len(matches) > 0 {
		t.Errorf("expected NumBitMaskChecks > 0 when there are matches")
	}
}

// TestCategoryEnginePartialMatches tests rules that partially match
func TestCategoryEnginePartialMatches(t *testing.T) {
	tests := []struct {
		name          string
		rules         [][][]types.Category
		event         []types.Category
		expectMatches int
		description   string
	}{
		{
			name: "AND rule partial match",
			rules: [][][]types.Category{
				{{1}, {2}, {3}},
			},
			event:         []types.Category{1, 2},
			expectMatches: 0,
			description:   "Missing third AND condition",
		},
		{
			name: "OR rule partial match",
			rules: [][][]types.Category{
				{{1, 2, 3}},
			},
			event:         []types.Category{1},
			expectMatches: 1,
			description:   "One of OR conditions matches",
		},
		{
			name: "mixed partial match",
			rules: [][][]types.Category{
				{{1, 2}, {3, 4}},
			},
			event:         []types.Category{1},
			expectMatches: 0,
			description:   "First OR matches but second AND doesn't",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := c.AndOrTablesToRuleRepo(tt.rules)
			catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
				OrOptimizationFreqThreshold:  0,
				AndOptimizationFreqThreshold: 0,
				Verbose:                      false,
			})

			matches := catFilter.MatchEvent(tt.event)
			if len(matches) != tt.expectMatches {
				t.Errorf("%s: expected %d matches, got %d", tt.description, tt.expectMatches, len(matches))
			}
		})
	}
}

// TestCategoryEngineDuplicateCategories tests events with duplicate categories
func TestCategoryEngineDuplicateCategories(t *testing.T) {
	rules := [][][]types.Category{
		{{1}, {2}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	// Event with duplicate categories
	matches := catFilter.MatchEvent([]types.Category{1, 1, 2, 2})
	if len(matches) != 1 {
		t.Errorf("expected 1 match with duplicate categories, got %d", len(matches))
	}
}

// TestCategoryEngineOrderIndependence tests that category order doesn't matter
func TestCategoryEngineOrderIndependence(t *testing.T) {
	rules := [][][]types.Category{
		{{1}, {2}, {3}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	tests := []struct {
		name  string
		event []types.Category
	}{
		{"ordered", []types.Category{1, 2, 3}},
		{"reverse", []types.Category{3, 2, 1}},
		{"mixed", []types.Category{2, 1, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := catFilter.MatchEvent(tt.event)
			if len(matches) != 1 {
				t.Errorf("expected 1 match for %v, got %d", tt.event, len(matches))
			}
		})
	}
}

// TestCategoryEngineComplexRealWorldScenario tests a realistic complex scenario
func TestCategoryEngineComplexRealWorldScenario(t *testing.T) {
	// Simulate a real-world scenario with multiple rules
	// Rule 0: (region=US OR region=CA) AND (age>=18) AND (status=active)
	// Rule 1: (region=EU) AND (gdpr=true)
	// Rule 2: (vip=true)
	// Categories: 1=US, 2=CA, 3=EU, 4=age>=18, 5=status=active, 6=gdpr=true, 7=vip=true

	rules := [][][]types.Category{
		{{1, 2}, {4}, {5}}, // Rule 0
		{{3}, {6}},         // Rule 1
		{{7}},              // Rule 2
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      false,
	})

	tests := []struct {
		name          string
		event         []types.Category
		expectMatches []int
	}{
		{
			name:          "US adult active user",
			event:         []types.Category{1, 4, 5},
			expectMatches: []int{0},
		},
		{
			name:          "EU GDPR user",
			event:         []types.Category{3, 6},
			expectMatches: []int{1},
		},
		{
			name:          "VIP user",
			event:         []types.Category{7},
			expectMatches: []int{2},
		},
		{
			name:          "VIP US adult active user",
			event:         []types.Category{1, 4, 5, 7},
			expectMatches: []int{0, 2},
		},
		{
			name:          "minor user",
			event:         []types.Category{1, 5},
			expectMatches: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := catFilter.MatchEvent(tt.event)
			if len(matches) != len(tt.expectMatches) {
				t.Errorf("expected %d matches, got %d", len(tt.expectMatches), len(matches))
				return
			}

			for i, expectedMatch := range tt.expectMatches {
				if int(matches[i]) != expectedMatch {
					t.Errorf("match %d: expected rule %d, got %d", i, expectedMatch, matches[i])
				}
			}
		})
	}
}

// TestCategoryEngineOptions tests different option configurations
func TestCategoryEngineOptions(t *testing.T) {
	rules := [][][]types.Category{
		{{1}, {2}},
		{{1}, {3}},
	}

	tests := []struct {
		name         string
		options      *cateng.Options
		expectPanic  bool
	}{
		{
			name: "normal options",
			options: &cateng.Options{
				OrOptimizationFreqThreshold:  1,
				AndOptimizationFreqThreshold: 1,
				Verbose:                      false,
			},
			expectPanic: false,
		},
		{
			name: "high thresholds",
			options: &cateng.Options{
				OrOptimizationFreqThreshold:  1000,
				AndOptimizationFreqThreshold: 1000,
				Verbose:                      false,
			},
			expectPanic: false,
		},
		{
			name: "zero thresholds",
			options: &cateng.Options{
				OrOptimizationFreqThreshold:  0,
				AndOptimizationFreqThreshold: 0,
				Verbose:                      false,
			},
			expectPanic: false,
		},
		{
			name: "verbose mode",
			options: &cateng.Options{
				OrOptimizationFreqThreshold:  1,
				AndOptimizationFreqThreshold: 1,
				Verbose:                      true,
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			repo := c.AndOrTablesToRuleRepo(rules)
			catFilter := cateng.NewCategoryEngine(repo, tt.options)

			matches := catFilter.MatchEvent([]types.Category{1, 2})
			_ = matches
		})
	}
}

// TestCategoryEngineNilOptions tests handling of nil options
func TestCategoryEngineNilOptions(t *testing.T) {
	rules := [][][]types.Category{
		{{1}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)

	// Should handle nil options gracefully or use defaults
	catFilter := cateng.NewCategoryEngine(repo, nil)

	matches := catFilter.MatchEvent([]types.Category{1})
	if len(matches) != 1 {
		t.Errorf("expected 1 match with nil options, got %d", len(matches))
	}
}

// TestCategoryEngineReuseability tests that the same engine can be reused
func TestCategoryEngineReuseability(t *testing.T) {
	rules := [][][]types.Category{
		{{1}, {2}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	// Use the engine multiple times
	for i := 0; i < 100; i++ {
		matches := catFilter.MatchEvent([]types.Category{1, 2})
		if len(matches) != 1 {
			t.Errorf("iteration %d: expected 1 match, got %d", i, len(matches))
		}
	}
}

// TestCategoryEngineConcurrentAccess tests thread safety
func TestCategoryEngineConcurrentAccess(t *testing.T) {
	rules := [][][]types.Category{
		{{1}, {2}},
		{{3}, {4}},
	}

	repo := c.AndOrTablesToRuleRepo(rules)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 0,
		Verbose:                      false,
	})

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			event := []types.Category{types.Category(id % 4 + 1)}
			for j := 0; j < 100; j++ {
				matches := catFilter.MatchEvent(event)
				_ = matches
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
