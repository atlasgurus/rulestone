# Strict Mode Design Document

## Executive Summary

This document summarizes the research into implementing a "strict mode" for rulestone that prevents rules from matching when referenced fields are missing from events.

## Problem Statement

**Current Permissive Behavior**:
```yaml
Rule: age != 18
Event: { name: "john" }  # age field missing

Result: Rule MATCHES ✓
Reason: null != 18 → true (null semantics)
```

**Desired Strict Behavior**:
```yaml
Rule: age != 18
Event: { name: "john" }  # age field missing

Result: Rule does NOT match
Reason: Field not present = rule not applicable
```

## Root Cause Analysis

### Discovery: The DefaultCatList Mechanism

Through debugging and code analysis, I discovered that negative comparisons (`age != 18`) are handled through a sophisticated mechanism called **DefaultCatList**, which is distinct from `AlwaysEvaluateCategories`.

#### How It Works

**Step 1: Expression Parsing** (`engine/engine_impl.go:1289-1291`)
```
age != 18
  → Converted to: NOT(CompareCondition(age == 18))
```

**Step 2: Category Assignment** (`cateng/builder.go:276-285`)
```
NOT(CategoryCond(123))  // Category 123 = "age == 18"
  → processNotOp() converts to:
CategoryCond(1000000123)  // Negative category = 123 + MaxCategory
```

**Step 3: DefaultCatList Population** (`cateng/builder.go:636-639`)
```go
for cat := range fb.NegCats {  // For every category that has a negation
    result.DefaultCategories[cat] = len(result.DefaultCategories)
    result.DefaultCatList = append(result.DefaultCatList, cat)  // ← KEY!
}
```

**Step 4: Runtime Evaluation** (`cateng/category_engine.go:83-95`)
```go
// Process default categories
for i, cat := range f.FilterTables.DefaultCatList {
    if !defaultCatMap[i] {  // ← If category didn't fire (field missing)
        negCat, found := f.FilterTables.NegCats[cat]
        csml := catToCatSetMask.Get(negCat)
        if csml != nil {
            applyCatSetMasks(csml, matchMaskArray, &result, f)  // ← Fire negative!
        }
    }
}
```

### Key Insight

**DefaultCatList categories are "default true"** - they're assumed to match **unless proven otherwise**. When a field is missing:
1. Normal category (`age == 18`) doesn't fire (no attribute callback)
2. Runtime sees category didn't fire in `defaultCatMap`
3. Automatically evaluates the negative category (`NOT age == 18`)
4. Result: `age != 18` matches even though age is missing!

## Null Handling Semantics

### No Distinction Between Missing and Explicit Null

**Critical Design Decision**: Rulestone treats these identically:
- Field doesn't exist in event: `{}`
- Field exists with null value: `{age: null}`

Both become `NullOperand` internally.

**Rationale**:
- JSON doesn't meaningfully distinguish between them
- Rule authors typically mean "has a value" when checking `!= null`
- Simpler implementation (single code path)
- Better performance

### Null Comparison Rules (SQL-like)

```
null == null  → true
null == value → false
null != value → true  ← This causes the issue with missing fields!
null > value  → false (not orderable)
null < value  → false (not orderable)
```

### Test Coverage

Comprehensive null handling tests in `tests/data/types_null_handling.yaml`:
- 52 test cases covering all scenarios
- Missing fields vs explicit null
- All comparison operators
- Logical operations (AND, OR)
- Arithmetic operations
- Function behavior (`length()`, `hasValue()`)

## Implementation Strategy for Strict Mode

### Recommended Approach

**Add a flag to disable DefaultCatList processing**

#### Option 1: Build-Time Flag (RECOMMENDED)
```go
// In cateng/builder.go:636-639
func (fb *FilterBuilder) Build(...) *FilterTables {
    // ...
    if !fb.StrictMode {  // NEW: Only populate in permissive mode
        for cat := range fb.NegCats {
            result.DefaultCategories[cat] = len(result.DefaultCategories)
            result.DefaultCatList = append(result.DefaultCatList, cat)
        }
    }
    // ...
}
```

