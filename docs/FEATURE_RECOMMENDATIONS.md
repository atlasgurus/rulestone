# Feature Recommendations Based on Competitive Analysis

## Executive Summary

After analyzing Grule, expr-lang, and other Go rule engines, this document recommends high-value features to add to rulestone while maintaining its core design philosophy of high-performance, read-only pattern matching.

**Key Finding**: Rulestone is well-positioned as the fastest Go rule engine for high-throughput pattern matching, but lacks some convenience features that would improve usability.

---

## Recommended Features (Priority Order)

### Tier 1: Implement Next (High Value, Low Effort)

#### 1. Ternary Operator ⭐⭐⭐⭐⭐

**Syntax**:
```yaml
expression: age >= 18 ? "adult" : "minor"
expression: status == "VIP" ? discount * 2 : discount
expression: score > 90 ? "A" : score > 80 ? "B" : "C"  # Nested
```

**Why**:
- Extremely common pattern
- Much cleaner than `(age >= 18 && "adult") || (age < 18 && "minor")`
- Supported by SQL, JavaScript, Go, Java
- Currently requires awkward workarounds

**Implementation**:
- Add `token.QUESTION` and `token.COLON` handling in AST parser
- Create TernaryOperand or handle inline
- ~50-100 lines of code

**Effort**: 2-3 hours
**Value**: High - immediate usability improvement

---

#### 2. Salience/Priority Metadata ⭐⭐⭐⭐

**Syntax**:
```yaml
- metadata:
    id: critical-rule
    salience: 100  # Higher = higher priority
  expression: critical_condition

- metadata:
    id: fallback-rule
    salience: 10
  expression: default_condition
```

**Returns**: Matched rules sorted by salience (descending)

**Why**:
- Deterministic ordering (currently random from map iteration)
- Useful for fallback patterns
- Common in production rule systems
- Grule's most-used feature

**Implementation**:
- Add Salience field to metadata
- Sort matched rules by salience before returning
- ~30 lines of code

**Effort**: 1-2 hours
**Value**: High - solves real user pain point

---

#### 3. Additional Math Functions ⭐⭐⭐⭐

**Functions**:
```yaml
abs(value)           # Absolute value
ceil(value)          # Round up
floor(value)         # Round down
round(value, digits) # Round to n digits
min(a, b, ...)       # Minimum value
max(a, b, ...)       # Maximum value
pow(base, exp)       # Power
```

**Examples**:
```yaml
expression: abs(balance) > 1000
expression: ceil(price * 1.08) <= budget
expression: round(score, 2) >= 95.50
expression: min(price, competitor_price) < threshold
expression: max(quantity, min_order) >= 10
```

**Why**:
- Already have sqrt(), natural extension
- Common operations
- Round out standard library

**Implementation**:
- Similar to existing funcLength, funcDays patterns
- ~20 lines per function
- ~150 lines total

**Effort**: 3-4 hours
**Value**: Medium-High - convenience, completeness

---

### Tier 2: Consider for Future (Medium Value/Effort)

#### 4. Filter/Map Functions ⭐⭐⭐

**Syntax**:
```yaml
# Filter returns filtered array
expression: length(filter("items", "item", item.price > 10)) >= 5

# Map returns array of values
expression: sum(map("items", "item", item.price)) > 1000

# Combined
expression: filter("users", "u", u.age >= 18)
```

**Why**:
- Complement to forAll/forSome
- Common functional programming pattern
- Useful for complex array operations

**Implementation**:
- Return ListOperand with filtered/mapped values
- Need reduce/sum for aggregation
- ~200-300 lines

**Effort**: 1-2 days
**Value**: Medium - advanced users

---

#### 5. More String Functions ⭐⭐⭐

**Functions**:
```yaml
contains(haystack, needle)    # "hello".contains("ell") → true
startsWith(str, prefix)        # "hello".startsWith("hel") → true
endsWith(str, suffix)          # "hello".endsWith("lo") → true
toLowerCase(str)               # "Hello" → "hello"
toUpperCase(str)               # "hello" → "HELLO"
trim(str)                      # "  hello  " → "hello"
split(str, sep)                # "a,b,c" → ["a", "b", "c"]
join(array, sep)               # ["a", "b"] → "a,b"
```

**Why**:
- Common string operations
- Currently limited string support
- Easy to implement

**Implementation**:
- Use Go's strings package
- ~20 lines per function
- ~200 lines total

**Effort**: 4-6 hours
**Value**: Medium - convenience

---

#### 6. Date Manipulation Functions ⭐⭐⭐

