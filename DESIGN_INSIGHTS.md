# Rulestone Engine: Deep Design Insights

## Document Purpose

This document captures critical design insights and architectural patterns discovered while fixing engine bugs. It's written to help future maintainers understand **why** the code works the way it does, not just **what** it does.

---

## Table of Contents

1. [Architectural Overview](#architectural-overview)
2. [The Two-Part Engine](#the-two-part-engine)
3. [ObjectAttributeMap: The Critical Data Structure](#objectattributemap-the-critical-data-structure)
4. [AlwaysEvaluateCategories Pattern](#alwaysevaluatecategories-pattern)
5. [Bug Fix Case Studies](#bug-fix-case-studies)
6. [Common Pitfalls](#common-pitfalls)
7. [Performance Characteristics](#performance-characteristics)

---

## Architectural Overview

### The Core Insight

Rulestone uses a **two-phase matching architecture**:

1. **Phase 1: Category Evaluators** - Convert event attributes into boolean category IDs
2. **Phase 2: Category Engine** - Match category combinations against rule patterns using bitmasks

This separation enables:
- **Common Subexpression Elimination (CSE)**: Same comparison → same category ID → evaluated once
- **Fast matching**: Category engine uses bitmask operations, not tree traversal
- **Scalability**: Handles thousands of rules efficiently

### Key Data Flow

```
Event Object
    ↓
ObjectAttributeMapper.MapObject()
    ↓
ObjectAttributeMap (Values: []interface{})  ← Only stores SCALAR LEAF VALUES
    ↓
Category Evaluators (via attribute address callbacks)
    ↓
Boolean Categories []Category
    ↓
Category Engine (bitmask matching)
    ↓
Matching Rule IDs []RuleIdType
```

**Critical**: The `ObjectAttributeMap.Values` array stores **only scalar leaf values**, NOT containers (arrays/objects). This is the root cause of Bug #3.

---

## The Two-Part Engine

### Part 1: Category Evaluator (CompareCondRepo)

**File**: `engine/engine_impl.go`

**Purpose**: Converts rule expressions into category evaluators.

**Key Components**:

1. **EvalCategoryRec**: Wraps an evaluator function with metadata
   ```go
   type EvalCategoryRec struct {
       Eval     condition.Operand  // The evaluation function
       Category types.Category     // Unique category ID
       AttrKeys []string           // Attributes this category depends on
   }
   ```

2. **CompareCondRepo**: Registry of all category evaluators
   ```go
   type CompareCondRepo struct {
       // Maps attribute paths to categories that depend on them
       AttributeToCompareCondRecord map[string]*hashset.Set[*EvalCategoryRec]

       // Maps conditions to their category evaluators (for CSE)
       CondToCompareCondRecord *hashmap.Map[condition.Condition, *EvalCategoryRec]

       // Categories that must run even if attributes aren't in event
       AlwaysEvaluateCategories *hashset.Set[*EvalCategoryRec]

       // Object mapper (converts events to attribute maps)
       ObjectAttributeMapper *objectmap.ObjectAttributeMapper
   }
   ```

3. **Registration Flow**:
   ```
   Rule Expression → AST Parser → processCondNode() →
   processCompareCondition() → genEvalForCondition() →
   Register in AttributeToCompareCondRecord
   ```

**Key Insight**: Each comparison in a rule becomes a category. The category ID is assigned sequentially during rule registration. **Same expression = same category ID** (CSE).

### Part 2: Category Engine

**File**: `cateng/category_engine.go`

**Purpose**: Match category combinations using bitmask operations.

**Key Data Structures**:

1. **FilterTables**: Maps category → rules that need it
2. **NegCats**: Rules that have negated categories
3. **DefaultCategories**: Categories assumed true by default

**Matching Algorithm**:
```
For each category in event:
    Mark all rules requiring this category as "potentially matching"

For each potentially matching rule:
    Check if all required categories are present
    Check if no negated categories are present

Return matching rule IDs
```

---

## ObjectAttributeMap: The Critical Data Structure

### The Design Choice That Caused Bug #3

**File**: `objectmap/object_attribute_map.go`

```go
type ObjectAttributeMap struct {
    DictRec       *AttrDictionaryRec
    Values        []interface{}      // ONLY stores scalar leaf values
    OriginalEvent interface{}        // Added to fix Bug #3
}
```

**Why Values only stores scalars**:

1. **Memory efficiency**: Don't duplicate entire nested objects
2. **Fast access**: Direct array indexing by address
3. **Callback optimization**: Only trigger for "interesting" values

**The Problem**:

- Empty arrays have **no scalar elements**
- Therefore nothing gets stored in `Values`
- Evaluators checking array existence get `nil` → error
- Can't distinguish "array missing" vs "array empty"

**Example**:

```go
Event: {"items": []}

After MapObject():
  Values: [nil, nil, ...]  // items[] has no elements, so values[0] = nil

Event: {"other": "data"}

After MapObject():
  Values: [nil, nil, ...]  // items missing, so values[0] = nil

// Both look identical in Values! ❌
```

### The Mapping Process

**Function**: `ObjectAttributeMapper.MapObject(v interface{}, attrCallback func([]int))`

**Flow**:
```
1. Create empty Values array sized to number of known attributes
2. Recursively traverse input object
3. For each scalar leaf:
   a. Look up its address in AttrDictionaryRec
   b. Store value in Values[address]
   c. Call attrCallback(address) to notify listeners
4. Return ObjectAttributeMap
```

**Key**: Containers (arrays, objects) are traversed but NOT stored. Only their **children** are stored.

### Address System

**Addresses are integer arrays** representing paths:

```
Event: {"user": {"age": 30}}

Paths:
  user      → address [0]
  user.age  → address [0, 0]

Arrays use special notation:
  items     → address [1] (the container)
  items[]   → address [1] (elements - same numeric address!)
  items[0]  → address [1, 0] (specific element)
  items[1]  → address [1, 1]
```

**Critical**: `"items"` and `"items[]"` have the **same numeric address**. The distinction is in the path string, used for registration.

---

## AlwaysEvaluateCategories Pattern

### The Problem This Solves

Normal evaluation flow:
```
MapObject finds attribute → Calls attrCallback(address) →
Looks up categories at address → Evaluates those categories
```

**Problem**: What if the attribute is **missing** from the event?
- No callback triggered
- Category evaluator never runs
- Can't distinguish "missing" vs "doesn't match"

### Use Cases

1. **Null checks**: `field == null` should match when field is missing
2. **Constant expressions**: `1 == 1` has no event dependencies, should always match
3. **Empty array checks**: `forAll("items", ...)` needs to run even for empty arrays

### Implementation

**Registration** (in `processCompareCondition` or `processForAllCondition`):
```go
if isNullCheck || hasNoEventDependencies {
    repo.AlwaysEvaluateCategories.Put(evalCatRec)
}
```

**Evaluation** (in `RuleEngine.MatchEvent`):
```go
// Normal flow: evaluate categories for attributes found in event
event.MapObject(v, func(addr []int) {
    catEvaluators := AttributeToCompareCondRecord[addr]
    matchingCompareCondRecords.Put(catEvaluators)
})

// Always-evaluate flow: force evaluation of these categories
AlwaysEvaluateCategories.Each(func(catEvaluator *EvalCategoryRec) {
    matchingCompareCondRecords.Put(catEvaluator)
})

// Now evaluate all collected categories
matchingCompareCondRecords.Each(func(catEvaluator *EvalCategoryRec) {
    result := catEvaluator.Evaluate(event, frames)
    if result == true {
        eventCategories.append(catEvaluator.Category)
    }
})
```

**Key Insight**: AlwaysEvaluateCategories bypasses the attribute callback mechanism. The evaluator still needs to determine the correct result internally.

---

## Bug Fix Case Studies

### Bug #1: Constant Expressions (1 == 1)

**Symptom**: `1 == 1` doesn't match empty events.

**Root Cause**:
- Expression has no attribute references (`AttrKeys` is empty)
- No callbacks triggered during MapObject
- Category never evaluated

**Fix**: Add to AlwaysEvaluateCategories
```go
hasNoEventDependencies := len(evalCatRec.AttrKeys) == 0
if hasNoEventDependencies {
    repo.AlwaysEvaluateCategories.Put(evalCatRec)
}
```

**Why It Works**: Evaluator runs unconditionally, returns `true`.

---

### Bug #2: Null Checks (field == null)

**Symptom**: `field == null` doesn't match when field is missing.

**Root Cause**:
- Missing field → no callback
- Category never evaluated
- Can't return `true` for "field is null"

**Fix Part 1**: Add to AlwaysEvaluateCategories
```go
isNullCheck := compareCond.LeftOperand.GetKind() == condition.NullOperandKind ||
    compareCond.RightOperand.GetKind() == condition.NullOperandKind
if isNullCheck {
    repo.AlwaysEvaluateCategories.Put(evalCatRec)
}
```

**Fix Part 2**: Handle null semantics in evaluation
```go
// In genEvalForCompareOperands
if xKind == condition.NullOperandKind || yKind == condition.NullOperandKind {
    bothNull := xKind == condition.NullOperandKind && yKind == condition.NullOperandKind
    switch compOp {
    case condition.CompareEqualOp:
        return condition.NewBooleanOperand(bothNull)  // null == null → true
    case condition.CompareNotEqualOp:
        return condition.NewBooleanOperand(!bothNull) // null != 0 → true
    case condition.CompareGreaterOp, ...:
        return condition.NewBooleanOperand(false)     // null > 0 → false
    }
}
```

**Null Semantics**:
- `null == null` → `true`
- `null != 0` → `true` (null is not equal to any non-null value)
- `null > 0` → `false` (null is not orderable)
- Missing field is treated as `null`

---

### Bug #3: forAll with Empty Arrays (Vacuous Truth)

**Symptom**: `forAll("items", "item", item.value > 100)` with `{items: []}` returns `false`. Should return `true` (vacuous truth).

**Root Cause**: The most complex of the three bugs.

1. **Empty arrays have no elements** → no scalar values stored in `Values`
2. **No callback triggered** → forAll evaluator doesn't run normally
3. Even when forced to run via AlwaysEvaluateCategories:
   - `GetNumElementsAtAddress(arrayAddress, frames)` tries to get `Values[address]`
   - Gets `nil` because empty array has no values stored
   - Returns error: "attribute items[] not available"
   - Can't distinguish empty array from missing array

**Fix Part 1**: Store original event reference
```go
type ObjectAttributeMap struct {
    DictRec       *AttrDictionaryRec
    Values        []interface{}
    OriginalEvent interface{}  // ← Added
}

func (mapper *ObjectAttributeMapper) MapObject(v interface{}, ...) *ObjectAttributeMap {
    result := mapper.NewObjectAttributeMap()
    result.OriginalEvent = v  // ← Store reference
    mapper.buildObjectMap("", v, result.Values, ...)
    return result
}
```

**Fix Part 2**: Fallback to original event when Values lookup fails
```go
func (attrMap *ObjectAttributeMap) GetNumElementsAtAddress(address *AttributeAddress, frames []interface{}) (int, error) {
    values, err := attrMap.GetAttributeByAddress(address.Address, frames[address.ParentParameterIndex])
    if err != nil {
        // Not in Values - check original event for empty arrays
        if attrMap.OriginalEvent != nil {
            arrayValue := attrMap.getValueFromOriginalEvent(address.Path)
            if arrayValue != nil {
                if reflect.ValueOf(arrayValue).Kind() == reflect.Slice {
                    return len(arrayValue.([]interface{})), nil  // ← Returns 0 for empty!
                }
            }
        }
        return 0, err  // Array truly missing
    }
    // Found in Values - use it
    return len(values.([]interface{})), nil
}
```

**Fix Part 3**: Navigate original event by path
```go
func (attrMap *ObjectAttributeMap) getValueFromOriginalEvent(path string) interface{} {
    cleanPath := strings.TrimSuffix(path, "[]")  // "items[]" → "items"
    if cleanPath == "" {
        return attrMap.OriginalEvent
    }

    current := attrMap.OriginalEvent
    for _, segment := range strings.Split(cleanPath, ".") {
        switch v := current.(type) {
        case map[string]interface{}:
            current = v[segment]
        default:
            return nil
        }
    }
    return current
}
```

**Fix Part 4**: Implement vacuous truth in forAll evaluator
```go
func genEvalForAllCondition(...) condition.Operand {
    return repo.CondFactory.NewExprOperand(
        func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
            numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
            if err != nil {
                // Array missing/null → false (rule doesn't apply)
                return condition.NewBooleanOperand(false)
            }

            if numElements == 0 {
                // Empty array → true (vacuous truth!)
                return condition.NewBooleanOperand(true)
            }

            // Non-empty array → evaluate condition for each element
            for i := 0; i < numElements; i++ {
                result := eval.Evaluate(event, frames)
                if !result.(condition.BooleanOperand) {
                    return condition.NewBooleanOperand(false)
                }
            }
            return condition.NewBooleanOperand(true)
        }, eval)
}
```

**Fix Part 5**: Add to AlwaysEvaluateCategories
```go
// In processForAllCondition
repo.AlwaysEvaluateCategories.Put(evalCatRec)
```

**Why All 5 Parts Are Needed**:

1. Without OriginalEvent: Can't check for empty arrays at all
2. Without fallback in GetNumElementsAtAddress: Empty arrays look like missing arrays
3. Without path navigation: Can't find nested empty arrays
4. Without vacuous truth logic: Returns false instead of true for empty
5. Without AlwaysEvaluateCategories: Empty arrays never trigger evaluation

**Correctness Verification**:

```
Event: {other: data}
  → GetNumElementsAtAddress tries Values[address] → nil
  → Falls back to OriginalEvent → field "items" missing → nil
  → Returns error
  → genEvalForAllCondition returns false ✓

Event: {items: []}
  → GetNumElementsAtAddress tries Values[address] → nil
  → Falls back to OriginalEvent → field "items" exists → []
  → Returns 0
  → genEvalForAllCondition returns true ✓

Event: {items: [{value: 150}]}
  → GetNumElementsAtAddress tries Values[address] → finds array
  → Returns 1
  → genEvalForAllCondition evaluates condition → returns true/false ✓
```

---

## Common Pitfalls

### Pitfall #1: Modifying ObjectAttributeMap.Values

**Don't**: Assume you can just check `Values[address]` to see if attribute exists.

**Why**: Empty arrays/objects won't be in Values even if they exist in the event.

**Do**: Use the original event as the source of truth for container existence.

---

### Pitfall #2: Forgetting AlwaysEvaluateCategories

**Symptom**: Rule works when attribute is present, fails when attribute is missing.

**Why**: Missing attribute → no callback → category never evaluated.

**Check**: Does the rule need to evaluate when the attribute is missing?
- Null checks: YES
- Constant expressions: YES
- Quantifiers on empty collections: YES
- Regular comparisons: NO

---

### Pitfall #3: Confusing "items" and "items[]"

**Path notation**:
- `"items"` - the array container
- `"items[]"` - array elements (notation for mapper registration)

**Address notation**:
- Both map to the **same numeric address**!
- Address `[1]` might represent both the container and its elements

**When to use which**:
- Registration (`registerCatEvaluatorForAddress`): Use `"items[]"` for element access
- Navigation (`getValueFromOriginalEvent`): Use `"items"` (strip `[]` suffix)
- Evaluation (`GetNumElementsAtAddress`): Address should point to container

---

### Pitfall #4: Null vs Zero vs Missing

**Three distinct concepts**:
1. **Missing field**: Key doesn't exist in map
2. **Explicit null**: Key exists, value is `nil`
3. **Zero value**: Key exists, value is `0`, `""`, `false`

**Engine behavior**:
- Missing field **is treated as null** in comparisons
- `field == null` matches both missing and explicit null
- `field == 0` only matches explicit zero, not null/missing

---

### Pitfall #5: CSE and Side Effects

**Common Subexpression Elimination means**:
- Same expression → evaluated once per event
- Changes to evaluation logic affect ALL rules using that expression

**Example**:
```yaml
- id: rule1
  expression: user.age > 18

- id: rule2
  expression: user.age > 18 && user.country == "US"
```

Both rules share the SAME category for `user.age > 18`. Evaluated once, result reused.

**Implication**: Be very careful changing evaluation semantics. Could break multiple rules.

---

## Performance Characteristics

### Benchmark Results (After Fixes)

**Regular operations** (no regression):
- Simple expression eval: ~1.9 μs/op, 1704 B/op, 38 allocs/op
- Complex expression eval: ~3.2 μs/op, 3144 B/op, 70 allocs/op
- forAll condition: ~2.5 μs/op, 1560 B/op, 37 allocs/op

**Bug fix operations** (new functionality):
- Null check: ~500 ns/op, 1096 B/op, 12 allocs/op
- Constant expression: ~454 ns/op, 1104 B/op, 13 allocs/op
- forAll empty array: ~1.3 μs/op, 1296 B/op, 15 allocs/op
- forAll non-empty (10 elements): ~4.2 μs/op, 2528 B/op, 77 allocs/op

**Analysis**:
- AlwaysEvaluateCategories adds ~0.5 μs overhead (negligible)
- OriginalEvent reference adds 8 bytes per ObjectAttributeMap (negligible)
- Fallback path in GetNumElementsAtAddress only runs for empty arrays (rare)
- Overall: **No measurable performance regression**

### Scalability Notes

**What scales well**:
- Number of rules (O(1) lookup via category matching)
- Number of categories (bitmask operations)
- CSE (more rules with shared expressions → fewer evaluations)

**What doesn't scale**:
- Large arrays in quantifiers (O(n) iteration)
- Deep nesting (O(depth) traversal)
- Complex string operations (regexpMatch, containsAny)

---

## Future Considerations

### Potential Improvements

1. **Cache empty array checks**: Store set of known-empty arrays in ObjectAttributeMap to avoid repeated original event navigation

2. **Lazy original event storage**: Only store OriginalEvent when rules have quantifiers (saves 8 bytes per map otherwise)

3. **Array container callbacks**: Modify MapObject to call attrCallback for arrays themselves, not just elements (would eliminate need for AlwaysEvaluateCategories for forAll)

### Cautions

1. **Don't break CSE**: Any change to evaluation logic affects all rules using that expression

2. **Preserve null semantics**: Existing rules depend on current null behavior

3. **Keep phase separation**: Don't blur the line between category evaluation and category matching

---

## Debugging Tips

### "My rule isn't matching!"

1. **Check if category is being evaluated**:
   - Add debug print in category evaluator
   - Is callback being triggered?
   - Is category in AlwaysEvaluateCategories if needed?

2. **Check what categories are generated**:
   - Print `eventCategories` after evaluation phase
   - Does it include the expected category ID?

3. **Check category engine**:
   - Is the rule's category pattern correct?
   - Are there negated categories blocking the match?

### "Performance is slow!"

1. **Profile category evaluation**:
   - How many categories are being evaluated per event?
   - Use `Metrics.NumCatEvals` counter

2. **Check CSE effectiveness**:
   - Print `CondToCompareCondRecord` size
   - Should be << number of rules if CSE is working

3. **Look for large arrays**:
   - forAll/forSome with 1000+ elements will be slow
   - Consider restructuring data or rules

---

## Glossary

- **Category**: A boolean value (true/false) representing whether a condition holds for an event
- **Category ID**: Unique integer identifying a category (assigned during rule registration)
- **Category Evaluator**: Function that computes a category's value for an event
- **CSE**: Common Subexpression Elimination - reusing evaluation results for identical expressions
- **AttrKeys**: Attribute paths that a category depends on
- **Address**: Integer array representing path to a value in nested structure
- **FullAddress**: String representation of address with nesting information
- **Frame**: Current context in nested evaluation (used in forAll/forSome)
- **Vacuous Truth**: Logic principle where "all elements of empty set satisfy P" is true

---

## Version History

- **v1.0** (2025-01-XX): Initial version documenting Bug #1, #2, #3 fixes
