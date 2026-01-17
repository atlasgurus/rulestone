# Design: Iterator-Based Filter/Map/Reduce

## Overview

Implements composable `filter()`, `map()`, and `reduce()` functions using **lazy iterators** with zero intermediate array allocations.

---

## Core Design Principle

**Lazy Evaluation with Iterator Fusion**

Functions compose to build an execution plan, which executes in a **single pass** when consumed by an aggregation function.

**Example**:
```yaml
sum(map(filter("items", "item", item.active), "item", item.price))
```

**Execution**:
1. Builds iterator chain: Filter → Map → Sum
2. Single pass iteration: Check active, extract price, accumulate
3. **Zero intermediate arrays**
4. **One iteration over source data**

---

## User-Facing Syntax

### filter(array_path, element_name, condition)

**Returns**: IteratorOperand (internal type, appears like array to users)

**Syntax**:
```yaml
filter("items", "item", item.price > 10)
filter("users", "user", user.age >= 18 && user.verified)
```

**Usage**:
```yaml
# Count filtered elements
length(filter("items", "item", item.active)) > 5

# Sum filtered values
sum(filter("items", "item", item.active), "item", item.price) > 100

# Chain filters
filter(filter("items", "item", item.active), "item", item.price > 10)
```

---

### map(iterator_or_array, element_name, expression)

**Returns**: IteratorOperand

**Syntax**:
```yaml
map("items", "item", item.price)
map(filter("items", "item", item.active), "item", item.price * 1.08)
```

**Usage**:
```yaml
# Sum mapped values
sum(map("items", "item", item.price)) > 1000

# Min of transformed values
min(map("items", "item", item.rating * 2)) >= 8

# Chain map after filter
max(map(filter("items", "item", item.active), "item", item.price)) < 500
```

---

### reduce(iterator_or_array, accumulator_name, element_name, expression, initial_value)

**Returns**: Single value (type depends on initial value)

**Syntax**:
```yaml
reduce("items", "sum", "item", sum + item.price, 0)
reduce("values", "product", "val", product * val, 1)
reduce("names", "result", "name", result + ", " + name, "")
```

**Usage**:
```yaml
# Custom aggregation
reduce("items", "total", "item", total + if(item.taxable, item.price * 1.08, item.price), 0) > 100

# Product
reduce("factors", "prod", "f", prod * f, 1) == 24

# Filtered reduce
reduce(filter("items", "item", item.active), "sum", "item", sum + item.price, 0) > 500
```

---

### Consuming Functions

**Functions that consume iterators**:
- `length(iterator)` - Count elements
- `sum(iterator, elem, expr)` - Sum values
- `min(iterator, elem, expr)` - Min value
- `max(iterator, elem, expr)` - Max value
- `avg(iterator, elem, expr)` - Average value (NEW)

**Updated existing functions** to accept IteratorOperand.

---

## Internal Implementation

### IteratorOperand Type

```go
// condition/condition.go

type IteratorOperand struct {
    ArrayAddress *objectmap.AttributeAddress
    Steps        []IteratorStep
    Scope        *ForEachScope
}

type IteratorStep struct {
    Type    IteratorStepType
    Operand condition.Operand
    Scope   *ForEachScope
}

type IteratorStepType int

const (
    FilterStepType IteratorStepType = iota
    MapStepType
)

const IteratorOperandKind OperandKind = 8

func NewIteratorOperand(arrayAddr *objectmap.AttributeAddress, scope *ForEachScope) *IteratorOperand {
    return &IteratorOperand{
        ArrayAddress: arrayAddr,
        Steps:        []IteratorStep{},
        Scope:        scope,
    }
}

// Implement Operand interface
func (v *IteratorOperand) GetKind() OperandKind {
    return IteratorOperandKind
}

func (v *IteratorOperand) GetHash() uint64 {
    // Hash based on array address and steps
    h := immutable.HashInt(int64(IteratorOperandKind))
    h = h ^ immutable.HashInts(v.ArrayAddress.Address)
    for _, step := range v.Steps {
        h = h ^ step.Operand.GetHash()
    }
    return h
}

func (v *IteratorOperand) Equals(o immutable.SetElement) bool {
    other, ok := o.(*IteratorOperand)
    if !ok {
        return false
    }
    // Compare array address and steps
    return reflect.DeepEqual(v.ArrayAddress, other.ArrayAddress) &&
           len(v.Steps) == len(other.Steps)
    // TODO: Deep compare steps
}

func (v *IteratorOperand) IsConst() bool {
    return false // Iterators are not constants
}

func (v *IteratorOperand) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) Operand {
    // Iterators can't be directly evaluated - must be consumed by aggregation
    return NewErrorOperand(fmt.Errorf("iterator cannot be evaluated directly - use with sum(), length(), etc."))
}

func (v *IteratorOperand) Convert(to OperandKind) Operand {
    // Iterators don't convert
    return NewErrorOperand(fmt.Errorf("iterator cannot be converted"))
}

func (v *IteratorOperand) Greater(o Operand) bool {
    panic("iterator cannot be compared")
}
```

