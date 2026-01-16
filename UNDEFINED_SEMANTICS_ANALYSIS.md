# Undefined Semantics Analysis: Can We Eliminate DefaultCatList?

## The Critical Question

If we introduce `UndefinedOperand` to distinguish missing fields from null, what should these expressions return?

```
undefined != 0
undefined == 0
!(undefined == 0)
```

**Why This Matters**: The answer determines whether we can eliminate DefaultCatList!

---

## Two Possible Approaches

### Option A: Three-Valued Logic (SQL-like, Undefined Propagates)

**Comparison Returns**:
```
undefined != 0      → undefined (not false, not true)
undefined == 0      → undefined (not false, not true)
undefined > 0       → undefined
undefined < 0       → undefined
null != 0           → true (current behavior)
null == 0           → false (current behavior)
```

**Negation Returns**:
```
!(undefined == 0)   → !(undefined) → undefined (not true!)
!(undefined)        → undefined (undefined propagates)
```

**Category Evaluation**:
```
Category result: undefined
  → Don't add to eventCategories
  → Treated as falsey/absent
```

**Impact on Negations**:
```yaml
Rule: age != 18
Event: { name: "john" }  # age missing

Evaluation:
1. age → undefined
2. undefined == 18 → undefined
3. !(undefined) → undefined
4. Category evaluation returns undefined
5. undefined → don't add to categories
6. Rule doesn't match ✓

Result: DefaultCatList NOT NEEDED! ✅
```

**Pros**:
- ✅ **Can eliminate DefaultCatList completely**
- ✅ Aligns with SQL three-valued logic
- ✅ Correct negation semantics
- ✅ Performance improvement

**Cons**:
- ❌ Need to handle undefined throughout codebase
- ❌ More complex than binary logic

---

### Option B: Coerce to False (Binary Logic)

**Comparison Returns**:
```
undefined != 0      → false (coerce undefined to false)
undefined == 0      → false (undefined doesn't equal anything)
undefined > 0       → false
null != 0           → true (current behavior)
null == 0           → false (current behavior)
```

**Negation Returns**:
```
!(undefined == 0)   → !(false) → true
!(undefined)        → true (undefined is falsey)
```

**Category Evaluation**:
```
Category result: true (from negation)
  → Add to eventCategories
  → Rule matches
```

**Impact on Negations**:
```yaml
Rule: age != 18
Event: { name: "john" }  # age missing

Evaluation:
1. age → undefined
2. undefined == 18 → false (coerced)
3. !(false) → true
4. Category evaluation returns true
5. Category fires → Rule matches ✗

Result: **DefaultCatList STILL NEEDED** ❌
```

**Pros**:
- ✅ Simpler than three-valued logic
- ✅ Familiar to JavaScript developers

**Cons**:
- ❌ **Still need DefaultCatList** (same problem)
- ❌ Doesn't solve the core issue
- ❌ Missing vs null distinction buys us nothing

---

## How Other Systems Handle This

### SQL (Three-Valued Logic)

```sql
-- undefined/missing columns don't exist at compile time
-- NULL comparisons:
NULL != 0      → UNKNOWN (third value)
!(NULL != 0)   → !(UNKNOWN) → UNKNOWN
NULL == NULL   → UNKNOWN

-- WHERE clause interpretation:
-- Only TRUE passes, UNKNOWN and FALSE both filtered out
```

**Result**: Negations don't match NULL ✓

---

### MongoDB 8.0

```javascript
// Missing field
db.users.find({ age: { $ne: 18 } })
// Documents without 'age' field: NOT matched

// Explicit null
db.users.find({ age: { $ne: 18 } })
// Documents with age=null: MATCHED (null != 18 → true)
```

**Behavior**: Missing field → query doesn't match (implicit EXISTS check)

**Equivalent logic**:
```
undefined != 18 → (field doesn't exist) → don't match
null != 18      → true → match
```

---

### OPA Rego (Undefined Halts Evaluation)