**Functions**:
```yaml
addDays(time, n)               # date + n days
addHours(time, n)              # time + n hours
startOfDay(time)               # Midnight of that day
endOfDay(time)                 # 23:59:59 of that day
dayOfWeek(time)                # 0-6 (Sunday=0)
dayOfMonth(time)               # 1-31
month(time)                    # 1-12
year(time)                     # 2025
```

**Examples**:
```yaml
expression: dayOfWeek(event_time) == 0  # Sunday
expression: month(created_at) == 12     # December
expression: event_time >= startOfDay(now())  # Today
```

**Why**:
- Extend time arithmetic capabilities
- Useful for time-based rules
- Currently have basic time support

**Implementation**:
- Use Go's time package
- ~30 lines per function
- ~250 lines total

**Effort**: 6-8 hours
**Value**: Medium - time rules are common

---

### Tier 3: Nice to Have (Lower Priority)

#### 7. Compile-Time Type Checking (Optional) ⭐⭐

**Syntax**:
```go
schema := &EventSchema{
    Fields: map[string]FieldType{
        "age": IntType,
        "name": StringType,
        "premium": BoolType,
    },
}

result := repo.LoadRules(reader,
    engine.WithTypeCheck(schema))
// Catches type errors at load time
```

**Why**:
- Catch errors earlier
- Better IDE support
- Safer for large rule sets

**Cons**:
- Requires schema definition
- Validation already catches many issues
- High implementation cost

**Effort**: 1-2 weeks
**Value**: Medium - nice-to-have, not essential

---

### Tier 4: Don't Implement (Conflicts with Design)

#### ❌ Actions/Mutations

**Why not**:
- Conflicts with read-only design philosophy
- Makes engine stateful (complexity)
- Users can handle actions in application code
- Would require architectural changes

**Grule has this, but**: Different design goals (inference vs matching)

---

#### ❌ Rule Chaining

**Why not**:
- Single-pass is a feature, not limitation
- Simplifies reasoning about rules
- Better performance
- Users can chain externally if needed

---

#### ❌ Bytecode Compilation

**Why not**:
- Current performance is already excellent
- High implementation complexity
- Maintenance burden
- Category optimization is sufficient

---

## Quick Wins for Next Release

**Bundle these together** for immediate impact:

1. **Ternary Operator** (3 hours)
2. **Salience/Priority** (2 hours)
3. **Math Functions** (4 hours)

**Total**: ~9 hours development, ~150-200 lines of code

**Impact**: Significantly improved developer experience with minimal complexity.

---

## Performance Benchmark Plan

Create `benchmarks/comparative_test.go`:

```go
// Benchmark against Grule for same scenarios
func BenchmarkRulestone_100Rules(b *testing.B)
func BenchmarkGrule_100Rules(b *testing.B)

func BenchmarkRulestone_1000Rules(b *testing.B)
func BenchmarkGrule_1000Rules(b *testing.B)

func BenchmarkRulestone_LoadTime(b *testing.B)
func BenchmarkGrule_LoadTime(b *testing.B)
```

**Goal**: Validate performance claims with data.

---

## Positioning Statement (Draft)

**Rulestone**: The fastest Go rule engine for high-throughput pattern matching

**vs Grule**:
- ✅ Faster: Category pre-filtering beats RETE at scale
- ✅ Simpler: Read-only, no state management
- ✅ Testing: Built-in test framework
- ❌ No actions: By design (separation of concerns)

**vs expr-lang**:
- ✅ Multi-rule: Optimized for thousands of rules
- ✅ Management: Loading, validation, versioning
- ❌ No bytecode: Direct evaluation (still fast)

**Best for**:
- Event classification (10k+ events/sec)
- Large rule sets (1000+ rules)
- Real-time matching
- Fraud detection, feature flags, routing

**Not for**:
- Complex workflows with state
- Multi-step inference
- Forward chaining

---

## Recommendation Summary

### Implement Immediately

1. ✅ **Ternary operator** - Essential syntax sugar
2. ✅ **Salience** - Deterministic ordering
3. ✅ **Math functions** - Complete standard library

### Consider for v2.x

4. Filter/map/reduce functions
5. Extended string functions
6. Extended date functions
7. Performance benchmarks vs Grule

### Don't Implement

- Actions/mutations (wrong paradigm)
- Rule chaining (wrong paradigm)
- Bytecode VM (unnecessary complexity)

**Philosophy**: Keep rulestone focused on what it does best - blazing-fast, read-only pattern matching at scale.