**Estimated**: ~150 lines for IteratorOperand type

---

### filter() Implementation

```go
// engine/engine_impl.go

func (repo *CompareCondRepo) funcFilter(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
    if len(n.Args) != 3 {
        return condition.NewErrorOperand(fmt.Errorf("filter() requires 3 arguments: array_or_iterator, element_name, condition"))
    }

    // Parse first argument - could be array path or iterator
    firstArg := repo.evalAstNode(n.Args[0], scope)
    if firstArg.GetKind() == condition.ErrorOperandKind {
        return firstArg
    }

    // Get element name
    elemOperand := repo.evalAstNode(n.Args[1], scope)
    if elemOperand.GetKind() != condition.StringOperandKind {
        return condition.NewErrorOperand(fmt.Errorf("filter() element name must be string"))
    }
    elementName := string(elemOperand.(condition.StringOperand))

    // Get condition expression AST
    condExpr := n.Args[2]

    var iterator *condition.IteratorOperand
    var arrayAddress *objectmap.AttributeAddress
    var parentScope *ForEachScope

    // Check if first arg is an iterator
    if firstArg.GetKind() == condition.IteratorOperandKind {
        // Chaining iterators
        iterator = firstArg.(*condition.IteratorOperand)
        arrayAddress = iterator.ArrayAddress
        parentScope = iterator.Scope
    } else {
        // Starting from array path
        if firstArg.GetKind() != condition.StringOperandKind {
            return condition.NewErrorOperand(fmt.Errorf("filter() first argument must be array path or iterator"))
        }
        arrayPath := string(firstArg.(condition.StringOperand))

        // Get array address
        var err error
        arrayAddress, err = getAttributePathAddress(arrayPath+"[]", scope)
        if err != nil {
            return condition.NewErrorOperand(err)
        }

        // Create new iterator
        iterator = condition.NewIteratorOperand(arrayAddress, scope)
        parentScope = scope
    }

    // Create scope for this iteration level
    newScope := &ForEachScope{
        Element:      elementName,
        Path:         arrayAddress.Path,
        NestingLevel: parentScope.NestingLevel + 1,
        ParentScope:  parentScope,
        AttrDictRec:  arrayAddress.DictRec,
    }

    // Evaluate condition in new scope
    condOperand := repo.evalAstNode(condExpr, newScope)
    if condOperand.GetKind() == condition.ErrorOperandKind {
        return condOperand
    }

    // Append filter step to iterator
    iterator.Steps = append(iterator.Steps, condition.IteratorStep{
        Type:    condition.FilterStepType,
        Operand: condOperand,
        Scope:   newScope,
    })

    return iterator
}
```

**Estimated**: ~100 lines

---

### map() Implementation