```rego
allow {
    input.age != 18
}

# Missing age: input.age → undefined → comparison fails → rule undefined
# Explicit null: input.age → null → null != 18 → rule might succeed
```

**Behavior**: undefined **halts evaluation** (strongest distinction)

---

### CEL (Skip Evaluation for Missing)

```cel
// Missing field
age != 18  → (evaluation skipped for this path)

// Can't even compare undefined
```

**Behavior**: Missing fields cause expression to be skipped

---

### JavaScript (Coercion in Comparisons)

```javascript
let obj = { name: "john" }
obj.age != 18       → true (undefined != 18 → true due to coercion)
obj.age == 18       → false (undefined == 18 → false)
!(obj.age == 18)    → true (!(false) → true)

// Strict equality
obj.age !== 18      → true (undefined !== 18 → true)
obj.age === undefined → true
```

**Behavior**: Coerces to false-like but `!= value` returns true

**Implication**: JavaScript still has the same issue! That's why TypeScript added strict null checks.

---

## My Recommendation: Three-Valued Logic (Option A)

### Why This Works

**Undefined as a Third State**:
- TRUE: Condition met
- FALSE: Condition explicitly not met
- UNDEFINED: Condition couldn't be evaluated (field missing)

**Category Evaluation Handling**:
```go
// In MatchEvent (engine_api.go:569-614)
result := catEvaluator.Evaluate(event, FrameStack[:])
switch r := result.(type) {
case condition.BooleanOperand:
    if r {  // Only TRUE
        eventCategories = append(eventCategories, cat)
    }
case condition.UndefinedOperand:  // NEW
    // Don't add to categories (neither true nor false)
    // Treated as "not applicable"
case condition.NullOperand:
    // Don't add to categories (falsey)
}
```

**Negation Handling**:
```go
// In genEvalForNotCondition (engine_impl.go:700-716)
func (repo *CompareCondRepo) genEvalForNotCondition(...) condition.Operand {
    result := eval.Evaluate(event, frames)

    switch result.GetKind() {
    case condition.ErrorOperandKind:
        return result  // Errors propagate
    case condition.UndefinedOperandKind:  // NEW
        return result  // Undefined propagates! !(undefined) → undefined
    case condition.BooleanOperandKind:
        return condition.NewBooleanOperand(!bool(result.(condition.BooleanOperand)))
    default:
        return result  // Other types pass through
    }
}
```

**Comparison Semantics**:
```go
// In genEvalForCompareOperands (engine_impl.go:226-238)
if xKind == condition.UndefinedOperandKind || yKind == condition.UndefinedOperandKind {
    // Undefined in any operand → comparison returns undefined
    return condition.NewUndefinedOperand(nil)
}

// Existing null handling continues:
if xKind == condition.NullOperandKind || yKind == condition.NullOperandKind {
    // Current null semantics (null != value → true)
}
```

### Complete Flow with Three-Valued Logic

```yaml
Rule: age != 18
Event: { name: "john" }  # age missing

Step-by-step:
1. Evaluate age → undefined (field missing)
2. undefined == 18 → undefined (comparison with undefined)
3. NOT(undefined) → undefined (negation propagates undefined)
4. Category returns undefined
5. undefined → category doesn't fire
6. Rule doesn't match ✓

NO DefaultCatList needed!
```

**Contrast with explicit null**:
```yaml
Rule: age != 18
Event: { age: null }  # age explicitly null

Step-by-step:
1. Evaluate age → null (field exists, value null)
2. null == 18 → false (null semantics)
3. NOT(false) → true
4. Category returns true
5. Category fires → Rule matches ✓

Correct behavior!
```

---

## Answer to Your Specific Questions

> are you saying that undefined != 0 => false and so is undefined == 0 => false?

**With Three-Valued Logic**:
```
undefined != 0  → undefined (not false, not true)
undefined == 0  → undefined (not false, not true)
```

> what about !(undefined == 0)?

