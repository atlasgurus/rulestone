# Implementation Plan: time.Time Support for Rulestone

## Executive Summary

The rulestone library currently panics when processing events containing `time.Time` or `*time.Time` fields. While the library has `TimeOperand` infrastructure in place, there are two critical gaps:

1. **Input Gap**: `NewInterfaceOperand()` doesn't recognize Go's `time.Time` or `*time.Time` types
2. **Output Gap**: `MatchEvent()` doesn't handle `TimeOperand` (and other operand types) in category evaluation results

## Root Cause Analysis

### Problem 1: NewInterfaceOperand() Missing time.Time Handlers

**Location**: `condition/condition.go:938-962`

```go
func NewInterfaceOperand(v interface{}, ctx *types.AppContext) Operand {
    switch n := v.(type) {
    case nil:
        return NewNullOperand(nil)
    case int:
        return NewFloatOperand(float64(n))
    case int64:
        return NewFloatOperand(float64(n))
    case string:
        return NewStringOperand(n)
    case float64:
        return NewFloatOperand(n)
    case bool:
        return NewBooleanOperand(n)
    case map[string]interface{}:
        return NewErrorOperand(...)
    case []interface{}:
        return NewErrorOperand(...)
    default:
        panic("Should not get here")  // ← time.Time falls through here
    }
}
```

**Issue**: When a `time.Time` value is passed, it doesn't match any case and triggers the panic.

### Problem 2: MatchEvent() Incomplete Operand Handling

**Location**: `engine/engine_api.go:546-595`

```go
func (f *RuleEngine) MatchEvent(v interface{}) []condition.RuleIdType {
    // ... event mapping and category evaluation ...

    switch r := result.(type) {
    case condition.ErrorOperand:
        // Handle error
    case condition.BooleanOperand:
        // Handle boolean
    case *condition.ListOperand:
        // Handle list
    case condition.IntOperand:
        // Handle int
    default:
        panic("should not get here")  // ← Missing: StringOperand, FloatOperand, TimeOperand, NullOperand
    }
}
```

**Issue**: Category evaluation can return ANY operand type, but only 4 are handled. This affects not just `TimeOperand`, but also `StringOperand`, `FloatOperand`, and `NullOperand`.

## Current TimeOperand Implementation Status

### ✅ Already Implemented

The codebase already has robust `TimeOperand` support:

1. **Type Definition** (condition.go:508):
   ```go
   type TimeOperand time.Time
   ```

2. **Core Methods**:
   - `NewTimeOperand(val time.Time) Operand`
   - `GetKind() OperandKind` → Returns `TimeOperandKind = 5`
   - `GetHash() uint64` → Uses `UnixNano()` for hashing
   - `Equals(o SetElement) bool` → Uses `time.Time.Equal()`
   - `Greater(o Operand) bool` → Uses `time.Time.After()`
   - `Evaluate(...)` → Returns self
   - `IsConst() bool` → Returns `true`

3. **Conversion Methods** (condition.go:514-530):
   - To `IntOperand`: Converts to Unix nanoseconds
   - To `FloatOperand`: Converts to Unix nanoseconds as float64
   - To `StringOperand`: Formats as RFC3339Nano
   - To `TimeOperandKind`: Returns self
   - To `NullOperandKind`: Returns null
   - To `BooleanOperandKind`: Panics (intentionally unsupported)

4. **Reverse Conversions** (from other types to TimeOperand):
   - `IntOperand.Convert(TimeOperandKind)`: `time.Unix(0, int64(v))`
   - `StringOperand.Convert(TimeOperandKind)`: Uses `dateparse.ParseAny()`
   - `FloatOperand`: Similar to IntOperand

### ❌ Missing Implementation

1. No `time.Time` or `*time.Time` handling in `NewInterfaceOperand()`
2. No `TimeOperand` handling in `MatchEvent()` result switch
3. No tests for time values in events
4. No documentation for time field usage

## Implementation Plan

### Phase 1: Core Fixes (Required for Basic Functionality)

#### Fix 1.1: Add time.Time Support to NewInterfaceOperand()