```go
func (repo *CompareCondRepo) funcMap(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
    if len(n.Args) != 3 {
        return condition.NewErrorOperand(fmt.Errorf("map() requires 3 arguments: array_or_iterator, element_name, expression"))
    }

    // Parse first argument
    firstArg := repo.evalAstNode(n.Args[0], scope)
    if firstArg.GetKind() == condition.ErrorOperandKind {
        return firstArg
    }

    // Get element name
    elemOperand := repo.evalAstNode(n.Args[1], scope)
    if elemOperand.GetKind() != condition.StringOperandKind {
        return condition.NewErrorOperand(fmt.Errorf("map() element name must be string"))
    }
    elementName := string(elemOperand.(condition.StringOperand))

    // Get expression AST
    exprAst := n.Args[2]

    var iterator *condition.IteratorOperand
    var arrayAddress *objectmap.AttributeAddress
    var parentScope *ForEachScope

    // Check if first arg is iterator (chaining)
    if firstArg.GetKind() == condition.IteratorOperandKind {
        iterator = firstArg.(*condition.IteratorOperand)
        arrayAddress = iterator.ArrayAddress
        parentScope = iterator.Scope
    } else {
        // Starting from array path
        if firstArg.GetKind() != condition.StringOperandKind {
            return condition.NewErrorOperand(fmt.Errorf("map() first argument must be array path or iterator"))
        }
        arrayPath := string(firstArg.(condition.StringOperand))

        var err error
        arrayAddress, err = getAttributePathAddress(arrayPath+"[]", scope)
        if err != nil {
            return condition.NewErrorOperand(err)
        }

        iterator = condition.NewIteratorOperand(arrayAddress, scope)
        parentScope = scope
    }

    // Create scope
    newScope := &ForEachScope{
        Element:      elementName,
        Path:         arrayAddress.Path,
        NestingLevel: parentScope.NestingLevel + 1,
        ParentScope:  parentScope,
        AttrDictRec:  arrayAddress.DictRec,
    }

    // Evaluate expression in scope
    exprOperand := repo.evalAstNode(exprAst, newScope)
    if exprOperand.GetKind() == condition.ErrorOperandKind {
        return exprOperand
    }

    // Append map step
    iterator.Steps = append(iterator.Steps, condition.IteratorStep{
        Type:    condition.MapStepType,
        Operand: exprOperand,
        Scope:   newScope,
    })

    return iterator
}
```

**Estimated**: ~100 lines

---

### reduce() Implementation

```go
func (repo *CompareCondRepo) funcReduce(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
    if len(n.Args) != 5 {
        return condition.NewErrorOperand(fmt.Errorf("reduce() requires 5 arguments: array_or_iterator, accumulator_name, element_name, expression, initial_value"))
    }

    // Parse arguments
    firstArg := repo.evalAstNode(n.Args[0], scope)
    accumName := string(repo.evalAstNode(n.Args[1], scope).(condition.StringOperand))
    elemName := string(repo.evalAstNode(n.Args[2], scope).(condition.StringOperand))
    exprAst := n.Args[3]
    initialValue := repo.evalAstNode(n.Args[4], scope)

    // Get iterator or create from array path
    var iterator *condition.IteratorOperand
    // ... similar to filter/map

    // Create scope with TWO variables (accumulator + element)
    // Evaluate expression that uses both
    // Return operand that executes iterator + reduce

    return repo.CondFactory.NewExprOperand(
        func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
            accumulator := initialValue.Evaluate(event, frames)

            // Execute iterator chain with reduce
            executeIteratorWithReduce(iterator, accumulator, exprOperand, event, frames)

            return accumulator
        }, ...)
}
```

**Estimated**: ~150 lines

---

### Iterator Execution Engine

**Core execution function** that all consuming functions use:

```go
// Execute iterator chain in single pass
func executeIterator(
    iterator *IteratorOperand,
    event *ObjectAttributeMap,
    frames []interface{},
    consumer func(elementValue Operand) bool, // Return false to stop iteration
) error {
    // Get number of elements in source array
    numElements, err := event.GetNumElementsAtAddress(iterator.ArrayAddress, frames)
    if err != nil {
        return err // Array missing or error
    }

    // Iterate source array once
    for i := 0; i < numElements; i++ {
        // Get element from source array
        elemValue := getElementAtIndex(event, iterator.ArrayAddress, i, frames)

        // Apply each step in the chain
        currentValue := elemValue
        shouldInclude := true

        for _, step := range iterator.Steps {
            // Bind current value to this step's scope
            frames[step.Scope.NestingLevel] = currentValue

            switch step.Type {
            case FilterStepType:
                // Evaluate filter condition
                result := step.Operand.Evaluate(event, frames)

                // Check if passes filter
                boolResult := result.Convert(BooleanOperandKind)
                if boolResult.GetKind() != BooleanOperandKind ||
                   !bool(boolResult.(BooleanOperand)) {
                    shouldInclude = false
                    break // Skip to next element
                }

            case MapStepType:
                // Transform value
                currentValue = step.Operand.Evaluate(event, frames)

                // Handle undefined/null from map
                if currentValue.GetKind() == UndefinedOperandKind {
                    shouldInclude = false
                    break
                }
            }
        }

        // If element passed all steps, pass to consumer
        if shouldInclude {
            if !consumer(currentValue) {
                break // Consumer says stop
            }
        }
    }

    return nil
}
```

