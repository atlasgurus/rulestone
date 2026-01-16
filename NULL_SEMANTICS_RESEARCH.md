# NULL Semantics Research: Industry Comparison & Design Analysis

## Executive Summary

This document compares how major rule engines and data systems handle the distinction between **missing fields** and **explicit null values**, and proposes potential approaches for rulestone.

**Key Finding**: The industry trend is **moving toward distinguishing** missing from null, with MongoDB making a breaking change in version 8.0 to enforce this distinction.

---

## Industry Survey: How Others Handle Missing vs Null

### 1. SQL - Three-Valued Logic (The Standard)

**Approach**: NULL is a special value representing "unknown" - creates three-valued logic (TRUE, FALSE, UNKNOWN)

**Semantics**:
```sql
-- NULL comparisons always return UNKNOWN (not TRUE or FALSE)
NULL = NULL     → UNKNOWN (not TRUE!)
NULL = 5        → UNKNOWN
NULL != 5       → UNKNOWN (not TRUE!)
NULL > 5        → UNKNOWN

-- WHERE clause only accepts TRUE
SELECT * FROM users WHERE age > 18;
-- If age is NULL, UNKNOWN is treated as FALSE (row excluded)

-- Special operators for NULL
age IS NULL      → TRUE/FALSE (binary result)
age IS NOT NULL  → TRUE/FALSE (binary result)
```

**Impact on Negative Comparisons**:
```sql
-- Critically: != NULL does NOT work
SELECT * FROM users WHERE age != 18;
-- If age is NULL → UNKNOWN → row EXCLUDED (not included!)

-- Correct way:
SELECT * FROM users WHERE age != 18 OR age IS NULL;
```

**Missing Columns**: SQL doesn't have "missing columns" - columns either exist in schema or don't compile.