**File**: `condition/condition.go`
**Location**: After line 948 (in `NewInterfaceOperand()` function)

```go
func NewInterfaceOperand(v interface{}, ctx *types.AppContext) Operand {
    switch n := v.(type) {
    case nil:
        return NewNullOperand(nil)
    case int:
        return NewFloatOperand(float64(n))
    case int64:
        return NewFloatOperand(float64(n))
    case string:
        return NewStringOperand(n)
    case float64:
        return NewFloatOperand(n)
    case bool:
        return NewBooleanOperand(n)
    // NEW: Handle time.Time values
    case time.Time:
        return NewTimeOperand(n)
    // NEW: Handle *time.Time pointers
    case *time.Time:
        if n == nil {
            return NewNullOperand(nil)
        }
        return NewTimeOperand(*n)
    case map[string]interface{}:
        return NewErrorOperand(...)
    case []interface{}:
        return NewErrorOperand(...)
    default:
        panic("Should not get here")
    }
}
```

**Key Decisions**:
- `time.Time` values → `TimeOperand` directly
- `*time.Time` pointers → Dereference if non-nil, `NullOperand` if nil
- Maintains consistency with other type handling

#### Fix 1.2: Complete MatchEvent() Operand Handling

**File**: `engine/engine_api.go`
**Location**: Around line 580-591

```go
// Current incomplete implementation:
switch r := result.(type) {
case condition.ErrorOperand:
    // error handling
case condition.BooleanOperand:
    // boolean handling
case *condition.ListOperand:
    // list handling
case condition.IntOperand:
    // int handling
default:
    panic("should not get here")
}

// NEW: Complete implementation:
switch r := result.(type) {
case condition.ErrorOperand:
    // Keep existing error handling

case condition.BooleanOperand:
    // Keep existing boolean handling
    cat := catEvaluator.GetCategory()
    if r {
        eventCategories = append(eventCategories, cat)
    }

case *condition.ListOperand:
    // Keep existing list handling
    for _, c := range r.List {
        cat := types.Category(c.(condition.IntOperand))
        eventCategories = append(eventCategories, cat)
    }

case condition.IntOperand:
    // Keep existing int handling
    if r != 0 {
        eventCategories = append(eventCategories, types.Category(r))
    }

// NEW CASES: Handle remaining operand types
case condition.FloatOperand:
    // Treat non-zero floats as truthy for category matching
    if r != 0.0 {
        cat := catEvaluator.GetCategory()
        eventCategories = append(eventCategories, cat)
    }

case condition.StringOperand:
    // Treat non-empty strings as truthy for category matching
    if len(r) > 0 {
        cat := catEvaluator.GetCategory()
        eventCategories = append(eventCategories, cat)
    }

case condition.TimeOperand:
    // Treat non-zero times as truthy for category matching
    if !time.Time(r).IsZero() {
        cat := catEvaluator.GetCategory()
        eventCategories = append(eventCategories, cat)
    }

case condition.NullOperand:
    // Null operands are falsy - don't add category
    // (do nothing)

default:
    // This should now be unreachable for standard operand types
    panic(fmt.Sprintf("Unexpected operand type in category evaluation: %T", result))
}
```

**Design Rationale**:

The new cases follow the "truthiness" pattern used by existing handlers:
- `IntOperand`: Truthy if non-zero
- `FloatOperand`: Truthy if non-zero (NEW)
- `StringOperand`: Truthy if non-empty (NEW)
- `TimeOperand`: Truthy if not zero time (NEW)
- `BooleanOperand`: Truthy if true
- `NullOperand`: Always falsy (NEW)

This enables category expressions that evaluate to different types:
```yaml
# Boolean expression
expression: total_accounts > 5

# Numeric expression
expression: risk_score * 2

# String expression
expression: user_name

# Time expression
expression: last_login_time
```

### Phase 2: Comprehensive Testing Strategy

#### Test Suite 2.1: Unit Tests for NewInterfaceOperand()

**File**: `condition/condition_test.go` (new or existing)

