# Iterator Implementation Details

## Core Architecture

This document describes the internal iterator architecture that enables zero-allocation filter/map/reduce composition.

---

## Key Insight: Lazy Evaluation

**No intermediate arrays are created**. Instead:

1. **Build phase**: Construct iterator chain (filter → map → etc.)
2. **Execute phase**: Single pass when consumed by aggregation
3. **Inline transformations**: All steps applied per element

**Result**: O(n) time, O(1) space

---

## Iterator Chain Building

### Example: `sum(map(filter("items", "item", item.active), "item", item.price))`

**Parsing order** (inside-out):

```
1. filter("items", "item", item.active)
   → Creates IteratorOperand with FilterStep

2. map(iterator, "item", item.price)
   → Appends MapStep to same IteratorOperand

3. sum(iterator)
   → Consumes iterator, executes chain
```

**Built structure**:
```go
IteratorOperand {
    ArrayAddress: &AttributeAddress{Path: "items", Address: [0]},
    Steps: [
        {Type: FilterStepType, Operand: conditionForActive, Scope: scope1},
        {Type: MapStepType, Operand: exprForPrice, Scope: scope2},
    ],
    Scope: rootScope,
}
```

---

## Scope Chaining

**Challenge**: Each iterator step needs its own scope for element binding.

**Solution**: Use ForEachScope nesting (already exists for forAll/forSome)

```go
// filter() creates scope level 1
scope1 := &ForEachScope{
    Element: "item",
    NestingLevel: 1,
    ParentScope: rootScope,
}

// map() creates scope level 2
scope2 := &ForEachScope{
    Element: "item",  // Reuse name or new name
    NestingLevel: 2,
    ParentScope: scope1,
}
```

**Frame binding**:
```go
// During iteration
frames[1] = currentElement  // For filter step
frames[2] = transformedValue // For map step
```

**Reuses existing scope mechanism** from forAll/forSome ✓

---

## Execution Flow

### executeIterator() - Core Iterator Executor

```go
func executeIterator(
    iterator *IteratorOperand,
    event *ObjectAttributeMap,
    frames []interface{},
    consumer func(Operand) bool,
) error {
    numElements, err := event.GetNumElementsAtAddress(iterator.ArrayAddress, frames)
    if err != nil {
        return err
    }

    for i := 0; i < numElements; i++ {
        // Get raw element
        elem := objectmap.GetNestedAttributeByAddress(
            frames[iterator.ArrayAddress.ParameterIndex],
            append(iterator.ArrayAddress.Address, i))

        currentValue := elem.(Operand)
        shouldInclude := true

        // Apply each transformation step
        for _, step := range iterator.Steps {
            frames[step.Scope.NestingLevel] = currentValue

            switch step.Type {
            case FilterStepType:
                result := step.Operand.Evaluate(event, frames)

                // Convert to boolean
                if !isTrue(result) {
                    shouldInclude = false
                    break
                }

            case MapStepType:
                currentValue = step.Operand.Evaluate(event, frames)

                if currentValue.GetKind() == UndefinedOperandKind {
                    shouldInclude = false
                    break
                }
            }
        }

        // If passes all steps, give to consumer
        if shouldInclude && !consumer(currentValue) {
            break
        }
    }

    return nil
}
```

**Flow**:
1. Iterate source array once
2. For each element, apply steps sequentially
3. Short-circuit on filter failure
4. Pass transformed value to consumer
5. Consumer controls continuation

---

## Optimization: Short-Circuit Evaluation

**Filter failure stops processing**:

```yaml
filter(filter(filter("items", "item", item.a), "item", item.b), "item", item.c)
```

**Execution**:
```go
for elem in items {
    if !elem.a { continue }  // Stop, don't check item.b or item.c
    if !elem.b { continue }  // Stop, don't check item.c
    if !elem.c { continue }
    // All passed, include element
}
```

**Early exit** saves evaluation of later steps ✓

---

## Memory Analysis

### Traditional Array-Based Approach (Avoided)

```yaml
sum(map(filter("items", "item", item.active), "item", item.price))
```

**Memory**:
```
items: 1000 elements = input
filter: 500 elements = 500 * sizeof(Operand) bytes allocated
map: 500 elements = 500 * sizeof(Operand) bytes allocated
sum: single value

Total allocations: 2 arrays, 1000 * sizeof(Operand) bytes
```

---

### Our Iterator Approach

**Memory**:
```
IteratorOperand: ~100 bytes (struct with 2 steps)
Execution: 0 bytes (uses existing frames array)

Total allocations: 0 (just stack for iteration)
```

**Savings**: ~16KB for 1000-element array (assuming 16-byte Operand)

---

## Performance Characteristics

### Best Case: Filter Early

```yaml
filter("items", "item", item.active)  # Passes 10% (100 elements)
  → map("item", expensiveCalculation(item))
  → sum()
```

