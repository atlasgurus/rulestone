package tests

import (
	"testing"

	"github.com/atlasgurus/rulestone/cateng"
	c "github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/engine"
	"github.com/atlasgurus/rulestone/types"
	"github.com/atlasgurus/rulestone/utils"
)

// Benchmark rule registration from file
func BenchmarkRuleRegistration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		repo := engine.NewRuleEngineRepo()
		_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
		if err != nil {
			b.Fatalf("failed RegisterRulesFromFile: %v", err)
		}
	}
}

// Benchmark engine creation
func BenchmarkEngineCreation(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.NewRuleEngine(repo)
		if err != nil {
			b.Fatalf("failed NewRuleEngine: %v", err)
		}
	}
}

// Benchmark simple expression evaluation
func BenchmarkSimpleExpressionEval(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test0.json")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark complex expression with nested attributes
func BenchmarkComplexExpressionEval(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test2.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test2.json")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark expression with logical operators
func BenchmarkLogicalOperatorsEval(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test3.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test3.json")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark forAll condition
func BenchmarkForAllCondition(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_for_each_1.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_for_each_test1.yaml")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark forSome condition with large array
func BenchmarkForSomeConditionLargeArray(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_for_each_test2.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_for_each_test2.json")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark multiple rules per file
func BenchmarkMultipleRulesPerFile(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/multiple_rules_per_file_test.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_multiple_rules_per_file_test0.json")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark category engine with simple rules
func BenchmarkCategoryEngineSimple(b *testing.B) {
	ruleSlice := [][][]types.Category{
		{{1}, {2}},
		{{1}, {2}, {3}},
	}

	repo := c.AndOrTablesToRuleRepo(ruleSlice)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      false,
	})

	event := []types.Category{1, 2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := catFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark category engine with complex rules
func BenchmarkCategoryEngineComplex(b *testing.B) {
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
		Verbose:                      false,
	})

	event := []types.Category{42, 3, 4}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := catFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark category engine with OR optimization
func BenchmarkCategoryEngineOrOptimization(b *testing.B) {
	ruleSlice := [][][]types.Category{
		{{1, 2, 3}, {4}},
		{{1, 2, 3}, {5}},
	}

	repo := c.AndOrTablesToRuleRepo(ruleSlice)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      false,
	})

	event := []types.Category{1, 2, 3, 4}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := catFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark category engine with AND optimization
func BenchmarkCategoryEngineAndOptimization(b *testing.B) {
	ruleSlice := [][][]types.Category{
		{{1}, {2}, {3}},
		{{1}, {2}, {4}},
	}

	repo := c.AndOrTablesToRuleRepo(ruleSlice)
	catFilter := cateng.NewCategoryEngine(repo, &cateng.Options{
		OrOptimizationFreqThreshold:  1,
		AndOptimizationFreqThreshold: 1,
		Verbose:                      false,
	})

	event := []types.Category{1, 2, 4}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := catFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark common expression elimination optimization
func BenchmarkCommonExpressionElimination(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test6.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test6.json")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark no match scenario (important for performance)
func BenchmarkNoMatch(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile("../examples/rules/rule_expression_test0.yaml")
	if err != nil {
		b.Fatalf("failed RegisterRulesFromFile: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	// Create event that won't match
	event, err := utils.ReadEvent("../examples/data/data_expression_test1.json")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) > 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark with multiple rules (scalability test)
func BenchmarkManyRules(b *testing.B) {
	repo := engine.NewRuleEngineRepo()

	// Register multiple rule files to simulate many rules
	ruleFiles := []string{
		"../examples/rules/rule_expression_test0.yaml",
		"../examples/rules/rule_expression_test1.yaml",
		"../examples/rules/rule_expression_test2.yaml",
		"../examples/rules/rule_expression_test3.yaml",
		"../examples/rules/rule_expression_test4.yaml",
		"../examples/rules/rule_expression_test5.yaml",
	}

	for _, ruleFile := range ruleFiles {
		_, err := repo.RegisterRulesFromFile(ruleFile)
		if err != nil {
			b.Fatalf("failed RegisterRulesFromFile %s: %v", ruleFile, err)
		}
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event, err := utils.ReadEvent("../examples/data/data_expression_test0.json")
	if err != nil {
		b.Fatalf("failed ReadEvent: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		_ = matches // Don't check count as multiple rules may match
	}
}

// Benchmark null check performance (Bug #2 fix)
func BenchmarkNullCheck(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	rules := `- metadata: {id: null-check}
  expression: field == null`
	_, err := repo.LoadRulesFromString(rules, engine.WithValidate(true))
	if err != nil {
		b.Fatalf("failed LoadRulesFromString: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event := map[string]interface{}{} // Missing field

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark constant expression performance (Bug #1 fix)
func BenchmarkConstantExpression(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	rules := `- metadata: {id: constant-expr}
  expression: 1 == 1`
	_, err := repo.LoadRulesFromString(rules, engine.WithValidate(true))
	if err != nil {
		b.Fatalf("failed LoadRulesFromString: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event := map[string]interface{}{} // Empty event

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark forAll with empty array (Bug #3 fix)
func BenchmarkForAllEmptyArray(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	rules := `- metadata: {id: forall-empty}
  expression: forAll("items", "item", item.value > 100)`
	_, err := repo.LoadRulesFromString(rules, engine.WithValidate(true))
	if err != nil {
		b.Fatalf("failed LoadRulesFromString: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	event := map[string]interface{}{"items": []interface{}{}} // Empty array

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}

// Benchmark forAll with non-empty array (ensure no regression)
func BenchmarkForAllNonEmptyArray(b *testing.B) {
	repo := engine.NewRuleEngineRepo()
	rules := `- metadata: {id: forall-nonempty}
  expression: forAll("items", "item", item.value > 50)`
	_, err := repo.LoadRulesFromString(rules, engine.WithValidate(true))
	if err != nil {
		b.Fatalf("failed LoadRulesFromString: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		b.Fatalf("failed NewRuleEngine: %v", err)
	}

	// Array with 10 matching elements
	items := make([]interface{}, 10)
	for i := 0; i < 10; i++ {
		items[i] = map[string]interface{}{"value": 100}
	}
	event := map[string]interface{}{"items": items}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) != 1 {
			b.Fatalf("unexpected match count: %d", len(matches))
		}
	}
}