```go
func TestNewInterfaceOperand_TimeTypes(t *testing.T) {
    ctx := types.NewAppContext()

    t.Run("time.Time value", func(t *testing.T) {
        now := time.Now()
        op := NewInterfaceOperand(now, ctx)

        require.Equal(t, TimeOperandKind, op.GetKind())
        require.Equal(t, now, time.Time(op.(TimeOperand)))
    })

    t.Run("*time.Time non-nil pointer", func(t *testing.T) {
        now := time.Now()
        op := NewInterfaceOperand(&now, ctx)

        require.Equal(t, TimeOperandKind, op.GetKind())
        require.Equal(t, now, time.Time(op.(TimeOperand)))
    })

    t.Run("*time.Time nil pointer", func(t *testing.T) {
        var nilTime *time.Time = nil
        op := NewInterfaceOperand(nilTime, ctx)

        require.Equal(t, NullOperandKind, op.GetKind())
    })

    t.Run("time.Time zero value", func(t *testing.T) {
        var zeroTime time.Time
        op := NewInterfaceOperand(zeroTime, ctx)

        require.Equal(t, TimeOperandKind, op.GetKind())
        require.True(t, time.Time(op.(TimeOperand)).IsZero())
    })

    t.Run("time.Time Unix epoch", func(t *testing.T) {
        epoch := time.Unix(0, 0)
        op := NewInterfaceOperand(epoch, ctx)

        require.Equal(t, TimeOperandKind, op.GetKind())
        require.Equal(t, epoch, time.Time(op.(TimeOperand)))
    })

    t.Run("time.Time with location", func(t *testing.T) {
        loc, _ := time.LoadLocation("America/New_York")
        now := time.Now().In(loc)
        op := NewInterfaceOperand(now, ctx)

        require.Equal(t, TimeOperandKind, op.GetKind())
        require.True(t, now.Equal(time.Time(op.(TimeOperand))))
    })
}
```

**Coverage**: Type conversion, nil handling, edge cases, timezone awareness

#### Test Suite 2.2: Integration Tests for MatchEvent()

**File**: `engine/engine_test.go` or new test file

```go
func TestMatchEvent_TimeOperandCategories(t *testing.T) {
    repo := engine.NewRuleEngineRepo()

    // Rule that references a time field
    yamlRule := `- metadata:
    id: 1
    name: Recent Login Check
  expression: last_login != null
`

    result, err := repo.LoadRulesFromString(yamlRule,
        engine.WithValidate(true),
        engine.WithFileFormat("yaml"),
    )
    require.NoError(t, err)
    require.True(t, result.ValidationOK)

    ruleEngine, err := engine.NewRuleEngine(repo)
    require.NoError(t, err)

    t.Run("time.Time value triggers match", func(t *testing.T) {
        event := map[string]interface{}{
            "last_login": time.Now(),
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("*time.Time pointer triggers match", func(t *testing.T) {
        now := time.Now()
        event := map[string]interface{}{
            "last_login": &now,
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("nil *time.Time does not match", func(t *testing.T) {
        var nilTime *time.Time = nil
        event := map[string]interface{}{
            "last_login": nilTime,
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.NotContains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("zero time.Time triggers match", func(t *testing.T) {
        // Zero time is still a valid time value, not null
        var zeroTime time.Time
        event := map[string]interface{}{
            "last_login": zeroTime,
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })
}
```

**Coverage**: Event matching with time fields, nil handling, zero time behavior

#### Test Suite 2.3: Time Comparison Operations in Rules

**File**: `engine/engine_time_comparison_test.go` (new)

```go
func TestRuleEngine_TimeComparisons(t *testing.T) {
    repo := engine.NewRuleEngineRepo()

    // Reference time: 2024-01-15 12:00:00 UTC
    refTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
    before := refTime.Add(-1 * time.Hour)
    after := refTime.Add(1 * time.Hour)

    yamlRules := `
- metadata:
    id: 1
    name: After Reference Time
  expression: event_time > "2024-01-15T12:00:00Z"

- metadata:
    id: 2
    name: Before Reference Time
  expression: event_time < "2024-01-15T12:00:00Z"

- metadata:
    id: 3
    name: Equal Reference Time
  expression: event_time == "2024-01-15T12:00:00Z"

