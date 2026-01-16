# Implementation Plan: UndefinedOperand & Three-Valued Logic

## Decision: Distinguish Missing from Null EVERYWHERE

No modes, no flags. This is THE behavior.

---

## Core Semantics

### Undefined Propagation (Three-Valued Logic)

```
undefined == value    → undefined (except undefined==undefined → true)
undefined != value    → undefined (except undefined!=undefined → false)
undefined > value     → undefined
undefined < value     → undefined
!(undefined)          → undefined
undefined && false    → false (short circuit)
undefined && true     → undefined
undefined || true     → true (short circuit)
undefined || false    → undefined

Special cases:
undefined == undefined → true (checking "is field missing?" returns true)
undefined != undefined → false (same type)
undefined == null      → false (different types: missing ≠ explicit null)
undefined != null      → true (different types)
```

### Category Result Handling

```go
// In MatchEvent (engine_api.go:572)
switch r := result.(type) {
case condition.UndefinedOperand:
    // Don't add to eventCategories (not applicable)
case condition.BooleanOperand:
    if r {
        eventCategories = append(eventCategories, cat)
    }
case condition.NullOperand:
    // Don't add to eventCategories (falsey)
}
```

---

## Efficient Implementation: Keep Optimized DefaultCatList

### Key Insight: Track Evaluation Separately from Firing

**Current**: Single boolean map (did category fire?)
```go
defaultCatMap := make([]bool, len(DefaultCategories))
```

**New**: Two separate sets (cleaner than state tracking)
```go
// Track which default categories were evaluated (triggered by field presence)
evaluatedDefaultCats := types.NewHashSet[types.Category]()

// Track which categories fired (returned true) - already have this as 'cats' parameter
```

**Advantages**:
- Simple set membership checks
- No need for state structs or enums
- Clear separation of concerns

---

## Pattern 1: `field == undefined` - Efficient DefaultCatList

```yaml
expression: age == undefined
Event: { name: "john" }  # age missing
```

### Compilation

```go
// In processCompareCondition
if isUndefinedEqualityCheck(compareCond) {
    // Create FALSE category triggered by "age"
    evalCatRec := NewEvalCategoryRec(
        func(event, frames) Operand {
            // Evaluate the field
            field := evaluateField("age", event, frames)
            // Will return UndefinedOperand if missing, actual value if present

            // Compare to undefined
            if field.GetKind() == UndefinedOperandKind {
                return NewBooleanOperand(true)  // Field is undefined
            } else {
                return NewBooleanOperand(false)  // Field exists
            }
        })

    // Mark as undefined check
    evalCatRec.IsUndefinedEqualityCheck = true
    evalCatRec.FieldPath = "age"

    // Register against "age" attribute (efficient triggering)
    registerCatEvaluatorForAddress("age", evalCatRec)

    // Will be added to DefaultCatList during Build()
    return NewCategoryCond(evalCatRec.GetCategory())
}
```

Wait, this is confusing again. Let me reconsider what the category should evaluate to...

**Actually, the category should directly evaluate** `age == undefined`:
```go
evalCatRec := NewEvalCategoryRec(
    func(event, frames) Operand {
        age := evaluateField("age", event, frames)
        undefined := NewUndefinedOperand(nil)

        // Perform comparison: age == undefined
        return compareOperands(age, undefined, CompareEqualOp)
        // Missing: undefined == undefined → true
        // Present: value == undefined → false
    })
```

**Triggering**:
- If added to AlwaysEvaluateCategories: Evaluates every event (always knows result)
- If registered against "age": Only evaluates when age present (missing when age missing!)

**The problem**: When age missing and NOT in AlwaysEvaluateCategories:
- Category not triggered → doesn't evaluate
- Can't return true!

**So we need EITHER**:
1. AlwaysEvaluateCategories (evaluate every event)
2. DefaultCatList with inverted logic

---

## DefaultCatList with Three-State Tracking

### The Efficient Approach

**Create INVERTED category** for `age == undefined`:

```go
// Category checks: "does age exist?" (inverted from what we want)
evalCatRec := NewEvalCategoryRec(
    func(event, frames) Operand {
        age := evaluateField("age", event, frames)
        // Return TRUE if age EXISTS (has any value, even null)
        if age.GetKind() != UndefinedOperandKind {
            return NewBooleanOperand(true)  // Age exists
        } else {
            return NewBooleanOperand(false)  // Age missing
        }
    })

// Register against "age" attribute
registerCatEvaluatorForAddress("age", evalCatRec)

// Mark for DefaultCatList
evalCatRec.IsUndefinedEqualityCheck = true

// Add to DefaultCatList during Build()
// Create negative category
negCat := cat + MaxCategory
```