**Sources**:
- [SQL NULL Values - W3Schools](https://www.w3schools.com/sql/sql_null_values.asp)
- [SQL Three-Valued Logic - Wikibooks](https://en.wikibooks.org/wiki/Structured_Query_Language/NULLs_and_the_Three_Valued_Logic)
- [SQL NULL Comparison Behavior](https://sqlpey.com/sql/sql-null-comparison-behavior/)

---

### 2. MongoDB - Recent Breaking Change (v8.0)

**Critical Update**: MongoDB **changed behavior in version 8.0** to distinguish undefined from null!

**Pre-8.0 Behavior** (Permissive):
```javascript
db.collection.find({ field: null })
// Returns: Documents where field is null OR missing
// Rationale: Convenience for users
```

**8.0+ Behavior** (Strict):
```javascript
db.collection.find({ field: null })
// Returns: Documents where field is ONLY null (not missing!)
// Missing fields must use: { field: { $exists: false } }
```

**Explicit Distinction**:
```javascript
// Query for missing field
{ field: { $exists: false } }

// Query for explicit null
{ field: null, field: { $exists: true } }

// Query for null OR missing (old behavior)
{ $or: [{ field: null }, { field: { $exists: false } }] }
```

**Rationale for Change**: Better align with user expectations and reduce ambiguity.

**Sources**:
- [MongoDB Query for Null or Missing Fields](https://www.mongodb.com/docs/manual/tutorial/query-for-null-fields/)
- [MongoDB Migrate Undefined Data](https://www.mongodb.com/docs/manual/reference/bson-types/migrate-undefined/)
- [MongoDB Null Handling Best Practices - MyDBOps](https://www.mydbops.com/blog/null-handling-in-mongodb)

**Key Insight**: MongoDB's breaking change shows the industry recognizes that **treating missing and null as identical is problematic**.

---

### 3. OPA Rego - Strong Undefined Semantics

**Approach**: **undefined** is a first-class concept, completely different from **null**

**Behavior**:
```rego
# Missing field
allow { input.user.age > 18 }
# If input.user.age doesn't exist → undefined → rule fails (allow remains undefined)

# Explicit null
input = { "user": { "age": null } }
allow { input.user.age > 18 }
# null > 18 → comparison fails → rule fails

# Default handling
default allow = false  # If allow is undefined, use false
```

**Critical Difference**:
- **Missing field**: Produces `undefined`, halts evaluation
- **Explicit null**: Produces `null`, comparison proceeds (usually fails)
- **Can NOT pass undefined to functions**: It's not a value, it's absence of value

**Checking for Missing Fields**:
```rego
# Check if field exists
has_age { input.user.age }  # True if field exists (even if null)

# Check if field is not null
is_not_null { not is_null(input.user.age) }
```

**Sources**:
- [OPA Issue #1241: Evaluate non existing entry](https://github.com/open-policy-agent/opa/issues/1241)
- [OPA Issue #5211: Allow undefined to be passed to function](https://github.com/open-policy-agent/opa/issues/5211)
- [Rego Keyword: default - Styra Docs](https://docs.styra.com/opa/rego-by-example/keywords/default)
- [Policy Language - OPA](https://www.openpolicyagent.org/docs/policy-language)

---

### 4. CEL (Common Expression Language - Google)

**Approach**: Missing fields **skip evaluation**, null is explicit

**Behavior**:
```cel
// Missing field
message.user.age > 18
// If message.user is missing → evaluation skipped for this path

// Explicit null
message.user == null  // Can compare to null

// Field presence check
has(message.user.age)  // Returns true if field exists
```

**Key Design**:
- **Missing fields**: Evaluation is skipped (not an error)
- **Null**: Explicit value with its own type (null_type)
- **Proto-based defaults**: `has()` distinguishes default values from set values

**Sources**:
- [CEL Language Definition - Google](https://github.com/google/cel-spec/blob/master/doc/langdef.md)
- [CEL Overview](https://cel.dev/)
- [Firebase CEL Reference](https://firebase.google.com/docs/data-connect/cel-reference)

---

### 5. json-rules-engine (JavaScript)

**Approach**: Missing fields are **falsey**, but configurable

**Default Behavior**:
```javascript
// Rule: age > 18
// Fact: { name: "john" }  // age missing

// Result: Rule does NOT trigger (undefined is falsey)
```

**allowUndefinedFacts Option**:
```javascript
let engine = new Engine(undefined, { allowUndefinedFacts: true })
// Now undefined facts are explicitly undefined (not automatic false)
```

**Explicit Undefined**:
```javascript
// Throws error: "facts must have a value or method"
facts.age = undefined

// Allowed
facts.age = null  // Can trigger events
```

**Custom Operators**: Can create operators like `isUndefinedOrNull` to check both

**Sources**:
- [json-rules-engine Issue #215: undefined facts](https://github.com/CacheControl/json-rules-engine/issues/215)
- [json-rules-engine Issue #111: Dealing with undefined facts](https://github.com/CacheControl/json-rules-engine/issues/111)
- [json-rules-engine npm docs](https://www.npmjs.com/package/json-rules-engine/v/1.0.0-beta9)

---

### 6. Drools (JBoss Rules Engine)

**Approach**: Java-based, follows Java null semantics with special operators

**Null-Safe Dereferencing**:
```drools
// Operator !. for null-safe access
person!.address!.city == "NYC"  // Won't NPE if person or address is null

// Explicit null checks required
person != null && person.age > 18
```

**Comparison Behavior**:
- `matches` against null → always false
- `not matches` against null → always true
- Must use explicit null guards to avoid NullPointerException

**Missing Fields**: Not applicable - Java objects have defined schemas

**Sources**:
- [Drools Language Reference](https://docs.drools.org/latest/drools-docs/drools/language-reference/index.html)
- [Red Hat BRMS Documentation](https://access.redhat.com/solutions/389723)

---

### 7. JavaScript/TypeScript Language Semantics

**Strong Distinction**:
```javascript
// undefined: Property doesn't exist
const obj = { name: "john" }
obj.age  // → undefined

// null: Explicitly set to empty
const obj2 = { age: null }
obj2.age  // → null

// Comparison
undefined == null   // → true (loose equality)
undefined === null  // → false (strict equality)
```

**TypeScript Best Practices**:
- **undefined**: For optional/missing properties
- **null**: For intentional absence
- **Optional chaining**: `obj?.user?.age` safely handles missing
- **Nullish coalescing**: `age ?? 18` defaults on null/undefined

**Recommendation from TS community**: Use optional properties (`age?: number`) rather than explicit null (`age: number | null`)

**Sources**:
- [TypeScript Optional Properties - Better Stack](https://betterstack.com/community/guides/scaling-nodejs/typescript-optional-properties/)
- [TypeScript Null vs Undefined - W3Schools](https://www.w3schools.com/typescript/typescript_null.php)
- [Null vs Undefined Deep Dive](https://basarat.gitbook.io/typescript/recap/null-undefined)
- [Microsoft TypeScript Issue #9653](https://github.com/microsoft/TypeScript/issues/9653)

---

## Comparison Matrix

| System | Missing Field | Explicit Null | Comparison null != value | Can Distinguish? |
|--------|---------------|---------------|------------------------|------------------|
| **SQL** | N/A (schema) | UNKNOWN | UNKNOWN (excluded) | N/A |
| **MongoDB 8.0+** | Not matched | Matched | Not matched | ✅ YES ($exists) |
| **MongoDB <8.0** | Matched | Matched | Matched | ❌ NO (breaking change!) |
| **OPA Rego** | undefined (halts) | null (proceeds) | Depends on logic | ✅ YES (strongly) |
| **CEL** | Skips evaluation | null value | Depends on context | ✅ YES (has()) |
| **json-rules-engine** | Falsey | null value | Depends on config | ✅ YES (conditional) |
| **Drools** | N/A (Java) | null (NPE risk) | Requires null guards | N/A |
| **JavaScript/TS** | undefined | null | true (!=) | ✅ YES (===) |
| **Rulestone Current** | NullOperand | NullOperand | true (!=) | ❌ NO |

---

## Critical Insights

### 1. Industry Consensus: Distinction is Important

**Evidence**:
- **MongoDB made a BREAKING CHANGE in 8.0** to distinguish them
- OPA Rego treats undefined as a halt condition
- CEL skips evaluation for missing fields
- TypeScript best practices recommend optional properties over null

**Rationale**:
- Missing = "not provided/don't know"
- Null = "explicitly empty/known absence"
- Semantic difference matters for business logic

### 2. The Negative Comparison Problem

**Every system that treats missing=null has the same issue**:

```
Rule: age != 18
Event: { name: "john" }  // age missing

Question: Should this match?

SQL Answer: NO (NULL != 18 → UNKNOWN → excluded from WHERE)
MongoDB 8.0 Answer: NO (field doesn't exist)
OPA Rego Answer: NO (undefined halts evaluation)
Rulestone Current Answer: YES (null != 18 → true) ← OUTLIER!
```

**Rulestone's DefaultCatList is a workaround** to make negations work, but it creates the opposite problem!

### 3. Performance Implications

**Current Rulestone (missing=null)**:
- ❌ Must evaluate negative categories even when fields missing (DefaultCatList)
- ❌ Performance overhead for negative comparisons
- ❌ Memory overhead for DefaultCatList

**If we distinguish missing from null**:
- ✅ No need for DefaultCatList mechanism
- ✅ Missing field → rule doesn't apply → skip evaluation
- ✅ Better performance AND correct semantics!

---

## Design Space Analysis

### Dimension 1: Missing vs Null Handling

**Option A: Current Behavior (missing = null)**
```yaml
Event: {}
Event: {age: null}
Both treated identically as NullOperand
```

**Pros**:
- Simpler mental model
- JSON-esque (JSON doesn't encode undefined)
- Single code path

**Cons**:
- Industry outlier
- Negative comparisons match when fields missing (unintuitive)
- Requires DefaultCatList workaround (performance cost)

**Option B: Distinguish Missing from Null**
```yaml
Event: {}              → UndefinedOperand (new type)
Event: {age: null}     → NullOperand
Different behavior
```

**Pros**:
- Matches industry best practices
- Negative comparisons work naturally (no DefaultCatList needed)
- Performance improvement
- More expressive rules (can check for missing vs null)

**Cons**:
- Breaking change for existing users
- More complex mental model
- Need new operand type (UndefinedOperand)

### Dimension 2: Strict Mode (DefaultCatList)

**Option A: Disable DefaultCatList**
```yaml
strictMode: true
age != 18 with missing age → No match
```

**Option B: Keep DefaultCatList, Add Stricter Null Semantics**
```yaml
strictMode: true + distinguish missing from null
age != 18 with missing age → No match (field undefined)
age != 18 with null age → Matches (null != 18)
```

### Dimension 3: Configuration Granularity

**Option A: Single Flag (strictMode)**
```go
WithStrictMode(true)
// Controls both DefaultCatList and null behavior
```

**Option B: Separate Flags**
```go
WithDistinguishMissingFromNull(true)  // Controls undefined vs null
WithStrictNegations(true)             // Controls DefaultCatList
```

**Option C: Null Semantics Modes**
```go
WithNullSemantics("sql")        // NULL=NULL → UNKNOWN, missing treated as NULL
WithNullSemantics("javascript") // undefined ≠ null, both distinct
WithNullSemantics("permissive") // Current behavior (missing=null)
```

---

## Recommended Approaches (Three Alternatives)

### Approach 1: Minimal Change - Just Disable DefaultCatList (Quick Win)

**What Changes**:
```go
WithStrictNegativeCategories(false)  // Disable DefaultCatList in strict mode
```

**Behavior**:
```
Permissive (default):
  age != 18 with missing age → matches (current behavior)

Strict:
  age != 18 with missing age → no match
  age != null with missing age → no match (because no DefaultCatList)
```

**Pros**:
- ✅ Smallest change (2 lines of code)
- ✅ Solves immediate problem
- ✅ Backward compatible
- ✅ Performance improvement in strict mode

**Cons**:
- ❌ Doesn't address underlying missing=null issue
- ❌ `age != null` doesn't work as expected in strict mode
- ❌ Still an industry outlier

**Files Modified**: 2 files (cateng/builder.go, cateng/category_engine.go)

---

### Approach 2: Introduce UndefinedOperand (Industry Alignment)

**What Changes**:
Add a new operand type to distinguish missing from null:
```go
type UndefinedOperand struct{}  // Represents missing field

// In evalOperandAccess:
if val == nil {
    // Check if field exists in original event
    if fieldExistsInOriginalEvent(path) {
        return condition.NewNullOperand(address)  // Explicit null
    } else {
        return condition.NewUndefinedOperand(address)  // Missing
    }
}
```

**Comparison Semantics**:
```
undefined == null    → false (different types)
undefined != 18      → undefined (not true!)
null != 18           → true
undefined == undefined → true
```

**DefaultCatList Impact**:
- Can REMOVE DefaultCatList entirely!
- undefined != value → undefined → rule doesn't match
- No need for special-casing

**Pros**:
- ✅ Aligns with industry (MongoDB 8.0, Rego, CEL, TypeScript)
- ✅ Eliminates need for DefaultCatList (performance win!)
- ✅ More expressive (can check for missing vs null)
- ✅ Correct negative comparison semantics
- ✅ Can add `hasValue()` or similar functions

**Cons**:
- ❌ **BREAKING CHANGE** for existing users
- ❌ More complex implementation
- ❌ Need to handle undefined in all operators
- ❌ Migration guide needed

**Files Modified**: 6+ files (condition/condition.go, engine/engine_impl.go, etc.)

---

### Approach 3: Configurable Semantics (Maximum Flexibility)

**What Changes**:
Separate flags for each concern:

```go
repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRules(reader,
    // Control null semantics
    engine.WithNullBehavior(engine.NullBehavior{
        TreatMissingAsNull: false,  // Distinguish missing from null
        ThreeValuedLogic: false,     // undefined propagates vs converts to false
    }),

    // Control negative category behavior
    engine.WithNegativeCategoryBehavior(engine.NegativeBehavior{
        EnableDefaultCatList: false,  // Strict: don't match on missing
    }),
)
```

**Possible Configurations**:

| Config | missing=null? | DefaultCatList? | age != 18 with missing age |
|--------|---------------|-----------------|---------------------------|
| **Permissive (current)** | Yes | Yes | Matches |
| **Strict + unified null** | Yes | No | No match |
| **Distinguish + permissive** | No | Yes | Matches (weird!) |
| **Distinguish + strict** | No | No | No match |

**Pros**:
- ✅ Maximum flexibility
- ✅ Can migrate gradually (start with strict mode, later add distinction)
- ✅ Backward compatible (defaults to current)
- ✅ Each concern controlled independently

**Cons**:
- ❌ Most complex API
- ❌ Confusing combinations possible
- ❌ Higher maintenance burden
- ❌ Harder to document

**Files Modified**: 8+ files

---

## In-Depth Analysis: Why Distinguish Missing from Null?

### Semantic Clarity

**Missing Field** (undefined):
- "I don't know"
- "Not applicable"
- "User didn't provide this information"
- Example: Optional survey question not answered

**Explicit Null** (null):
- "I know it's empty"
- "Intentionally cleared"
- "User explicitly said 'none'"
- Example: User cleared their phone number field

### Real-World Example

**E-commerce Risk Rules**:
```yaml
# Rule: Flag high-risk orders
expression: previous_order_count != 0

Scenario 1: New customer (never ordered before)
Event: { user_id: 123, name: "John" }  # previous_order_count missing

Current Rulestone: MATCHES (null != 0 → true) → Flagged as risky!
Should it? NO - new customer is different from risky customer

Scenario 2: Known customer with zero orders (account exists but inactive)
Event: { user_id: 123, previous_order_count: 0 }

Current Rulestone: No match (0 != 0 → false) ✓ Correct

Scenario 3: Data explicitly nulled (anomaly/data issue)
Event: { user_id: 123, previous_order_count: null }

Current Rulestone: MATCHES (null != 0 → true)
Should it? Maybe - this is an anomaly worth flagging
```

**With Distinction**:
```yaml
# Check for risky known customers
expression: previous_order_count != null && previous_order_count != 0

# Check for new customers (missing data)
expression: previous_order_count == undefined

# Check for data anomalies
expression: previous_order_count == null
```

### API Evolution Example

**Versioned APIs**:
```yaml
# v1 API: Didn't have 'email_verified' field
Event: { user_id: 123, email: "test@example.com" }

# v2 API: Added 'email_verified' field
Event: { user_id: 123, email: "test@example.com", email_verified: false }

# Rule: Require email verification
expression: email_verified == true

v1 Event (missing field):
  - Current: false (null == true → false) ✓ Works by accident
  - With distinction: undefined == true → undefined → no match ✓ Explicit

# Better rule with distinction:
expression: hasField("email_verified") && email_verified == true
```

---

## MongoDB's Breaking Change - Lessons Learned

### What MongoDB Changed (v8.0)

**Before 8.0**:
```javascript
db.users.find({ age: null })
// Returned: Documents with age=null AND documents without age field
// Rationale: "Convenient for users"
```

**After 8.0**:
```javascript
db.users.find({ age: null })
// Returns: ONLY documents with age explicitly null
// Missing field: Must use { age: { $exists: false } }
// Rationale: "Reduce ambiguity, align with user expectations"
```

### Why They Made This Breaking Change

From MongoDB documentation and community feedback:
1. **User confusion**: Hard to reason about query results
2. **Unintended matches**: Queries matched more than expected
3. **Performance**: Could optimize better with distinction
4. **Correctness**: Aligns with JSON/JavaScript semantics

### Their Migration Strategy

1. **Clear documentation** of the change
2. **Migration guide** showing how to update queries
3. **Deprecation warnings** in v7.x
4. **Backward compatibility option** (`$or` pattern for old behavior)

### What We Can Learn

- ✅ Even established systems change semantics when they're wrong
- ✅ Breaking changes are acceptable if well-communicated
- ✅ Provide migration path and backward compatibility
- ✅ Industry is moving toward distinction

---

## Design Recommendations

### Short-Term (Minimal Risk)

**Approach 1: Add StrictMode flag**
```go
WithStrictMode(true)  // Disables DefaultCatList only
```

**Behavior**:
- `age != 18` with missing age → no match
- Still treats missing=null internally
- Backward compatible (default false)

**Rationale**:
- Solves immediate user pain point
- Low implementation risk
- Buys time for deeper null semantics work

---

### Long-Term (Industry Alignment)

**Approach 2: Introduce UndefinedOperand (Phased)**

**Phase 1: Add operand type (opt-in)**
```go
WithDistinguishMissingFromNull(true)  // Opt-in to new semantics
```

**Phase 2: Make it default in v2.0**
```go
// v2.0: Distinguish by default
WithTreatMissingAsNull(true)  // Opt-in to OLD behavior
```

**Rationale**:
- Aligns with MongoDB, Rego, CEL, TypeScript
- Eliminates DefaultCatList (performance win!)
- More expressive rule language
- Future-proof

---

### Ultra-Flexible (Maximum Control)

**Approach 3: Orthogonal flags**

```go
repo.LoadRules(reader,
    // Null semantics
    engine.WithTreatMissingAsNull(false),  // false = distinguish

    // Negative category behavior
    engine.WithDefaultCatList(false),  // false = strict negations
)
```

**Four Combinations**:

| TreatMissingAsNull | DefaultCatList | Behavior |
|--------------------|----------------|----------|
| true | true | **Current** (missing=null, negations match) |
| true | false | **Strict mode v1** (missing=null, negations don't match) |
| false | true | **Weird** (missing≠null, but negations still match??) |
| false | false | **Strict mode v2** (missing≠null, negations only on present) |

**Rationale**:
- Maximum flexibility
- Can migrate incrementally
- Each concern controlled independently

**Concerns**:
- Complex API
- Some combinations don't make sense
- Harder to reason about

---

## Open Questions for Design Discussion

### 1. Breaking Change Appetite

**Question**: Are we willing to make a breaking change to align with industry?

**Options**:
- **A**: No breaking changes (Approach 1 - just add StrictMode flag)
- **B**: Breaking change in major version (Approach 2 - introduce UndefinedOperand)
- **C**: Opt-in new behavior, deprecate old (Approach 2 phased)

### 2. Null Check Semantics in Strict Mode

**Question**: Should `age == null` match missing fields in strict mode?

**Current**:
```
age == null
  Event: {}          → matches (missing=null)
  Event: {age: null} → matches (explicit null)
```

**If we distinguish**:
```
age == null
  Event: {}          → ??? What should happen?
  Event: {age: null} → matches

Options:
  A) No match (only explicit null)
  B) Still match (null check is special)
  C) Add separate check: age == undefined
```

### 3. Migration Path

**Question**: How do users migrate existing rules?

**If we distinguish missing from null**:
```yaml
# Old rule
expression: age != 18

# New behavior: Doesn't match missing age
# Migration options:
  A) expression: (age != undefined && age != 18)  # Only non-missing
  B) expression: (age != 18 || age == undefined)  # Old behavior
  C) Auto-migration tool rewrites rules
```

### 4. hasValue() Function

**Question**: What should `hasValue()` return?

**Current**:
```
hasValue("age")
  Event: {}          → false
  Event: {age: null} → false (null is not a "value")
  Event: {age: 0}    → true
```

**If we distinguish**:
```
hasField("age")  // New function
  Event: {}          → false
  Event: {age: null} → true  (field exists, even if null)
  Event: {age: 0}    → true

hasValue("age")  // Existing function
  Event: {}          → false
  Event: {age: null} → false (same as current)
  Event: {age: 0}    → true
```

### 5. forAll/forSome with Missing Arrays

**Current**:
```
forAll("items", "item", item.active == true)
  Event: {}              → false (missing array)
  Event: {items: null}   → false (null array, same as missing)
  Event: {items: []}     → true  (vacuous truth)
  Event: {items: [{active: true}]} → true
```

**If we distinguish**:
```
forAll("items", "item", item.active == true)
  Event: {}              → undefined (field doesn't exist)
  Event: {items: null}   → null or error (can't iterate null)
  Event: {items: []}     → true  (vacuous truth)
```

**Question**: Should missing array and null array behave differently?

---

## Performance Analysis

### Current Implementation Costs

**DefaultCatList Processing** (cateng/category_engine.go:83-95):
```go
// Runs for EVERY event with negative categories
for i, cat := range f.FilterTables.DefaultCatList {
    if !defaultCatMap[i] {
        // Process negative category
    }
}
```

**Cost**:
- Loop through all default categories (could be hundreds)
- Check defaultCatMap for each
- Look up negative category
- Process catset masks

**Estimated Overhead**: ~5-10% for rule sets with many negations

### With UndefinedOperand

**No DefaultCatList Needed**:
```
age != 18
  ↓ Category evaluation
  age is undefined → comparison returns undefined → category false
  No special processing needed!
```

**Performance Win**:
- ✅ Eliminate DefaultCatList construction
- ✅ Eliminate DefaultCatList processing loop
- ✅ Fewer category evaluations
- ✅ Simpler mental model

**Estimated Improvement**: ~5-10% for rule sets with many negations

---

## Test Coverage Impact

### If We Distinguish Missing from Null

**Current Test Suite**: `tests/data/types_null_handling.yaml` (52 tests)
- All assume missing = null
- **Would need updates**

**Migration Strategy**:
1. Add new test file: `types_undefined_handling.yaml`
2. Update `types_null_handling.yaml` to test explicit null
3. Add tests for undefined behavior
4. Add backward compatibility tests

**Estimated**: ~100 additional test cases

---

## Comparison with Rulestone's Current Design

### Where Rulestone Differs

| Aspect | Industry Norm | Rulestone Current | Alignment |
|--------|---------------|-------------------|-----------|
| Missing vs Null | Distinguished | Same (NullOperand) | ❌ Outlier |
| Negative comparisons | Don't match missing | Match missing | ❌ Outlier |
| Performance | Skip missing fields | Evaluate DefaultCatList | ❌ Overhead |
| null != value | false/undefined | true | ❌ Different |
| IS NULL operator | Special operator | field == null | ✅ Similar |

### Where Rulestone Excels

- ✅ Clear syntax (Go-like expressions)
- ✅ High performance (category engine)
- ✅ Common subexpression elimination
- ✅ Rich function library
- ✅ Comprehensive test suite

---

## Recommended Decision Framework

### Priority 1: What Do Users Need Most?

**User Story**: "I want `age != 18` to NOT match when age is missing"

**Solutions**:
- Quick: Approach 1 (StrictMode disables DefaultCatList)
- Complete: Approach 2 (UndefinedOperand)
- Flexible: Approach 3 (Configurable)

### Priority 2: Industry Alignment vs Stability

**Industry Trend**: Distinguish missing from null (MongoDB 8.0 evidence)

**Trade-off**:
- Align with industry → Breaking change risk
- Maintain current → Remain outlier

### Priority 3: Performance vs Complexity

**Simplest**: Keep current, add StrictMode flag
**Fastest**: Introduce UndefinedOperand (remove DefaultCatList)
**Most Flexible**: Configurable semantics

---

## Proposed Decision Tree

```
START: Should we change null semantics?
  │
  ├─ NO → Just add StrictMode flag (Approach 1)
  │     → Quick win, minimal risk
  │     → Technical debt remains
  │
  └─ YES → Are we OK with breaking changes?
        │
        ├─ NO → Make it opt-in (Approach 2 phased)
        │     → WithDistinguishMissingFromNull(true)
        │     → Deprecate old behavior
        │     → v2.0: Make it default
        │
        └─ YES → Introduce UndefinedOperand now (Approach 2)
              → Major version bump
              → Migration guide
              → Align with industry
```

---

## Recommendation

**My recommendation: Approach 2 (Phased)**

**Phase 1 (v1.1)**:
```go
// Opt-in to new semantics
WithDistinguishMissingFromNull(true)
```

**Phase 2 (v2.0)**:
```go
// Default behavior changed
// Opt-in to OLD behavior if needed
WithTreatMissingAsNull(true)
```

**Rationale**:
1. ✅ Aligns with industry (MongoDB, Rego, CEL, TypeScript)
2. ✅ Eliminates DefaultCatList (performance win)
3. ✅ More correct semantics for negative comparisons
4. ✅ Gives users time to migrate
5. ✅ Future-proof design

---

## Next Steps

1. **Decide on approach** (1, 2, or 3)
2. **Design the API** (flag names, behavior)
3. **Plan migration** (if breaking change)
4. **Implement with tests**
5. **Document thoroughly**
6. **Gather user feedback**

---

## Sources

### Rule Engines
- [Drools Language Reference](https://docs.drools.org/latest/drools-docs/drools/language-reference/index.html)
- [OPA Rego - Undefined Decision Fix](https://policyascode.dev/guides/rego-undefined-decision-fix/)
- [OPA Issue #1241: Evaluate non existing entry](https://github.com/open-policy-agent/opa/issues/1241)
- [OPA Issue #5211: Allow undefined to be passed to function](https://github.com/open-policy-agent/opa/issues/5211)
- [CEL Language Definition](https://github.com/google/cel-spec/blob/master/doc/langdef.md)
- [json-rules-engine Issue #215: undefined facts](https://github.com/CacheControl/json-rules-engine/issues/215)
- [json-rules-engine Issue #111: Dealing with undefined facts](https://github.com/CacheControl/json-rules-engine/issues/111)

### Database Systems
- [SQL Three-Valued Logic - Wikibooks](https://en.wikibooks.org/wiki/Structured_Query_Language/NULLs_and_the_Three_Valued_Logic)
- [MongoDB Query for Null or Missing Fields](https://www.mongodb.com/docs/manual/tutorial/query-for-null-fields/)
- [MongoDB Null Handling Best Practices](https://www.mydbops.com/blog/null-handling-in-mongodb)

### Programming Languages
- [TypeScript Optional Properties - Better Stack](https://betterstack.com/community/guides/scaling-nodejs/typescript-optional-properties/)
- [TypeScript Null vs Undefined Deep Dive](https://basarat.gitbook.io/typescript/recap/null-undefined)
- [Microsoft TypeScript Issue #9653](https://github.com/microsoft/TypeScript/issues/9653)