- metadata:
    id: 4
    name: Recent Event (within 1 day)
  expression: now() - event_time < 86400
`

    result, err := repo.LoadRulesFromString(yamlRules,
        engine.WithValidate(true),
        engine.WithFileFormat("yaml"),
    )
    require.NoError(t, err)
    require.True(t, result.ValidationOK)

    ruleEngine, err := engine.NewRuleEngine(repo)
    require.NoError(t, err)

    t.Run("time.Time after reference", func(t *testing.T) {
        event := map[string]interface{}{
            "event_time": after,
        }
        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
        require.NotContains(t, matchedIDs, condition.RuleIdType(2))
        require.NotContains(t, matchedIDs, condition.RuleIdType(3))
    })

    t.Run("time.Time before reference", func(t *testing.T) {
        event := map[string]interface{}{
            "event_time": before,
        }
        matchedIDs := ruleEngine.MatchEvent(event)
        require.NotContains(t, matchedIDs, condition.RuleIdType(1))
        require.Contains(t, matchedIDs, condition.RuleIdType(2))
        require.NotContains(t, matchedIDs, condition.RuleIdType(3))
    })

    t.Run("time.Time equal reference", func(t *testing.T) {
        event := map[string]interface{}{
            "event_time": refTime,
        }
        matchedIDs := ruleEngine.MatchEvent(event)
        require.NotContains(t, matchedIDs, condition.RuleIdType(1))
        require.NotContains(t, matchedIDs, condition.RuleIdType(2))
        require.Contains(t, matchedIDs, condition.RuleIdType(3))
    })

    t.Run("*time.Time pointer comparisons", func(t *testing.T) {
        afterPtr := after
        event := map[string]interface{}{
            "event_time": &afterPtr,
        }
        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })
}

func TestRuleEngine_TimezoneHandling(t *testing.T) {
    repo := engine.NewRuleEngineRepo()

    // Same instant in different timezones
    utcTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
    estLoc, _ := time.LoadLocation("America/New_York")
    estTime := utcTime.In(estLoc) // 07:00:00 EST

    yamlRule := `- metadata:
    id: 1
    name: Same Instant Check
  expression: event_time == "2024-01-15T12:00:00Z"
`

    result, err := repo.LoadRulesFromString(yamlRule,
        engine.WithValidate(true),
        engine.WithFileFormat("yaml"),
    )
    require.NoError(t, err)

    ruleEngine, err := engine.NewRuleEngine(repo)
    require.NoError(t, err)

    t.Run("UTC time matches", func(t *testing.T) {
        event := map[string]interface{}{"event_time": utcTime}
        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("EST time matches same instant", func(t *testing.T) {
        event := map[string]interface{}{"event_time": estTime}
        matchedIDs := ruleEngine.MatchEvent(event)
        // Should match because time.Time.Equal() compares instants
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })
}
```

**Coverage**: Comparison operators, timezone handling, time arithmetic

#### Test Suite 2.4: Time Conversion Between Operand Types

**File**: `condition/condition_time_conversion_test.go` (new)

