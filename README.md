# Rulestone

![Go Tests](https://github.com/atlasgurus/rulestone/actions/workflows/tests.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/atlasgurus/rulestone)](https://goreportcard.com/report/github.com/atlasgurus/rulestone)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/atlasgurus/rulestone/blob/main/LICENSE)

Lightweight and fast [rule engine](https://en.wikipedia.org/wiki/Business_rules_engine) written in Go, with API for other languages:
* Go (this!)
* [Java](https://github.com/atlasgurus/rulestone-java)

With Rulestone you can define thousands of rules and then process tens of thousands events/objects per second, getting the matching rules for each object.

## Features

- **Fast**: Process tens of thousands of events per second
- **Flexible**: Load rules from files, databases, S3, HTTP, or any io.Reader
- **Validated**: Optional expression validation during load
- **Tested**: Built-in test cases in rule metadata
- **Data-Driven**: Self-documenting rules with inline tests
- **Production-Ready**: Skip validation for trusted sources in production

## Installation

Install the package:

```bash
go get github.com/atlasgurus/rulestone
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/atlasgurus/rulestone/engine"
)

func main() {
    // Create repository
    repo := engine.NewRuleEngineRepo()

    // Load rules with validation and testing
    result, err := repo.LoadRulesFromFile("rules.yaml", engine.LoadOptions{
        Validate:   true,  // Validate expressions
        RunTests:   true,  // Execute built-in tests
        FileFormat: "yaml",
    })
    if err != nil {
        panic(err)
    }

    // Check validation and test results
    if !result.ValidationOK {
        for _, err := range result.Errors {
            fmt.Printf("Validation error: %v\n", err)
        }
        return
    }

    summary := result.GetTestSummary()
    fmt.Printf("%s\n", summary.FormatTestSummary())

    // Create engine
    ruleEngine, err := engine.NewRuleEngine(repo)
    if err != nil {
        panic(err)
    }

    // Match events
    event := map[string]interface{}{
        "name": "Frank",
        "age":  20,
    }

    matches := ruleEngine.MatchEvent(event)
    for _, ruleID := range matches {
        rule := ruleEngine.GetRuleDefinition(uint(ruleID))
        if id, ok := rule.Metadata["id"].(string); ok {
            fmt.Printf("Rule matched: %s\n", id)
        }
    }
}
```

## Rule Format

### Simple Rule

```yaml
- metadata:
    id: simple-rule
    description: Match specific name and age
    created: "2023-03-29"
  expression: name == "Frank" && age == 20
```

### Rule with Built-in Tests

```yaml
- metadata:
    id: premium-user
    description: Check premium user eligibility
  expression: user.age >= 18 && user.verified == true
  tests:
    - name: eligible user
      event:
        user:
          age: 25
          verified: true
      expect: true
    - name: underage user
      event:
        user:
          age: 16
          verified: true
      expect: false
```

### Multiple Rules in One File

```yaml
- metadata:
    id: rule-1
  expression: status == "active"
  tests:
    - name: active status
      event: {status: active}
      expect: true

- metadata:
    id: rule-2
  expression: amount > 1000
  tests:
    - name: high amount
      event: {amount: 1500}
      expect: true
```

## API Reference

### LoadOptions

Controls rule loading behavior:

```go
type LoadOptions struct {
    Validate   bool   // If true, validate expressions during load
    RunTests   bool   // If true, execute test cases
    FileFormat string // "yaml", "json", or "" for auto-detect
}
```

### LoadResult

Contains results of rule loading:

```go
type LoadResult struct {
    RuleIDs      []uint       // IDs of loaded rules
    ValidationOK bool         // True if all rules validated
    TestResults  []TestResult // Results from test execution
    Errors       []error      // Validation or test errors
}

// Get test statistics
summary := result.GetTestSummary()
fmt.Println(summary.FormatTestSummary())

// Get only failed tests
for _, ft := range result.GetFailedTests() {
    fmt.Println(ft.FormatTestResult())
}
```

### Loading Rules

```go
// From file
result, err := repo.LoadRulesFromFile("rules.yaml", engine.LoadOptions{
    Validate: true,
    RunTests: true,
})

// From string
rulesYAML := `- metadata: {id: test}
  expression: a == 1`
result, err := repo.LoadRulesFromString(rulesYAML, engine.LoadOptions{
    Validate: true,
})

// From io.Reader (database, S3, HTTP, etc.)
rulesData := fetchFromDatabase()
reader := bytes.NewReader(rulesData)
result, err := repo.LoadRules(reader, engine.LoadOptions{
    Validate:   false,  // Skip validation for trusted source
    FileFormat: "json",
})
```

## Workflows

### Development Workflow

Full validation and testing during development:

```go
repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRulesFromFile("rules.yaml", engine.LoadOptions{
    Validate: true,  // Validate all expressions
    RunTests: true,  // Run all built-in tests
})

if !result.ValidationOK {
    log.Fatal("Validation failed")
}

summary := result.GetTestSummary()
if summary.Failed > 0 || summary.Errors > 0 {
    log.Fatalf("Tests failed: %s", summary.FormatTestSummary())
}
```

### CI/CD Workflow

Validate rules in CI, skip validation in production:

```go
// CI: Validate and test
tmpRepo := engine.NewRuleEngineRepo()
result, err := tmpRepo.LoadRulesFromFile("rules.yaml", engine.LoadOptions{
    Validate: true,
    RunTests: true,
})
if err != nil || !result.ValidationOK {
    os.Exit(1)  // Fail CI build
}

// Production: Skip validation (already validated in CI)
repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRulesFromFile("rules.yaml", engine.LoadOptions{
    Validate: false,  // Trusted source
    RunTests: false,  // Skip tests in prod
})
```

### Hot Reload Workflow

Validate new rules before swapping:

```go
// Validate in temporary repository first
tmpRepo := engine.NewRuleEngineRepo()
result, err := tmpRepo.LoadRules(newRulesReader, engine.LoadOptions{
    Validate: true,
    RunTests: true,
})
if err != nil || !result.ValidationOK {
    return fmt.Errorf("invalid rules")
}

// Build new engine
newEngine, err := engine.NewRuleEngine(tmpRepo)
if err != nil {
    return err
}

// Atomic swap
atomic.StorePointer(&currentEngine, unsafe.Pointer(newEngine))
```

## Expression Language

### Operators

- Comparison: `==`, `!=`, `>`, `>=`, `<`, `<=`
- Arithmetic: `+`, `-`, `*`, `/`
- Logical: `&&`, `||`, `!`
- Parentheses: `(`, `)`

### Literals

- Strings: `"text"`
- Numbers: `1`, `2.3`, `-5`
- Booleans: `true`, `false`
- Null: `null`

### Field Access

- Simple: `field1`
- Nested: `user.name`, `order.items[0].price`
- Arrays: `items[0]`, `items[1].value`

### Functions

#### String Functions

```yaml
# Regular expression matching
expression: regexpMatch("^[A-Z]{2}[0-9]{4}$", code)

# Check if value exists
expression: hasValue(user.email)

# Check if value matches any
expression: isEqualToAny(status, "active", "pending")
```

#### Date Functions

```yaml
# Date comparison
expression: date(dob) < date("11/29/1968")

# Date in different formats
expression: date("2023-03-29") > date(user.registered)
```

#### Quantifier Functions

```yaml
# All elements must satisfy condition
expression: forAll("items", "item", item.price > 0)

# At least one element must satisfy condition
expression: forSome("items", "item", item.status == "shipped")
```

#### Array Functions

```yaml
# Get array length
expression: length("items") > 0

# Check array size range
expression: length("items") >= 2 && length("items") <= 10

# Combine with other conditions
expression: length("items") > 0 && forAll("items", "item", item.validated == true)
```

**Note:** `length()` returns `null` for missing or null arrays, allowing proper null semantics:
- `length("items") > 0` handles all cases correctly (missing→false, empty→false, non-empty→true)
- `length("items") != 0` is true for both missing (null != 0) and non-empty arrays
- Use `hasValue("items")` only if you need to distinguish missing/null from empty arrays

## Testing

### Built-in Tests

Rules can include test cases that are executed during load:

```yaml
- metadata:
    id: discount-rule
  expression: order.total > 100 && user.premium == true
  tests:
    - name: premium user with high total
      event:
        order: {total: 150}
        user: {premium: true}
      expect: true
    - name: non-premium user
      event:
        order: {total: 150}
        user: {premium: false}
      expect: false
```

### Data-Driven Testing

Create test files in `tests/data/`:

```go
// tests/data_driven_test.go automatically discovers
// and runs all *.yaml files in tests/data/
func TestDataDrivenRules(t *testing.T) {
    files, _ := filepath.Glob("data/*.yaml")
    for _, file := range files {
        t.Run(filepath.Base(file), func(t *testing.T) {
            repo := engine.NewRuleEngineRepo()
            result, err := repo.LoadRulesFromFile(file, engine.LoadOptions{
                Validate: true,
                RunTests: true,
            })
            // Check results...
        })
    }
}
```

## Examples

See comprehensive examples in:
- `tests/data/simple_rules_with_tests.yaml` - Basic examples
- `tests/data/comprehensive_tests.yaml` - Advanced patterns
- `tests/data/boolean_literals.yaml` - Boolean operations
- `examples/rules/` - Various rule patterns

## Migration from v1 API

Old API (v1):
```go
repo := engine.NewRuleEngineRepo()
_, err := repo.RegisterRulesFromFile("rules.yaml")
eng, _ := engine.NewRuleEngine(repo)
```

New API (v2):
```go
repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRulesFromFile("rules.yaml", engine.LoadOptions{
    Validate: true,
    RunTests: true,
})
if !result.ValidationOK {
    // Handle validation errors
}
eng, _ := engine.NewRuleEngine(repo)
```

## Performance

- Process tens of thousands of events per second
- Support thousands of rules
- Category-based optimization reduces matching complexity
- O(1) attribute lookup with object mapping
- Zero allocation for event matching (after warm-up)

## Contributing

We love contributions! If you have suggestions, bug reports, or feature requests, please open an issue in our [tracker](https://github.com/atlasgurus/rulestone/issues).

## License

This project is licensed under the MIT License - see the LICENSE file for details.