**With Three-Valued Logic**:
```
!(undefined == 0) → !(undefined) → undefined (not false, not true!)
```

> if the answer is still false, then we may be able to do away with the default list and negative categories

**Exactly right!** If `!(undefined == 0)` returns undefined (not true), then:
- Category evaluation returns undefined
- Undefined categories don't fire
- No need for DefaultCatList to force-fire negative categories!

**But**: This requires three-valued logic where undefined propagates through negation.

---

## Comparison: Binary vs Three-Valued Logic

### Scenario: age != 18 with missing age

**Binary Logic (coerce undefined to false)**:
```
Step 1: age → undefined
Step 2: undefined == 18 → false (coerced)
Step 3: !(false) → true
Result: Category fires → Rule matches
Conclusion: ❌ Still need DefaultCatList
```

**Three-Valued Logic (undefined propagates)**:
```
Step 1: age → undefined
Step 2: undefined == 18 → undefined (propagated)
Step 3: !(undefined) → undefined (propagated)
Result: Category doesn't fire → Rule doesn't match
Conclusion: ✅ DefaultCatList NOT NEEDED!
```

---

## Implementation Requirements for Three-Valued Logic

### 1. New Operand Type

```go
// condition/condition.go
type UndefinedOperand struct {
    Source interface{}  // For debugging: what was undefined
}

func NewUndefinedOperand(source interface{}) Operand {
    return &UndefinedOperand{Source: source}
}

func (v *UndefinedOperand) GetKind() OperandKind {
    return UndefinedOperandKind  // New constant
}
```

### 2. Undefined Propagation in Comparisons

```go
// In genEvalForCompareOperands
// Check for undefined BEFORE null
if xKind == condition.UndefinedOperandKind || yKind == condition.UndefinedOperandKind {
    return condition.NewUndefinedOperand(nil)  // Propagate
}

// Then existing null check...
```

### 3. Undefined Propagation in Logical Operators

```go
// AND operator
func genEvalForAndCondition(...) {
    // If ANY operand is undefined → result is undefined
    // (Short circuit like error propagation)
}

// OR operator
func genEvalForOrCondition(...) {
    // If one side is true → true (short circuit)
    // If all are false/undefined → false
    // If mix of false and undefined → undefined
}

// NOT operator (CRITICAL!)
func genEvalForNotCondition(...) {
    result := eval.Evaluate(event, frames)
    if result.GetKind() == condition.UndefinedOperandKind {
        return result  // !(undefined) → undefined
    }
    // ... existing boolean negation
}
```

### 4. Category Evaluation Handling

```go
// In MatchEvent (engine_api.go:569-614)
switch r := result.(type) {
case condition.UndefinedOperand:
    // Don't add to categories (not applicable)
    // This is the KEY - undefined = absent, not false
case condition.BooleanOperand:
    if r {
        eventCategories = append(eventCategories, cat)
    }
// ... rest
}
```

### 5. Remove DefaultCatList

```go
// cateng/builder.go:636-639 - DELETE or make conditional
// DELETE: for cat := range fb.NegCats { ... }

// cateng/category_engine.go:83-95 - DELETE or make conditional
// DELETE: for i, cat := range f.FilterTables.DefaultCatList { ... }
```

---

## Proof: Why DefaultCatList Becomes Unnecessary

**Current Problem**:
```
age != 18 with missing age

Without DefaultCatList:
- age missing → age == 18 doesn't fire
- NOT(category) at category engine level has no category to negate
- Result: Rule doesn't match (but we WANT it to match in permissive mode)

With DefaultCatList (current workaround):
- Detect age == 18 didn't fire
- Force-evaluate NOT(age == 18)
- Result: Rule matches ✓
```

**With Three-Valued Undefined Logic**:
```
age != 18 with missing age

Step 1: age → undefined
Step 2: undefined == 18 → undefined
Step 3: !(undefined) → undefined (NOT at evaluation level, not category level!)
Step 4: Category returns undefined (not boolean)
Step 5: undefined → don't add to categories
Step 6: Rule doesn't match ✓

No DefaultCatList needed because:
- Negation happens at EVALUATION level (before category)
- undefined propagates through negation
- Category never fires (neither true nor false)
```