### Runtime with Three-State Tracking

**When age missing**:
```go
// Category not triggered
defaultCatMap[cat].Evaluated = false
defaultCatMap[cat].Fired = false

// DefaultCatList processing:
if !defaultCatMap[cat].Evaluated {  // Not evaluated (field missing)
    // Fire negative category (represents "age == undefined")
    negCat := NegCats[cat]
    processCat(negCat)  // Returns true ✓
}
```

**When age present (age=25)**:
```go
// Category triggered
// Evaluates: 25 != undefined → returns true (age exists)
// Category fires → added to eventCategories

defaultCatMap[cat].Evaluated = true  // WAS triggered
defaultCatMap[cat].Fired = true      // Returned true

// DefaultCatList processing:
if !defaultCatMap[cat].Evaluated {  // Evaluated! Skip.
    // Don't fire negative
}

// Result: Negative doesn't fire → false ✓
```

**Perfect!** The negative represents "age == undefined" and only fires when age is missing.

---

## Updated category_engine.go Code

### Current Code (Binary Tracking)

```go
defaultCatMap := make([]bool, len(DefaultCategories))

for _, cat := range cats {  // Categories that fired
    if i, ok := DefaultCategories[cat]; ok {
        defaultCatMap[i] = true
    }
}

for i, cat := range DefaultCatList {
    if !defaultCatMap[i] {  // Didn't fire
        negCat := NegCats[cat]
        processCat(negCat)
    }
}
```

### New Code (Two-Set Tracking)

```go
// Track which default categories were evaluated (field was present)
evaluatedDefaultCats := types.NewHashSet[types.Category]()

// Populate evaluated set
matchingCompareCondRecords.Each(func(catEval *EvalCategoryRec) {
    cat := catEval.GetCategory()
    if _, ok := f.FilterTables.DefaultCategories[cat]; ok {
        evaluatedDefaultCats.Put(cat)
    }
})

// Process default categories
for _, cat := range f.FilterTables.DefaultCatList {
    if !evaluatedDefaultCats.Contains(cat) {  // NOT evaluated (field missing)
        negCat := f.FilterTables.NegCats[cat]
        csml := catToCatSetMask.Get(negCat)
        if csml != nil {
            applyCatSetMasks(csml, matchMaskArray, &result, f)
        }
    }
}
```

**Simpler!** Just check set membership instead of tracking state.

---

## Complete Pattern Handling

| Pattern | Category Returns | Register Against | Default List? | Always Evaluate? | When Missing | When Present |
|---------|------------------|------------------|---------------|------------------|--------------|--------------|
| `age == undefined` | TRUE if exists | "age" | YES (inverted) | NO | Neg fires → true | Cat fires → neg suppressed → false |
| `age != undefined` | Same cat | Same | YES (negated) | NO | Doesn't fire → false | Fires → neg fires → true |
| `age != 18` | undefined!=18→undef | "age" | NO | NO | Doesn't fire → false | Fires → varies |
| `age == null` | Checks null | "age" | NO | YES | Eval → false | Eval → varies |

**Actually, let me reconsider** `age == null` - should it also use efficient triggering? Or is AlwaysEvaluateCategories acceptable since null checks are also meta-checks?

---

## Proposed Final Architecture

### Small Optimized DefaultCatList

**Only contains**: Undefined-equality checks (`field == undefined`)
- Category returns TRUE when field EXISTS
- Negative category represents "field missing"
- Only evaluates when field present (optimal)

**Size**: Typically 5-10 entries (vs 100+ currently)

### AlwaysEvaluateCategories

**Contains**:
- Null checks: `age == null`, `age != null`
- Constants: `1 == 1`, `true`
- Quantifiers: `forAll(...)`, `forSome(...)`
- (NOT undefined checks - those use DefaultCatList)

### Regular Categories (No Special Casing)

**All negations work naturally**:
- `age != 18` → undefined propagation → doesn't match when missing ✓
- `status != "active"` → undefined propagation ✓
- `!(age > 18)` → undefined propagation ✓

---

## Implementation Changes

