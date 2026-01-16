# Time/Date Arithmetic Analysis & Recommendations

## Current Capabilities

### ✅ What We Have

**1. now() Function** (engine_impl.go:1609-1617)
```yaml
expression: now()
Returns: Current time as TimeOperand
```

**2. date() Function** (engine_impl.go:1602)
```yaml
expression: date("2023-03-29")
expression: date(user.registered)
Returns: Parsed date as TimeOperand
```

**3. Time Comparisons**
```yaml
expression: created_at > "2023-01-01T00:00:00Z"
expression: last_login < now()
expression: event_time >= date("2023-03-29")
```

**4. Time Arithmetic (Nanoseconds)**
```yaml
# Subtract times to get duration in nanoseconds
expression: (now() - event_time) < 86400000000000  # 1 day in nanoseconds
expression: (now() - created_at) < 432000000000000  # 5 days in nanoseconds
```

**5. Time Conversions**
- TimeOperand → IntOperand: Unix nanoseconds
- TimeOperand → StringOperand: RFC3339Nano format
- StringOperand → TimeOperand: Flexible date parsing (dateparse library)

### ❌ What's Missing

**1. Duration Literals**
```yaml
# Want this:
expression: created_at <= now() - 5d
expression: session_start > now() - 2h
expression: timestamp < now() - 30m

# Currently must write:
expression: (now() - created_at) <= 432000000000000  # 5 days in nanoseconds
```

**2. Duration Constants**
No support for:
- `5d` (5 days)
- `2h` (2 hours)
- `30m` (30 minutes)
- `10s` (10 seconds)

**3. Explicit Duration Arithmetic**
```yaml
# Want:
now() - 5d
now() + 2h
timestamp - 30m

# Currently: Must use nanoseconds
```

---

## Proposed Solution

### Option 1: Duration Literal Syntax (Recommended)

**Add duration literals to expression language**:

```yaml
# Days
expression: created_at <= now() - 5d
expression: last_login > now() - 7d

# Hours
expression: session_start > now() - 2h

# Minutes
expression: event_time >= now() - 30m

# Seconds
expression: timestamp < now() - 10s

# Combined
expression: deadline <= now() + 1d + 12h  # 1.5 days from now
```

**Syntax**: `<number><unit>` where unit is:
- `d` - days
- `h` - hours
- `m` - minutes
- `s` - seconds
- `ms` - milliseconds (optional)

### Implementation Approach

**1. Add DurationOperand Type** (condition/condition.go)

```go
type DurationOperand time.Duration

func NewDurationOperand(val time.Duration) Operand {
    return DurationOperand(val)
}

const DurationOperandKind OperandKind = 7  // After UndefinedOperandKind (6)

// Duration can convert to:
// - IntOperand: nanoseconds
// - FloatOperand: nanoseconds as float
// - StringOperand: formatted duration (e.g., "5d", "2h30m")
```

**2. Parse Duration Literals** (engine/engine_impl.go)

In `evalAstNode`, detect pattern like `5d`:

```go
case *ast.BinaryExpr:
    // Check if this looks like a duration: <number><ident>
    if n.Op == token.MUL {  // Go parser might see "5d" as multiplication
        // Or we might need to parse as a single identifier/literal
    }

// OR handle in BasicLit:
case *ast.BasicLit:
    if n.Kind == token.INT {
        // Check if next token is a duration unit (requires custom parsing)
    }
```

**Better approach**: Use function syntax initially:

```go
case "days", "hours", "minutes", "seconds":
    // days(5) → 5 * 24 * 60 * 60 * 1e9 nanoseconds
```

**3. Time Arithmetic Operations**

```go
// In arithmetic operations (engine_impl.go ~1691-1738)
switch n.Op {
case token.ADD:
    // TimeOperand + DurationOperand → TimeOperand
    // now() + days(5)
case token.SUB:
    // TimeOperand - DurationOperand → TimeOperand
    // now() - days(5)

    // TimeOperand - TimeOperand → DurationOperand
    // now() - created_at
}
```

---

## Option 2: Duration Functions (Simpler, No Parser Changes)

**Use function syntax** (easier to implement):

