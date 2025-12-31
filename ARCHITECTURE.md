# Rulestone Engine Architecture

**📖 For comprehensive design insights and debugging guidance, see [DESIGN_INSIGHTS.md](DESIGN_INSIGHTS.md).**

## Overview

Your memory is correct! The engine has a two-part architecture:

1. **Category Engine** (`cateng/category_engine.go`) - Evaluates complex logical expressions (AND/OR/NOT) against boolean categories
2. **Category Evaluator** (`engine/engine_impl.go`) - Evaluates comparison/arithmetic expressions (e.g., `a > 10`, `field == null`) and maps them to categories

## Part 1: Category Evaluator (`EvalCategoryRec`)

### Purpose
Converts non-logical expressions into boolean category values that can be fed into the Category Engine.

### Key Data Structures

```go
type EvalCategoryRec struct {
    Cat      types.Category         // Unique category ID
    Eval     condition.Operand      // The evaluation function
    AttrKeys []string               // Attribute paths this category depends on
}

type CompareCondRepo struct {
    // Maps attribute addresses (e.g., "a.b.c") to the category evaluators that reference them
    AttributeToCompareCondRecord map[string]*hashset.Set[*EvalCategoryRec]

    // Cache for common subexpression elimination
    CondToCompareCondRecord *hashmap.Map[condition.Condition, *EvalCategoryRec]

    // All category evaluators
    EvalCategoryRecs []*EvalCategoryRec

    RuleRepo              condition.RuleRepo
    ObjectAttributeMapper *objectmap.ObjectAttributeMapper
}
```

### Common Subexpression Elimination (CSE)

When multiple rules contain the same boolean subexpression (e.g., `a > 10`):
- The system assigns them the **same category** (via `CondToCompareCondRecord` cache)
- The expression is only evaluated **once per event**
- All rules referencing it share the result

Example:
```
Rule 1: a > 10 && b < 20    → Categories [Cat1, Cat2]
Rule 2: a > 10 && c == 5    → Categories [Cat1, Cat3]
                              ↑ Same category reused
```

### Attribute-to-Category Mapping

During compilation, for each comparison expression:
1. Extract all attribute paths it references (e.g., `a.b.c` in `a.b.c == 10`)
2. Register the category evaluator against those paths in `AttributeToCompareCondRecord`
3. Result: When an event contains `a.b.c`, we know to evaluate all categories that depend on it

```
AttributeToCompareCondRecord = {
    "a.b.c" → {EvalCategoryRec(Cat1), EvalCategoryRec(Cat5)},
    "x.y"   → {EvalCategoryRec(Cat2), EvalCategoryRec(Cat7)},
}
```

## Part 2: Category Engine (`cateng/category_engine.go`)

### Purpose
Evaluates logical expressions (AND/OR/NOT operations on categories) to determine which rules match.

### Key Data Structures

```go
type FilterTables struct {
    CatToCatSetMask    *CatSetMaskArray                     // Maps category → bitmask updates
    CatSetFilters      []*CatSetFilter                      // Bitmask patterns to match
    NegCats            map[types.Category]types.Category    // Normal cat → negative cat mapping
    DefaultCategories  map[types.Category]int               // Categories with default value true
    DefaultCatList     []types.Category                     // List of default categories
}
```

### Normal vs Negative Categories

**Key Insight**: The engine supports "negative" categories for handling absence/negation:

- **Normal Category**: Represents that a condition evaluated to `true`
- **Negative Category**: Represents that a condition's negation would be `true`

Mapping:
```
Normal Category Cat    → Negative Category (Cat + MaxCategory)
```

### Default Categories

For categories that have "default value true" (meaning they should be considered true unless proven false):
- Stored in `DefaultCategories` map and `DefaultCatList`
- Processed specially in `CategoryEngine.MatchEvent()` (lines 80-92)

## Event Matching Flow

### Step 1: Map Object to Attributes (`RuleEngine.MatchEvent` lines 466-479)