### 1. Three-State Tracking (category_engine.go)

**Minimal change**:
```go
// OLD:
defaultCatMap := make([]bool, len(DefaultCategories))

// NEW:
defaultCatState := make([]DefaultCatState, len(DefaultCategories))
```

Track both "evaluated" and "fired" separately.

### 2. DefaultCatList Population (builder.go:636-639)

**OLD** (adds all negations):
```go
for cat := range fb.NegCats {
    result.DefaultCategories[cat] = len(result.DefaultCategories)
    result.DefaultCatList = append(result.DefaultCatList, cat)
}
```

**NEW** (only undefined-equality checks):
```go
for cat, evalCatRec := range fb.CategoryRecords {
    if evalCatRec.IsUndefinedEqualityCheck {
        result.DefaultCategories[cat] = len(result.DefaultCategories)
        result.DefaultCatList = append(result.DefaultCatList, cat)
    }
}
```

### 3. Mark Undefined Checks (engine_impl.go:605-617)

Add detection for undefined comparisons, mark the category.

---

---

## Complete Implementation Summary

### What Changes

**1. Add UndefinedOperand Type** (condition/condition.go):
```go
type UndefinedOperand struct {
    Source interface{}
}

const UndefinedOperandKind OperandKind = 6

func NewUndefinedOperand(source interface{}) Operand {
    return &UndefinedOperand{Source: source}
}

// Implement Operand interface (GetKind, GetHash, Equals, etc.)
```

**2. Parse `undefined` Keyword** (engine/engine_impl.go):
```go
case *ast.Ident:
    switch n.Name {
    case "undefined":  // NEW
        return condition.NewUndefinedOperand(nil)
    case "null":
        return condition.NewNullOperand(nil)
    // ...
    }
```

**3. Return Undefined for Missing Fields** (engine/engine_impl.go:1796):
```go
val := objectmap.GetNestedAttributeByAddress(...)
if val == nil {
    // Check if field exists in original event
    if fieldExistsInOriginalEvent(event, attributePath) {
        return condition.NewNullOperand(address)  // Explicit null
    } else {
        return condition.NewUndefinedOperand(address)  // Missing field
    }
}
```

**4. Undefined Propagation in Comparisons** (engine/engine_impl.go:226):
```go
// Check undefined FIRST (before null)
if xKind == UndefinedOperandKind && yKind == UndefinedOperandKind {
    // Both undefined
    switch compOp {
    case CompareEqualOp:
        return NewBooleanOperand(true)  // undefined == undefined
    case CompareNotEqualOp:
        return NewBooleanOperand(false)
    default:
        return NewUndefinedOperand(nil)  // Can't order
    }
}

if xKind == UndefinedOperandKind || yKind == UndefinedOperandKind {
    // One undefined
    switch compOp {
    case CompareEqualOp:
        return NewBooleanOperand(false)  // undefined != value
    case CompareNotEqualOp:
        return NewBooleanOperand(true)  // undefined != value
    default:
        return NewUndefinedOperand(nil)  // Propagate
    }
}

// Then existing null handling (undefined != null handled above)
```

**5. Undefined Propagation in Negation** (engine/engine_impl.go:709):
```go
result := eval.Evaluate(event, frames)

if result.GetKind() == UndefinedOperandKind {
    return result  // !(undefined) → undefined
}

// Existing boolean negation
return NewBooleanOperand(!bool(result.(BooleanOperand)))
```

**6. Undefined Propagation in AND** (engine/engine_impl.go:660):
```go
hasUndefined := false
for _, eval := range condEvaluators {
    result := eval.Func(event, frames)

    if result.GetKind() == ErrorOperandKind {
        return result  // Error propagates
    }

    // False short-circuits
    if result.GetKind() == BooleanOperandKind && !bool(result.(BooleanOperand)) {
        return NewBooleanOperand(false)
    }

    // Track undefined (doesn't short-circuit)
    if result.GetKind() == UndefinedOperandKind {
        hasUndefined = true
    }
}

// If any was undefined and none were false → undefined
if hasUndefined {
    return NewUndefinedOperand(nil)
}

// All were true
return NewBooleanOperand(true)
```