**Estimated**: ~80 lines

---

### Consuming Functions

#### length() - Updated to Accept Iterators

```go
func funcLength(arg) {
    // Check if iterator
    if arg.GetKind() == IteratorOperandKind {
        iterator := arg.(*IteratorOperand)

        return NewExprOperand(func(event, frames) {
            count := 0

            // Execute iterator, count elements
            executeIterator(iterator, event, frames, func(elem Operand) bool {
                count++
                return true // Continue
            })

            return NewIntOperand(count)
        })
    }

    // Existing logic for array paths
    // ...
}
```

**Estimated**: +20 lines to existing function

---

#### sum() - NEW, Consumes Iterators

```go
func funcSum(arrayOrIterator, elemName, expression) {
    // If first arg is iterator
    if arg.GetKind() == IteratorOperandKind {
        iterator := arg.(*IteratorOperand)

        // Expression is identity (already transformed by map)
        return NewExprOperand(func(event, frames) {
            sum := 0.0

            executeIterator(iterator, event, frames, func(elem Operand) bool {
                // Element already filtered and mapped
                numeric := elem.Convert(FloatOperandKind)
                if numeric.GetKind() != ErrorOperandKind {
                    sum += float64(numeric.(FloatOperand))
                }
                return true
            })

            return NewFloatOperand(sum)
        })
    }

    // Direct array iteration (no filter/map)
    arrayPath := string(arg.(StringOperand))

    // Parse element name and expression
    elemName := ...
    expr := ...

    // Setup scope (like forAll)
    arrayAddress, newScope := setupArrayIteration(arrayPath, elemName, scope)

    return NewExprOperand(func(event, frames) {
        sum := 0.0

        numElements := getNumElements(arrayAddress, frames)
        for i := 0; i < numElements; i++ {
            elem := getElementAtIndex(i)
            frames[newScope.NestingLevel] = elem

            result := expression.Evaluate(event, frames)

            // Skip undefined/null
            if result is numeric {
                sum += result
            }
        }

        return sum
    })
}
```

**Estimated**: ~120 lines

---

#### avg() - NEW

```go
func funcAvg(iterator, elemName, expression) {
    // Similar to sum, but divide by count
    // Handle empty array → return undefined
}
```

**Estimated**: ~80 lines

---

#### min/max() - Updated

```go
// Existing min/max: min(a, b, c) - scalar values
// NEW: min(iterator, elem, expr) - from iterated values

func funcMin(arg1, ...) {
    // Check first argument type
    if arg1.GetKind() == IteratorOperandKind {
        // Iterator min
        return minFromIterator(arg1, elemName, expression)
    }

    // Existing scalar min logic
}

func minFromIterator(iterator, elemName, expr) {
    return NewExprOperand(func(event, frames) {
        minVal := math.Inf(1)
        hasValue := false

        executeIterator(iterator, event, frames, func(elem) {
            // Element already filtered/mapped if iterator has steps
            // Evaluate expression
            result := expression.Evaluate(event, frames)

            if numeric(result) {
                if !hasValue || result < minVal {
                    minVal = result
                    hasValue = true
                }
            }
            return true
        })

        if !hasValue {
            return NewUndefinedOperand(nil)
        }
        return NewFloatOperand(minVal)
    })
}
```

**Estimated**: +80 lines to existing min/max

---

## Complete Example: Iterator Fusion

### User Expression
```yaml
expression: sum(map(filter("items", "item", item.active), "item", item.price * 1.08)) > 1000
```

### Compilation Phase

**Step 1**: Parse `filter("items", "item", item.active)`
```
→ IteratorOperand {
    ArrayAddress: address of "items"
    Steps: [FilterStep(item.active)]
  }
```

**Step 2**: Parse `map(iterator, "item", item.price * 1.08)`
```
→ IteratorOperand {
    ArrayAddress: address of "items"
    Steps: [
      FilterStep(item.active),
      MapStep(item.price * 1.08)
    ]
  }
```

