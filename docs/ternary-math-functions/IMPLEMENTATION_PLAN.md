# Implementation Plan: Ternary Operator & Math Functions

## Features to Implement

### 1. Ternary Operator

**Syntax**: `condition ? true_value : false_value`

**Examples**:
```yaml
expression: age >= 18 ? "adult" : "minor"
expression: premium ? discount * 2 : discount
expression: score > 90 ? "A" : score > 80 ? "B" : "C"  # Nested
```

**Implementation**: Add handling in `evalAstNode` for ternary expressions in Go 1.18+ (Go doesn't have native ternary, so we parse it manually or use if-expr syntax)

Actually, Go doesn't have ternary operator in AST. We need to either:
- Option A: Parse as custom syntax (complex)
- Option B: Use if-expr syntax like expr-lang
- Option C: Add custom ternary function

**Decision**: Start with function syntax, add operator syntax later if needed.

```yaml
expression: ternary(age >= 18, "adult", "minor")
expression: if(premium, discount * 2, discount)
```

### 2. Math Functions

**Functions**:
- `abs(x)` - Absolute value
- `ceil(x)` - Round up to integer
- `floor(x)` - Round down to integer
- `round(x, digits)` - Round to n decimal places
- `min(a, b, ...)` - Minimum of values
- `max(a, b, ...)` - Maximum of values
- `pow(base, exp)` - Power/exponentiation

**Examples**:
```yaml
expression: abs(balance) > 1000
expression: ceil(price * 1.08) <= budget
expression: round(score, 2) >= 95.50
expression: min(price1, price2, price3) < threshold
expression: max(age, 18) > 21
expression: pow(base, 2) > 100
```

## Implementation Steps

1. Add math functions to `evalAstNode` switch
2. Implement each function following pattern of existing functions
3. Add comprehensive tests
4. Update README

Estimated: 4-6 hours total