**7. Undefined Propagation in OR** (engine/engine_impl.go:685):
```go
hasUndefined := false
for _, eval := range condEvaluators {
    result := eval.Func(event, frames)

    if result.GetKind() == ErrorOperandKind {
        return result
    }

    // True short-circuits
    if result.GetKind() == BooleanOperandKind && bool(result.(BooleanOperand)) {
        return NewBooleanOperand(true)
    }

    // Track undefined
    if result.GetKind() == UndefinedOperandKind {
        hasUndefined = true
    }
}

// If any was undefined and none were true → undefined
if hasUndefined {
    return NewUndefinedOperand(nil)
}

// All were false
return NewBooleanOperand(false)
```

**8. Handle Undefined in Category Results** (engine/engine_api.go:572):
```go
switch r := result.(type) {
case condition.UndefinedOperand:  // NEW
    // Don't add to eventCategories (not applicable)
case condition.BooleanOperand:
    if r {
        eventCategories = append(eventCategories, cat)
    }
// ...
}
```

**9. Mark Undefined-Equality Checks** (engine/engine_impl.go:610):
```go
isUndefinedCheck := compareCond.LeftOperand.GetKind() == UndefinedOperandKind ||
    compareCond.RightOperand.GetKind() == UndefinedOperandKind

isUndefinedEqualityCheck := isUndefinedCheck &&
    compareCond.CompareOp == CompareEqualOp

if isUndefinedEqualityCheck {
    evalCatRec.IsUndefinedEqualityCheck = true
    evalCatRec.FieldPath = extractFieldPath(compareCond)
}
```

**10. Update DefaultCatList Population** (cateng/builder.go:636-639):
```go
// OLD: Add ALL negated categories
// for cat := range fb.NegCats {
//     result.DefaultCategories[cat] = ...
// }

// NEW: Only add undefined-equality checks
for catId, evalCatRec := range fb.CategoryRecords {
    if evalCatRec.IsUndefinedEqualityCheck {
        result.DefaultCategories[catId] = len(result.DefaultCategories)
        result.DefaultCatList = append(result.DefaultCatList, catId)
    }
}
```

**11. Update DefaultCatList Processing** (cateng/category_engine.go:64-95):
```go
func (f *CategoryEngine) MatchEvent(cats []types.Category) []condition.RuleIdType {
    matchMaskArray := make([]types.Mask, len(f.FilterTables.NegCats)+len(f.FilterTables.CatSetFilters))
    result := make([]condition.RuleIdType, 0, 100)

    // NEW: Track which default categories were evaluated
    evaluatedDefaultCats := types.NewHashSet[types.Category]()

    catToCatSetMask := f.FilterTables.CatToCatSetMask
    for _, cat := range cats {
        // Track if this cat is a default category that was evaluated
        if _, ok := f.FilterTables.DefaultCategories[cat]; ok {
            evaluatedDefaultCats.Put(cat)  // Mark as evaluated
        }

        csml := catToCatSetMask.Get(cat)
        if csml != nil {
            applyCatSetMasks(csml, matchMaskArray, &result, f)
        }
    }

    // Process default categories
    for _, cat := range f.FilterTables.DefaultCatList {
        if !evaluatedDefaultCats.Contains(cat) {  // NOT evaluated (field missing)
            negCat, found := f.FilterTables.NegCats[cat]
            if !found {
                panic("negCat must exist for default category")
            }
            csml := catToCatSetMask.Get(negCat)
            if csml != nil {
                applyCatSetMasks(csml, matchMaskArray, &result, f)
            }
        }
    }

    return result
}
```

**Wait**, I need to also track evaluated categories from `matchingCompareCondRecords`, not just from fired categories...

Let me reconsider. The `cats` parameter contains categories that **fired** (returned true). But we need to know which categories were **evaluated** (even if they returned false).

**In MatchEvent** (engine_api.go:569):
```go
matchingCompareCondRecords.Each(func(catEvaluator *EvalCategoryRec) {
    result := catEvaluator.Evaluate(event, FrameStack[:])
    // This catEvaluator WAS evaluated (regardless of result)

    switch r := result.(type) {
    case condition.BooleanOperand:
        if r {
            eventCategories = append(eventCategories, cat)  // Fired
        }
        // Even if false, we know it was evaluated
    }
})
```

So we need to pass **which categories were evaluated** to the category engine, not just which fired.