```go
func TestTimeOperand_Conversions(t *testing.T) {
    refTime := time.Date(2024, 1, 15, 12, 30, 45, 123456789, time.UTC)
    expectedNano := refTime.UnixNano()

    t.Run("TimeOperand to IntOperand", func(t *testing.T) {
        timeOp := NewTimeOperand(refTime)
        intOp := timeOp.Convert(IntOperandKind)

        require.Equal(t, IntOperandKind, intOp.GetKind())
        require.Equal(t, IntOperand(expectedNano), intOp)
    })

    t.Run("TimeOperand to FloatOperand", func(t *testing.T) {
        timeOp := NewTimeOperand(refTime)
        floatOp := timeOp.Convert(FloatOperandKind)

        require.Equal(t, FloatOperandKind, floatOp.GetKind())
        require.Equal(t, FloatOperand(expectedNano), floatOp)
    })

    t.Run("TimeOperand to StringOperand", func(t *testing.T) {
        timeOp := NewTimeOperand(refTime)
        strOp := timeOp.Convert(StringOperandKind)

        require.Equal(t, StringOperandKind, strOp.GetKind())
        // Should be RFC3339Nano format
        expectedStr := refTime.Format(time.RFC3339Nano)
        require.Equal(t, StringOperand(expectedStr), strOp)
    })

    t.Run("IntOperand to TimeOperand (Unix nano)", func(t *testing.T) {
        intOp := NewIntOperand(expectedNano)
        timeOp := intOp.Convert(TimeOperandKind)

        require.Equal(t, TimeOperandKind, timeOp.GetKind())
        // Should reconstruct the time
        reconstructed := time.Time(timeOp.(TimeOperand))
        require.True(t, refTime.Equal(reconstructed))
    })

    t.Run("StringOperand to TimeOperand (RFC3339)", func(t *testing.T) {
        strOp := NewStringOperand("2024-01-15T12:30:45Z")
        timeOp := strOp.Convert(TimeOperandKind)

        require.Equal(t, TimeOperandKind, timeOp.GetKind())
        parsed := time.Time(timeOp.(TimeOperand))
        expected := time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)
        require.True(t, expected.Equal(parsed))
    })

    t.Run("StringOperand to TimeOperand (various formats)", func(t *testing.T) {
        testCases := []string{
            "2024-01-15",
            "2024-01-15T12:30:45Z",
            "2024-01-15 12:30:45",
            "Jan 15, 2024",
            "01/15/2024",
        }

        for _, timeStr := range testCases {
            t.Run(timeStr, func(t *testing.T) {
                strOp := NewStringOperand(timeStr)
                timeOp := strOp.Convert(TimeOperandKind)

                // Should not be ErrorOperand
                require.NotEqual(t, ErrorOperandKind, timeOp.GetKind())
                require.Equal(t, TimeOperandKind, timeOp.GetKind())
            })
        }
    })

    t.Run("Invalid string to TimeOperand returns error", func(t *testing.T) {
        strOp := NewStringOperand("not a date")
        timeOp := strOp.Convert(TimeOperandKind)

        require.Equal(t, ErrorOperandKind, timeOp.GetKind())
    })
}

func TestTimeOperand_Comparison(t *testing.T) {
    time1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
    time2 := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)

    t.Run("Greater", func(t *testing.T) {
        op1 := NewTimeOperand(time1)
        op2 := NewTimeOperand(time2)

        require.False(t, op1.Greater(op2))
        require.True(t, op2.Greater(op1))
    })

    t.Run("Equals", func(t *testing.T) {
        op1 := NewTimeOperand(time1)
        op2 := NewTimeOperand(time1)
        op3 := NewTimeOperand(time2)

        require.True(t, op1.Equals(op2))
        require.False(t, op1.Equals(op3))
    })

    t.Run("Equals with different timezones, same instant", func(t *testing.T) {
        utcTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
        estLoc, _ := time.LoadLocation("America/New_York")
        estTime := utcTime.In(estLoc)

        op1 := NewTimeOperand(utcTime)
        op2 := NewTimeOperand(estTime)

        // Should be equal because they represent the same instant
        require.True(t, op1.Equals(op2))
    })
}

func TestTimeOperand_Hashing(t *testing.T) {
    time1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
    time2 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
    time3 := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)

    op1 := NewTimeOperand(time1)
    op2 := NewTimeOperand(time2)
    op3 := NewTimeOperand(time3)

    t.Run("Same times have same hash", func(t *testing.T) {
        require.Equal(t, op1.GetHash(), op2.GetHash())
    })

    t.Run("Different times have different hash", func(t *testing.T) {
        require.NotEqual(t, op1.GetHash(), op3.GetHash())
    })
}
```

**Coverage**: All conversion paths, comparison operations, hashing

#### Test Suite 2.5: Edge Cases and Error Handling

**File**: `engine/engine_time_edge_cases_test.go` (new)

```go
func TestMatchEvent_TimeEdgeCases(t *testing.T) {
    repo := engine.NewRuleEngineRepo()

    yamlRule := `- metadata:
    id: 1
    name: Time Field Check
  expression: event_time != null
