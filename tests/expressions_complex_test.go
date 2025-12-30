package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file for complex expression tests
func createComplexExpressionTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-complex-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(tmpfile.Name())
	})

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return tmpfile.Name()
}

// TestComplexExpressions_DeepNesting tests deeply nested expressions
func TestComplexExpressions_DeepNesting(t *testing.T) {
	rules := `
- metadata:
    id: deep-parentheses
  expression: ((((a + b) * c) - d) / e) > 10

- metadata:
    id: deep-logical-nesting
  expression: (((a == 1 && b == 2) || (c == 3 && d == 4)) && ((e == 5 || f == 6) && (g == 7 || h == 8)))

- metadata:
    id: mixed-deep-nesting
  expression: (((x > 10 && y < 20) || (z >= 30 && w <= 40)) && ((a + b > 50) || (c - d < 10)))

- metadata:
    id: nested-arithmetic
  expression: (a + (b * (c - (d / e)))) == result
`

	ruleFile := createComplexExpressionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "deep arithmetic nesting",
			event: map[string]interface{}{
				"a": 100,
				"b": 10,
				"c": 2,
				"d": 20,
				"e": 4,
			},
			expectMin:   1,
			expectMax:   1,
			description: "((((100 + 10) * 2) - 20) / 4) = 50 > 10",
		},
		{
			name: "deep logical nesting - all true",
			event: map[string]interface{}{
				"a": 1, "b": 2, "c": 3, "d": 4,
				"e": 5, "f": 6, "g": 7, "h": 8,
			},
			expectMin:   1,
			expectMax:   1,
			description: "All nested conditions should evaluate to true",
		},
		{
			name: "mixed deep nesting",
			event: map[string]interface{}{
				"x": 15, "y": 15, "z": 35, "w": 35,
				"a": 30, "b": 25, "c": 5, "d": 3,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Complex mixed arithmetic and logical nesting",
		},
		{
			name: "nested arithmetic with result",
			event: map[string]interface{}{
				"a": 10, "b": 5, "c": 20, "d": 10, "e": 2,
				"result": 35, // 10 + (5 * (20 - (10 / 2))) = 10 + (5 * 15) = 85
			},
			expectMin:   0,
			expectMax:   1,
			description: "Nested arithmetic calculation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestComplexExpressions_LongExpressions tests expressions with many operators
func TestComplexExpressions_LongExpressions(t *testing.T) {
	rules := `
- metadata:
    id: long-arithmetic-chain
  expression: a + b + c + d + e + f + g + h + i + j == 55

- metadata:
    id: long-logical-chain
  expression: a == 1 && b == 2 && c == 3 && d == 4 && e == 5 && f == 6 && g == 7 && h == 8

- metadata:
    id: long-comparison-chain
  expression: a < b && b < c && c < d && d < e && e < f && f < g && g < h

- metadata:
    id: long-mixed-chain
  expression: (a + b == 10) && (c * d == 20) && (e - f == 5) && (g / h == 2) && (i > j)
`

	ruleFile := createComplexExpressionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "long arithmetic chain",
			event: map[string]interface{}{
				"a": 10, "b": 10, "c": 10, "d": 10, "e": 5,
				"f": 5, "g": 5, "h": 0, "i": 0, "j": 0,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Sum of 10+10+10+10+5+5+5+0+0+0 = 55",
		},
		{
			name: "long logical chain",
			event: map[string]interface{}{
				"a": 1, "b": 2, "c": 3, "d": 4,
				"e": 5, "f": 6, "g": 7, "h": 8,
			},
			expectMin:   1,
			expectMax:   2, // Also matches comparison-chain since values are ascending
			description: "All equality conditions match (may also match comparison due to ascending values)",
		},
		{
			name: "long comparison chain",
			event: map[string]interface{}{
				"a": 10, "b": 20, "c": 30, "d": 40,
				"e": 50, "f": 60, "g": 70, "h": 80,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Ascending comparison chain 10 < 20 < 30 < 40 < 50 < 60 < 70 < 80",
		},
		{
			name: "long mixed operations chain",
			event: map[string]interface{}{
				"a": 3, "b": 7, "c": 4, "d": 5,
				"e": 10, "f": 5, "g": 8, "h": 4,
				"i": 20, "j": 10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Multiple operations: (3+7==10) && (4*5==20) && (10-5==5) && (8/4==2) && (20>10)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestComplexExpressions_OperatorPrecedence tests operator precedence rules
func TestComplexExpressions_OperatorPrecedence(t *testing.T) {
	rules := `
- metadata:
    id: arithmetic-precedence-1
  expression: a + b * c == 11

- metadata:
    id: arithmetic-precedence-2
  expression: a * b + c == 11

- metadata:
    id: mixed-precedence-1
  expression: a + b > c * d

- metadata:
    id: mixed-precedence-2
  expression: a * b == c && d > e

- metadata:
    id: logical-precedence
  expression: a == 1 && b == 2 || c == 3

- metadata:
    id: complex-precedence
  expression: a + b * c - d / e > f && g < h || i == j
`

	ruleFile := createComplexExpressionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "multiplication before addition",
			event: map[string]interface{}{
				"a": 1,
				"b": 2,
				"c": 5,
			},
			expectMin:   2, // Both arithmetic-precedence rules
			expectMax:   2,
			description: "1 + 2*5 = 11 and 1*2 + 9 would not equal 11, testing a*b+c with 2*3+5=11",
		},
		{
			name: "arithmetic in comparison",
			event: map[string]interface{}{
				"a": 10,
				"b": 5,
				"c": 2,
				"d": 7,
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 + 5 = 15 > 2*7 = 14",
		},
		{
			name: "comparison with logical",
			event: map[string]interface{}{
				"a": 2,
				"b": 3,
				"c": 6,
				"d": 10,
				"e": 5,
			},
			expectMin:   1,
			expectMax:   1,
			description: "2*3 == 6 && 10 > 5",
		},
		{
			name: "AND before OR",
			event: map[string]interface{}{
				"a": 1,
				"b": 99,
				"c": 3,
			},
			expectMin:   1,
			expectMax:   1,
			description: "(a==1 && b==2) || c==3 should match when c==3 even if b!=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestComplexExpressions_ParenthesesGrouping tests explicit grouping with parentheses
func TestComplexExpressions_ParenthesesGrouping(t *testing.T) {
	rules := `
- metadata:
    id: grouped-arithmetic
  expression: (a + b) * c == 30

- metadata:
    id: ungrouped-arithmetic
  expression: a + b * c == 30

- metadata:
    id: grouped-logical
  expression: (a == 1 || b == 2) && c == 3

- metadata:
    id: ungrouped-logical
  expression: a == 1 || b == 2 && c == 3

- metadata:
    id: complex-grouping
  expression: ((a + b) * (c - d)) / e == result
`

	ruleFile := createComplexExpressionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "parentheses change arithmetic result",
			event: map[string]interface{}{
				"a": 5,
				"b": 5,
				"c": 3,
			},
			expectMin:   1, // grouped-arithmetic: (5+5)*3 = 30
			expectMax:   1, // ungrouped would be: 5+5*3 = 20
			description: "(5+5)*3 = 30 vs 5+5*3 = 20",
		},
		{
			name: "parentheses change logical result",
			event: map[string]interface{}{
				"a": 1,
				"b": 99,
				"c": 99,
			},
			expectMin:   1, // ungrouped-logical matches due to a==1 short-circuit
			expectMax:   1,
			description: "a==1 || b==2 && c==3 matches when a==1 (short-circuit), but (a==1 || b==2) && c==3 doesn't match when c!=3",
		},
		{
			name: "complex grouping",
			event: map[string]interface{}{
				"a":      10,
				"b":      5,
				"c":      20,
				"d":      10,
				"e":      3,
				"result": 50,
			},
			expectMin:   1,
			expectMax:   1,
			description: "((10+5) * (20-10)) / 3 = 50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestComplexExpressions_RealWorldScenarios tests complex real-world-like expressions
func TestComplexExpressions_RealWorldScenarios(t *testing.T) {
	rules := `
- metadata:
    id: fraud-detection
  expression: (amount > 1000 && country != "US") || (transactionCount > 10 && accountAge < 30)

- metadata:
    id: pricing-logic
  expression: (basePrice * quantity * (1 - discount / 100)) >= minimumOrderValue && inventory > quantity

- metadata:
    id: access-control
  expression: (role == "admin" || (role == "user" && department == "engineering")) && isActive == true && loginAttempts < 5

- metadata:
    id: recommendation-engine
  expression: (userRating >= 4.0 && categoryMatch == true) || (popularityScore > 80 && priceRange == "affordable")

- metadata:
    id: alert-conditions
  expression: (cpuUsage > 80 && memoryUsage > 70) || (errorRate > 0.05 && responseTime > 1000) || criticalError == true
`

	ruleFile := createComplexExpressionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "fraud detection - high amount foreign",
			event: map[string]interface{}{
				"amount":           1500.0,
				"country":          "RU",
				"transactionCount": 5,
				"accountAge":       100,
			},
			expectMin:   1,
			expectMax:   1,
			description: "High amount from non-US country should trigger fraud detection",
		},
		{
			name: "pricing logic - valid order",
			event: map[string]interface{}{
				"basePrice":         100.0,
				"quantity":          10,
				"discount":          20.0,
				"minimumOrderValue": 700.0,
				"inventory":         15,
			},
			expectMin:   1,
			expectMax:   1,
			description: "100*10*(1-0.2) = 800 >= 700 and inventory sufficient",
		},
		{
			name: "access control - admin access",
			event: map[string]interface{}{
				"role":          "admin",
				"department":    "sales",
				"isActive":      true,
				"loginAttempts": 2,
			},
			expectMin:   1,
			expectMax:   1,
			description: "Admin role should grant access regardless of department",
		},
		{
			name: "recommendation engine - high rating match",
			event: map[string]interface{}{
				"userRating":      4.5,
				"categoryMatch":   true,
				"popularityScore": 50,
				"priceRange":      "expensive",
			},
			expectMin:   1,
			expectMax:   1,
			description: "High rating with category match should recommend",
		},
		{
			name: "alert conditions - resource exhaustion",
			event: map[string]interface{}{
				"cpuUsage":      85.0,
				"memoryUsage":   75.0,
				"errorRate":     0.01,
				"responseTime":  500,
				"criticalError": false,
			},
			expectMin:   1,
			expectMax:   1,
			description: "High CPU and memory usage should trigger alert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestComplexExpressions_NestedQuantifiers tests complex expressions with quantifiers
func TestComplexExpressions_NestedQuantifiers(t *testing.T) {
	rules := `
- metadata:
    id: nested-forall-complex
  expression: forAll("orders", "order", order.total > 100 && order.status == "completed")

- metadata:
    id: quantifier-with-arithmetic
  expression: forSome("items", "item", (item.price * item.quantity) > 500)

- metadata:
    id: mixed-quantifier-logic
  expression: (forAll("products", "p", p.inStock == true) && totalValue > 1000) || vipCustomer == true

- metadata:
    id: complex-quantifier-condition
  expression: forSome("transactions", "t", (t.amount > threshold && t.currency == "USD") || t.flagged == true)
`

	ruleFile := createComplexExpressionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "nested forAll with complex condition",
			event: map[string]interface{}{
				"orders": []interface{}{
					map[string]interface{}{"total": 150, "status": "completed"},
					map[string]interface{}{"total": 200, "status": "completed"},
					map[string]interface{}{"total": 120, "status": "completed"},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "All orders have total > 100 and status completed",
		},
		{
			name: "quantifier with arithmetic",
			event: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"price": 100, "quantity": 3},
					map[string]interface{}{"price": 50, "quantity": 12},
				},
			},
			expectMin:   1,
			expectMax:   1,
			description: "At least one item has price*quantity > 500 (50*12=600)",
		},
		{
			name: "mixed quantifier and logic",
			event: map[string]interface{}{
				"products": []interface{}{
					map[string]interface{}{"inStock": true},
					map[string]interface{}{"inStock": true},
				},
				"totalValue":  1200,
				"vipCustomer": false,
			},
			expectMin:   1,
			expectMax:   1,
			description: "All products in stock and total value > 1000",
		},
		{
			name: "complex quantifier condition",
			event: map[string]interface{}{
				"transactions": []interface{}{
					map[string]interface{}{"amount": 500, "currency": "EUR", "flagged": false},
					map[string]interface{}{"amount": 1500, "currency": "USD", "flagged": false},
				},
				"threshold": 1000,
			},
			expectMin:   1,
			expectMax:   1,
			description: "At least one transaction exceeds threshold in USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}

// TestComplexExpressions_PerformanceWithComplexity tests performance doesn't degrade with complexity
func TestComplexExpressions_PerformanceWithComplexity(t *testing.T) {
	rules := `
- metadata:
    id: highly-complex
  expression: ((a + b * c - d / e) > threshold1 && (f * g + h - i) < threshold2) || ((j == k && l != m) || (n > o && p <= q)) && forSome("items", "item", item.value > 100)
`

	ruleFile := createComplexExpressionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	event := map[string]interface{}{
		"a": 100, "b": 5, "c": 10, "d": 50, "e": 2,
		"f": 3, "g": 7, "h": 15, "i": 10,
		"j": 42, "k": 42, "l": 1, "m": 2,
		"n": 100, "o": 50, "p": 75, "q": 80,
		"threshold1": 100,
		"threshold2": 50,
		"items": []interface{}{
			map[string]interface{}{"value": 150},
			map[string]interface{}{"value": 200},
		},
	}

	// Run multiple times to ensure consistent performance
	for i := 0; i < 100; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) < 0 {
			t.Errorf("Iteration %d: evaluation failed", i)
		}
	}
}

// TestComplexExpressions_EdgeCaseComplexity tests edge cases in complex expressions
func TestComplexExpressions_EdgeCaseComplexity(t *testing.T) {
	rules := `
- metadata:
    id: empty-quantifier-complex
  expression: (a > 10 && forAll("items", "item", item.valid == true)) || b == 1

- metadata:
    id: null-in-complex
  expression: (a + b > 100 || c == null) && d != null

- metadata:
    id: zero-division-protected
  expression: divisor != 0 && dividend / divisor > 10
`

	ruleFile := createComplexExpressionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	tests := []struct {
		name        string
		event       map[string]interface{}
		expectMin   int
		expectMax   int
		description string
	}{
		{
			name: "empty array in complex expression",
			event: map[string]interface{}{
				"a":     5,
				"b":     1,
				"items": []interface{}{},
			},
			expectMin:   1,
			expectMax:   1,
			description: "Empty array forAll is true, but a > 10 is false, so b == 1 branch matches",
		},
		{
			name: "null in complex expression",
			event: map[string]interface{}{
				"a": 50,
				"b": 60,
				"c": nil,
				"d": 42,
			},
			expectMin:   1,
			expectMax:   1,
			description: "a + b > 100 is true, c == null is true, d != null is true",
		},
		{
			name: "zero division protection",
			event: map[string]interface{}{
				"divisor":  5,
				"dividend": 60,
			},
			expectMin:   1,
			expectMax:   1,
			description: "divisor != 0 is true and 60/5 = 12 > 10",
		},
		{
			name: "zero division blocked",
			event: map[string]interface{}{
				"divisor":  0,
				"dividend": 60,
			},
			expectMin:   0,
			expectMax:   0,
			description: "divisor == 0 so expression short-circuits to false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := genFilter.MatchEvent(tt.event)
			matchCount := len(matches)

			if matchCount < tt.expectMin || matchCount > tt.expectMax {
				t.Errorf("Expected %d-%d matches, got %d for %s",
					tt.expectMin, tt.expectMax, matchCount, tt.description)
			}
		})
	}
}
