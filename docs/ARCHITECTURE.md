# Rulestone Architecture

## Table of Contents
1. [System Overview](#system-overview)
2. [Core Components](#core-components)
   - [AlwaysEvaluateCategories Pattern](#alwaysevaluatecategories-pattern)
   - [Negative Categories & DefaultCatList Pattern](#negative-categories--defaultcatlist-pattern)
3. [Category Engine](#category-engine)
4. [Expression Evaluation Pipeline](#expression-evaluation-pipeline)
   - [Null Handling Semantics](#7-null-handling-semantics)
5. [Memory Management](#memory-management)
6. [Performance Characteristics](#performance-characteristics)
7. [Design Patterns](#design-patterns)
8. [Maintainer's Guide: Common Pitfalls & Debugging](#maintainers-guide-common-pitfalls--debugging)

## System Overview

Rulestone is a high-performance business rule engine designed to efficiently evaluate thousands of rules against events at high throughput (tens of thousands of events per second).

### Key Design Goals
- **Performance**: Tens of thousands of evaluations per second
- **Scalability**: Handle thousands of rules efficiently
- **Expressiveness**: Rich expression language with Go-like syntax
- **Optimization**: Automatic common sub-expression elimination
- **Memory Efficiency**: Object pooling and deduplication

### Architecture Layers

```
┌──────────────────────────────────────┐
│         Application Layer             │
│  (User Code, Rule Registration)       │
└────────────────┬─────────────────────┘
                 │
┌────────────────┴─────────────────────┐
│         Engine API Layer              │
│   (RuleEngine, RuleEngineRepo)        │
└────────────────┬─────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
┌───────┴───────┐  ┌──────┴──────────┐
│  Expression   │  │   Category      │
│  Processing   │  │   Engine        │
│  (AST Parse)  │  │   (Bit Masks)   │
└───────┬───────┘  └──────┬──────────┘
        │                 │
┌───────┴─────────────────┴──────────┐
│     Condition & Operand Layer      │
│   (Logical Operators, Comparisons) │
└────────────────┬───────────────────┘
                 │
┌────────────────┴───────────────────┐
│     Object Attribute Mapping       │
│   (Event Data Access, Caching)     │
└────────────────────────────────────┘
```

## Core Components

### 1. RuleEngineRepo
**File**: `engine/engine_api.go`

Repository pattern implementation for managing rules.

**Responsibilities**:
- Rule registration from YAML/JSON files
- Rule ID management (auto-incrementing)
- Application context for error accumulation
- Interface to underlying condition repository

**Key Methods**:
```go
func NewRuleEngineRepo() *RuleEngineRepo
func (repo *RuleEngineRepo) RegisterRulesFromFile(filename string) ([]uint, error)
func (repo *RuleEngineRepo) GetAppCtx() *types.AppContext
```

### 2. RuleEngine
**File**: `engine/engine_api.go`

Main engine for event matching against registered rules.

**Responsibilities**:
- Event matching and rule evaluation
- Category engine coordination
- Metrics collection (evaluations, matches)
- Object attribute map pooling for memory efficiency

**Key Methods**:
```go
func NewRuleEngine(repo *RuleEngineRepo) (*RuleEngine, error)
func (engine *RuleEngine) MatchEvent(event interface{}) []condition.RuleIdType
```

**Performance Optimizations**:
- Object pooling for `ObjectAttributeMap` (`sync.Pool`)
- Reusable evaluation frames to avoid allocations
- Category-based pre-filtering before full evaluation

### 3. CompareCondRepo
**File**: `engine/engine_impl.go`

Core expression processing and condition generation engine.

**Responsibilities**:
- Parse Go-like expressions into AST
- Convert AST to internal condition representation
- Generate category evaluators for optimization
- Implement common sub-expression elimination
- Handle all/any scoping

**Key Data Structures**:
```go
type CompareCondRepo struct {
    // Maps attribute paths to categories that depend on them
    AttributeToCompareCondRecord map[string]*hashset.Set[*EvalCategoryRec]

    // CSE cache: same condition → same category
    CondToCompareCondRecord *hashmap.Map[condition.Condition, *EvalCategoryRec]

    // Categories that must evaluate even when attributes are missing
    // Used for: null checks, constant expressions, quantifiers on empty arrays
    AlwaysEvaluateCategories *hashset.Set[*EvalCategoryRec]

    ObjectAttributeMapper *objectmap.ObjectAttributeMapper
}
```

**Key Operations**:
- Expression parsing using Go's `go/parser` package
- AST traversal and transformation
- Type reconciliation between operands
- Scope management for array quantifiers

#### AlwaysEvaluateCategories Pattern

**Design Challenge**: The normal evaluation flow relies on attribute callbacks:
```
Event has attribute → Callback fires → Category evaluated
```

But what if:
- Attribute is missing from event (null checks: `field == null`)
- Expression has no event dependencies (constants: `1 == 1`)
- Container exists but is empty (quantifiers: `all("items", ...)` with `{items: []}`)

**Solution**: `AlwaysEvaluateCategories` bypasses the callback mechanism.

**Registration**: Categories are added during rule compilation:
```go
// Null checks
if isNullCheck {
    repo.AlwaysEvaluateCategories.Put(evalCatRec)
}

// Constant expressions (no event dependencies)
if len(evalCatRec.AttrKeys) == 0 {
    repo.AlwaysEvaluateCategories.Put(evalCatRec)
}

// Quantifiers (need to handle empty arrays)
if isQuantifier {
    repo.AlwaysEvaluateCategories.Put(evalCatRec)
}
```

**Evaluation**: In `RuleEngine.MatchEvent()`:
```go
// Normal flow: evaluate categories for found attributes
event.MapObject(v, func(addr []int) {
    catEvaluators := AttributeToCompareCondRecord[addr]
    matchingRecords.Put(catEvaluators)
})

// Force evaluation of always-evaluate categories
AlwaysEvaluateCategories.Each(func(catEvaluator *EvalCategoryRec) {
    matchingRecords.Put(catEvaluator)
})
```

**Critical Insight**: The evaluator still needs to determine the correct result internally. AlwaysEvaluateCategories only ensures it *runs* - the logic must handle missing data appropriately.

### Negative Categories & DefaultCatList Pattern

**Background**: Negated comparisons (`age != 18`) create a unique challenge - they should match when the field is **present and not equal**, but the current design makes them match when the field is **missing** as well.

#### The Problem

```yaml
Rule: age != 18
Event: { name: "john" }  # age field missing

Current Behavior: Rule MATCHES ✓
Desired Strict Behavior: Rule should NOT match (field missing = not applicable)
```

**Why this happens**:
1. Normal categories require attributes to be present to fire
2. Negated categories need special handling to work correctly
3. The "absence of a category" needs to mean "negative category fires"

#### Implementation Mechanism

**File**: `cateng/builder.go`

**Step 1: Negative Category Registration (Line 149-156)**
```go
func (fb *FilterBuilder) registerNegativeCat(cat types.Category) types.Category {
    negCat, ok := fb.NegCats[cat]
    if !ok {
        negCat = cat + types.MaxCategory  // ← Negative category ID
        fb.NegCats[cat] = negCat
    }
    return negCat
}
```

**Step 2: NOT Processing (Line 276-285)**
```go
func (fb *FilterBuilder) processNotOp(cond condition.Condition) CatFilter {
    switch cond.GetKind() {
    case condition.CategoryCondKind:
        cat := cond.(*condition.CategoryCond).Cat
        // Substitute NOT(Category) with NegativeCategory
        negCat := fb.registerNegativeCat(cat)
        return fb.computeCatFilter(condition.NewCategoryCond(negCat))
```

**Example**:
```
Expression: age != 18
  ↓ Parsing converts to:
NOT(CompareCondition(age == 18))
  ↓ Category assignment:
NOT(CategoryCond(123))   // Category 123 = "age == 18"
  ↓ processNotOp converts to:
CategoryCond(1000000123)  // Negative category = 123 + MaxCategory
```

**Step 3: DefaultCatList Construction (Line 636-639)**
```go
for cat := range fb.NegCats {  // ← For every category that has a negation
    result.DefaultCategories[cat] = len(result.DefaultCategories)
    result.DefaultCatList = append(result.DefaultCatList, cat)  // ← Add to DefaultCatList
}
```

**Critical Insight**: Categories in `DefaultCatList` are "default true" - they're assumed to match **unless proven otherwise**. This is the mechanism that makes negative categories work.

**Step 4: Runtime Evaluation (category_engine.go:83-95)**
```go
// Now process default categories
for i, cat := range f.FilterTables.DefaultCatList {
    if !defaultCatMap[i] {  // ← If category didn't match (field missing or false)
        negCat, found := f.FilterTables.NegCats[cat]
        csml := catToCatSetMask.Get(negCat)
        if csml != nil {
            applyCatSetMasks(csml, matchMaskArray, &result, f)  // ← Eval negative category!
        }
    }
}
```

#### Complete Flow Example

```
Rule: age != 18
Event: { name: "john" }  # age field missing

Compilation:
1. age != 18 → NOT(CategoryCond(cat_123))  // cat_123 = "age == 18"
2. processNotOp → CategoryCond(neg_cat_123)  // neg_cat_123 = cat_123 + MaxCategory
3. DefaultCatList.append(cat_123)  // Mark cat_123 as "default category"

Runtime:
1. MapObject: No "age" field → cat_123 not evaluated
2. defaultCatMap[cat_123] = false  // Category didn't fire
3. Process DefaultCatList:
   - cat_123 didn't fire → evaluate neg_cat_123
   - neg_cat_123 fires → Rule matches! ✓

Result: age != 18 matches when age is missing
```

#### Design Rationale

**Why not just add `age != 18` to AlwaysEvaluateCategories?**

Because the semantics are different:
- `AlwaysEvaluateCategories`: Always **evaluate** the expression (needed for null checks)
- `DefaultCatList`: Assume **default value** when not evaluated (needed for negations)

**Key Difference**:
```
age == null (AlwaysEvaluateCategories):
  → Always run the comparison, return result
  → age missing: evaluates to true ✓

age != 18 (DefaultCatList):
  → If age present: evaluate "age == 18", negate result
  → If age missing: assume "age == 18" is false, negate to true ✓
```

#### Implications for Strict Mode

**Current Permissive Behavior**: Negative comparisons match when fields are missing
- `age != 18` → matches when age missing
- `status != "active"` → matches when status missing
- `count != 0` → matches when count missing

**Desired Strict Behavior**: Fields must be present for any comparison
- `age != 18` → no match when age missing
- `status != "active"` → no match when status missing
- `count != 0` → no match when count missing

**Implementation Approach**: Disable `DefaultCatList` processing in strict mode
- Skip lines 636-639 in builder.go (don't populate DefaultCatList)
- Skip lines 83-95 in category_engine.go (don't process default categories)
- Result: Negated categories only fire when field is present

## Category Engine

The category engine is Rulestone's "secret sauce" for performance. It uses bit masks and Aho-Corasick string matching to quickly filter rules before full expression evaluation.

### Concept

**Category**: A numeric identifier representing a simple boolean condition.

Example:
- Category 1 = `age > 18`
- Category 2 = `status == "active"`
- Category 3 = `country == "US"`

**Rule as Categories**:
```
Original Rule: (age > 18 AND status == "active") OR country == "US"
Category Form: (1 AND 2) OR 3
Representation: [[1, 2], [3]]  // OR of AND terms
```

### How It Works

#### 1. Rule Compilation

**File**: `cateng/builder.go`

During engine creation:
1. Extract simple comparisons from complex expressions
2. Assign each unique comparison a category ID
3. Convert rules to category tables (AND-OR form)
4. Build optimization data structures

#### 2. Bit Mask Optimization

Rules are grouped into "CatSets" with bit masks:

```
CatSet for Categories {1, 2}:
- Bit 0: Category 1 present
- Bit 1: Category 2 present
- Mask: 0b11 (both bits must be set)

Event has categories {1, 2, 5}:
- Set bit 0 (category 1)
- Set bit 1 (category 2)
- Check: 0b11 == 0b11 ✓ Match!
```

**Benefits**:
- O(1) matching for AND terms using bit operations
- Early termination when categories missing
- CPU cache-friendly operations

#### 3. Frequency-Based Optimization

**File**: `cateng/builder.go:276-319`

Common categories are factored out:

```
Before:
Rule 1: Cat1 AND Cat2 AND Cat3
Rule 2: Cat1 AND Cat2 AND Cat4

After (if Cat1 AND Cat2 frequent):
Synthetic Cat100 = Cat1 AND Cat2
Rule 1: Cat100 AND Cat3
Rule 2: Cat100 AND Cat4
```

**Threshold Parameters**:
- `OrOptimizationFreqThreshold`: Min frequency for OR optimization
- `AndOptimizationFreqThreshold`: Min frequency for AND optimization

#### 4. String Matching Optimization

**File**: `engine/lookups.go`

For rules with many string equality comparisons:
- Build Aho-Corasick automaton
- Single pass matches multiple strings
- Significantly faster than individual comparisons

### Category Engine Flow

```
Event → ObjectAttributeMap
          │
          ↓
    Evaluate Categories
    (Simple comparisons)
          │
          ↓
    Category Matching
    (Bit mask operations)
          │
          ↓
    Candidate Rules
          │
          ↓
    Full Expression Eval
    (Only for candidates)
          │
          ↓
    Matched Rule IDs
```

## Expression Evaluation Pipeline

### 1. Expression Parsing

**File**: `engine/engine_impl.go:1144-1163`

Uses Go's native parser:
```go
import "go/parser"
import "go/ast"

// Parse expression string
expr, err := parser.ParseExpr(expressionString)
// Returns: *ast.Expr
```

### 2. AST Preprocessing

**File**: `engine/engine_impl.go:1163-1336`

Recursive traversal converting AST nodes to operands:

```
AST Node Types:
- BinaryExpr  → Comparison/Logical Operations
- CallExpr    → Function Calls (all, hasValue, etc.)
- SelectorExpr → Nested Attribute Access (user.name)
- IndexExpr   → Array Indexing (items[0])
- Ident       → Variable References
- BasicLit    → Constants (strings, numbers)
```

**Key Transformations**:
- Attribute paths → Address operands
- Nested access → Chain of operations
- Array indexing → Index address calculations
- Function calls → Operand generators

### 3. Condition Generation

**File**: `engine/engine_impl.go:579-799`

Build condition tree:
```
Expression: age > 18 AND status == "active"

Condition Tree:
    AndCondition
    ├─ CompareCondition (GreaterThan)
    │  ├─ AddressOperand (age)
    │  └─ InterfaceOperand (18)
    └─ CompareCondition (Equality)
       ├─ AddressOperand (status)
       └─ InterfaceOperand ("active")
```

### 4. all/any Handling

**File**: `engine/engine_impl.go:599-758`

Quantifiers over arrays require special scoping:

```go
// Original: all(item, items, item.active == true)
// Compiled:
ForAllCondition {
    Element: "item",
    Array: "items",
    Scope: {
        ParentScope: nil,
        Element: "item",
        ArrayPath: "items",
    },
    Condition: CompareCondition {
        Left: AddressOperand("item.active"),  // Resolves in scope
        Right: BoolOperand(true),
    }
}
```

**Evaluation**:
1. Retrieve array from event
2. For each element, bind to scope variable
3. Evaluate condition in element scope
4. all: all must be true
5. any: at least one must be true

**Design Insight: Vacuous Truth**:
- `all` on an **empty array** returns `true` (vacuous truth principle)
- `all` on a **missing array** returns `false` (rule doesn't apply)
- This distinction is critical for correct semantics

**Implementation Challenge**: Empty arrays are invisible in normal evaluation:
```go
Event: {"items": []}
After MapObject():
  Values[address] = nil  // No elements stored

Event: {"other": "data"}
After MapObject():
  Values[address] = nil  // Array missing

// Both look identical in Values!
```

**Solution**:
1. all added to `AlwaysEvaluateCategories` (runs unconditionally)
2. `ObjectAttributeMap.OriginalEvent` stores reference to unmapped event
3. Evaluator falls back to original event to check array existence
4. Can distinguish: missing (false) vs empty (true) vs non-empty (evaluate)

### 5. ObjectAttributeMap Design

**File**: `objectmap/object_attribute_map.go`

**Core Design Decision**: `ObjectAttributeMap.Values` stores **only scalar leaf values**, not containers.

```go
type ObjectAttributeMap struct {
    DictRec       *AttrDictionaryRec
    Values        []interface{}      // ONLY scalars
    OriginalEvent interface{}        // Fallback for edge cases
}
```

**What Gets Stored**:

| Input | Stored in Values |
|-------|------------------|
| `{"age": 30}` | `Values[0] = 30` ✅ |
| `{"user": {"age": 30}}` | `Values[0] = 30` (user object NOT stored) |
| `{"items": []}` | `Values[0] = nil` (empty array NOT stored) |
| `{"items": [{"val": 10}]}` | `Values[0] = [[10]]` (array of element-values) |

**Rationale**:
1. **Memory efficiency**: Don't duplicate nested structures
2. **Fast access**: Direct array indexing by address
3. **Callback optimization**: Only trigger for "interesting" values

**Tradeoff**:
- ✅ Works perfectly for 99% of cases (non-empty structures)
- ❌ Empty containers are invisible in Values
- ✅ Solved by storing `OriginalEvent` reference (8 bytes overhead)

**Address System**:
```
Event: {"user": {"age": 30}, "items": [{"val": 10}]}

Paths and Addresses:
  user        → address [0]     (container, not stored)
  user.age    → address [0, 0]  (scalar, stored)
  items       → address [1]     (container, not stored)
  items[]     → address [1]     (element notation, same as items)
  items[0]    → address [1, 0]  (element 0)
  items[0].val → address [1, 0, 0] (scalar, stored)
```

**Critical Nuance**: `"items"` and `"items[]"` have the **same numeric address** but different semantic meaning:
- `"items"` - The array container itself
- `"items[]"` - Array elements (used in registration)

**Mapping Process**:
```
MapObject(event) → for each field:
  - Scalars → Store in Values[address], call attrCallback(address)
  - Objects → Recurse into children (object itself not stored)
  - Arrays → Store array of element-values, call attrCallback(address)
           Note: Empty arrays → nothing stored, but callback still fires!
```

### 6. Type Reconciliation

**File**: `condition/condition.go:935-997`

Operands may have different types. Reconciliation rules:

```
Numeric Types:
- int + float → both become float
- int8 + int64 → both become int64

String vs Other:
- string + int → error (no implicit conversion)
- string + string → string comparison

Boolean:
- bool is NOT converted to int/float (intentional isolation)
- bool + numeric → no conversion, direct comparison
```

### 7. Null Handling Semantics

**Critical Design Decision**: Rulestone makes **NO distinction** between missing fields and explicit null values. Both are represented internally as `NullOperand` and behave identically.

#### Why No Distinction?

**Problem**: In JSON and Go maps, there are conceptually three states:
1. Field doesn't exist in map (missing)
2. Field exists with `nil` value (explicit null)
3. Field exists with zero value (`0`, `""`, `false`)

**Rulestone's Approach**: Treat (1) and (2) **identically as null**

**Rationale**:
- **JSON Semantics**: JSON doesn't distinguish between `{"field": null}` and `{}` in most use cases
- **Rule Author Intent**: When writing `age != null`, authors typically mean "age has a value"
- **Simplicity**: No need for separate "missing" vs "null" checks in most cases
- **Performance**: Single code path for both cases

#### Implementation Details

**File**: `engine/engine_impl.go:1775-1801`

```go
func (repo *CompareCondRepo) evalOperandAccess(operand condition.Operand, ...) condition.Operand {
    return repo.CondFactory.NewExprOperand(
        func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
            val := objectmap.GetNestedAttributeByAddress(...)
            if val == nil {
                // ← Both missing and explicit null end up here
                return condition.NewNullOperand(address)
            }
            return val.(condition.Operand)
        }, operand)
}
```

**Key Point**: `GetNestedAttributeByAddress()` returns `nil` for:
- Missing fields (no key in map)
- Explicit null values (key exists, value is `nil`)

Both become `NullOperand` - **no distinction preserved**.

#### Null Comparison Semantics

**File**: `engine/engine_impl.go:226-238`

```go
if xKind == condition.NullOperandKind || yKind == condition.NullOperandKind {
    bothNull := xKind == condition.NullOperandKind && yKind == condition.NullOperandKind
    switch compOp {
    case condition.CompareEqualOp:
        return condition.NewBooleanOperand(bothNull)
    case condition.CompareNotEqualOp:
        return condition.NewBooleanOperand(!bothNull)
    case condition.CompareGreaterOp, condition.CompareLessOp, etc:
        return condition.NewBooleanOperand(false)  // null is not orderable
    }
}
```

**Null Comparison Rules** (SQL-like semantics):
- `null == null` → `true`
- `null == value` → `false` (null doesn't equal any non-null value)
- `null != value` → `true` (null is different from any non-null value)
- `null > value` → `false` (null is not orderable)
- `null < value` → `false` (null is not orderable)
- `null >= value` → `false` (null is not orderable)
- `null <= value` → `false` (null is not orderable)

#### Practical Examples

**Test File**: `tests/data/types_null_handling.yaml`

```yaml
# Example 1: Equality
expression: age == 0
Event: {}                    → false (missing field = null, null != 0)
Event: {age: null}           → false (explicit null, null != 0)
Event: {age: 0}              → true  (zero value matches)

# Example 2: Inequality
expression: age != 18
Event: {}                    → true  (missing field = null, null != 18) ← Strict mode issue!
Event: {age: null}           → true  (explicit null, null != 18)
Event: {age: 18}             → false (value matches, negation fails)
Event: {age: 25}             → true  (value doesn't match, negation succeeds)

# Example 3: Ordering
expression: age > 18
Event: {}                    → false (missing field = null, null is not orderable)
Event: {age: null}           → false (explicit null, null is not orderable)
Event: {age: 25}             → true  (25 > 18)

# Example 4: Null Check
expression: age == null
Event: {}                    → true  (missing field = null)
Event: {age: null}           → true  (explicit null)
Event: {age: 0}              → false (zero value is not null)

# Example 5: Not Null Check
expression: age != null
Event: {}                    → false (missing field = null, null != null is false)
Event: {age: null}           → false (explicit null, null != null is false)
Event: {age: 0}              → true  (zero value exists)
```

#### Functions and Null Handling

**length() Function** (`engine/engine_impl.go`):
```go
// length() returns null for missing or null arrays
expression: length("items") > 0

Event: {}                    → false (missing array → null, null > 0 → false)
Event: {items: null}         → false (explicit null → null, null > 0 → false)
Event: {items: []}           → false (empty array → 0, 0 > 0 → false)
Event: {items: [1, 2]}       → true  (array length = 2, 2 > 0 → true)
```

**hasValue() Function** - Only Way to Distinguish

```go
// hasValue() returns true if field exists and is not null
expression: hasValue("field")

Event: {}                    → false (field missing)
Event: {field: null}         → false (field explicitly null)
Event: {field: 0}            → true  (field has value, even if zero)
Event: {field: ""}           → true  (field has value, even if empty string)
```

**Design Insight**: `hasValue()` is the **only** way to check if a field is present with a non-null value, but it **still** treats missing and explicit null identically (both return false).

#### Why This Matters for Strict Mode

**Current Permissive Behavior**:
```
age != 18  with missing age → matches (null != 18 → true)
```

**Problem**: Rule triggers when field doesn't exist, which is often unintended.

**Strict Mode Goal**: Only match when field is **present**
```
age != 18  with missing age → no match (field not applicable)
```

**Implementation Challenge**: Can't just check `field == null` because:
1. Null checks need special handling (AlwaysEvaluateCategories)
2. Negative comparisons use DefaultCatList mechanism
3. No distinction between missing and explicit null at runtime

**Solution**: Disable DefaultCatList in strict mode, requiring all fields to be present for any comparison (including negations).

## Memory Management

### 1. Object Pooling

**File**: `engine/engine_api.go:177`

```go
var objectMapPool = sync.Pool{
    New: func() interface{} {
        return objectmap.NewObjectAttributeMap(nil)
    },
}

// Get from pool
oam := objectMapPool.Get().(*objectmap.ObjectAttributeMap)
oam.Init(event, mapper)

// Use...

// Return to pool
oam.Reset()
objectMapPool.Put(oam)
```

**Benefits**:
- Reduce GC pressure
- Reuse allocated memory
- Faster allocation for hot paths

### 2. Condition Deduplication

**File**: `condition/factory.go`

Factory pattern with caching:

```go
type Factory struct {
    ConditionToConditionMap *hashmap.Map[Condition, Condition]
    OperandToOperandMap     *hashmap.Map[Operand, Operand]
}

// Deduplicate identical conditions
func (f *Factory) NewAndCondition(conds []Condition) Condition {
    newCond := &AndCondition{Conditions: conds}
    if existing, ok := f.ConditionToConditionMap.Get(newCond); ok {
        return existing  // Reuse existing
    }
    f.ConditionToConditionMap.Put(newCond, newCond)
    return newCond
}
```

**Benefits**:
- Identical sub-expressions share memory
- Reduces memory footprint
- Enables pointer-based equality checks

### 3. Common Expression Elimination

**File**: `engine/engine_impl.go:111-167`

Track and eliminate duplicate evaluations:

```
Rule 1: age > 18 AND status == "active"
Rule 2: age > 18 AND country == "US"

Common: age > 18 (appears in both)

Result: Evaluate age > 18 once, reuse for both rules
```

**Implementation**:
- Hash condition to category ID
- Track category → condition mapping
- Multiple rules share category evaluator
- Single evaluation, multiple rule checks

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Rule Registration | O(n * m) | n=rules, m=avg expression nodes |
| Engine Creation | O(r * c) | r=rules, c=categories, build optimization |
| Category Evaluation | O(c) | c=unique categories in event |
| Bit Mask Check | O(1) | Per AND-term |
| String Matching | O(n + m) | Aho-Corasick, n=text, m=patterns |
| Full Rule Eval | O(k * e) | k=candidate rules, e=expression nodes |
| **Overall Match** | O(c + k*e) | Usually c >> k*e due to filtering |

### Space Complexity

| Component | Complexity | Notes |
|-----------|------------|-------|
| Rule Storage | O(r * e) | r=rules, e=expression nodes |
| Category Maps | O(c) | c=unique categories |
| Bit Masks | O(s * w) | s=catsets, w=word size |
| Object Pool | O(p) | p=pool size (bounded) |
| Event Attributes | O(a) | a=attributes in event |

### Scalability

**Tested Performance**:
- **10,000 rules**: ~2-5ms per event (single threaded)
- **1,000 rules**: ~500µs per event
- **100 rules**: ~50µs per event

**Scaling Factors**:
- Category pre-filtering provides near-constant time for large rule sets
- Linear growth only in matched rules (usually small)
- Thread-safe for concurrent event evaluation

## Design Patterns

### 1. Repository Pattern
- `RuleEngineRepo` encapsulates rule storage and management
- Clean separation of concerns
- Testable interface

### 2. Factory Pattern
- `condition.Factory` for condition/operand creation
- Centralized caching and deduplication
- Consistent object construction

### 3. Visitor Pattern
- AST traversal in expression processing
- Clean separation of tree structure and operations
- Extensible for new node types

### 4. Object Pool Pattern
- `sync.Pool` for `ObjectAttributeMap`
- Reduces allocations and GC pressure
- Automatic cleanup

### 5. Strategy Pattern
- Different evaluation strategies per operand type
- Polymorphic `Evaluate()` method
- Easy to add new operand kinds

### 6. Builder Pattern
- `cateng.BuildFilterTables` constructs optimized structures
- Step-by-step complex object creation
- Encapsulates optimization logic

## Key Algorithms

### 1. Aho-Corasick String Matching

**File**: `engine/lookups.go`

**Purpose**: Match multiple string patterns in a single pass.

**Usage**:
```
Patterns: ["error", "warning", "critical"]
Text: "This is a critical error in the system"
Result: {1: "critical", 0: "error"} (found 2 patterns)
```

**Complexity**: O(n + m + z)
- n = text length
- m = total pattern length
- z = number of matches

### 2. Bit Mask Matching

**File**: `cateng/category_engine.go:62`

**Purpose**: Fast AND-term evaluation using bitwise operations.

```go
func applyCatSetMasks(csmList []*CatSetMask, matchMaskArray []types.Mask, ...) {
    for _, csm := range csmList {
        v := matchMaskArray[csm.Index1-1]
        if v != -1 {
            newV := v | csm.Mask   // Set bits for this category
            matchMaskArray[csm.Index1-1] = newV
            if newV == -1 {        // All bits set?
                // We have a match for this AND-term!
                ...
            }
        }
    }
}
```

### 3. Common Sub-Expression Elimination

**File**: `engine/engine_impl.go:111-167`

**Purpose**: Evaluate identical expressions only once.

**Algorithm**:
1. Parse expression into AST
2. Hash each comparison node
3. Assign category ID to unique comparisons
4. Multiple rules referencing same comparison share category
5. Evaluate category once, check multiple rules

**Example**:
```
Input:
Rule 1: age > 18 && status == "active"
Rule 2: age > 18 && country == "US"
Rule 3: status == "active" && verified == true

Common Expressions:
Cat1: age > 18 (appears in Rule 1, 2)
Cat2: status == "active" (appears in Rule 1, 3)
Cat3: country == "US" (Rule 2 only)
Cat4: verified == true (Rule 3 only)

Evaluation:
1. Evaluate Cat1 once → result R1
2. Evaluate Cat2 once → result R2
3. Evaluate Cat3 once → result R3
4. Evaluate Cat4 once → result R4
5. Rule 1: R1 AND R2
6. Rule 2: R1 AND R3
7. Rule 3: R2 AND R4

Savings: 7 evaluations instead of 10 (30% reduction)
```

## Future Enhancements

### Potential Optimizations
1. **JIT Compilation**: Compile hot expressions to native code
2. **SIMD Operations**: Vectorize category evaluations
3. **Rule Clustering**: Group similar rules for better cache locality
4. **Incremental Evaluation**: Track changed attributes between events

### Feature Additions
1. **Rule Priority**: Control evaluation order
2. **Rule Dependencies**: Express prerequisites between rules
3. **Partial Matching**: Return "almost matches" with scores
4. **Rule Debugging**: Step-through evaluation with breakpoints
5. **Rule Profiling**: Identify slow rules and bottlenecks

### Architectural Improvements
1. **Plugin System**: Custom functions and operators
2. **Streaming API**: Process event streams
3. **Distributed Evaluation**: Scale across multiple machines
4. **Persistent Cache**: Save compiled rules to disk

---

## Maintainer's Guide: Common Pitfalls & Debugging

### Common Pitfalls

#### 1. Assuming Values Contains All Event Data

**Pitfall**: Checking `Values[address]` to see if attribute exists.

**Why It Fails**: Empty arrays/objects won't be in Values even if they exist in the event.

**Correct Approach**: Use the original event as the source of truth for container existence, or use the `AlwaysEvaluateCategories` pattern.

#### 2. Forgetting AlwaysEvaluateCategories

**Symptom**: Rule works when attribute is present, fails when attribute is missing.

**Root Cause**: Missing attribute → no callback → category never evaluated.

**When Needed**:
- ✅ Null checks (`field == null`)
- ✅ Constant expressions (`1 == 1`)
- ✅ Quantifiers on collections (`all`, `any`)
- ❌ Regular comparisons (`field > 10`)

**Check**: Does the rule need to evaluate when the attribute is missing?

#### 3. Confusing "items" and "items[]"

**Path Notation**:
- `"items"` - The array container (conceptual)
- `"items[]"` - Array elements (registration notation)

**Address Notation**:
- Both map to the **same numeric address**!
- Address `[1]` might represent both the container and its elements

**When to Use Which**:
- **Registration** (`registerCatEvaluatorForAddress`): Use `"items[]"` for element access
- **Navigation** (`getValueFromOriginalEvent`): Use `"items"` (strip `[]` suffix)
- **Evaluation** (`GetNumElementsAtAddress`): Address points to container

#### 4. Null vs Zero vs Missing

**Three Distinct Concepts**:
1. **Missing field**: Key doesn't exist in map
2. **Explicit null**: Key exists, value is `nil`
3. **Zero value**: Key exists, value is `0`, `""`, `false`

**Engine Behavior**:
- Missing field **is treated as null** in comparisons
- `field == null` matches both missing and explicit null
- `field == 0` only matches explicit zero, not null/missing

**Example**:
```go
Event: {}                    → field == null ✅, field == 0 ❌
Event: {field: null}         → field == null ✅, field == 0 ❌
Event: {field: 0}            → field == null ❌, field == 0 ✅
```

#### 5. CSE Side Effects

**Common Subexpression Elimination Means**:
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

### Debugging Guide

#### "My rule isn't matching!"

**Step 1: Check if category is being evaluated**
```go
// Add debug print in category evaluator
fmt.Printf("[DEBUG] Evaluating category %d for rule %s\n", cat.ID, ruleID)
```

Questions:
- Is the callback being triggered?
- Is the category in `AlwaysEvaluateCategories` if needed?
- Does the attribute path match registration?

**Step 2: Check what categories are generated**
```go
// In RuleEngine.MatchEvent, after category evaluation
fmt.Printf("[DEBUG] Event categories: %v\n", eventCategories)
```

Questions:
- Does it include the expected category ID?
- Are there unexpected categories?
- Are categories being suppressed?

**Step 3: Check category engine**
```go
// In CategoryEngine.MatchEvent
fmt.Printf("[DEBUG] Matching categories %v against %d rules\n", cats, len(rules))
```

Questions:
- Is the rule's category pattern correct?
- Are there negated categories blocking the match?
- Is the rule even registered?

**Step 4: Check attribute mapping**
```go
// In MapObject callback
fmt.Printf("[DEBUG] Found attribute at address: %v\n", addr)
```

Questions:
- Is the attribute being found in the event?
- Does the address match what you expect?
- Is the value being stored correctly?

#### "Performance is slow!"

**Profile 1: Category Evaluation Count**
```go
// Check Metrics.NumCatEvals counter
fmt.Printf("Categories evaluated per event: %d\n", engine.Metrics.NumCatEvals)
```

**Expected**: Should be roughly proportional to number of unique comparisons in rules that could match.

**Red Flag**: If this number is very high (>1000 per event), you may have too many rules or poor CSE.

**Profile 2: CSE Effectiveness**
```go
// Print CondToCompareCondRecord size
fmt.Printf("Unique categories: %d\n", len(compCondRepo.CondToCompareCondRecord))
fmt.Printf("Total rules: %d\n", len(rules))
```

**Expected**: Unique categories << (rules * avg comparisons per rule) if CSE is working.

**Red Flag**: If unique categories ≈ total comparisons, CSE isn't helping.

**Profile 3: Large Arrays**
```go
// Check for arrays with many elements
for _, rule := range rules {
    if hasQuantifier(rule) && arraySize > 1000 {
        fmt.Printf("WARNING: Rule %s has large array quantifier\n", rule.ID)
    }
}
```

**Expected**: `all`/`any` with 10-100 elements is fine.

**Red Flag**: 1000+ elements will be slow. Consider restructuring data or rules.

#### "Rule used to work, now it doesn't!"

**Check 1: Did data format change?**
- Attribute paths are case-sensitive
- Nested structure changes break rules
- Array vs scalar changes break type checks

**Check 2: Did another rule get added?**
- CSE might have changed evaluation order
- Category ID assignment might have shifted (shouldn't matter, but check)
- New rule might have conflicting logic

**Check 3: Was engine rebuilt?**
- Engine optimization parameters changed?
- Category engine threshold adjusted?
- Rule order matters for some operations

### Performance Tuning

**Benchmark Results After Optimizations**:
- Simple expression eval: ~1.9 μs/op
- Complex expression eval: ~3.2 μs/op
- Null check: ~500 ns/op
- Constant expression: ~454 ns/op
- all condition: ~2.5 μs/op
- all empty array: ~1.3 μs/op

**Key Metrics**:
- AlwaysEvaluateCategories adds ~0.5 μs overhead (negligible)
- OriginalEvent reference adds 8 bytes per ObjectAttributeMap (negligible)
- Fallback path only runs for empty arrays (rare, acceptable)

**What Scales Well**:
- Number of rules (O(1) lookup via category matching)
- Number of categories (bitmask operations)
- CSE (more rules with shared expressions → fewer evaluations)

**What Doesn't Scale**:
- Large arrays in quantifiers (O(n) iteration)
- Deep nesting (O(depth) traversal)
- Complex string operations (`regexpMatch`, `containsAny`)

### Glossary for Maintainers

- **Category**: Boolean value (true/false) representing whether a condition holds
- **Category ID**: Unique integer identifying a category (assigned during rule registration)
- **Category Evaluator**: Function that computes a category's value for an event
- **CSE**: Common Subexpression Elimination - reusing evaluation results for identical expressions
- **AttrKeys**: Attribute paths that a category depends on
- **Address**: Integer array representing path to a value in nested structure
- **FullAddress**: String representation of address with nesting information
- **Frame**: Current context in nested evaluation (used in all/any)
- **Vacuous Truth**: Logic principle where "all elements of empty set satisfy P" is true
- **AlwaysEvaluateCategories**: Categories that must run even when their attributes are missing from the event
