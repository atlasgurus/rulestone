package tests

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
)

// Regression tests for numeric threshold collisions.  Before the fix,
// FloatOperand.GetHash truncated to the integer part and the Equals methods
// compared only hashes, so any two comparisons on one attribute with the same
// operator and constants sharing an integer part were treated as one condition.
// Every later comparison then reused the first one's evaluator and constant.

// matchExpressions loads the given expressions as one rule each, in the order
// given, builds a single engine and returns the expressions whose rule matched.
func matchExpressions(t *testing.T, expressions []string, event map[string]interface{}) []string {
	t.Helper()

	var yaml strings.Builder
	for i, expr := range expressions {
		fmt.Fprintf(&yaml, "- metadata: {id: \"%d\"}\n  expression: '%s'\n", i, expr)
	}

	repo := engine.NewRuleEngineRepo()
	if _, err := repo.LoadRulesFromString(yaml.String()); err != nil {
		t.Fatalf("failed to load rules %v: %v", expressions, err)
	}
	ruleEngine, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("failed to build engine for %v: %v", expressions, err)
	}

	var matched []string
	for _, ruleID := range ruleEngine.MatchEvent(event) {
		definition := ruleEngine.GetRuleDefinition(uint(ruleID))
		if definition == nil {
			t.Fatalf("no definition for matched rule %d", ruleID)
		}
		index := 0
		if _, err := fmt.Sscanf(definition.Metadata["id"].(string), "%d", &index); err != nil {
			t.Fatalf("unexpected rule id %v", definition.Metadata["id"])
		}
		matched = append(matched, expressions[index])
	}
	sort.Strings(matched)
	return matched
}

func assertMatches(t *testing.T, expressions []string, event map[string]interface{}, want []string) {
	t.Helper()
	expected := append([]string(nil), want...)
	sort.Strings(expected)
	got := matchExpressions(t, expressions, event)
	if strings.Join(got, " | ") != strings.Join(expected, " | ") {
		t.Errorf("rules %v on event %v:\n  want match: %v\n  got match:  %v",
			expressions, event, expected, got)
	}
}

// TestFloatOperandHash covers the hashing fault directly.
func TestFloatOperandHash(t *testing.T) {
	t.Run("distinct constants with the same integer part hash differently", func(t *testing.T) {
		values := []float64{0, 0.08, 0.12, 0.2, 0.5, 0.7, 0.95, 0.97, 0.999999,
			1, 1.5, -0.5, -0.95, 100, 100.25}
		seen := make(map[uint64]float64, len(values))
		for _, value := range values {
			hash := condition.FloatOperand(value).GetHash()
			if previous, collides := seen[hash]; collides {
				t.Errorf("%v and %v share hash %d", previous, value, hash)
			}
			seen[hash] = value
		}
	})

	t.Run("plus and minus zero hash alike", func(t *testing.T) {
		positive := condition.FloatOperand(0).GetHash()
		negative := condition.FloatOperand(math.Copysign(0, -1)).GetHash()
		if positive != negative {
			t.Errorf("+0 hashes to %d but -0 hashes to %d; they compare equal so must hash alike",
				positive, negative)
		}
	})

	t.Run("hashes spread over the low bits", func(t *testing.T) {
		// The hashmap and hashset pick a bucket with hash & (capacity-1), so a
		// hash that only varies in its high bits degrades every lookup to a
		// linear scan.  The raw IEEE-754 bits of these values differ only above
		// bit 51.
		buckets := make(map[uint64]bool)
		for _, value := range []float64{0, 0.25, 0.5, 1, 2, 4, 100, 1024} {
			buckets[condition.FloatOperand(value).GetHash()&0x7] = true
		}
		if len(buckets) < 4 {
			t.Errorf("round values landed in only %d of 8 buckets", len(buckets))
		}
	})
}