```yaml
expression: created_at <= now() - days(5)
expression: last_login > now() - hours(2)
expression: event_time >= now() - minutes(30)
expression: timestamp < now() - seconds(10)
```

**Implementation**:

```go
// In evalAstNode (engine_impl.go ~1600)
case "days":
    return funcDays(repo, n, scope)
case "hours":
    return funcHours(repo, n, scope)
case "minutes":
    return funcMinutes(repo, n, scope)
case "seconds":
    return funcSeconds(repo, n, scope)

func funcDays(repo *CompareCondRepo, n *ast.CallExpr, scope *ForEachScope) condition.Operand {
    if len(n.Args) != 1 {
        return condition.NewErrorOperand(fmt.Errorf("days() requires exactly one argument"))
    }

    argOperand := repo.evalAstNode(n.Args[0], scope)
    if argOperand.GetKind() == condition.ErrorOperandKind {
        return argOperand
    }

    return repo.CondFactory.NewExprOperand(
        func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
            arg := argOperand.Evaluate(event, frames)

            // Convert to numeric
            num := arg.Convert(condition.FloatOperandKind)
            if num.GetKind() == condition.ErrorOperandKind {
                return num
            }

            // Calculate nanoseconds: days * 24 * 60 * 60 * 1e9
            nanos := float64(num.(condition.FloatOperand)) * 24 * 60 * 60 * 1e9
            return condition.NewIntOperand(int64(nanos))
        }, argOperand)
}
```

**Pros**:
- ✅ No parser changes needed
- ✅ Works with current expression syntax
- ✅ Simple to implement (~100 lines)
- ✅ Immediately available

**Cons**:
- ❌ More verbose than literals (`days(5)` vs `5d`)
- ❌ Less elegant syntax

---

## Option 3: Hybrid Approach (Best of Both)

**Phase 1**: Implement duration functions (simple, immediate)
```yaml
expression: created_at <= now() - days(5)
```

**Phase 2**: Add duration literal syntax sugar (later)
```yaml
expression: created_at <= now() - 5d  # Parsed as days(5) internally
```

---

## Recommended Implementation: Option 2 (Duration Functions)

### Functions to Add

```go
days(n)      // n * 24 * 60 * 60 * 1e9 nanoseconds
hours(n)     // n * 60 * 60 * 1e9 nanoseconds
minutes(n)   // n * 60 * 1e9 nanoseconds
seconds(n)   // n * 1e9 nanoseconds
millis(n)    // n * 1e6 nanoseconds (optional)
```

### Usage Examples

```yaml
# Check if event happened within last 5 days
expression: created_at > now() - days(5)

# Check if session is recent (2 hours)
expression: session_start > now() - hours(2)

# Check if within 30 minutes
expression: event_time >= now() - minutes(30)

# Check if older than 7 days
expression: created_at <= now() - days(7)

# Compound: within 1.5 days
expression: timestamp > now() - days(1) - hours(12)

# Duration comparison (time difference)
expression: (now() - last_login) < days(30)
```

### Return Type Considerations

**Option A: Return IntOperand (Nanoseconds)**
```go
days(5) → IntOperand(432000000000000)
```
- Simple addition/subtraction with TimeOperand
- TimeOperand converts to/from int nanoseconds
- Works with existing arithmetic

**Option B: Return DurationOperand**
```go
days(5) → DurationOperand(432000000000000)
```
- More explicit type
- Need to handle Duration + Time, Time - Duration
- More complex but clearer semantics

**Recommendation**: Start with **Option A** (IntOperand) - works with existing code.

---

## Implementation Plan

### Files to Modify

**1. engine/engine_impl.go** (~100 lines)
- Add funcDays, funcHours, funcMinutes, funcSeconds
- Register in evalAstNode switch (~1600)
- Each function ~20 lines

**2. README.md** (~20 lines)
- Add "Time Arithmetic Functions" section
- Examples and usage

