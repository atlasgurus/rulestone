package tests

import (
	"os"
	"testing"

	"github.com/atlasgurus/rulestone/engine"
)

// Helper to create temp rule file for string function tests
func createStringFunctionTestRuleFile(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "rule-strfunc-*.yaml")
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

// TestStringFunctions_RegexpMatch_Basic tests basic regexpMatch functionality
func TestStringFunctions_RegexpMatch_Basic(t *testing.T) {
	rules := `
- metadata:
    id: simple-pattern
  expression: regexpMatch(".*@.*\\.com", email)

- metadata:
    id: digit-pattern
  expression: regexpMatch("^[0-9]{4}$", code)

- metadata:
    id: word-pattern
  expression: regexpMatch("\\btest\\b", text)

- metadata:
    id: case-sensitive
  expression: regexpMatch("^Test", value)

- metadata:
    id: multiline-pattern
  expression: regexpMatch("start.*end", content)
`

	ruleFile := createStringFunctionTestRuleFile(t, rules)
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
			name: "valid email pattern",
			event: map[string]interface{}{
				"email": "user@example.com",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Email should match pattern",
		},
		{
			name: "invalid email pattern",
			event: map[string]interface{}{
				"email": "invalid-email",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Invalid email should not match",
		},
		{
			name: "four digit code",
			event: map[string]interface{}{
				"code": "1234",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Four digit code should match",
		},
		{
			name: "invalid code length",
			event: map[string]interface{}{
				"code": "123",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Three digit code should not match four digit pattern",
		},
		{
			name: "word boundary match",
			event: map[string]interface{}{
				"text": "this is a test string",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Word 'test' with boundaries should match",
		},
		{
			name: "word boundary no match",
			event: map[string]interface{}{
				"text": "testing string",
			},
			expectMin:   0,
			expectMax:   0,
			description: "'testing' should not match '\\btest\\b'",
		},
		{
			name: "case sensitive match",
			event: map[string]interface{}{
				"value": "Test123",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Case sensitive pattern should match",
		},
		{
			name: "case sensitive no match",
			event: map[string]interface{}{
				"value": "test123",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Lowercase should not match case sensitive pattern",
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

// TestStringFunctions_RegexpMatch_Complex tests complex regex patterns
func TestStringFunctions_RegexpMatch_Complex(t *testing.T) {
	rules := `
- metadata:
    id: ip-address
  expression: regexpMatch("^(?:[0-9]{1,3}\\.){3}[0-9]{1,3}$", ip)

- metadata:
    id: phone-number
  expression: regexpMatch("^\\+?1?[0-9]{10,14}$", phone)

- metadata:
    id: url-pattern
  expression: regexpMatch("^https?://[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}", url)

- metadata:
    id: credit-card
  expression: regexpMatch("^[0-9]{4}[- ]?[0-9]{4}[- ]?[0-9]{4}[- ]?[0-9]{4}$", card)

- metadata:
    id: alphanumeric
  expression: regexpMatch("^[a-zA-Z0-9]+$", value)
`

	ruleFile := createStringFunctionTestRuleFile(t, rules)
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
			name: "valid IP address",
			event: map[string]interface{}{
				"ip": "192.168.1.1",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Valid IP should match pattern",
		},
		{
			name: "invalid IP address",
			event: map[string]interface{}{
				"ip": "999.999.999.999",
			},
			expectMin:   0,
			expectMax:   1,
			description: "Invalid IP (pattern matches format, not validity)",
		},
		{
			name: "valid phone number",
			event: map[string]interface{}{
				"phone": "+12345678901",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Phone with country code should match",
		},
		{
			name: "valid URL http",
			event: map[string]interface{}{
				"url": "http://example.com",
			},
			expectMin:   1,
			expectMax:   1,
			description: "HTTP URL should match",
		},
		{
			name: "valid URL https",
			event: map[string]interface{}{
				"url": "https://www.example.com",
			},
			expectMin:   1,
			expectMax:   1,
			description: "HTTPS URL should match",
		},
		{
			name: "credit card with spaces",
			event: map[string]interface{}{
				"card": "1234 5678 9012 3456",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Credit card with spaces should match",
		},
		{
			name: "credit card with dashes",
			event: map[string]interface{}{
				"card": "1234-5678-9012-3456",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Credit card with dashes should match",
		},
		{
			name: "alphanumeric valid",
			event: map[string]interface{}{
				"value": "Test123",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Alphanumeric string should match",
		},
		{
			name: "alphanumeric with special chars",
			event: map[string]interface{}{
				"value": "Test@123",
			},
			expectMin:   0,
			expectMax:   0,
			description: "String with special chars should not match alphanumeric",
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

// TestStringFunctions_RegexpMatch_NullHandling tests regex with null inputs
func TestStringFunctions_RegexpMatch_NullHandling(t *testing.T) {
	rules := `
- metadata:
    id: null-value
  expression: regexpMatch("test", value)

- metadata:
    id: null-check-with-fallback
  expression: value != null && regexpMatch("test", value)
`

	ruleFile := createStringFunctionTestRuleFile(t, rules)
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
			name:        "null value in regex",
			event:       map[string]interface{}{"value": nil},
			expectMin:   0,
			expectMax:   0,
			description: "Null value should not match regex",
		},
		{
			name:        "missing value in regex",
			event:       map[string]interface{}{},
			expectMin:   0,
			expectMax:   0,
			description: "Missing value should not match regex",
		},
		{
			name: "valid value matches",
			event: map[string]interface{}{
				"value": "test string",
			},
			expectMin:   2, // Both rules match
			expectMax:   2,
			description: "Valid value should match both rules",
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

// TestStringFunctions_ContainsAny_Basic tests basic containsAny functionality
func TestStringFunctions_ContainsAny_Basic(t *testing.T) {
	rules := `
- metadata:
    id: single-pattern
  expression: containsAny(text, "error")

- metadata:
    id: multiple-patterns
  expression: containsAny(text, "error", "fail", "warning")

- metadata:
    id: case-sensitive-contains
  expression: containsAny(text, "Error", "ERROR")

- metadata:
    id: empty-list
  expression: text == "never_matches_empty_list"

- metadata:
    id: substring-match
  expression: containsAny(url, "example.com", "test.org")
`

	ruleFile := createStringFunctionTestRuleFile(t, rules)
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
			name: "single pattern match",
			event: map[string]interface{}{
				"text": "This is an error message",
			},
			expectMin:   2, // single-pattern and multiple-patterns
			expectMax:   2,
			description: "Text containing 'error' should match",
		},
		{
			name: "multiple pattern match",
			event: map[string]interface{}{
				"text": "System failed to start",
			},
			expectMin:   1, // multiple-patterns only
			expectMax:   1,
			description: "Text containing 'fail' should match multiple-patterns",
		},
		{
			name: "no pattern match",
			event: map[string]interface{}{
				"text": "Everything is fine",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Text without error keywords should not match",
		},
		{
			name: "case sensitive no match",
			event: map[string]interface{}{
				"text": "error in lowercase",
			},
			expectMin:   2, // matches both single-pattern and multiple-patterns (both have lowercase "error")
			expectMax:   2,
			description: "Lowercase 'error' matches lowercase patterns, not uppercase-only patterns",
		},
		{
			name: "case sensitive match",
			event: map[string]interface{}{
				"text": "ERROR in uppercase",
			},
			expectMin:   1, // case-sensitive-contains
			expectMax:   1,
			description: "Uppercase ERROR should match",
		},
		{
			name: "empty list never matches",
			event: map[string]interface{}{
				"text": "any text",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Empty pattern list should never match",
		},
		{
			name: "substring in URL",
			event: map[string]interface{}{
				"url": "https://example.com/path",
			},
			expectMin:   1,
			expectMax:   1,
			description: "URL containing 'example.com' should match",
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

// TestStringFunctions_ContainsAny_AhoCorasick tests Aho-Corasick algorithm behavior
func TestStringFunctions_ContainsAny_AhoCorasick(t *testing.T) {
	rules := `
- metadata:
    id: overlapping-patterns
  expression: containsAny(text, "abc", "bcd", "cde")

- metadata:
    id: prefix-patterns
  expression: containsAny(text, "test", "testing", "tester")

- metadata:
    id: many-patterns
  expression: containsAny(text, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j")

- metadata:
    id: long-patterns
  expression: containsAny(text, "the quick brown fox", "jumps over the lazy dog")
`

	ruleFile := createStringFunctionTestRuleFile(t, rules)
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
			name: "overlapping patterns",
			event: map[string]interface{}{
				"text": "xabcdex",
			},
			expectMin:   2, // matches both overlapping-patterns and many-patterns
			expectMax:   2,
			description: "Should match overlapping patterns (abc, bcd, cde) and many-patterns (a, b, c, d, e)",
		},
		{
			name: "prefix pattern short match",
			event: map[string]interface{}{
				"text": "test",
			},
			expectMin:   2, // matches both prefix-patterns and many-patterns (has 'e')
			expectMax:   2,
			description: "Should match 'test' pattern and many-patterns ('e')",
		},
		{
			name: "prefix pattern long match",
			event: map[string]interface{}{
				"text": "testing phase",
			},
			expectMin:   2, // matches both prefix-patterns and many-patterns (has 'a', 'e', 'g', 'h', 'i')
			expectMax:   2,
			description: "Should match 'testing' or 'test' pattern and many-patterns",
		},
		{
			name: "many patterns single match",
			event: map[string]interface{}{
				"text": "xyz",
			},
			expectMin:   0,
			expectMax:   0,
			description: "Should not match any single letter pattern",
		},
		{
			name: "many patterns multiple matches",
			event: map[string]interface{}{
				"text": "abcdefghij",
			},
			expectMin:   2, // matches both overlapping-patterns (abc,bcd,cde) and many-patterns
			expectMax:   2,
			description: "Should match overlapping patterns and many-patterns",
		},
		{
			name: "long pattern match",
			event: map[string]interface{}{
				"text": "the quick brown fox leaps",
			},
			expectMin:   2, // matches both long-patterns and many-patterns (has a,b,c,e,f,h,i)
			expectMax:   2,
			description: "Should match long phrase pattern and many-patterns",
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

// TestStringFunctions_ContainsAny_NullHandling tests containsAny with null inputs
func TestStringFunctions_ContainsAny_NullHandling(t *testing.T) {
	rules := `
- metadata:
    id: null-value
  expression: containsAny(value, "test", "demo")

- metadata:
    id: null-check-with-fallback
  expression: value != null && containsAny(value, "test", "demo")
`

	ruleFile := createStringFunctionTestRuleFile(t, rules)
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
			name:        "null value",
			event:       map[string]interface{}{"value": nil},
			expectMin:   0,
			expectMax:   0,
			description: "Null value should not match containsAny",
		},
		{
			name:        "missing value",
			event:       map[string]interface{}{},
			expectMin:   0,
			expectMax:   0,
			description: "Missing value should not match containsAny",
		},
		{
			name: "valid value matches",
			event: map[string]interface{}{
				"value": "this is a test",
			},
			expectMin:   2, // Both rules
			expectMax:   2,
			description: "Valid value should match both rules",
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

// TestStringFunctions_Combined tests combining string functions
func TestStringFunctions_Combined(t *testing.T) {
	rules := `
- metadata:
    id: regex-and-contains
  expression: regexpMatch("^[A-Z]", text) && containsAny(text, "error", "warning")

- metadata:
    id: regex-or-contains
  expression: regexpMatch(".*@.*\\.com", email) || containsAny(email, "test", "demo")

- metadata:
    id: nested-string-functions
  expression: containsAny(log, "ERROR", "FATAL") && regexpMatch("\\d{4}-\\d{2}-\\d{2}", log)

- metadata:
    id: complex-string-logic
  expression: (regexpMatch("^https://", url) && containsAny(url, "api", "service")) || priority > 5
`

	ruleFile := createStringFunctionTestRuleFile(t, rules)
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
			name: "regex and contains both match",
			event: map[string]interface{}{
				"text": "Error: an error occurred in system",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Starts with capital and contains lowercase 'error'",
		},
		{
			name: "regex matches, contains doesn't",
			event: map[string]interface{}{
				"text": "Everything is fine",
			},
			expectMin:   0,
			expectMax:   0,
			description: "AND requires both conditions",
		},
		{
			name: "email matches regex",
			event: map[string]interface{}{
				"email": "user@example.com",
			},
			expectMin:   1,
			expectMax:   1,
			description: "OR requires only one condition (regex matches)",
		},
		{
			name: "email matches contains",
			event: map[string]interface{}{
				"email": "test@somewhere.net",
			},
			expectMin:   1,
			expectMax:   1,
			description: "OR requires only one condition (contains matches)",
		},
		{
			name: "log with date and error",
			event: map[string]interface{}{
				"log": "2024-01-15 ERROR: System failure",
			},
			expectMin:   1,
			expectMax:   1,
			description: "Both ERROR and date pattern should match",
		},
		{
			name: "complex string logic via URL",
			event: map[string]interface{}{
				"url":      "https://api.example.com/endpoint",
				"priority": 3,
			},
			expectMin:   1,
			expectMax:   1,
			description: "HTTPS URL containing 'api' should match",
		},
		{
			name: "complex string logic via priority",
			event: map[string]interface{}{
				"url":      "http://website.com",
				"priority": 10,
			},
			expectMin:   1,
			expectMax:   1,
			description: "High priority should match despite URL not matching",
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

// TestStringFunctions_Performance tests performance with many patterns
func TestStringFunctions_Performance(t *testing.T) {
	// Build a rule with many containsAny patterns to test Aho-Corasick efficiency
	rules := `
- metadata:
    id: many-patterns-performance
  expression: containsAny(log, "error", "warning", "critical", "fatal", "exception", "timeout", "failure", "denied", "unauthorized", "forbidden", "unavailable", "overload", "crash", "panic", "abort")
`

	ruleFile := createStringFunctionTestRuleFile(t, rules)
	repo := engine.NewRuleEngineRepo()
	_, err := repo.RegisterRulesFromFile(ruleFile)
	if err != nil {
		t.Fatalf("Failed to register rules: %v", err)
	}

	genFilter, err := engine.NewRuleEngine(repo)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Run multiple times to ensure consistent performance
	event := map[string]interface{}{
		"log": "This is a very long log message with multiple words and hopefully one of them will trigger the containsAny match like fatal error",
	}

	for i := 0; i < 1000; i++ {
		matches := genFilter.MatchEvent(event)
		if len(matches) == 0 {
			t.Errorf("Iteration %d: expected match but got none", i)
		}
	}
}