**Iterations**:
- Source: 1000 elements
- After filter: 100 elements continue to map
- **Total work**: 1000 filter checks + 100 map evals

**vs Array-Based**:
- Filter: 1000 elements → create 100-element array
- Map: 100 elements → create 100-element array
- **Total work**: Same, but with 2 allocations

**Winner**: Iterator (same work, zero allocations)

---

### Worst Case: Filter Late or No Filter

```yaml
map("items", "item", item.price)      # All 1000 elements
  → filter("item", item > 10)          # Filters to 100
  → sum()
```

**Iterations**:
- Map: 1000 elements
- Filter: 1000 checks
- **Total work**: 2000 operations

**vs Direct**:
```yaml
sum("items", "item", if(item.price > 10, item.price, 0))
```

**Iterations**:
- 1000 elements, single pass
- **Total work**: 1000 operations

**Winner**: Direct (half the work)

**Lesson**: Encourage users to put filters early, or use inline if() for simple cases

---

## Undefined/Null Handling

### FilterStep with Undefined

```yaml
filter("items", "item", item.price > 10)
items: [{price: 20}, {name: "x"}, {price: 5}]
```

**Execution**:
```go
elem 0: price=20 → 20 > 10 → true → include
elem 1: price=undefined → undefined > 10 → undefined → false → exclude
elem 2: price=5 → 5 > 10 → false → exclude
```

**Result**: Filters to [elem 0]

---

### MapStep with Undefined

```yaml
map("items", "item", item.price)
items: [{price: 20}, {name: "x"}]
```

**Execution**:
```go
elem 0: price=20 → map to 20 → include
elem 1: price=undefined → map to undefined → exclude (shouldInclude=false)
```

**Result**: Maps to [20]

**Undefined elements are skipped** ✓

---

## Implementation Strategy

### Phase 1: Core Infrastructure (Day 1)

1. IteratorOperand type
2. executeIterator() function
3. Basic filter() and map()
4. Update length() to consume iterators

**Deliverable**: `length(filter("items", "item", item.active)) > 5` works

---

### Phase 2: Aggregations (Day 2)

1. sum() consuming iterators
2. avg() function
3. Update min/max to consume iterators

**Deliverable**: `sum(map(filter(...), "item", item.price)) > 100` works

---

### Phase 3: Reduce (Day 3)

1. reduce() function
2. Dual-variable scoping (accumulator + element)

**Deliverable**: `reduce("items", "sum", "item", sum + item, 0) > 100` works

---

### Phase 4: Testing & Polish (Days 4-5)

1. Comprehensive tests (~1000 lines)
2. Edge cases (undefined, null, empty arrays, nesting)
3. Performance validation
4. Documentation

**Deliverable**: Production-ready with full test coverage

---

## Code Reuse from Existing Functions

**Extract common patterns from forAll/forSome**:

```go
// Common: Array iteration setup
func setupArrayIteration(arrayPath, elemName, scope) (*AttributeAddress, *ForEachScope, error) {
    arrayAddress, err := getAttributePathAddress(arrayPath+"[]", scope)
    if err != nil {
        return nil, nil, err
    }

    newScope := &ForEachScope{
        Element:      elemName,
        Path:         arrayPath,
        NestingLevel: scope.NestingLevel + 1,
        ParentScope:  scope,
        AttrDictRec:  arrayAddress.DictRec,
    }

    return arrayAddress, newScope, nil
}

// Common: Element access
func getElementAtIndex(event, arrayAddress, index, frames) Operand {
    // Reuse existing logic from forAll/forSome
}
```

**Estimated savings**: ~100 lines by extracting helpers

---

## Edge Cases

### Empty Array

```yaml
sum(filter("empty_array", "item", item.value))
```

**Result**: 0 (no elements to sum)

---

### Missing Array

```yaml
sum(filter("missing", "item", item.value))
```

**Result**: undefined (array doesn't exist)

---

### All Elements Filtered Out

```yaml
sum(filter("items", "item", item.price > 1000000))
```

**Result**: 0 (no elements pass filter)

---

### Nested Iterators

```yaml
map(map("items", "item", item.price), "price", price * 2)
```

**Second map operates on first map's output**:
```
IteratorOperand {
  Steps: [
    MapStep(item.price),
    MapStep(price * 2),  # Uses "price" from previous step
  ]
}
```

**Scoping**: Each step gets its own scope level

---

## Summary

**Iterator composition with zero allocations**:
- ✅ Lazy evaluation
- ✅ Single pass execution
- ✅ Iterator fusion
- ✅ Composable syntax
- ✅ Follows forAll/forSome pattern
- ✅ Reuses existing scope mechanism

**Estimated**: ~2100 lines, 5 days, production-ready filter/map/reduce

Ready to implement!