// TestNumericThresholdCollisions covers the reported cases.  Each pair is run
// in both load orders: the defect was order dependent, producing a false
// positive in one order and a lost rule in the other.
func TestNumericThresholdCollisions(t *testing.T) {
	cases := []struct {
		name  string
		rules []string
		event map[string]interface{}
		want  []string
	}{
		{
			name:  "greater or equal thresholds in [0,1)",
			rules: []string{"x >= 0.95", "x >= 0.97"},
			event: map[string]interface{}{"x": 0.96},
			want:  []string{"x >= 0.95"},
		},
		{
			name:  "integer and decimal constant",
			rules: []string{"x > 0", "x > 0.5"},
			event: map[string]interface{}{"x": 0.3},
			want:  []string{"x > 0"},
		},
		{
			name:  "equality against distinct constants",
			rules: []string{"x == 0.5", "x == 0.7"},
			event: map[string]interface{}{"x": 0.7},
			want:  []string{"x == 0.7"},
		},
		{
			name:  "arithmetic operand",
			rules: []string{"x * 0.5 > 1", "x * 0.9 > 1"},
			event: map[string]interface{}{"x": 1.5},
			want:  []string{"x * 0.9 > 1"},
		},
		{
			name:  "zero constant and null",
			rules: []string{"x == 0", "x == null"},
			event: map[string]interface{}{"x": 0.0},
			want:  []string{"x == 0"},
		},
		{
			name:  "three thresholds on one attribute",
			rules: []string{"score >= 0.08", "score >= 0.12", "score >= 0.2"},
			event: map[string]interface{}{"score": 0.15},
			want:  []string{"score >= 0.08", "score >= 0.12"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assertMatches(t, testCase.rules, testCase.event, testCase.want)

			reversed := make([]string, len(testCase.rules))
			for i, rule := range testCase.rules {
				reversed[len(testCase.rules)-1-i] = rule
			}
			assertMatches(t, reversed, testCase.event, testCase.want)
		})
	}
}

// TestCollisionInsideOneRule covers the shape that cannot be avoided by
// building one engine per rule: both thresholds live in the same rule.
func TestCollisionInsideOneRule(t *testing.T) {
	rule := "(a == 1 && x >= 0.2) || (a == 0 && x >= 0.12)"

	cases := []struct {
		event map[string]interface{}
		want  bool
	}{
		{map[string]interface{}{"a": 0, "x": 0.15}, true},  // over 0.12
		{map[string]interface{}{"a": 0, "x": 0.11}, false}, // under 0.12
		{map[string]interface{}{"a": 1, "x": 0.15}, false}, // under 0.2
		{map[string]interface{}{"a": 1, "x": 0.25}, true},  // over 0.2
	}

	for _, testCase := range cases {
		var want []string
		if testCase.want {
			want = []string{rule}
		}
		assertMatches(t, []string{rule}, testCase.event, want)
	}
}

// TestThresholdPositiveControls keeps the shapes that evaluated correctly
// before the fix, so the fix cannot regress them.
func TestThresholdPositiveControls(t *testing.T) {
	t.Run("constants with different integer parts", func(t *testing.T) {
		rules := []string{"x >= 1.5", "x >= 0.5"}
		assertMatches(t, rules, map[string]interface{}{"x": 0.75}, []string{"x >= 0.5"})
		assertMatches(t, rules, map[string]interface{}{"x": 1.75}, rules)
	})

	t.Run("different attributes", func(t *testing.T) {
		rules := []string{"x >= 0.95", "y >= 0.97"}
		assertMatches(t, rules, map[string]interface{}{"x": 0.96, "y": 0.96}, []string{"x >= 0.95"})
	})

	t.Run("equal and not equal", func(t *testing.T) {
		rules := []string{"x == 0.5", "x != 0.7"}
		assertMatches(t, rules, map[string]interface{}{"x": 0.5}, rules)
		assertMatches(t, rules, map[string]interface{}{"x": 0.7}, nil)
		assertMatches(t, rules, map[string]interface{}{"x": 0.9}, []string{"x != 0.7"})
	})

	t.Run("different operators on one attribute", func(t *testing.T) {
		rules := []string{"x > 0.5", "x >= 0.5", "x < 0.5", "x <= 0.5", "x == 0.5", "x != 0.5"}
		assertMatches(t, rules, map[string]interface{}{"x": 0.5},
			[]string{"x >= 0.5", "x <= 0.5", "x == 0.5"})
		assertMatches(t, rules, map[string]interface{}{"x": 0.75},
			[]string{"x > 0.5", "x >= 0.5", "x != 0.5"})
	})

	t.Run("identical comparisons still share one evaluator at scale", func(t *testing.T) {
		// Dedupe must keep merging the shared comparison across many rules.
		var rules []string
		for i := 0; i < 200; i++ {
			rules = append(rules, fmt.Sprintf("score >= 0.9 && tier == %d", i))
		}
		assertMatches(t, rules, map[string]interface{}{"score": 0.95, "tier": 7},
			[]string{"score >= 0.9 && tier == 7"})
		assertMatches(t, rules, map[string]interface{}{"score": 0.85, "tier": 7}, nil)
	})

	t.Run("identical comparisons still share one evaluator", func(t *testing.T) {
		// Dedupe is still expected to merge genuinely identical comparisons.
		rules := []string{"x >= 0.95", "x >= 0.95 && y == 1"}
		assertMatches(t, rules, map[string]interface{}{"x": 0.96, "y": 1}, rules)
		assertMatches(t, rules, map[string]interface{}{"x": 0.96, "y": 2}, []string{"x >= 0.95"})
		assertMatches(t, rules, map[string]interface{}{"x": 0.94, "y": 1}, nil)
	})
}

