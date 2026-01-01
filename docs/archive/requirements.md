# Rulestone Engine - Test Requirements & Capabilities

## Overview

This document defines all engine capabilities, test requirements, and known limitations of the Rulestone rule engine. Tests are organized by **engine capability** (what the engine can do), not by application domain. The domain is merely test data to exercise engine features.

## Core Capabilities

### 1. Operand Types (13 types)

The engine supports 13 operand kinds with automatic type reconciliation:

- `String` - String literals and values
- `Int` - Integer numbers
- `Float` - Floating-point numbers
- `Boolean` - Boolean values (true/false, 0/1)
- `Time` - Time/date values
- `Null` - Null/nil values
- `ObjectAttributeAddress` - Object field references
- `ArrayLen` - Array length operations
- `Function` - Function call results
- `Condition` - Nested conditions
- `ForCondition` - forAll/forSome quantifiers
- `ArithmeticExpression` - Arithmetic operations
- `BooleanExpression` - Boolean logical operations

**Test Priority**: P0 (Critical)
**Current Coverage**: ~30% (many type interactions untested)

### 2. Arithmetic Operators

Operators: `+`, `-`, `*`, `/`

**Test Priority**: P0 (Critical - COMPLETELY UNTESTED)
**Current Coverage**: 0%
**Required Tests**:
- Basic operations with integers and floats
- Type mixing (int + float, float - int, etc.)
- Division by zero handling
- Overflow/underflow behavior
- Null operand handling
- Precedence with other operators
- Nested arithmetic expressions
- Integration with comparison operators

### 3. Comparison Operators (7 operators)

Operators: `==`, `!=`, `<`, `<=`, `>`, `>=`, `contains`

**Test Priority**: P1 (High)
**Current Coverage**: ~60%
**Required Tests**:
- All operators with all compatible type combinations
- String comparisons (lexicographic)
- Numeric comparisons (int, float, mixed)
- Time comparisons
- Null comparisons (null == null, null != value)
- Case sensitivity in string operations
- `contains` operator with substrings
- Type coercion rules

### 4. Logical Operators

Operators: `&&`, `||`, `!`

**Test Priority**: P1 (High)
**Current Coverage**: ~50%
**Required Tests**:
- Basic AND/OR/NOT operations
- Short-circuit evaluation behavior
- Complex nested expressions
- Operator precedence
- Truth tables with null values
- Deep nesting (multiple levels)
- Mixed logical and comparison operators

### 5. Quantifiers: forAll

Syntax: `forAll("array.path", "element", condition)`

**Test Priority**: P0 (Critical)
**Current Coverage**: ~40%
**Required Tests**:
- Empty arrays (vacuous truth - should return true)
- Single element arrays
- Multi-element arrays (all match, some match, none match)
- Nested forAll (forAll inside forAll)
- Frame stack limit (20 levels)
- Element variable shadowing
- Null array handling
- Non-array field handling (error case)
- Complex conditions in quantifiers
- Accessing element fields (element.field)
- Performance with large arrays

### 6. Quantifiers: forSome

Syntax: `forSome("array.path", "element", condition)`

**Test Priority**: P0 (Critical)
**Current Coverage**: ~40%
**Required Tests**:
- Empty arrays (should return false)
- Single element arrays
- Multi-element arrays (first match, last match, multiple matches)
- Nested forSome
- Mixed forAll/forSome nesting
- Frame stack limit
- Element variable shadowing
- Null array handling
- Short-circuit evaluation (stops at first match)
- Performance optimization verification

### 7. Null Handling

**Test Priority**: P0 (Critical)
**Current Coverage**: ~20%
**Required Tests**:
- Null in comparisons (==, !=, <, >, etc.)
- Null in arithmetic (null + 5, 10 / null, etc.)
- Null in logical operations (null && true, null || false)
- Null in functions (regexpMatch(null, pattern))
- Null array in quantifiers
- Null element in quantifiers
- Missing object fields (should be treated as null)
- Explicit null in test data
- Type reconciliation with null
- Error propagation from null operations

### 8. Built-in Functions

#### String Functions
- `regexpMatch(value, pattern)` - Regular expression matching
- `containsAny(value, [...])` - Multi-pattern matching (Aho-Corasick)

#### Date Functions
- `date(string, format)` - Date parsing and conversion

