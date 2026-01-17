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
    result, err := repo.LoadRulesFromFile("rules.yaml",
        engine.WithValidate(true),   // Validate expressions
        engine.WithRunTests(true),    // Execute built-in tests
        engine.WithFileFormat("yaml"),
    )
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

### Load Options

Control rule loading behavior using functional options:

```go
// Available options:
engine.WithValidate(bool)      // Enable/disable expression validation
engine.WithRunTests(bool)      // Enable/disable test execution
engine.WithFileFormat(string)  // Set format: "yaml", "json", or "" for auto-detect
engine.WithOptimize(bool)      // Enable/disable category engine optimizations

// Example usage:
result, err := repo.LoadRulesFromFile("rules.yaml",
    engine.WithValidate(true),
    engine.WithRunTests(true),
)
```

### Engine Optimization

Control category engine optimizations using the `WithOptimize` option:

```go
repo := engine.NewRuleEngineRepo()

// Default is non-optimized (false)
result, err := repo.LoadRulesFromFile("rules.yaml",
    engine.WithValidate(true),
)

// Enable optimization for better performance
result, err := repo.LoadRulesFromFile("rules.yaml",
    engine.WithValidate(true),
    engine.WithOptimize(true),
)
```

**Optimization modes:**
- **Non-optimized** (default): Simpler engine structure, useful for debugging
- **Optimized**: Applies AND-set optimizations for better matching performance

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

### Rule Hashing

Each compiled rule has a unique cryptographic hash computed from its condition tree:

```go
repo := engine.NewRuleEngineRepo()
result, _ := repo.LoadRulesFromString(`
- expression: a == 1 && b == 2
`, engine.LoadOptions{Validate: true})

// Get the rule hash
ruleID := result.RuleIDs[0]
hash := repo.Rules[ruleID].GetHash()
fmt.Printf("Rule hash: %x\n", hash)
```

**Properties:**
- **Cryptographic hash**: Uses recursive hash computation for security
- **Semantic equality**: Identical expressions produce identical hashes
- **Deterministic**: Same rule always produces the same hash
- **Unique**: Different rules produce different hashes (collision-resistant)

**Use cases:**
- Detect duplicate rules across rule sets
- Cache compiled rules by hash
- Track rule versions and changes
- Validate rule integrity

### Loading Rules

```go
// From file
result, err := repo.LoadRulesFromFile("rules.yaml",
    engine.WithValidate(true),
    engine.WithRunTests(true),
)

// From string
rulesYAML := `- metadata: {id: test}
  expression: a == 1`
result, err := repo.LoadRulesFromString(rulesYAML,
    engine.WithValidate(true),
)

// From io.Reader (database, S3, HTTP, etc.)
rulesData := fetchFromDatabase()
reader := bytes.NewReader(rulesData)
result, err := repo.LoadRules(reader,
    engine.WithValidate(false),  // Skip validation for trusted source
    engine.WithFileFormat("json"),
)
```

## Workflows

### Development Workflow

Full validation and testing during development:

```go
repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRulesFromFile("rules.yaml",
    engine.WithValidate(true),  // Validate all expressions
    engine.WithRunTests(true),  // Run all built-in tests
)

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
result, err := tmpRepo.LoadRulesFromFile("rules.yaml",
    engine.WithValidate(true),
    engine.WithRunTests(true),
)
if err != nil || !result.ValidationOK {
    os.Exit(1)  // Fail CI build
}

// Production: Skip validation (already validated in CI)
repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRulesFromFile("rules.yaml",
    engine.WithValidate(false),  // Trusted source
    engine.WithRunTests(false),  // Skip tests in prod
)
```

### Hot Reload Workflow

Validate new rules before swapping:

```go
// Validate in temporary repository first
tmpRepo := engine.NewRuleEngineRepo()
result, err := tmpRepo.LoadRules(newRulesReader,
    engine.WithValidate(true),
    engine.WithRunTests(true),
)
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
- Undefined: `undefined`

### Field Access

- Simple: `field1`
- Nested: `user.name`, `order.items[0].price`
- Arrays: `items[0]`, `items[1].value`

### Null vs Undefined Semantics

Rulestone distinguishes between **missing fields** and **explicit null values**:

```yaml
# Missing field (not in event)
Event: { name: "john" }
age → undefined