**Solution**: Pass both sets to CategoryEngine.MatchEvent():
```go
func (f *CategoryEngine) MatchEvent(
    firedCats []types.Category,        // Categories that returned true
    evaluatedCats *hashset.Set[types.Category],  // NEW: Categories that were evaluated
) []condition.RuleIdType {

    // Process default categories
    for _, cat := range f.FilterTables.DefaultCatList {
        if !evaluatedCats.Contains(cat) {  // NOT evaluated (field missing)
            negCat := f.FilterTables.NegCats[cat]
            // Process negative
        }
    }
}
```

---

## Updated MatchEvent Flow

```go
func (f *RuleEngine) MatchEvent(v interface{}) []condition.RuleIdType {
    matchingCompareCondRecords := types.NewHashSet[*EvalCategoryRec]()

    // ... existing MapObject code ...

    // ... existing AlwaysEvaluateCategories code ...

    // NEW: Track which categories were evaluated
    evaluatedCategories := types.NewHashSet[types.Category]()

    var eventCategories []types.Category
    var FrameStack = [20]interface{}{event.Values}

    matchingCompareCondRecords.Each(func(catEvaluator *EvalCategoryRec) {
        cat := catEvaluator.GetCategory()
        evaluatedCategories.Put(cat)  // NEW: Mark as evaluated

        result := catEvaluator.Evaluate(event, FrameStack[:])

        switch r := result.(type) {
        case condition.UndefinedOperand:  // NEW
            // Don't add to eventCategories
        case condition.BooleanOperand:
            if r {
                eventCategories = append(eventCategories, cat)
            }
        // ... rest
        }
    })

    f.compCondRepo.ObjectAttributeMapper.FreeObject(event)

    // Pass BOTH fired categories and evaluated set
    return f.catEngine.MatchEvent(eventCategories, evaluatedCategories)
}
```

**That's it!** Simple addition of a second set parameter.

---

## Final Architecture Summary

### Special Casing by Pattern Type