**3. tests/** (~150 lines)
- Test file: tests/time_arithmetic_test.go
- Test each function
- Test combinations
- Test with now()
- Edge cases

### Example Implementation

```go
// In engine_impl.go, add to evalAstNode switch around line 1630:
case "days":
    return repo.funcDays(n, scope)
case "hours":
    return repo.funcHours(n, scope)
case "minutes":
    return repo.funcMinutes(n, scope)
case "seconds":
    return repo.funcSeconds(n, scope)

// Implement functions:
func (repo *CompareCondRepo) funcDays(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
    if len(n.Args) != 1 {
        return condition.NewErrorOperand(fmt.Errorf("days() requires exactly one argument"))
    }

    argOperand := repo.evalAstNode(n.Args[0], scope)
    if argOperand.GetKind() == condition.ErrorOperandKind {
        return argOperand
    }

    if !argOperand.IsConst() {
        return condition.NewErrorOperand(fmt.Errorf("days() requires constant argument"))
    }

    return repo.CondFactory.NewExprOperand(
        func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
            arg := argOperand.Evaluate(event, frames)

            // Handle different input types
            switch arg.GetKind() {
            case condition.IntOperandKind:
                days := int64(arg.(condition.IntOperand))
                nanos := days * 24 * 60 * 60 * 1e9
                return condition.NewIntOperand(nanos)

            case condition.FloatOperandKind:
                days := float64(arg.(condition.FloatOperand))
                nanos := int64(days * 24 * 60 * 60 * 1e9)
                return condition.NewIntOperand(nanos)

            default:
                return condition.NewErrorOperand(fmt.Errorf("days() requires numeric argument"))
            }
        }, argOperand)
}

// Similar for hours, minutes, seconds with different multipliers
```

---

## Test Coverage

```go
func TestDurationFunctions(t *testing.T) {
    repo := engine.NewRuleEngineRepo()

    t.Run("days function", func(t *testing.T) {
        rule := `- metadata: {id: 1}
  expression: created_at > now() - days(5)`

        // Test with recent timestamp (3 days ago)
        // Test with old timestamp (7 days ago)
    })

    t.Run("hours function", func(t *testing.T) {
        rule := `- metadata: {id: 1}
  expression: session_start > now() - hours(2)`
    })

    t.Run("compound duration", func(t *testing.T) {
        rule := `- metadata: {id: 1}
  expression: timestamp > now() - days(1) - hours(12)`
        // 1.5 days
    })

    t.Run("duration comparison", func(t *testing.T) {
        rule := `- metadata: {id: 1}
  expression: (now() - last_login) < days(30)`
    })
}
```

---

## Alternative: Literal Syntax (Future Enhancement)

If we want `5d` syntax later, we'd need custom parsing:

**Option A: Postfix in Identifier**
```
Parse "5d" as: INT(5) followed by IDENT(d)
→ Convert to days(5) internally
```

**Option B: Custom Token Type**
```
Add DURATION token type to lexer
Recognize patterns: \d+[dhms]
```

**Option C: String Literal with Parser**
```yaml
expression: created_at <= now() - "5d"
# Parse string "5d" as duration
```

**Recommendation**: Start with functions, add literal syntax later if needed.

---

## Migration from Current Nanosecond Approach

**Existing usage**:
```yaml
expression: (now() - event_time) < 86400000000000
```

**With duration functions**:
```yaml
expression: (now() - event_time) < days(1)
```

**Backward compatible** - both work!

---

## Summary & Recommendation

### Current State
- ✅ now() function exists
- ✅ Time comparisons work
- ✅ Time arithmetic returns nanoseconds
- ❌ No human-readable duration units

### Recommended: Add Duration Functions

**Implement**:
- `days(n)` - Returns nanoseconds for n days
- `hours(n)` - Returns nanoseconds for n hours
- `minutes(n)` - Returns nanoseconds for n minutes
- `seconds(n)` - Returns nanoseconds for n seconds

**Usage**:
```yaml
created_at <= now() - days(5)
session_start > now() - hours(2)
```

**Effort**: ~2-3 hours
- 4 simple functions (~100 lines)
- Tests (~150 lines)
- Documentation (~20 lines)

**Files Modified**: 3 files
- engine/engine_impl.go
- README.md
- tests/time_arithmetic_test.go (new)

Should I implement this?