**Step 3**: Parse `sum(iterator)`
```
→ ExprOperand that will execute iterator and sum results
```

### Runtime Execution (Single Pass!)

```go
sum := 0.0

for i := 0; i < len(items); i++ {
    elem := items[i]
    currentValue := elem

    // Step 1: FilterStep(item.active)
    frames[1] = elem
    if !item.active {
        continue  // Skip element
    }

    // Step 2: MapStep(item.price * 1.08)
    frames[1] = currentValue
    currentValue = elem.price * 1.08

    // Consumer (sum)
    sum += currentValue
}

return sum
```

**Result**:
- **1 iteration**
- **0 allocations**
- **All transformations inline**

---

## Implementation Checklist

### Core Iterator System

- [ ] Add IteratorOperand type (condition/condition.go) - ~150 lines
- [ ] Add IteratorStepType enum
- [ ] Add IteratorStep struct
- [ ] Implement Operand interface for IteratorOperand

### Filter/Map/Reduce Functions

- [ ] Implement funcFilter (engine/engine_impl.go) - ~100 lines
- [ ] Implement funcMap (engine/engine_impl.go) - ~100 lines
- [ ] Implement funcReduce (engine/engine_impl.go) - ~150 lines

### Iterator Execution

- [ ] Add executeIterator() helper function - ~80 lines
- [ ] Helper for binding values to scope frames
- [ ] Helper for getting array elements

### Update Existing Functions

- [ ] Update length() to accept IteratorOperand - +20 lines
- [ ] Update min() to accept IteratorOperand - +80 lines
- [ ] Update max() to accept IteratorOperand - +80 lines

### New Aggregation Functions

- [ ] Add sum(iterator, elem, expr) - ~120 lines
- [ ] Add avg(iterator, elem, expr) - ~80 lines

### Testing

- [ ] filter() tests - ~200 lines
- [ ] map() tests - ~200 lines
- [ ] reduce() tests - ~200 lines
- [ ] Composition tests (filter+map, etc.) - ~200 lines
- [ ] Iterator with undefined/null - ~150 lines
- [ ] Performance tests - ~100 lines

### Documentation

- [ ] README examples
- [ ] ARCHITECTURE.md update
- [ ] Migration guide (if needed)

---

## Total Effort Estimate

| Component | Lines | Effort |
|-----------|-------|--------|
| IteratorOperand type | ~150 | 0.5 day |
| filter/map/reduce functions | ~350 | 1 day |
| Iterator execution | ~80 | 0.5 day |
| Update existing functions | ~180 | 0.5 day |
| New aggregations | ~200 | 0.5 day |
| **Subtotal Code** | **~960** | **3 days** |
| Tests | ~1050 | 1.5 days |
| Documentation | ~100 | 0.5 day |
| **Total** | **~2110** | **5 days** |

---

## File Structure

```
condition/
  condition.go          # IteratorOperand type (~150 lines)

engine/
  engine_impl.go        # filter/map/reduce + helpers (~730 lines)

tests/
  filter_test.go        # NEW (~200 lines)
  map_test.go           # NEW (~200 lines)
  reduce_test.go        # NEW (~200 lines)
  iterator_composition_test.go  # NEW (~200 lines)
  iterator_undefined_test.go    # NEW (~150 lines)
  iterator_performance_test.go  # NEW (~100 lines)
```

---

## Examples

```yaml
# Simple filter
expression: length(filter("items", "item", item.active)) > 5

# Simple map
expression: sum(map("items", "item", item.price)) > 1000

# Filter + map
expression: sum(map(filter("items", "item", item.active), "item", item.price * 1.08)) > 500

# Multiple filters
expression: length(filter(filter("items", "item", item.category == "food"), "item", item.price > 10)) > 0

# Custom reduce
expression: reduce("items", "sum", "item", sum + item.value, 0) > 100

# Complex chaining
expression: avg(map(filter("users", "u", u.age >= 18), "u", u.score)) >= 75
```

**All execute in single pass with zero allocations** ✓

---

## Next Steps

1. Review this iterator-based design
2. Approve for implementation
3. Implement IteratorOperand type
4. Implement filter/map/reduce
5. Update consuming functions
6. Comprehensive testing

Ready to implement when approved!