```go
event := f.compCondRepo.ObjectAttributeMapper.MapObject(v,
    // Callback for each attribute found in the event
    func(addr []int) {
        addrMatchId := objectmap.AddressMatchKey(addr)
        catEvaluators, ok := f.compCondRepo.AttributeToCompareCondRecord[addrMatchId]
        if ok {
            // Collect all category evaluators that depend on this attribute
            matchingCompareCondRecords.Put(catEvaluator)
        }
    })
```

**Problem Identified**: If an attribute is **missing** from the event, the callback is never called!
- Rules checking `field == null` never get evaluated
- Rules with no event dependencies (like `1 == 1`) never get evaluated

### Step 2: Evaluate Categories (lines 482-506)

For each category evaluator whose attributes were found:
```go
matchingCompareCondRecords.Each(func(catEvaluator *EvalCategoryRec) {
    result := catEvaluator.Evaluate(event, FrameStack[:])
    if result == true {
        eventCategories = append(eventCategories, cat)
    }
})
```

Produces a list of categories: `[Cat1, Cat5, Cat7, ...]`

### Step 3: Category Engine Matching (line 508, `category_engine.go` lines 62-94)

```go
func (f *CategoryEngine) MatchEvent(cats []types.Category) []condition.RuleIdType {
    // Process provided categories
    for _, cat := range cats {
        if i, ok := f.FilterTables.DefaultCategories[cat]; ok {
            // This default-true category evaluated to false, mark it
            defaultCatMap[i] = true
        }
        // Apply bitmasks...
    }

    // Process default categories (lines 81-92)
    for i, cat := range f.FilterTables.DefaultCatList {
        if !defaultCatMap[i] {
            // Category wasn't explicitly set to false, so use its negative category
            negCat := f.FilterTables.NegCats[cat]
            // Apply negative category bitmasks...
        }
    }
}
```

## The Bug: Missing Negative Category Registration

### Current Implementation Status

**What Works**:
✅ Negative categories exist in the category engine (`NegCats`, `DefaultCategories`)
✅ The category engine processes them correctly (lines 80-92 of `category_engine.go`)
✅ NOT operations create negative categories in the builder (`registerNegativeCat()` in `builder.go` lines 149-156)

**What's Broken**:
❌ **Category evaluators for null checks are NOT registered in the default/negative category system**
❌ When an attribute is missing, its evaluators are never triggered (callback not called)
❌ No mechanism to force evaluation of categories that should default to true

### Root Cause

In `engine_impl.go`, when processing comparisons like `field == null`:

1. The system creates an `EvalCategoryRec` and registers it against attribute `"field"` in `AttributeToCompareCondRecord`
2. During matching, if `"field"` is missing from the event → callback never fires → category never evaluated
3. The category engine never receives any signal that this category should be considered

**BUT**: The infrastructure for "always-evaluate" categories EXISTS in the category engine!
- `DefaultCategories` / `DefaultCatList` in FilterTables
- Special processing loop (lines 81-92 of category_engine.go)
- Just need to populate it correctly

## The Fix for Bug #2: Null Check Failure

### Strategy

For comparisons that check for null/missing values (e.g., `field == null`):

1. **Identify** which categories represent "negative" checks (absence of data)
2. **Register** them in a new `AlwaysEvaluateCategories` list in `CompareCondRepo`
3. **Evaluate** them unconditionally in `RuleEngine.MatchEvent`, even if their attributes weren't found
4. **Feed** their results (or lack thereof) into the Category Engine as negative categories

### Implementation Plan

#### Step 1: Track "Always-Evaluate" Categories

In `engine_impl.go`, add to `CompareCondRepo`:
```go
type CompareCondRepo struct {
    // ... existing fields ...

    // Categories that must be evaluated even if their attributes aren't in the event
    AlwaysEvaluateCategories *hashset.Set[*EvalCategoryRec]
}
```

#### Step 2: Register Null-Check Categories