**Test Priority**: P1 (High)
**Current Coverage**: ~30%
**Required Tests**:
- regexpMatch: valid patterns, invalid patterns, null inputs, complex regex
- containsAny: empty list, single item, multiple items, no matches, partial matches
- date: various formats, invalid formats, null inputs, timezone handling
- Function composition (nested functions)
- Functions in arithmetic expressions
- Functions in logical expressions

### 9. Type Conversion & Reconciliation

**Test Priority**: P1 (High)
**Current Coverage**: ~25%
**Required Tests**:
- String to number conversion
- Number to string conversion
- Boolean to numeric (true=1, false=0)
- Implicit conversions in operators
- Type reconciliation matrix (all combinations)
- Invalid conversions (error handling)
- Precision loss (float to int)
- String to time conversion

### 10. Object Attribute Access

Syntax: `object.field`, `object.nested.field`

**Test Priority**: P1 (High)
**Current Coverage**: ~60%
**Required Tests**:
- Simple field access
- Nested field access (multiple levels)
- Array element access via quantifiers
- Missing fields (should return null)
- Deep nesting (many levels)
- Field name conflicts with keywords
- Special characters in field names
- Case sensitivity

### 11. Category Engine Optimization

**Test Priority**: P2 (Medium)
**Current Coverage**: ~70%
**Existing Tests**: cateng_comprehensive_test.go
**Additional Tests Needed**:
- OR set optimization thresholds
- AND set optimization thresholds
- Mixed OR/AND optimization
- Bit mask generation correctness
- Category ID assignment
- Edge cases with large category counts
- Performance degradation scenarios

### 12. Common Sub-Expression Elimination (CSE)

**Test Priority**: P2 (Medium)
**Current Coverage**: ~50%
**Required Tests**:
- Identical expressions across rules (should share)
- Similar but different expressions (should not share)
- CSE in forAll/forSome conditions
- Cache key generation correctness
- Memory usage reduction verification
- Hash collision handling

### 13. Expression Complexity

**Test Priority**: P1 (High)
**Current Coverage**: ~40%
**Required Tests**:
- Deep nesting (10+ levels of parentheses)
- Long expressions (100+ operators)
- Operator precedence (mixed arithmetic, logical, comparison)
- Parentheses grouping
- Complex real-world expressions
- Performance with complexity

### 14. Multiple Rules Interaction

**Test Priority**: P1 (High)
**Current Coverage**: ~50%
**Required Tests**:
- Multiple rules matching same event
- Rule evaluation order independence
- Rules with overlapping conditions
- Rules with conflicting conditions
- Many rules (1000+) performance
- Rule priority/ordering (if applicable)

### 15. Error Handling & Validation

**Test Priority**: P1 (High)
**Current Coverage**: ~30%
**Required Tests**:
- Invalid rule syntax
- Invalid expression syntax
- Type errors in expressions
- Missing required fields in rules
- Invalid function arguments
- Stack overflow (deep nesting)
- Invalid YAML format
- Duplicate rule IDs
- Error message clarity

### 16. Edge Cases

**Test Priority**: P1 (High)
**Current Coverage**: ~35%
**Required Tests**:
- Empty events
- Empty rules
- Very large events (MB+)
- Very large arrays (10k+ elements)
- Unicode in strings
- Special characters in field names
- Extremely deep object nesting
- Maximum numeric values
- Minimum numeric values
- Boundary conditions for all operators

### 17. Concurrency & Thread Safety

**Test Priority**: P2 (Medium)
**Current Coverage**: ~40%
**Existing Tests**: engine_comprehensive_test.go has basic concurrency test
**Additional Tests Needed**:
- Concurrent rule evaluation
- Concurrent rule registration
- Race condition detection
- Thread safety of category engine
- Memory consistency under load

### 18. Performance & Benchmarks

**Test Priority**: P2 (Medium)
**Current Coverage**: Good (benchmarks_test.go exists)
**Existing Benchmarks**: 15 comprehensive benchmarks
**Additional Benchmarks Needed**:
- Arithmetic operations
- Complex null handling
- Deep expression nesting
- Large array quantifiers

## Known Limitations & TODOs

Based on source code analysis, the following are known limitations:

### From engine/engine_impl.go:

