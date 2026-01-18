# Design: Simple Array Aggregation Functions

## Overview

Add straightforward aggregation functions without filter/map complexity.

---

## Proposed Functions

### 1. count(array_path, element_name, condition)

**Returns**: Integer count of matching elements

```yaml
# Count active items
expression: count("items", "item", item.active == true) > 5

# Count adults
expression: count("users", "user", user.age >= 18) >= 10

# Count with complex condition
expression: count("orders", "order", order.status == "pending" && order.total > 100) > 0
```

**Similar to**: any, but returns count instead of boolean

---

### 2. sum(array_path, element_name, expression)

**Already implemented** - keep as-is:

```yaml
expression: sum("items", "item", item.price) > 1000
expression: sum("items", "item", item.price * item.quantity) > 500
expression: sum("items", "item", if(item.active, item.price, 0)) > 100
```

---

### 3. avg(array_path, element_name, expression)

**Already implemented** - keep as-is:

```yaml
expression: avg("ratings", "r", r) >= 4.0
expression: avg("items", "item", if(item.active, item.rating, undefined)) >= 4.5
```

---

### 4. min(array_path, element_name, expression)

**New aggregation form** (different from existing scalar min):

```yaml
# Minimum price in array
expression: min("items", "item", item.price) < 100

# Minimum age
expression: min("users", "user", user.age) >= 18

# Minimum with condition
expression: min("items", "item", if(item.active, item.price, 999999)) < 50
```

**Note**: Keep existing `min(a, b, c)` scalar form, add this array form

---

### 5. max(array_path, element_name, expression)

**New aggregation form**:

```yaml
# Maximum price
expression: max("items", "item", item.price) > 500

# Maximum rating
expression: max("reviews", "r", r.stars) <= 5

# Maximum with filter
expression: max("items", "item", if(item.category == "premium", item.price, 0)) < 1000
```

---

## Function Name Improvements

### Rename Quantifiers (Breaking Change)

**Current** → **Proposed**:
- `all()` → `all()` (more standard, shorter)
- `any()` → `any()` (matches SQL, Python, JavaScript)

**Rationale**:
- `all()` and `any()` are universal (Python, SQL, JavaScript)
- Shorter, clearer
- "all" sounds verbose
- Industry standard naming

**Migration**:
```yaml
# Old
expression: all("items", "item", item.valid == true)
expression: any("items", "item", item.shipped == true)

# New
expression: all("items", "item", item.valid == true)
expression: any("items", "item", item.shipped == true)
```

**Keep as aliases** for backward compatibility:
- all → calls all()
- any → calls any()

---

## Implementation Approach

### Pattern: All Follow all/any

Each function:
1. Parse arguments (array path, element name, expression/condition)
2. Setup iteration scope
3. Iterate array
4. Evaluate expression/condition per element
5. Aggregate result

**No iterators, no intermediate arrays** - just direct iteration like all/any.

---

### count() Implementation (~80 lines)

```go
func (repo *CompareCondRepo) funcCount(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
    // Parse args (same as all)
    arrayPath, elementName, condition := parseArrayIterationArgs(n)

    // Setup scope (same as all)
    arrayAddress, newScope := setupIterationScope(arrayPath, elementName, scope)

    // Evaluate condition in scope
    condOperand := evaluateCondition(condition, newScope)

    return NewExprOperand(func(event, frames) {
        count := 0

        // Iterate like all
        for i := 0; i < numElements; i++ {
            elem := getElement(i)
            frames[nestingLevel] = elem

            result := condOperand.Evaluate(event, frames)

            // If true, increment count
            if isTrue(result) {
                count++
            }
        }

        return NewIntOperand(count)
    })
}
```

**Identical pattern to all** - just counts instead of checking all.

---

### min/max array forms (~160 lines total)

Similar to count, but track min/max value instead of count.

---

## Comparison: With vs Without Filter/Map

### Use Case: Sum active items

**With filter/map** (complex):
```yaml
sum(map(filter("items", "item", item.active), "item", item.price))
```

**Without filter/map** (simple):
```yaml
sum("items", "item", if(item.active, item.price, 0))
```

**Winner**: Simple version is clearer!

---

### Use Case: Count adults

**With filter** (complex):
```yaml
length(filter("users", "user", user.age >= 18))
```

**With count** (simple):
```yaml
count("users", "user", user.age >= 18)
```

**Winner**: count() is clearer and direct!

---

### Use Case: Average adult scores

**With filter/map** (complex):
```yaml
avg(filter("users", "user", user.age >= 18), "user", user.score)
```

**Without filter** (simple):
```yaml
avg("users", "user", if(user.age >= 18, user.score, undefined))
```

**Winner**: Similar complexity, but second is more explicit about skipping.

---

## Proposed Implementation

### Functions to Add

1. ✅ `sum(array, elem, expr)` - Already exists
2. ✅ `avg(array, elem, expr)` - Already exists
3. **`count(array, elem, condition)`** - NEW (~80 lines)
4. **`min(array, elem, expr)`** - NEW (~80 lines)
5. **`max(array, elem, expr)`** - NEW (~80 lines)

### Functions to Rename

6. **`all → all`** (alias all for compatibility)
7. **`any → any`** (alias any for compatibility)

**Total new code**: ~240 lines
**Total test code**: ~300 lines

---

## Benefits Over Filter/Map

1. **Simpler**: No IteratorOperand type needed
2. **Clearer**: Intent is obvious from function name
3. **Consistent**: All follow same pattern as all/any
4. **No map chaining complexity**: Avoided entirely
5. **Smaller**: ~240 lines vs ~420 lines
6. **Easier to maintain**: Standard aggregation pattern

---

## Real-World Examples

```yaml
# E-commerce: Count items in cart
expression: count("cart_items", "item", item.quantity > 0) >= 1

# User management: Any admin exists
expression: any("users", "user", user.role == "admin") == true

# Inventory: All items in stock
expression: all("order_items", "item", item.in_stock == true) == true

# Pricing: Maximum price check
expression: max("items", "item", item.price) <= 500

# Quality: Minimum rating
expression: min("reviews", "review", review.stars) >= 3

# Finance: Total order value
expression: sum("items", "item", item.price * item.quantity) > 1000

# Analytics: Average session duration
expression: avg("sessions", "s", s.duration_seconds) < 300
```

**These are clear, practical rule matching use cases** ✓

---

## Recommendation

**Abandon filter/map complexity**, implement:
1. count(array, elem, condition)
2. min(array, elem, expr) - array form
3. max(array, elem, expr) - array form
4. Rename all → all, any → any (with backward compat aliases)

**Estimated**: 1 day vs 2-3 more days for filter/map debugging

What do you think?