In `processCompareCondition()`, detect null comparisons:
```go
// If comparing against null, this category must always be evaluated
if (compareCond.CompareOp == condition.CompareEqualOp ||
    compareCond.CompareOp == condition.CompareNotEqualOp) &&
   (compareCond.LeftOperand.IsNull() || compareCond.RightOperand.IsNull()) {
    repo.AlwaysEvaluateCategories.Put(evalCatRec)
}
```

#### Step 3: Evaluate Always-Evaluate Categories

In `RuleEngine.MatchEvent()`, after line 479:
```go
// Also evaluate categories that must always run (e.g., null checks)
f.compCondRepo.AlwaysEvaluateCategories.Each(func(catEvaluator *EvalCategoryRec) {
    matchingCompareCondRecords.Put(catEvaluator)
})
```

This ensures null-check categories are evaluated even when their fields are missing.

## Fix for Bug #1: Always-True Expressions

Similar approach: Detect constant expressions (no attribute references) and add to `AlwaysEvaluateCategories`.

## Fix for Bug #3: forAll with Empty Arrays (FIXED)

### The Challenge

Empty arrays presented a unique architectural challenge:
- `ObjectAttributeMap.Values` only stores **scalar leaf values**, not array containers
- Empty arrays have no elements, so nothing gets stored in `Values`
- When `GetNumElementsAtAddress` tried to access `Values[arrayIndex]`, it got `nil` and returned an error
- This caused empty arrays to be treated the same as missing arrays (both return `false`)

### The Solution

1. **Store Original Event**: Added `OriginalEvent` field to `ObjectAttributeMap` to keep a reference to the unmapped event
2. **Fallback Lookup**: Modified `GetNumElementsAtAddress` to check the original event when mapped Values don't contain the array
3. **Path Navigation**: Added `getValueFromOriginalEvent()` helper to navigate the original event by path (e.g., "items[]" → "items")
4. **Vacuous Truth**: Modified `genEvalForAllCondition` to return `true` when `numElements == 0`

This approach correctly distinguishes:
- **Missing array** (`{other: data}`): `GetNumElementsAtAddress` returns error → `false`
- **Empty array** (`{items: []}`): `GetNumElementsAtAddress` finds array in original event, returns `0` → `true`
- **Non-empty array** (`{items: [...]}`): Normal evaluation → evaluate condition for each element

---

## Summary: All Three Bugs Fixed

### Bug #1: Always-True Expressions ✅ FIXED
- **Solution**: Added to `AlwaysEvaluateCategories` when `AttrKeys` is empty
- **Performance**: ~454 ns/op (negligible overhead)

### Bug #2: Null Checks ✅ FIXED
- **Solution**: Added to `AlwaysEvaluateCategories` + special null comparison handling
- **Performance**: ~500 ns/op (negligible overhead)

### Bug #3: forAll with Empty Arrays ✅ FIXED
- **Solution**: Store `OriginalEvent` in `ObjectAttributeMap` + fallback lookup + vacuous truth logic
- **Performance**: ~1.3 μs/op for empty arrays, ~4.2 μs/op for 10 elements (no regression)

### Test Results
- **All 482 tests passing** (11 data-driven test files)
- **No performance regressions** in existing benchmarks
- **Comprehensive benchmark coverage** added for all three fixes

### Files Modified
1. `engine/engine_impl.go` - Added AlwaysEvaluateCategories logic, null handling, vacuous truth
2. `engine/engine_api.go` - Evaluate AlwaysEvaluateCategories in MatchEvent
3. `objectmap/object_attribute_map.go` - Added OriginalEvent field and fallback lookup
4. `tests/data/*.yaml` - Updated test expectations for fixed behavior
5. `tests/benchmarks_test.go` - Added benchmarks for bug fixes

### Documentation
- `ARCHITECTURE.md` - This file (architectural overview and bug analysis)
- `DESIGN_INSIGHTS.md` - Comprehensive design patterns and debugging guide

---

**For detailed implementation guidance, common pitfalls, and debugging tips, see [DESIGN_INSIGHTS.md](DESIGN_INSIGHTS.md).**