**The KEY**: Negation in `!(undefined == 0)` happens **during expression evaluation**, not at the category engine level. With three-valued logic, it returns undefined, so the category never fires.

---

## Comparison with Industry

| System | undefined != 0 | undefined == 0 | !(undefined == 0) | Negations Match Missing? |
|--------|----------------|----------------|-------------------|-------------------------|
| **SQL** | UNKNOWN | UNKNOWN | UNKNOWN | NO ✓ |
| **OPA Rego** | undefined (halt) | undefined (halt) | undefined (halt) | NO ✓ |
| **CEL** | (skipped) | (skipped) | (skipped) | NO ✓ |
| **MongoDB 8.0** | (not matched) | (not matched) | (not matched) | NO ✓ |
| **JavaScript** | true | false | true | YES ✗ |
| **Proposed Option A** | undefined | undefined | undefined | NO ✓ |
| **Proposed Option B** | false | false | true | YES ✗ |

**Conclusion**: Industry uses three-valued logic or equivalent to make negations NOT match missing fields.

---

## Recommended Implementation: Three-Valued Logic

### Core Semantics

**UndefinedOperand Behavior**:
```go
// All comparisons with undefined → undefined
undefined == value    → undefined
undefined != value    → undefined
undefined > value     → undefined
undefined < value     → undefined

// Logical operations
!(undefined)          → undefined
undefined && true     → undefined
undefined || true     → true (short circuit)
undefined || false    → undefined
true && undefined     → undefined
false && undefined    → false (short circuit)

// Special cases
undefined == undefined → undefined (or true? debatable)
undefined == null      → false (different types)
```

### Conversion Rules

**From Undefined**:
```go
func (v UndefinedOperand) Convert(to OperandKind) Operand {
    switch to {
    case UndefinedOperandKind:
        return v
    case ErrorOperandKind:
        return NewErrorOperand(fmt.Errorf("undefined value"))
    default:
        return NewUndefinedOperand(nil)  // Propagate undefined
    }
}
```

**To Undefined**: No conversions from other types

### Category Result Handling

```go
// In MatchEvent
switch r := result.(type) {
case condition.ErrorOperand:
    // Log error, don't add category
case condition.BooleanOperand:
    if r {
        eventCategories = append(eventCategories, cat)
    }
case condition.UndefinedOperand:  // NEW!
    // Don't add category (not applicable)
case condition.NullOperand:
    // Don't add category (falsey)
// ... rest
}
```

---

## Migration Strategy

### Phase 1: Introduce Undefined (v1.1, Opt-In)

```go
// Default: Current behavior (missing=null)
repo.LoadRules(reader)

// Opt-in to new semantics
repo.LoadRules(reader, engine.WithDistinguishMissingFromNull(true))
```

**Implementation**:
- Add UndefinedOperand type
- Add flag to control behavior
- Keep DefaultCatList when flag = false
- Disable DefaultCatList when flag = true

### Phase 2: Deprecation Notice (v1.2)

- Add deprecation warning for old semantics
- Documentation emphasizes new approach
- Community feedback period

### Phase 3: Make Default (v2.0)

```go
// v2.0: Distinguish by default
repo.LoadRules(reader)  // Uses undefined semantics

// Opt-in to OLD behavior if needed
repo.LoadRules(reader, engine.WithTreatMissingAsNull(true))
```

---

## Test Cases to Verify Semantics

### Test 1: Undefined in Comparisons

```go
func TestUndefinedComparisons(t *testing.T) {
    // With WithDistinguishMissingFromNull(true)

    event := map[string]interface{}{ "name": "john" }  // age missing

    // undefined != 0
    rule := "age != 0"
    expect: NO MATCH (undefined != 0 → undefined → category doesn't fire)

    // undefined == 0
    rule = "age == 0"
    expect: NO MATCH (undefined == 0 → undefined → category doesn't fire)

    // undefined > 0
    rule = "age > 0"
    expect: NO MATCH (undefined > 0 → undefined → category doesn't fire)
}
```