```go
// In cateng/category_engine.go:83-95
func (f *CategoryEngine) MatchEvent(cats []types.Category) []condition.RuleIdType {
    // ...
    if !f.StrictMode {  // NEW: Only process in permissive mode
        for i, cat := range f.FilterTables.DefaultCatList {
            if !defaultCatMap[i] {
                negCat, found := f.FilterTables.NegCats[cat]
                // ... process negative category
            }
        }
    }
    // ...
}
```

**Advantages**:
- Clean separation of concerns
- No runtime overhead in strict mode
- DefaultCatList is empty, saving memory
- Performance benefit (skip loop entirely)

#### Option 2: Runtime Flag
Check flag in the loop rather than skipping list construction.

**Disadvantages**:
- Still builds DefaultCatList (memory overhead)
- Still iterates empty loop (minor performance cost)

### Where to Add the Flag

**RuleEngineRepo** (engine/engine_api.go):
```go
type RuleEngineRepo struct {
    Rules      []*GeneralRuleRecord
    Optimize   bool
    StrictMode bool  // NEW: default false for backward compatibility
}
```

**LoadOption**:
```go
func WithStrictMode(strict bool) LoadOption {
    return func(c *loadConfig) {
        c.strictMode = strict
    }
}
```

**Propagate to CategoryEngine**:
```go
catEngine := cateng.NewCategoryEngine(&compCondRepo.RuleRepo, &cateng.Options{
    OrOptimizationFreqThreshold:  orThreshold,
    AndOptimizationFreqThreshold: andThreshold,
    StrictMode:                   repo.StrictMode,  // NEW
})
```

## Impact Analysis

### What Changes in Strict Mode

**Negative Comparisons**:
```
age != 18       → No match when age missing
status != "active" → No match when status missing
count != 0      → No match when count missing
```

**Null Checks** (UNCHANGED):
```
age == null     → Still matches when age missing
age != null     → Still works correctly
```

**Positive Comparisons** (UNCHANGED):
```
age > 18        → Already doesn't match when age missing
status == "active" → Already doesn't match when status missing
```

### Breaking Changes

**None** if default is `StrictMode = false`:
- Existing behavior preserved
- Opt-in via `WithStrictMode(true)`
- No API changes to `MatchEvent()`

### Performance Impact

**Strict Mode Benefits**:
- Skip DefaultCatList construction (memory savings)
- Skip default category processing loop (CPU savings)
- Fewer categories evaluated per event

**Estimated**: ~5-10% faster in strict mode for rules with many negations

## Test Coverage Needed

### Unit Tests
- DefaultCatList empty when StrictMode = true
- DefaultCatList populated when StrictMode = false
- Negative categories don't fire when field missing (strict)
- Negative categories still fire when field present (both modes)

### Integration Tests
- Rules with `!=` don't match missing fields (strict)
- Rules with `!=` still match when field present but not equal (strict)
- Rules with `==` null checks still work (both modes)
- Mixed positive/negative comparisons (both modes)

### Regression Tests
- All existing tests pass with StrictMode = false
- No behavior changes in permissive mode

## Documentation Updates

### ARCHITECTURE.md

Added two major sections:

**1. Negative Categories & DefaultCatList Pattern**
- Explains the mechanism discovered during this research
- Complete flow from parsing to runtime evaluation
- Design rationale and implications

**2. Null Handling Semantics** (Expanded)
- Why no distinction between missing and explicit null
- Comprehensive comparison rules
- Practical examples with test references
- Functions and edge cases

### README.md

Should add:
- WithStrictMode() usage example
- Behavior differences table
- Migration guide for existing users

## Open Questions

1. **Should explicit null be treated differently in strict mode?**
   - Current proposal: No, maintain existing null semantics
   - Alternative: Distinguish missing vs explicit null (major change)

2. **Should there be per-rule strict mode?**
   - Current proposal: Engine-wide setting
   - Alternative: Per-rule metadata flag (more complex)

3. **Should null checks work differently in strict mode?**
   - Current proposal: `field == null` still matches missing fields
   - Alternative: Only match explicit null (breaking change)

## Next Steps

1. Review this design document
2. Decide on open questions
3. Implement StrictMode flag with tests
4. Update documentation
5. Add examples and migration guide
