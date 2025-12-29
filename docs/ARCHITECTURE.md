# Rulestone Architecture

## Table of Contents
1. [System Overview](#system-overview)
2. [Core Components](#core-components)
3. [Category Engine](#category-engine)
4. [Expression Evaluation Pipeline](#expression-evaluation-pipeline)
5. [Memory Management](#memory-management)
6. [Performance Characteristics](#performance-characteristics)
7. [Design Patterns](#design-patterns)

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
- Handle forAll/forSome scoping

**Key Operations**:
- Expression parsing using Go's `go/parser` package
- AST traversal and transformation
- Type reconciliation between operands
- Scope management for array quantifiers

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
- CallExpr    → Function Calls (forAll, hasValue, etc.)
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

### 4. forAll/forSome Handling

**File**: `engine/engine_impl.go:599-758`

Quantifiers over arrays require special scoping:

```go
// Original: forAll(item, items, item.active == true)
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
4. forAll: all must be true
5. forSome: at least one must be true

### 5. Type Reconciliation

**File**: `condition/condition.go:935-997`

Operands may have different types. Reconciliation rules:

```
Numeric Types:
- int + float → both become float
- int8 + int64 → both become int64

String vs Other:
- string + int → error (no implicit conversion)
- string + string → string comparison

nil Handling:
- nil == nil → true
- nil == value → false
```

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