### Test 2: Negation Propagation

```go
func TestNegationWithUndefined(t *testing.T) {
    event := map[string]interface{}{ "name": "john" }

    // !(undefined == 0)
    rule := "!(age == 0)"
    expect: NO MATCH (!(undefined) → undefined → category doesn't fire)

    // Equivalent to age != 0
    rule := "age != 0"
    expect: NO MATCH (same as above)
}
```

### Test 3: Null Still Works

```go
func TestNullWithUndefined(t *testing.T) {
    event := map[string]interface{}{ "age": nil }  // Explicit null

    // null != 0
    rule := "age != 0"
    expect: MATCH (null != 0 → true → category fires)

    // null == 0
    rule := "age == 0"
    expect: NO MATCH (null == 0 → false)
}
```

### Test 4: Undefined vs Null Checks

```go
func TestCheckingForMissingVsNull(t *testing.T) {
    // New functions/checks needed

    event1 := map[string]interface{}{ "name": "john" }  // age missing
    rule := "!hasField('age')"
    expect: MATCH (field doesn't exist)

    event2 := map[string]interface{}{ "age": nil }  // age null
    rule := "age == null"
    expect: MATCH (explicit null check)

    rule := "!hasField('age')"
    expect: NO MATCH (field exists, just null)
}
```

---

## Answer to Your Question

> are you saying that undefined != 0 => false and so is undefined == 0 => false? what about !(undefined == 0)? if the answer is still false, then we may be able to do away with the default list and negative categories, but otherwise not.

**My Answer**:

**With Three-Valued Logic (RECOMMENDED)**:
```
undefined != 0      → undefined (neither true nor false)
undefined == 0      → undefined (neither true nor false)
!(undefined == 0)   → undefined (not false, not true!)
```

Result: **YES, we can eliminate DefaultCatList!** ✓

**With Binary Logic (JavaScript-style)**:
```
undefined != 0      → false (coerced)
undefined == 0      → false (coerced)
!(undefined == 0)   → !(false) → true
```

Result: **NO, still need DefaultCatList** ✗

---

## What the Recent Commits Tell Us

Looking at the `length()` function implementation (commit cefd34c):

```
length("missing") > 0 → false (null > 0 is false)
length("missing") != 0 → true (null != 0 is true)
```

This shows current semantics treat missing as null, and `null != 0` returns **true**.

**With three-valued undefined**:
```
length("missing") > 0 → undefined > 0 → undefined → false (doesn't match)
length("missing") != 0 → undefined != 0 → undefined → false (doesn't match!)
```

**Implication**: `length("items") != 0` would **NOT match** when items is missing. This might be a breaking change users need to adapt to.

**Migration**:
```yaml
# Old (matches missing):
expression: length("items") != 0

# New (only matches non-empty arrays):
expression: length("items") != 0

# To match old behavior (missing or non-empty):
expression: length("items") != 0 || !hasField("items")
```

---

## Final Recommendation

**Use three-valued logic with undefined propagation**:

1. ✅ **Can eliminate DefaultCatList** (performance win)
2. ✅ Aligns with SQL, Rego, CEL industry standards
3. ✅ Correct semantics for negative comparisons
4. ✅ More expressive rule language
5. ❌ Breaking change (needs migration)

**Proposed API**:
```go
// v1.1: Opt-in
engine.WithDistinguishMissingFromNull(true)

// v2.0: Default behavior
// (with opt-out via WithTreatMissingAsNull(true) if needed)
```

**Next Steps**:
1. Confirm three-valued logic approach
2. Design UndefinedOperand API
3. Plan migration for length() and other functions
4. Implement with comprehensive tests
5. Document breaking changes clearly
