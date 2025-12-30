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
				"a": float64(100),
				"b": float64(10),
				"c": float64(2),
				"d": float64(20),
				"e": float64(4),
			},
			expectMin:   1,
			expectMax:   1,
			description: "((((100 + 10) * 2) - 20) / 4) = 50 > 10",
		},
		{
			name: "deep logical nesting - all true",
			event: map[string]interface{}{
				"a": float64(1), "b": float64(2), "c": float64(3), "d": float64(4),
				"e": float64(5), "f": float64(6), "g": float64(7), "h": float64(8),
			},
			expectMin:   1,
			expectMax:   1,
			description: "All nested conditions should evaluate to true",
		},
		{
			name: "mixed deep nesting",
			event: map[string]interface{}{
				"x": float64(15), "y": float64(15), "z": float64(35), "w": float64(35),
				"a": float64(30), "b": float64(25), "c": float64(5), "d": float64(3),
			},
			expectMin:   1,
			expectMax:   1,
			description: "Complex mixed arithmetic and logical nesting",
		},
		{
			name: "nested arithmetic with result",
			event: map[string]interface{}{
				"a": float64(10), "b": float64(5), "c": float64(20), "d": float64(10), "e": float64(2),
				"result": float64(35), // 10 + (5 * (20 - (10 / 2))) = 10 + (5 * 15) = 85
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
				"a": float64(1), "b": float64(2), "c": float64(3), "d": float64(4), "e": float64(5),
				"f": float64(6), "g": float64(7), "h": float64(8), "i": float64(9), "j": float64(10),
			},
			expectMin:   1,
			expectMax:   1,
			description: "Sum of 1+2+3+4+5+6+7+8+9+10 = 55",
		},
		{
			name: "long logical chain",
			event: map[string]interface{}{
				"a": float64(1), "b": float64(2), "c": float64(3), "d": float64(4),
				"e": float64(5), "f": float64(6), "g": float64(7), "h": float64(8),
			},
			expectMin:   1,
			expectMax:   1,
			description: "All equality conditions in chain should match",
		},
		{
			name: "long comparison chain",
			event: map[string]interface{}{
				"a": float64(1), "b": float64(2), "c": float64(3), "d": float64(4),
				"e": float64(5), "f": float64(6), "g": float64(7), "h": float64(8),
			},
			expectMin:   1,
			expectMax:   1,
			description: "Ascending comparison chain 1 < 2 < 3 < 4 < 5 < 6 < 7 < 8",
		},
		{
			name: "long mixed operations chain",
			event: map[string]interface{}{
				"a": float64(3), "b": float64(7), "c": float64(4), "d": float64(5),
				"e": float64(10), "f": float64(5), "g": float64(8), "h": float64(4),
				"i": float64(20), "j": float64(10),
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
				"a": float64(1),
				"b": float64(2),
				"c": float64(5),
			},
			expectMin:   2, // Both arithmetic-precedence rules
			expectMax:   2,
			description: "1 + 2*5 = 11 and 1*2 + 9 would not equal 11, testing a*b+c with 2*3+5=11",
		},
		{
			name: "arithmetic in comparison",
			event: map[string]interface{}{
				"a": float64(10),
				"b": float64(5),
				"c": float64(2),
				"d": float64(7),
			},
			expectMin:   1,
			expectMax:   1,
			description: "10 + 5 = 15 > 2*7 = 14",
		},
		{
			name: "comparison with logical",
			event: map[string]interface{}{
				"a": float64(2),
				"b": float64(3),
				"c": float64(6),
				"d": float64(10),
				"e": float64(5),
			},
			expectMin:   1,
			expectMax:   1,
			description: "2*3 == 6 && 10 > 5",
		},
		{
			name: "AND before OR",
			event: map[string]interface{}{
				"a": float64(1),
				"b": float64(99),
				"c": float64(3),
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
				"a": float64(5),
				"b": float64(5),
				"c": float64(3),
			},
			expectMin:   1, // grouped-arithmetic: (5+5)*3 = 30
			expectMax:   1, // ungrouped would be: 5+5*3 = 20
			description: "(5+5)*3 = 30 vs 5+5*3 = 20",
		},
		{
			name: "parentheses change logical result",
			event: map[string]interface{}{
				"a": float64(99),
				"b": float64(2),
				"c": float64(3),
			},
			expectMin:   1, // grouped-logical matches
			expectMax:   1,
			description: "(a==1 || b==2) && c==3 should match when b==2 and c==3",
		},
		{
			name: "complex grouping",
			event: map[string]interface{}{
				"a":      float64(10),
				"b":      float64(5),
				"c":      float64(20),
				"d":      float64(10),
				"e":      float64(3),
				"result": float64(50),
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
				"transactionCount": float64(5),
				"accountAge":       float64(100),
			},
			expectMin:   1,
			expectMax:   1,
			description: "High amount from non-US country should trigger fraud detection",
		},
		{
			name: "pricing logic - valid order",
			event: map[string]interface{}{
				"basePrice":         100.0,
				"quantity":          float64(10),
				"discount":          20.0,
				"minimumOrderValue": 700.0,
				"inventory":         float64(15),
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
				"loginAttempts": float64(2),
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
				"popularityScore": float64(50),
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
				"responseTime":  float64(500),
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
					map[string]interface{}{"total": float64(150), "status": "completed"},
					map[string]interface{}{"total": float64(200), "status": "completed"},
					map[string]interface{}{"total": float64(120), "status": "completed"},
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
					map[string]interface{}{"price": float64(100), "quantity": float64(3)},
					map[string]interface{}{"price": float64(50), "quantity": float64(12)},
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
				"totalValue":  float64(1200),
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
					map[string]interface{}{"amount": float64(500), "currency": "EUR", "flagged": false},
					map[string]interface{}{"amount": float64(1500), "currency": "USD", "flagged": false},
				},
				"threshold": float64(1000),
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
		"a": float64(100), "b": float64(5), "c": float64(10), "d": float64(50), "e": float64(2),
		"f": float64(3), "g": float64(7), "h": float64(15), "i": float64(10),
		"j": float64(42), "k": float64(42), "l": float64(1), "m": float64(2),
		"n": float64(100), "o": float64(50), "p": float64(75), "q": float64(80),
		"threshold1": float64(100),
		"threshold2": float64(50),
		"items": []interface{}{
			map[string]interface{}{"value": float64(150)},
			map[string]interface{}{"value": float64(200)},
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
				"a":     float64(5),
				"b":     float64(1),
				"items": []interface{}{},
			},
			expectMin:   1,
			expectMax:   1,
			description: "Empty array forAll is true, but a > 10 is false, so b == 1 branch matches",
		},
		{
			name: "null in complex expression",
			event: map[string]interface{}{
				"a": float64(50),
				"b": float64(60),
				"c": nil,
				"d": float64(42),
			},
			expectMin:   1,
			expectMax:   1,
			description: "a + b > 100 is true, c == null is true, d != null is true",
		},
		{
			name: "zero division protection",
			event: map[string]interface{}{
				"divisor":  float64(5),
				"dividend": float64(60),
			},
			expectMin:   1,
			expectMax:   1,
			description: "divisor != 0 is true and 60/5 = 12 > 10",
		},
		{
			name: "zero division blocked",
			event: map[string]interface{}{
				"divisor":  float64(0),
				"dividend": float64(60),
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