# Explicit null (field exists with null value)
Event: { name: "john", age: null }
age → null

# Zero/empty values are NOT null or undefined
Event: { name: "john", age: 0 }
age → 0
```

**Comparison behavior**:
```yaml
# Check if field is missing
expression: age == undefined
  Event: {}          → true (field missing)
  Event: {age: null} → false (field exists, just null)
  Event: {age: 0}    → false (field has value)

# Check if field exists (even if null)
expression: age != undefined
  Event: {}          → false (field missing)
  Event: {age: null} → true (field exists)
  Event: {age: 0}    → true (field has value)

# Check if field is explicitly null
expression: age == null
  Event: {}          → false (field missing, not null)
  Event: {age: null} → true (explicit null)
  Event: {age: 0}    → false (has value)

# Check if field has a non-null value
expression: age != null
  Event: {}          → false (field missing)
  Event: {age: null} → false (explicit null)
  Event: {age: 0}    → true (has value)
```

**Negations with missing fields**:
```yaml
# Does NOT match when field is missing
expression: age != 18
  Event: {}          → false (field missing → undefined != 18 → undefined)
  Event: {age: null} → true (null != 18)
  Event: {age: 25}   → true (25 != 18)
  Event: {age: 18}   → false (18 != 18)

# To match when missing OR not equal (rare):
expression: age == undefined || age != 18
```

**Key principle**: Missing fields cause comparisons to return `undefined`, which is treated as "not applicable" - the rule doesn't match.

**Common patterns**:
```yaml
# Require field to exist and meet condition
expression: age != undefined && age > 18

# Optional field - default behavior if missing
expression: premium == undefined || premium == true

# Check field is neither missing nor null
expression: age != undefined && age != null && age > 18

# Distinguish three states
expression: |
  status == undefined     # Field not provided
  status == null          # Field explicitly cleared
  status == "active"      # Field has value
```

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
# Current time
expression: now()

# Date comparison
expression: date(dob) < date("11/29/1968")

# Date in different formats
expression: date("2023-03-29") > date(user.registered)
```

#### Duration Functions

Duration functions convert time units to nanoseconds for time arithmetic:

```yaml
# Check if event happened within last 5 days
expression: created_at > now() - days(5)

# Check if session is recent (within 2 hours)
expression: session_start > now() - hours(2)

# Check if event occurred within last 30 minutes
expression: event_time >= now() - minutes(30)

# Check if timestamp is older than 10 seconds
expression: timestamp < now() - seconds(10)

# Compound durations (1.5 days from now)
expression: deadline <= now() + days(1) + hours(12)

# Time difference comparisons
expression: (now() - last_login) < days(30)
expression: (now() - created_at) >= hours(24)

# Fractional durations
expression: event_time > now() - days(0.5)  # 12 hours
expression: session > now() - hours(2.5)    # 2.5 hours
```

**Available functions**:
- `days(n)` - Converts n days to nanoseconds
- `hours(n)` - Converts n hours to nanoseconds
- `minutes(n)` - Converts n minutes to nanoseconds
- `seconds(n)` - Converts n seconds to nanoseconds

**Note**: Arguments can be integers, floats, or constant expressions (e.g., `minutes(30 + 15)`).

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

**Note:** `length()` returns `undefined` for missing arrays and `null` for explicit null, allowing proper semantics:
- `length("items") > 0` handles all cases correctly (missing→false, null→false, empty→false, non-empty→true)
- `length("items") != 0` with missing array → false (undefined != 0 → undefined)
- To match missing OR non-empty: `items == undefined || length("items") > 0`
- Use `hasValue("items")` to check if array exists with a value
- `length(iterator)` also works with filter/map results

#### Array Transformation Functions (filter/map)

Transform and filter arrays using composable iterator functions with **zero intermediate allocations**:

```yaml
# Filter array elements
expression: length(filter("items", "item", item.active == true)) > 5

# Map array to values
expression: sum(map("items", "item", item.price)) > 1000

# Filter then map (composition)
expression: sum(map(filter("items", "item", item.active == true), "item", item.price)) > 500

# Multiple filters
expression: length(filter(filter("items", "item", item.category == "food"), "item", item.price > 10)) > 0

# Sum with direct array
expression: sum("items", "item", item.price * item.quantity) > 100

# Average values
expression: avg("ratings", "r", r) >= 4.0

# Average of filtered elements
expression: avg(filter("users", "user", user.age >= 18), "user", user.score) >= 75
```

**Available functions**:

**Transformation**:
- `filter(array_or_iterator, element_name, condition)` - Select elements matching condition
- `map(array_or_iterator, element_name, expression)` - Transform elements to values

**Aggregation**:
- `sum(iterator)` or `sum(array, elem, expr)` - Sum values
- `avg(iterator)` or `avg(array, elem, expr)` - Average values
- `length(iterator)` - Count filtered/mapped elements

**Key features**:
- **Zero allocations**: Iterators fuse into single pass
- **Composable**: Chain filter + filter, filter + map, etc.
- **Lazy evaluation**: Only executes when consumed by aggregation
- **Handles undefined**: Skips undefined/null values automatically

#### Conditional/Ternary Operator

The `if()` function provides conditional expressions for ternary logic:

```yaml
# Basic conditional - returns different values based on condition
expression: if(age >= 18, "adult", "minor") == "adult"

# Conditional with numeric values
expression: if(premium, discount * 2, discount) >= 10

# Nested conditionals for multiple conditions
expression: if(score >= 90, "A", if(score >= 80, "B", "C")) == "B"

# Use in arithmetic expressions
expression: if(premium, 100, 50) + bonus == 120

# Use in comparisons
expression: price < if(vip, 100, 200)

# Select between field values
expression: if(use_alt, alt_value, main_value) > 100
```

**Syntax**: `if(condition, true_value, false_value)`

**Behavior**:
- Evaluates `condition` to boolean
- Returns `true_value` if condition is true
- Returns `false_value` if condition is false
- Returns `undefined` if condition is `undefined`
- Can be nested for multiple conditions

#### Math Functions

Math functions for numeric operations:

```yaml
# abs(x) - Absolute value
expression: abs(balance) > 1000
expression: abs(-42) == 42

# ceil(x) - Round up to nearest integer
expression: ceil(price * 1.08) <= budget
expression: ceil(4.2) == 5

# floor(x) - Round down to nearest integer
expression: floor(rating) >= 4
expression: floor(4.8) == 4

# round(x) or round(x, digits) - Round to n decimal places
expression: round(score) >= 95
expression: round(price, 2) == 19.99
expression: round(3.14159, 2) == 3.14

# min(a, b, ...) - Minimum of multiple values
expression: min(price1, price2, price3) < 100
expression: min(age, min_age) >= 18

# max(a, b, ...) - Maximum of multiple values
expression: max(price1, price2, price3) <= budget
expression: max(age, min_age) >= 21

# pow(base, exponent) - Power/exponentiation
expression: pow(base, 2) == 100
expression: pow(value, 0.5) == 5  # square root
expression: pow(2, -1) == 0.5     # inverse
```

**Handling undefined/null**:
- Math operations on `undefined` return `undefined`
- Math operations on `null` return `undefined`
- `min()` and `max()` skip `undefined`/`null` values
- If all values are `undefined`/`null`, returns `undefined`

**Examples**:
```yaml
# Combining math functions
expression: ceil(abs(value)) == 3
expression: round(pow(value, 2), 1) == 6.2

# Math with ternary operator
expression: if(use_ceil, ceil(value), floor(value)) == 4
expression: abs(if(positive, value, -value)) > 10

# Complex calculations
expression: floor(price * 1.1) + ceil(tax) <= budget
expression: max(min(a, b), min(c, d)) == 15
```

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
            result, err := repo.LoadRulesFromFile(file,
                engine.WithValidate(true),
                engine.WithRunTests(true),
            )
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
result, err := repo.LoadRulesFromFile("rules.yaml",
    engine.WithValidate(true),
    engine.WithRunTests(true),
)
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
