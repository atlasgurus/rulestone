# Undefined Semantics: Efficient Implementation Strategy

## Decision Summary

**DECIDED**: Distinguish missing from null EVERYWHERE using three-valued logic with UndefinedOperand.

This document describes the **efficient implementation strategy** that minimizes special-casing while maintaining high performance.

---

## Core Semantics

### Three-Valued Logic

All operations with undefined propagate undefined (like SQL's UNKNOWN):

```
undefined != 0      → undefined
undefined == 0      → undefined
undefined > 0       → undefined
!(undefined)        → undefined
undefined && true   → undefined
undefined || true   → true (short circuit)
true && undefined   → undefined
false && undefined  → false (short circuit)
```

### Undefined vs Undefined

```
undefined == undefined → true
undefined != undefined → false
```

**Rationale**: Checking "is this field missing?" should return true when field is missing.

### Undefined vs Null

```
undefined == null → false (different concepts)
undefined != null → true (different types)
null == null      → true
null != null      → false
```

**Critical**: Missing and explicit null are **completely distinct**.

### Category Evaluation Result Handling

```go
// In MatchEvent (engine_api.go:572)
switch r := result.(type) {
case condition.UndefinedOperand:
    // Don't add to categories (not applicable)
    // Treated as absent/false at rule level
case condition.BooleanOperand:
    if r {
        eventCategories = append(eventCategories, cat)
    }
case condition.NullOperand:
    // Don't add to categories (falsey)
// ...
}
```

---

## Efficient Implementation for Undefined Checks

### Pattern 1: `field != undefined` - Works Naturally

```yaml
expression: age != undefined
Event: { name: "john" }  # age missing
```

**Natural Flow (No Special Casing)**:
1. "age" not in event → category NOT triggered (no attribute callback)
2. Category doesn't evaluate
3. Category doesn't fire
4. Rule doesn't match → **false ✓**

**When field present**:
```yaml
Event: { age: 25 }
```
1. "age" in event → category triggered
2. Evaluate: age → 25, 25 != undefined → true
3. Category fires → **true ✓**

**Optimization (Optional)**: Convert to "conditional TRUE category"
```go
// In processCompareCondition
if isUndefinedInequality(compareCond) {
    // Create category that returns constant true
    // Register against field (triggers only when field present)
    // Result: true when triggered, false when not triggered
}
```

**No AlwaysEvaluateCategories needed! No DefaultCatList needed!** ✅

---

### Pattern 2: `field == undefined` - Efficient DefaultCatList

```yaml
expression: age == undefined
Event: { name: "john" }  # age missing
```

**Naive approach** (Inefficient):
- Add to AlwaysEvaluateCategories
- Evaluate for EVERY event
- Breaks optimization (categories only triggered by present fields)

**Efficient approach** (Use DefaultCatList):

**Compilation**:
```go
// Create FALSE category triggered by field
evalCatRec := NewEvalCategoryRec(
    func(event, frames) Operand {
        // When evaluated (field present), return false
        return NewBooleanOperand(false)
    })

// Register against "age" attribute
registerCatEvaluatorForAddress("age", evalCatRec)

// Add to DefaultCatList
addToDefaultCatList(evalCatRec)

// Create negative category (like current NOT handling)
negCat := registerNegativeCat(evalCatRec.GetCategory())

// Rule uses the negative category
return NewCategoryCond(negCat)
```

**Runtime - age missing**:
1. "age" not in event → category NOT triggered (efficient!)
2. defaultCatMap[cat] = false (category didn't fire)
3. DefaultCatList processing: fire negative category
4. Result: **true ✓**

**Runtime - age present**:
1. "age" in event → category triggered
2. Returns false → defaultCatMap[cat] = true
3. DefaultCatList processing: negative category suppressed
4. Result: **false ✓**

**Performance**: Only evaluates when field is present! Much better than AlwaysEvaluateCategories.

---

### Pattern 3: `!(field == undefined)` - Same as Pattern 1

**Parser converts**:
```yaml
field != undefined
  ↓ (token.NEQ handling)
!(field == undefined)
  ↓ (category building)
NOT(CategoryCond(cat_field_eq_undefined))
```

**Detection in Category Builder**:
```go
// In cateng/builder.go processNotOp
func (fb *FilterBuilder) processNotOp(cond condition.Condition) CatFilter {
    switch cond.GetKind() {
    case condition.CategoryCondKind:
        cat := cond.(*condition.CategoryCond).Cat
        evalCatRec := fb.lookupCategoryRecord(cat)

        // Detect: NOT(field == undefined) → convert to conditional TRUE
        if evalCatRec.IsUndefinedEqualityCheck {
            return fb.createConditionalTrueCategory(evalCatRec.FieldPath)
        }

        // Other NOT cases shouldn't use DefaultCatList anymore
        // (undefined propagation handles them)
}
```

**Result**: `!(age == undefined)` becomes conditional TRUE category (same as `age != undefined`)

---

## Implementation Details

### 1. Mark Undefined Checks During Compilation

```go
// In processCompareCondition (engine_impl.go)
func (repo *CompareCondRepo) processCompareCondition(
    compareCond *condition.CompareCondition, scope *ForEachScope) condition.Condition {

    // ... existing code ...

    // Detect undefined equality check
    isUndefinedEqualityCheck :=
        (compareCond.CompareOp == condition.CompareEqualOp) &&
        (compareCond.LeftOperand.GetKind() == condition.UndefinedOperandKind ||
         compareCond.RightOperand.GetKind() == condition.UndefinedOperandKind)

    if isUndefinedEqualityCheck {
        // Mark the category
        evalCatRec.IsUndefinedEqualityCheck = true
        evalCatRec.FieldPath = extractFieldPath(compareCond)

        // This will be used in DefaultCatList
    }

    // ... rest of existing code ...
}
```

### 2. Category Builder Handling

```go
// In cateng/builder.go
func (fb *FilterBuilder) processNotOp(cond condition.Condition) CatFilter {
    switch cond.GetKind() {
    case condition.CategoryCondKind:
        cat := cond.(*condition.CategoryCond).Cat
        evalCatRec := fb.getCategoryRecord(cat)

        // Special case: NOT(field == undefined) → field exists check
        if evalCatRec != nil && evalCatRec.IsUndefinedEqualityCheck {
            // Create conditional TRUE category
            condTrueCat := fb.createConditionalTrueCategory(evalCatRec.FieldPath)
            return fb.computeCatFilter(condition.NewCategoryCond(condTrueCat))
        }

        // Other negations: Should NOT use DefaultCatList anymore
        // With undefined propagation, they work naturally
        // This code path should rarely be hit now

        // For safety, still create negative category (backward compat during migration)
        negCat := fb.registerNegativeCat(cat)
        return fb.computeCatFilter(condition.NewCategoryCond(negCat))

    // ... existing DeMorgan's law handling for AND/OR ...
}
```

### 3. DefaultCatList Population (Shrunk)

```go
// In cateng/builder.go Build() (line 636-639)
func (fb *FilterBuilder) Build(...) *FilterTables {
    // ... existing code ...

    result.DefaultCategories = make(map[types.Category]int)

    // OLD: Add all negated categories
    // for cat := range fb.NegCats {
    //     result.DefaultCategories[cat] = len(result.DefaultCategories)
    //     result.DefaultCatList = append(result.DefaultCatList, cat)
    // }

    // NEW: Only add undefined-equality checks
    for cat, evalCatRec := range fb.CategoryRecords {
        if evalCatRec.IsUndefinedEqualityCheck {
            result.DefaultCategories[cat] = len(result.DefaultCategories)
            result.DefaultCatList = append(result.DefaultCatList, cat)
        }
    }

    return result
}
```

**Result**: DefaultCatList contains only undefined-equality checks (typically <10 instead of 100+)

### 4. Undefined Propagation in Operators

```go
// In genEvalForCompareOperands (engine_impl.go:226)
func (repo *CompareCondRepo) genEvalForCompareOperands(
    compOp condition.CompareOp, xEval condition.Operand, yEval condition.Operand) condition.Operand {

    return repo.CondFactory.NewExprOperand(
        func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
            X := xEval.Evaluate(event, frames)
            Y := yEval.Evaluate(event, frames)

            xKind := X.GetKind()
            yKind := Y.GetKind()

            // Error propagation (existing)
            if xKind == condition.ErrorOperandKind {
                return X
            }
            if yKind == condition.ErrorOperandKind {
                return Y
            }

            // NEW: Undefined handling (BEFORE null)
            // Both undefined
            if xKind == condition.UndefinedOperandKind && yKind == condition.UndefinedOperandKind {
                switch compOp {
                case condition.CompareEqualOp:
                    return condition.NewBooleanOperand(true)  // undefined == undefined
                case condition.CompareNotEqualOp:
                    return condition.NewBooleanOperand(false)  // Same type
                default:
                    return condition.NewUndefinedOperand(nil)  // Can't order
                }
            }

            // One undefined, one not
            if xKind == condition.UndefinedOperandKind || yKind == condition.UndefinedOperandKind {
                switch compOp {
                case condition.CompareEqualOp:
                    return condition.NewBooleanOperand(false)  // Different
                case condition.CompareNotEqualOp:
                    return condition.NewBooleanOperand(true)  // Different
                default:
                    return condition.NewUndefinedOperand(nil)  // Propagate
                }
            }

            // Existing null handling
            if xKind == condition.NullOperandKind || yKind == condition.NullOperandKind {
                // Note: undefined != null (handled above)
                // ... existing null comparison logic ...
            }

            // ... rest of existing comparison logic ...
        }, xEval, yEval)
}
```

### 5. Undefined Propagation in Negation

```go
// In genEvalForNotCondition (engine_impl.go:700-716)
func (repo *CompareCondRepo) genEvalForNotCondition(
    cond *condition.NotCond, parentScope *ForEachScope) condition.Operand {

    eval := repo.genEvalForCondition(cond.Operand, parentScope)
    if eval.GetKind() == condition.ErrorOperandKind {
        return eval
    }

    return repo.CondFactory.NewExprOperand(
        func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
            result := eval.Evaluate(event, frames)

            // Error propagation
            if result.GetKind() == condition.ErrorOperandKind {
                return result
            }

            // NEW: Undefined propagation
            if result.GetKind() == condition.UndefinedOperandKind {
                return result  // !(undefined) → undefined
            }

            // Existing boolean negation
            return condition.NewBooleanOperand(!bool(result.(condition.BooleanOperand)))
        }, eval)
}
```

### 6. Undefined Propagation in AND/OR

```go
// In genEvalForAndCondition
func (repo *CompareCondRepo) genEvalForAndCondition(...) {
    return NewExprOperand(
        func(event, frames) Operand {
            for _, eval := range condEvaluators {
                result := eval.Func(event, frames)

                if result.GetKind() == ErrorOperandKind {
                    return result  // Error propagates
                }

                // NEW: False short-circuits (existing behavior)
                if result.GetKind() == BooleanOperandKind && !bool(result.(BooleanOperand)) {
                    return NewBooleanOperand(false)
                }

                // NEW: Undefined propagates (but doesn't short-circuit)
                // Keep track of undefined, continue checking others
            }

            // If any was undefined and none were false → undefined
            // If all were true → true
            // If any was false → false (already short-circuited above)
        })
}

// Similar logic for OR (true short-circuits)
```

**Actually, simpler approach for AND/OR**:

Since categories are evaluated separately and combined at category level, the category engine's AND/OR handles this naturally! Individual category returns undefined → doesn't fire → contributes "absent" to the AND/OR logic.

---

## Comparison of Approaches

### Regular Negation: `age != 18`

```yaml
Event: { name: "john" }  # age missing
```

**Flow**:
1. "age" not in event → category NOT evaluated
2. Category doesn't fire
3. No special casing needed ✓
4. Rule doesn't match

**No DefaultCatList! No AlwaysEvaluateCategories!** Just works via undefined propagation.

---

### Undefined Inequality: `age != undefined`

```yaml
Event: { name: "john" }  # age missing
```

**Approach A: Natural (Simple)**:
1. "age" not in event → category NOT evaluated
2. Category doesn't fire
3. Rule doesn't match → false ✓

**Approach B: Optimized (Conditional TRUE)**:
1. Create category that returns constant TRUE
2. Register against "age" attribute
3. "age" not in event → category NOT triggered → false ✓
4. "age" in event → category triggered → returns TRUE → true ✓

**Both work! Choose based on implementation complexity.**

**Decision**: Start with Approach A (simpler), optimize later if needed.

---

### Undefined Equality: `age == undefined`

```yaml
Event: { name: "john" }  # age missing
```

**Efficient Approach (Use DefaultCatList)**:

**Compilation**:
```go
// 1. Create FALSE category triggered by "age"
evalCatRec := NewEvalCategoryRec(
    func(event, frames) Operand {
        // When field present, return false
        return NewBooleanOperand(false)
    })

// 2. Mark as undefined check
evalCatRec.IsUndefinedEqualityCheck = true
evalCatRec.FieldPath = "age"

// 3. Register against "age" attribute
registerCatEvaluatorForAddress(addressOf("age"), evalCatRec)

// 4. Will be added to DefaultCatList during Build()
// (Automatically based on IsUndefinedEqualityCheck flag)

// 5. Create negative category
negCat := cat + MaxCategory
NegCats[cat] = negCat

// 6. Rule uses negative category
return NewCategoryCond(negCat)
```

**Runtime - age missing**:
1. "age" not in event → category NOT evaluated (efficient!)
2. defaultCatMap[cat] = false (category didn't fire)
3. Process DefaultCatList: cat didn't fire → evaluate negCat
4. negCat fires → **true ✓**

**Runtime - age present**:
1. "age" in event → category evaluated
2. Returns false → defaultCatMap[cat] = true
3. DefaultCatList processing: negCat suppressed
4. Result: **false ✓**

**Performance**: Only evaluates when field is present! Optimal.

---

### Negated Undefined Equality: `!(age == undefined)` or `age != undefined`

**Parser Flow**:
```yaml
age != undefined
  ↓ (token.NEQ → negate + CompareEqualOp)
!(age == undefined)
  ↓ (processCompareCondition)
NotCond(CategoryCond(cat_age_eq_undefined))
  ↓ (category builder processNotOp)
Detect: NOT of undefined-equality check
  ↓
Convert to conditional TRUE category
```

**Detection in processNotOp**:
```go
func (fb *FilterBuilder) processNotOp(cond condition.Condition) CatFilter {
    switch cond.GetKind() {
    case condition.CategoryCondKind:
        cat := cond.(*condition.CategoryCond).Cat
        evalCatRec := fb.getCategoryRecord(cat)

        // NEW: Detect NOT(field == undefined)
        if evalCatRec != nil && evalCatRec.IsUndefinedEqualityCheck {
            // This is "field != undefined" (field exists)
            // Create conditional TRUE category (or just return normally)
            // Field missing → not triggered → false ✓
            // Field present → triggered → true ✓

            // Simple approach: Just register negCat normally
            // It will work because the positive cat is FALSE when present
            negCat := fb.registerNegativeCat(cat)
            return fb.computeCatFilter(condition.NewCategoryCond(negCat))
        }

        // Other negations: No longer use DefaultCatList!
        // With undefined propagation, they work naturally
        // This should rarely execute now
}
```

**Actually simpler**: The existing negative category mechanism already works!
- Positive cat (field == undefined): Returns FALSE when field present
- Negative cat: fires when positive doesn't (field missing → fires, field present → doesn't fire)
- Wait, that's backward...

Let me reconsider:

**DefaultCatList logic**:
```go
for i, cat := range DefaultCatList {
    if !defaultCatMap[i] {  // Category DIDN'T fire
        negCat := NegCats[cat]
        // Evaluate negCat
    }
}
```

**For `age == undefined`**:
- Positive cat: FALSE when age present, doesn't fire when age missing
- When age missing: positive cat doesn't fire → negative cat fires → true ✓
- When age present: positive cat fires (returns false) → negative cat suppressed → false ✓

**For `!(age == undefined)` aka `age != undefined`**:
- Same positive cat (age == undefined)
- Negation creates negative category
- When age missing: negative cat fires → BUT we want false!
- **Wait, this is wrong...**

---

## Let Me Re-Think the DefaultCatList Logic

**DefaultCatList processes categories that DIDN'T fire**:

```go
for i, cat := range DefaultCatList {
    if !defaultCatMap[i] {  // Cat DIDN'T fire (field missing or returned false)
        negCat := NegCats[cat]
        processCat(negCat)  // Fire the negative
    }
}
```

**For `age == undefined`**:

**When age missing**:
- Positive cat (age == undefined) NOT triggered by attribute
- defaultCatMap[cat] = false (didn't fire)
- Negative cat fires → true ✓ **CORRECT**

**When age present (age=25)**:
- Positive cat triggered by "age" attribute
- Evaluates: 25 == undefined → false
- Category returns false → **doesn't add to eventCategories**
- defaultCatMap[cat] = false (didn't add to categories!)
- Negative cat fires → true ✗ **WRONG!**

**Oh no!** The issue is defaultCatMap tracks "did this category fire?" not "was it evaluated?"

---

## The Real Challenge

We need to distinguish:
1. **Category not evaluated** (field missing) → defaultCatMap[cat] = false
2. **Category evaluated but returned false** (field present) → defaultCatMap[cat] = true (suppress negative)

**Current code only tracks**: Did the category get added to eventCategories?

**For `age == undefined` to work with DefaultCatList**, we need:
- age missing → cat not evaluated → defaultCatMap = false → negative fires ✓
- age present → cat evaluated returns false → defaultCatMap = true → negative suppressed ✓

**But current tracking in category_engine.go:72-76**:
```go
for _, cat := range cats {  // cats = categories that fired (returned true)
    if i, ok := DefaultCategories[cat]; ok {
        defaultCatMap[i] = true  // Mark as "did fire"
    }
}
```

**This tracks categories that FIRED (returned true), not categories that were EVALUATED.**

---

## Solution: Track Evaluation, Not Just Firing

We need **two separate pieces of information**:
1. **Was category evaluated?** (field present)
2. **Did category fire?** (returned true)

**But wait** - let me check if there's already a mechanism for this...

Actually, maybe `age == undefined` should return TRUE when age is present!

```go
// Inverted logic:
evalCatRec := NewEvalCategoryRec(
    func(event, frames) Operand {
        // Evaluate age
        age := evaluateField("age", event, frames)
        if age.GetKind() == UndefinedOperandKind {
            return NewBooleanOperand(true)  // age is undefined
        } else {
            return NewBooleanOperand(false)  // age exists
        }
    })
```

Then it works naturally with the category engine - no DefaultCatList needed!

**But** this requires AlwaysEvaluateCategories (to evaluate when age missing).

---

## I Need to Reconsider the Whole Approach

Let me think about what's most efficient for each case:

### **Option 1: Use AlwaysEvaluateCategories for Both**

```yaml
age == undefined  → AlwaysEvaluateCategories
age != undefined  → AlwaysEvaluateCategories
```

**Simple**, but breaks optimization (evaluates every event).

### **Option 2: Use DefaultCatList Cleverly**

This requires categories to return TRUE when field PRESENT (not when condition met).

Gets complex and confusing.

### **Option 3: Mixed Approach**

```yaml
age == undefined  → AlwaysEvaluateCategories (small cost for explicit checks)
age != undefined  → Natural (no special casing, works via non-triggering)
age != 18         → Natural (undefined propagation)
```

**Advantages**:
- Simple to implement
- Each pattern uses most natural approach
- DefaultCatList can be DELETED entirely
- Only cost: `age == undefined` checks evaluate every event

---

## My Updated Recommendation

**Delete DefaultCatList entirely, use AlwaysEvaluateCategories for undefined-equality**

**Rationale**:
1. Undefined checks (`age == undefined`) are relatively rare
2. Simpler implementation (no complex DefaultCatList logic)
3. Most cases (negations, undefined inequality) work naturally
4. Small cost for explicit undefined checks is acceptable

**What do you think?** Or do you prefer the complexity of keeping DefaultCatList for the optimization?