`

    result, err := repo.LoadRulesFromString(yamlRule,
        engine.WithValidate(true),
        engine.WithFileFormat("yaml"),
    )
    require.NoError(t, err)

    ruleEngine, err := engine.NewRuleEngine(repo)
    require.NoError(t, err)

    t.Run("zero time.Time (default value)", func(t *testing.T) {
        var zeroTime time.Time
        event := map[string]interface{}{
            "event_time": zeroTime,
        }

        // Zero time is a valid time, not null
        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("nil *time.Time pointer", func(t *testing.T) {
        var nilTime *time.Time = nil
        event := map[string]interface{}{
            "event_time": nilTime,
        }

        // Nil pointer should be treated as null
        matchedIDs := ruleEngine.MatchEvent(event)
        require.NotContains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("Unix epoch", func(t *testing.T) {
        epoch := time.Unix(0, 0)
        event := map[string]interface{}{
            "event_time": epoch,
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("far future time", func(t *testing.T) {
        farFuture := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
        event := map[string]interface{}{
            "event_time": farFuture,
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("far past time", func(t *testing.T) {
        farPast := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
        event := map[string]interface{}{
            "event_time": farPast,
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("nanosecond precision", func(t *testing.T) {
        nanoTime := time.Date(2024, 1, 15, 12, 0, 0, 123456789, time.UTC)
        event := map[string]interface{}{
            "event_time": nanoTime,
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })
}

func TestMatchEvent_TimeWithOtherTypes(t *testing.T) {
    repo := engine.NewRuleEngineRepo()

    // Mixed type event
    yamlRule := `- metadata:
    id: 1
    name: Mixed Type Check
  expression: user_count > 5 && last_login != null && is_active == true
`

    result, err := repo.LoadRulesFromString(yamlRule,
        engine.WithValidate(true),
        engine.WithFileFormat("yaml"),
    )
    require.NoError(t, err)

    ruleEngine, err := engine.NewRuleEngine(repo)
    require.NoError(t, err)

    t.Run("mixed types including time.Time", func(t *testing.T) {
        event := map[string]interface{}{
            "user_count":  10,
            "last_login":  time.Now(),
            "is_active":   true,
            "device_name": "iPhone 12",
        }

        matchedIDs := ruleEngine.MatchEvent(event)
        require.Contains(t, matchedIDs, condition.RuleIdType(1))
    })

    t.Run("mixed types with nil *time.Time", func(t *testing.T) {
        var nilTime *time.Time = nil
        event := map[string]interface{}{
            "user_count": 10,
            "last_login": nilTime,
            "is_active":  true,
        }

        // Should not match because last_login is null
        matchedIDs := ruleEngine.MatchEvent(event)
        require.NotContains(t, matchedIDs, condition.RuleIdType(1))
    })
}
```

**Coverage**: Edge cases, boundary conditions, mixed type events

### Phase 3: Documentation and Examples

#### Doc 3.1: Update README or docs with time.Time usage

**Example documentation to add**:

```markdown
## Using Time Fields in Events

Rulestone supports `time.Time` and `*time.Time` fields in events. Time values can be compared against string literals or other time expressions.

### Supported Time Types

- `time.Time`: Go time values
- `*time.Time`: Pointers to time values (nil treated as null)

### Time Comparison Examples

```go
// Event with time fields
event := map[string]interface{}{
    "last_login": time.Now(),
    "account_created": time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
}

// Rules can compare times
rules := `
- metadata:
    id: 1
    name: Recent Login
  expression: last_login > "2024-01-01T00:00:00Z"

- metadata:
    id: 2
    name: Old Account
  expression: account_created < "2023-01-01T00:00:00Z"
`
```

### Time Conversions

Time values are automatically converted when compared with other types:
- To `int`/`float`: Unix nanoseconds
- To `string`: RFC3339Nano format
- From `string`: Parsed using flexible date parser

### Nil Handling

`*time.Time` nil pointers are treated as `null`:

```go
var nilTime *time.Time = nil
event := map[string]interface{}{
    "last_login": nilTime,  // Treated as null
}

