# Filter/Map/Reduce Functions - Implementation Plan

## Overview

This folder contains the design and implementation plan for adding composable filter/map/reduce functions to rulestone using **lazy iterators with zero intermediate allocations**.

---

## Approach: Iterator Composition

We're implementing functional programming primitives that compose via internal iterators, executing in a **single pass** with **zero intermediate arrays**.

### Key Design

**User writes**:
```yaml
sum(map(filter("items", "item", item.active), "item", item.price * 1.08))
```

**Executes as**:
- Single iteration over items array
- Filter inline: check if item.active
- Map inline: calculate item.price * 1.08
- Aggregate inline: sum values
- **Zero allocations, one pass** ✓

Following the same pattern as existing `forAll()` and `forSome()` functions.

---

## Documents

### DESIGN.md
Complete implementation specification:
- IteratorOperand internal type
- filter/map/reduce function signatures
- Iterator execution engine
- Integration with existing functions
- Complete code examples with line estimates
- Implementation checklist

### ITERATOR_DESIGN.md
Internal architecture details:
- Lazy evaluation mechanics
- Scope chaining (reuses ForEachScope)
- Iterator fusion optimization
- Memory analysis
- Performance characteristics
- Edge cases (undefined, null, empty arrays)

---

## Functions to Implement

### Core Iterators

**1. filter(array_or_iterator, element_name, condition)**
- Returns: IteratorOperand
- Chains with other iterators
- Zero allocations

**2. map(array_or_iterator, element_name, expression)**
- Returns: IteratorOperand
- Transforms values
- Zero allocations

**3. reduce(array_or_iterator, accumulator_name, element_name, expression, initial_value)**
- Returns: Accumulated value
- Custom aggregations
- Dual-variable scoping

### Consuming Functions

**NEW**:
- `sum(iterator, elem, expr)` - Sum values
- `avg(iterator, elem, expr)` - Average values

**UPDATED** (accept IteratorOperand):
- `length(iterator)` - Count elements
- `min(iterator, elem, expr)` - Minimum value
- `max(iterator, elem, expr)` - Maximum value

---

## Usage Examples

```yaml
# Simple filter
expression: length(filter("items", "item", item.active)) > 5

# Simple map
expression: sum(map("items", "item", item.price)) > 1000

# Filter + map composition
expression: sum(map(filter("items", "item", item.active), "item", item.price * 1.08)) > 500

# Multiple filters
expression: length(filter(filter("items", "item", item.category == "food"), "item", item.price > 10)) > 0

# Custom reduce
expression: reduce("items", "total", "item", total + if(item.taxable, item.price * 1.08, item.price), 0) > 100

# Complex chaining
expression: avg(map(filter("users", "u", u.age >= 18), "u", u.score)) >= 75

# Min/max of filtered values
expression: max(map(filter("items", "item", item.active), "item", item.price)) < 500
```

**All execute in single pass with zero allocations** ✓

---

## Implementation Estimate

| Component | Lines | Effort |
|-----------|-------|--------|
| **Code** | | |
| IteratorOperand type | ~150 | 0.5 day |
| filter/map/reduce functions | ~350 | 1 day |
| Iterator execution engine | ~80 | 0.5 day |
| Update existing functions | ~180 | 0.5 day |
| New aggregations (sum/avg) | ~200 | 0.5 day |
| **Tests** | | |
| filter/map/reduce tests | ~600 | 1 day |
| Composition tests | ~200 | 0.5 day |
| Undefined/null tests | ~150 | 0.5 day |
| Performance tests | ~100 | 0.5 day |
| **Documentation** | ~100 | 0.5 day |
| **Total** | **~2110** | **5 days** |

---

## Files to Modify

```
condition/
  condition.go          # IteratorOperand type (~150 lines)

engine/
  engine_impl.go        # filter/map/reduce + helpers (~730 lines)

tests/
  filter_test.go                # NEW (~200 lines)
  map_test.go                   # NEW (~200 lines)
  reduce_test.go                # NEW (~200 lines)
  iterator_composition_test.go  # NEW (~200 lines)
  iterator_undefined_test.go    # NEW (~150 lines)
  iterator_performance_test.go  # NEW (~100 lines)

README.md                        # Examples and documentation
```

---

## Key Benefits

### 1. Zero Allocations
- No intermediate arrays created
- Uses existing frames array for scoping
- Only final result allocated

### 2. Single Pass
- Iterator fusion combines all steps
- Early exit on filter failure
- Optimal performance

### 3. Composable
- filter + filter
- filter + map
- map + map
- Any combination works

### 4. Familiar Syntax
- Similar to JavaScript, Python, Scala
- Easy for developers to understand
- Follows functional programming conventions

### 5. Consistent with Existing
- Uses forAll/forSome scope pattern
- Same iteration mechanism
- Minimal new concepts

---

## Next Steps

1. ✅ Review and approve this design
2. Implement IteratorOperand type
3. Implement filter/map/reduce functions
4. Update consuming functions
5. Comprehensive testing
6. Documentation

Ready to implement!