| Expression Type | Example | Missing Field Result | Optimization Used | Evaluated When |
|-----------------|---------|---------------------|-------------------|----------------|
| **Regular comparison** | `age > 18` | false (doesn't match) | None | Field present |
| **Regular negation** | `age != 18` | false (doesn't match) | None | Field present |
| **Explicit negation** | `!(age > 18)` | false (doesn't match) | None | Field present |
| **Undefined equality** | `age == undefined` | true (matches) | DefaultCatList | Field present |
| **Undefined inequality** | `age != undefined` | false (doesn't match) | DefaultCatList | Field present |
| **Null equality** | `age == null` | false (doesn't match)* | AlwaysEvaluateCategories | Every event |
| **Null inequality** | `age != null` | true (matches)* | AlwaysEvaluateCategories | Every event |
| **Constants** | `1 == 1` | true (matches) | AlwaysEvaluateCategories | Every event |

*Behavior changes if we distinguish undefined from null in null checks (see below)

### Three Mechanisms Working Together

**1. Natural Triggering (Majority of Cases)**
- Category registered against field
- Only triggered when field present
- Missing field → undefined propagation → category doesn't fire
- No special casing needed!

**Examples**: `age != 18`, `age > 18`, `status == "active"`, `!(age > 18)`

**2. DefaultCatList (Small, Optimized)**
- **Only** for `field == undefined` checks
- Inverted category (returns true when field EXISTS)
- Negative fires when field missing
- Track "evaluated" separately from "fired"
- Size: ~5-10 entries (vs 100+ currently)

**Examples**: `age == undefined`, `!(age == undefined)`, `age != undefined`

**3. AlwaysEvaluateCategories (Meta-Checks)**
- For checks that need to run regardless of field presence
- Evaluates every event (acceptable for rare meta-checks)
- Used for: null checks, constants, quantifiers

**Examples**: `age == null`, `1 == 1`, `forAll(...)`

### Performance Impact

**Before (Current)**:
- DefaultCatList: 100+ entries (all negations)
- Loop cost: O(100+) per event
- All negations have special casing overhead

**After (With Undefined)**:
- DefaultCatList: ~5-10 entries (only undefined-equality)
- Loop cost: O(5-10) per event
- 90% of negations work naturally (no overhead)

**Estimated improvement**: 5-10% for typical rule sets

---

## Implementation Checklist

### Core Changes

- [ ] Add UndefinedOperand type (condition/condition.go)
- [ ] Parse `undefined` keyword (engine/engine_impl.go ~1410)
- [ ] Return undefined for missing fields (engine/engine_impl.go ~1796)
- [ ] Undefined propagation in comparisons (engine/engine_impl.go ~226)
- [ ] Undefined propagation in negation (engine/engine_impl.go ~709)
- [ ] Undefined propagation in AND (engine/engine_impl.go ~660)
- [ ] Undefined propagation in OR (engine/engine_impl.go ~685)
- [ ] Handle undefined in category results (engine/engine_api.go ~572)

### DefaultCatList Optimization

- [ ] Add IsUndefinedEqualityCheck field to EvalCategoryRec (engine/engine_impl.go)
- [ ] Mark undefined-equality checks (engine/engine_impl.go ~610)
- [ ] Update DefaultCatList population (cateng/builder.go ~636)
- [ ] Add evaluatedCategories set tracking (engine/engine_api.go ~569)
- [ ] Pass evaluatedCategories to CategoryEngine.MatchEvent (change signature)
- [ ] Update DefaultCatList processing to check evaluated set (cateng/category_engine.go ~83)

### Helper Functions

- [ ] Add fieldExistsInOriginalEvent() helper (objectmap/object_attribute_map.go)
- [ ] Optionally add hasField() function (engine/engine_impl.go)

### Testing

- [ ] Unit tests: UndefinedOperand comparisons (50+ tests)
- [ ] Integration tests: undefined in expressions (30+ tests)
- [ ] Edge cases: nested fields, null vs undefined (20+ tests)
- [ ] Performance tests: verify DefaultCatList shrinkage
- [ ] Migration tests: update existing null handling tests

### Documentation

- [ ] Update README with undefined semantics
- [ ] Add migration guide for breaking changes
- [ ] Update ARCHITECTURE.md with undefined details
- [ ] Add examples for common patterns

---

## Files Modified (Estimated)

| File | Lines Added | Lines Deleted | Change Type |
|------|-------------|---------------|-------------|
| condition/condition.go | ~150 | ~0 | UndefinedOperand type |
| engine/engine_impl.go | ~100 | ~20 | Undefined propagation |
| engine/engine_api.go | ~30 | ~5 | Track evaluated set |
| cateng/builder.go | ~20 | ~10 | DefaultCatList population |
| cateng/category_engine.go | ~15 | ~5 | Two-set tracking |
| objectmap/object_attribute_map.go | ~30 | ~0 | Field existence helper |
| **Total** | **~345** | **~40** | **Net +305 lines** |

**Test files**: ~150 new tests across 5-6 test files

---

## Breaking Changes & Migration

### Affected Patterns

**1. Negative comparisons with missing fields**
```yaml
# Before: Matched when field missing
expression: age != 18

# After: Doesn't match when field missing
# Migration (if old behavior desired):
expression: age == undefined || age != 18
```

**2. length() with missing arrays**
```yaml
# Before: Matched when array missing (null != 0 → true)
expression: length("items") != 0

# After: Doesn't match when array missing (undefined != 0 → undefined)
# Migration (if old behavior desired):
expression: !hasField("items") || length("items") > 0
```

**3. Null checks (if we distinguish fully)**
```yaml
# Before: Matched both missing and explicit null
expression: age == null

# After: Only matches explicit null
# Migration (if old behavior desired):
expression: age == null || age == undefined
```

### Migration Guide Template

```markdown
# Rulestone v2.0 Migration Guide

## Breaking Change: Undefined vs Null

Missing fields now return `undefined` instead of `null`.

### Quick Fixes

| Old Pattern | New Equivalent | When Needed |
|-------------|----------------|-------------|
| `field != value` | `field != value` | Usually no change needed! |
| `field != value` (want old behavior) | `field == undefined \|\| field != value` | Rare |
| `length("arr") != 0` | `length("arr") > 0` | Common |
| `field == null` (want old behavior) | `field == null \|\| field == undefined` | If needed |

### Test Your Rules

Run with test cases that have missing fields to verify behavior.
```

---

## Ready to Implement?

This design:
- ✅ Distinguishes missing from null everywhere
- ✅ Uses three-valued logic with undefined propagation
- ✅ Keeps optimized DefaultCatList (small, for undefined-equality only)
- ✅ Tracks evaluated separately from fired (simple two-set approach)
- ✅ Eliminates 90% of DefaultCatList entries
- ✅ All negations work naturally (no special casing)
- ✅ Clean, efficient, industry-aligned

Estimated: ~300 lines of code, ~150 tests, 2-3 days of implementation.

Should we proceed with implementation?