1. **Line 461**: `// TODO: handle functions in compare operand` - Functions in comparison operands not fully supported
2. **Line 553**: `// TODO: handle functions in operand` - Functions in arithmetic/boolean operands limited
3. **Line 1037**: `// TODO: verify that operation is compare` - Operation type verification needed
4. **Line 1051**: `// TODO: here we will need to check all function args` - Function argument validation incomplete
5. **Line 1087**: `// TODO: add support for time constants` - Time literals not supported
6. **Line 1382**: `// TODO: add len handling` - Array length handling incomplete
7. **Line 1420**: `// TODO: handle null explicitly` - Explicit null literal handling missing
8. **Line 1515**: `// TODO: support "len" for arrays` - Array length function not implemented
9. **Line 1559**: Frame limit comment - 20 level nesting limit for quantifiers
10. **Line 1713**: `// TODO: Handle this case` - Unary expression case unhandled

### Missing Functionality:

- Time literal constants (must use date() function)
- Array `len()` function
- Explicit null literal in expressions
- Some function argument validations
- Complete function support in all operand positions

## Test File Organization

Tests should be organized by **engine capability**, not application domain:

```
tests/
├── operators_arithmetic_test.go      # +, -, *, / operators
├── operators_comparison_test.go      # ==, !=, <, >, <=, >=, contains
├── operators_logical_test.go         # &&, ||, !
├── types_conversion_test.go          # Type conversions
├── types_reconciliation_test.go      # Type matching rules
├── types_null_handling_test.go       # Null in all contexts
├── functions_string_test.go          # regexpMatch, containsAny
├── functions_date_test.go            # date() parsing
├── quantifiers_forall_test.go        # forAll edge cases
├── quantifiers_forsome_test.go       # forSome edge cases
├── quantifiers_nesting_test.go       # Nested quantifiers
├── expressions_complex_test.go       # Deep nesting, precedence
├── expressions_edge_cases_test.go    # Boundary conditions
├── rules_multiple_test.go            # Multiple rules interaction
├── errors_validation_test.go         # Error handling
├── performance_stress_test.go        # Stress tests
├── benchmarks_test.go                # Performance benchmarks (existing)
├── engine_comprehensive_test.go      # Engine infrastructure (existing)
└── cateng_comprehensive_test.go      # Category engine (existing)
```

## Test Implementation Priorities

### Phase 2a (P0 - Critical, Untested)
1. `operators_arithmetic_test.go` - Arithmetic operators (0% coverage)
2. `types_null_handling_test.go` - Comprehensive null handling (20% coverage)
3. `quantifiers_forall_test.go` - forAll edge cases (40% coverage)
4. `quantifiers_forsome_test.go` - forSome edge cases (40% coverage)

### Phase 2b (P1 - High Priority Gaps)
5. `types_conversion_test.go` - Type conversions (25% coverage)
6. `expressions_complex_test.go` - Complex expressions (40% coverage)
7. `errors_validation_test.go` - Error handling (30% coverage)
8. `functions_string_test.go` - String functions (30% coverage)
9. `rules_multiple_test.go` - Multiple rules (50% coverage)

### Phase 2c (P2 - Completeness)
10. `operators_comparison_test.go` - Complete comparison coverage (60% coverage)
11. `operators_logical_test.go` - Complete logical coverage (50% coverage)
12. `functions_date_test.go` - Date function coverage
13. `quantifiers_nesting_test.go` - Nested quantifier edge cases
14. `expressions_edge_cases_test.go` - Boundary conditions

## Coverage Goals

- **Current**: 70.2%
- **Phase 2a Target**: 75%
- **Phase 2b Target**: 80%
- **Phase 2c Target**: 85%+

## Success Criteria

1. All P0 capabilities have comprehensive tests
2. All TODO items have corresponding tests documenting behavior
3. Coverage increased to 80%+
4. All edge cases documented and tested
5. Performance benchmarks for all major operations
6. Clear error messages for all validation failures
7. Thread safety verified under concurrent load
8. All tests pass consistently

## Notes

- Domain-specific tests (e-commerce, monitoring, etc.) are **test data**, not test organization
- Each test file should focus on ONE engine capability
- Test names should describe WHAT engine feature is being tested
- Use table-driven tests for comprehensive coverage
- Include both positive and negative test cases
- Document expected behavior for edge cases and TODOs