// TestThresholdDifferential compares the engine against a plain Go evaluator
// over random thresholds in [0,1) and random events.
func TestThresholdDifferential(t *testing.T) {
	random := rand.New(rand.NewSource(20261023))

	// Two decimal places keeps the YAML round trip exact and keeps every
	// threshold inside the integer part that used to collapse to 0.
	threshold := func() float64 { return float64(random.Intn(100)) / 100 }

	operators := []struct {
		text  string
		apply func(left, right float64) bool
	}{
		{">", func(l, r float64) bool { return l > r }},
		{">=", func(l, r float64) bool { return l >= r }},
		{"<", func(l, r float64) bool { return l < r }},
		{"<=", func(l, r float64) bool { return l <= r }},
		{"==", func(l, r float64) bool { return l == r }},
		{"!=", func(l, r float64) bool { return l != r }},
	}

	t.Run("one comparison per rule", func(t *testing.T) {
		for round := 0; round < 40; round++ {
			type comparison struct {
				text     string
				operator int
				constant float64
			}

			var rules []string
			var comparisons []comparison
			seen := make(map[string]bool)
			for len(comparisons) < 8 {
				operator := random.Intn(len(operators))
				constant := threshold()
				text := fmt.Sprintf("x %s %v", operators[operator].text, constant)
				if seen[text] {
					continue
				}
				seen[text] = true
				rules = append(rules, text)
				comparisons = append(comparisons, comparison{text, operator, constant})
			}

			for event := 0; event < 10; event++ {
				value := threshold()
				var want []string
				for _, c := range comparisons {
					if operators[c.operator].apply(value, c.constant) {
						want = append(want, c.text)
					}
				}
				assertMatches(t, rules, map[string]interface{}{"x": value}, want)
			}
		}
	})

	t.Run("negative and out-of-range constants", func(t *testing.T) {
		// Negative constants used to hash through an implementation-defined
		// float-to-uint64 conversion, so they were excluded from the original
		// report.  Hashing the bit pattern makes them well defined.
		wide := func() float64 { return float64(random.Intn(500)-250) / 100 }

		for round := 0; round < 40; round++ {
			type comparison struct {
				text     string
				operator int
				constant float64
			}

			var rules []string
			var comparisons []comparison
			seen := make(map[string]bool)
			for len(comparisons) < 10 {
				operator := random.Intn(len(operators))
				constant := wide()
				text := fmt.Sprintf("x %s %v", operators[operator].text, constant)
				if seen[text] {
					continue
				}
				seen[text] = true
				rules = append(rules, text)
				comparisons = append(comparisons, comparison{text, operator, constant})
			}

			for event := 0; event < 8; event++ {
				value := wide()
				var want []string
				for _, c := range comparisons {
					if operators[c.operator].apply(value, c.constant) {
						want = append(want, c.text)
					}
				}
				assertMatches(t, rules, map[string]interface{}{"x": value}, want)
			}
		}
	})

	t.Run("several comparisons inside one rule", func(t *testing.T) {
		for round := 0; round < 40; round++ {
			// group == i selects the i-th threshold, so each branch's
			// comparison is reachable on its own.
			var branches []string
			var constants []float64
			var operatorIndexes []int
			for i := 0; i < 4; i++ {
				operator := random.Intn(len(operators))
				constant := threshold()
				constants = append(constants, constant)
				operatorIndexes = append(operatorIndexes, operator)
				branches = append(branches,
					fmt.Sprintf("(group == %d && x %s %v)", i, operators[operator].text, constant))
			}
			rule := strings.Join(branches, " || ")

			for group := 0; group < 4; group++ {
				for event := 0; event < 5; event++ {
					value := threshold()
					var want []string
					if operators[operatorIndexes[group]].apply(value, constants[group]) {
						want = []string{rule}
					}
					assertMatches(t, []string{rule},
						map[string]interface{}{"group": group, "x": value}, want)
				}
			}
		}
	})
}

// TestManyThresholdsOnOneAttribute is the shape the report calls the normal
// case rather than an edge case: a model score field where every pair of
// thresholds used to collide.
func TestManyThresholdsOnOneAttribute(t *testing.T) {
	var rules []string
	var cutoffs []float64
	for i := 0; i < 100; i++ {
		cutoff := float64(i) / 100
		cutoffs = append(cutoffs, cutoff)
		rules = append(rules, fmt.Sprintf("score >= %v", cutoff))
	}

	for _, value := range []float64{0, 0.005, 0.37, 0.5, 0.995, 1} {
		var want []string
		for i, cutoff := range cutoffs {
			if value >= cutoff {
				want = append(want, rules[i])
			}
		}
		assertMatches(t, rules, map[string]interface{}{"score": value}, want)
	}
}