// Check for null
expression: last_login == null  // Matches
```
```

#### Doc 3.2: Add examples to tests or example files

Create `examples/time_usage_example.go` demonstrating common patterns.

## Testing Matrix

| Test Category | File | Test Count | Priority |
|--------------|------|------------|----------|
| Unit - Type Conversion | `condition/condition_test.go` | 6 | High |
| Unit - Time Conversion | `condition/condition_time_conversion_test.go` | 10 | High |
| Integration - MatchEvent | `engine/engine_test.go` | 4 | High |
| Integration - Comparisons | `engine/engine_time_comparison_test.go` | 6 | High |
| Edge Cases | `engine/engine_time_edge_cases_test.go` | 11 | Medium |
| User-provided Test | Existing test file | 5 | High |

**Total**: ~42 new tests + user's 5 tests = **47 tests**

## Implementation Checklist

- [ ] Fix 1.1: Add time.Time handling to NewInterfaceOperand()
- [ ] Fix 1.2: Add TimeOperand case to MatchEvent() switch
- [ ] Fix 1.3: Add FloatOperand case to MatchEvent() switch
- [ ] Fix 1.4: Add StringOperand case to MatchEvent() switch
- [ ] Fix 1.5: Add NullOperand case to MatchEvent() switch
- [ ] Test 2.1: Unit tests for NewInterfaceOperand() time handling
- [ ] Test 2.2: Integration tests for MatchEvent() with time fields
- [ ] Test 2.3: Time comparison operation tests
- [ ] Test 2.4: Time conversion tests
- [ ] Test 2.5: Edge case tests
- [ ] Test 2.6: Verify user's reproduction test passes
- [ ] Doc 3.1: Update documentation
- [ ] Doc 3.2: Add usage examples

## Risk Assessment

### Low Risk
- Adding time.Time cases to NewInterfaceOperand() - additive change
- TimeOperand already exists and is well-tested

### Medium Risk
- MatchEvent() switch statement expansion - affects core matching logic
- Need to ensure "truthiness" semantics are consistent

### Mitigation
- Comprehensive test coverage before and after changes
- Run full existing test suite to catch regressions
- Review coverage reports

## Performance Considerations

- Time conversion via `UnixNano()` is O(1)
- Time comparison via `time.Time.Equal()` and `After()` is O(1)
- No allocation overhead for TimeOperand (value type wrapping time.Time)
- String parsing via `dateparse.ParseAny()` is more expensive but only happens during conversion

## Alternative Approaches Considered

### Alternative 1: Convert time.Time to float64 immediately
```go
case time.Time:
    return NewFloatOperand(float64(n.UnixNano()))
```

**Pros**: No changes needed to MatchEvent()
**Cons**: Loses time semantics, makes time comparisons harder in rules

**Decision**: Rejected - Keep TimeOperand for semantic clarity

### Alternative 2: Add custom time comparison functions
```go
expression: time_after(last_login, "2024-01-01")
```

**Pros**: Explicit time operations
**Cons**: Verbose, not idiomatic for rule expressions

**Decision**: Rejected - Use natural comparison operators

## Success Criteria

1. ✅ User's test case passes without panic
2. ✅ All 47+ tests pass
3. ✅ No regressions in existing test suite
4. ✅ Code coverage maintained or improved
5. ✅ Documentation updated with time usage examples

## Future Enhancements (Out of Scope)

- Time arithmetic functions: `add_days()`, `add_hours()`, etc.
- Duration type support
- Time formatting functions
- Relative time expressions: `now() - 1d`

## Questions for Review

1. Should zero time.Time be treated as null or as a valid time value?
   - **Decision**: Valid time value (not null)
   - **Rationale**: Consistent with Go semantics; zero time ≠ nil pointer

2. Should MatchEvent() panic or return error for unexpected operand types?
   - **Decision**: Keep panic with improved error message
   - **Rationale**: Indicates programming error, should fail fast

3. Should FloatOperand/StringOperand be "truthy" for category matching?
   - **Decision**: Yes, non-zero/non-empty are truthy
   - **Rationale**: Consistent with existing IntOperand behavior
