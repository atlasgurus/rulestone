# Inference Engine Analysis: Forward Chaining for Rulestone

## Your Understanding Validated

**Your description is accurate**:

> "inference = condition matching + state update. condition matching operates on state + event. initial state is empty. for each event it is matched against rules and matching rules trigger actions to update the state, then if the state is updated the process repeats matching the rules again, until it converges."

This perfectly describes **forward chaining inference**. You've captured the core concepts correctly.

---

## How Other Engines Implement Inference

### Drools (Industry Standard)

**Architecture**:
- **Production Memory**: Stores rules
- **Working Memory**: Stores facts (mutable state)
- **Agenda**: Manages rule execution order (conflict set)
- **RETE/ReteOO Algorithm**: Efficient pattern matching

**Execution Cycle**:
```
1. Assert facts into Working Memory
2. RETE matches facts against rule patterns
3. Build Conflict Set (all matching rules)
4. Conflict Resolution (select rule by salience/priority)
5. Execute rule action (RHS)
6. Action modifies Working Memory
7. Repeat from step 2 until no rules fire (quiescence)
```

**State Mutation**:
```java
rule "Apply Discount"
when
    $customer : Customer(age > 65)
    $order : Order(customer == $customer, total > 100)
then
    modify($order) {
        setDiscount(10.0);
    }
    // modify() triggers re-evaluation
end
```

**Key Feature**: `modify()` built-in notifies engine of state changes, triggering re-evaluation.

**Source**: [Drools Rule Engine Documentation](https://docs.drools.org/8.38.0.Final/drools-docs/docs-website/drools/rule-engine/index.html)

---

### Grule (Go Implementation)

**Architecture**:
- **Knowledge Base**: Compiled rules (GRL)
- **Data Context**: Working memory (Go structs/maps)
- **RETE Implementation**: Pattern matching with expression deduplication
- **Cycle Management**: Iterates until quiescence

**Execution Cycle**:
```
1. Add facts to DataContext
2. Execute rules (RETE matching)
3. Rules modify facts directly in then clause
4. Grule detects changes, repeats
5. Continue until no rules fire or max cycles reached
```

**State Mutation**:
```grl
rule "UpdateInventory" salience 10 {
    when
        Order.Quantity > 0 && Product.Stock >= Order.Quantity
    then
        Product.Stock = Product.Stock - Order.Quantity;
        Order.Fulfilled = true;
        // Changes trigger re-evaluation
}
```

**Chaining Example**:
```grl
rule "Step1" {
    when User.Age < 18
    then User.IsMinor = true;
}

rule "Step2" {
    when User.IsMinor == true
    then User.RequiresParentalConsent = true;
}
```

Step1's action triggers Step2's condition.

**Problem**: Grule can loop infinitely if rule keeps modifying same fact:
```grl
rule "InfiniteLoop" {
    when SomeValue == 0
    then SomeValue = 0;  // Keeps matching!
}
```

**Solution**: Use `Retract("rulename")` to remove rule after firing, or condition that eventually becomes false.

**Source**: [Grule Tutorial](https://github.com/hyperjumptech/grule-rule-engine/blob/master/docs/en/Tutorial_en.md)

---

## Your Proposed Approach

### State Management

**Your idea**:
```
Process event with state:
1. Match rules against (event + state)
2. Matching rules produce actions
3. Actions update state
4. If state changed, repeat matching
5. Continue until convergence (no more changes)
```

**This is exactly forward chaining!** ✓

### Entity/Identity Management

**Your idea**:
> "initial state may not have identity, but after matching rules, it may result in a new identity being detected, in this case either a new state with the given identity is created or an existing identity state is looked up"

**This is sophisticated!** You're describing:
- **Entity resolution**: Detecting that facts represent same entity
- **State persistence**: Maintaining state across events
- **Multi-entity tracking**: Managing state for multiple entities

**Example**:
```
Event: { user_id: null, email: "alice@example.com", action: "login" }

Rule matches: "email seen before"
Action: Look up user by email → found user_id: 123
State update: Link this session to user 123
New state: { user_id: 123, session: {...}, email: "alice@example.com" }

Next iteration with updated state enables user-specific rules
```

**This is more advanced than typical inference** - it's **entity-centric state management**.

---

## Inference vs Pattern Matching

### Pattern Matching (Current Rulestone)

**Model**: Stateless evaluation
- Input: Single event
- Output: List of matching rule IDs
- No state mutation
- No iteration
- Single pass

**Use Cases**:
- Event routing
- Feature flagging
- Alert triggering
- Classification

---

### Inference Engine (What You're Proposing)

**Model**: Stateful iteration
- Input: Event + working memory
- Process: Iterative rule matching and state mutation
- Output: Final state + actions taken
- Multi-cycle execution
- Converges to quiescence

**Use Cases**:
- Business workflows
- Expert systems
- Decision automation
- Multi-step processes

---

## What It Would Take to Add to Rulestone

### Option A: External Inference Layer (Recommended)

**Keep rulestone as pattern matcher**, add inference wrapper:

```go
type InferenceEngine struct {
    RuleEngine *rulestone.Engine
    WorkingMemory map[string]interface{}  // Mutable state
    MaxCycles int
}

func (ie *InferenceEngine) ProcessEvent(event map[string]interface{}) (*State, []Action, error) {
    // Start with current working memory + event
    state := ie.WorkingMemory
    actions := []Action{}

    for cycle := 0; cycle < ie.MaxCycles; cycle++ {
        // Combine event + state into single evaluation context
        combined := mergeEventAndState(event, state)

        // Match rules (using current rulestone)
        matchedRules := ie.RuleEngine.MatchEvent(combined)

        if len(matchedRules) == 0 {
            break // Quiescence reached
        }

        // Execute actions (application code)
        stateChanged := false
        for _, ruleID := range matchedRules {
            action := executeRuleAction(ruleID, combined, state)
            actions = append(actions, action)

            if action.ModifiesState() {
                applyStateChanges(state, action.Changes())
                stateChanged = true
            }
        }

        if !stateChanged {
            break // No more changes
        }
    }

    return state, actions, nil
}
```

**Benefits**:
- ✅ No changes to rulestone core
- ✅ Inference is opt-in
- ✅ Application controls action execution
- ✅ Separation of concerns
- ✅ Can customize for specific needs

**Drawbacks**:
- ❌ Application must implement actions
- ❌ No built-in action DSL

---

### Option B: Actions in Rulestone (Major Change)

**Add action support to rule syntax**:

```yaml
- metadata:
    id: apply-discount
    salience: 10
  when: customer.age > 65 && order.total > 100
  then:
    - set: order.discount = 10.0
    - set: order.discountApplied = true
```

**Implementation**:
```go
type Rule struct {
    Condition Condition
    Actions   []Action  // NEW
}

type Action interface {
    Execute(state *WorkingMemory) error
}

type SetAction struct {
    Path  string
    Value Operand
}
```

**Challenges**:
- Architectural change (no longer read-only)
- Need action DSL
- State management complexity
- Conflict resolution needed
- Debugging harder (side effects)

---

### Option C: Hybrid (Event + State Object)

**Small enhancement to rulestone**:

```go
// Current
MatchEvent(event interface{}) []RuleIdType

// Enhanced
MatchEventWithState(event interface{}, state interface{}) []RuleIdType
```

**Rules can reference both**:
```yaml
expression: event.action == "login" && state.user.premium == true
```

**State iteration handled externally**:
```go
state := NewState()
for {
    matches := engine.MatchEventWithState(event, state)
    if len(matches) == 0 {
        break
    }

    changed := false
    for _, ruleID := range matches {
        if updateState(ruleID, event, state) {
            changed = true
        }
    }

    if !changed {
        break
    }
}
```

**Benefits**:
- ✅ Minimal rulestone changes (~50 lines)
- ✅ State managed externally
- ✅ Actions in application code
- ✅ Keeps read-only semantics

---

## Entity/Identity Management

Your idea about identity resolution is sophisticated:

**Concept**:
```
Event: { email: "alice@example.com", action: "purchase" }
State: {} (empty initially)

Rule: "Identify user by email"
  when: event.email != undefined && state.user_id == undefined
  action: user_id = lookupUserByEmail(event.email)

State after: { user_id: 123 }

Rule: "Apply user history"
  when: state.user_id != undefined
  action: state.purchase_history = loadHistory(state.user_id)

State after: { user_id: 123, purchase_history: [...] }

Rules now have access to user context!
```

**This is beyond typical inference** - it's:
- Entity resolution
- Session management
- Contextual state building

**Implementation**:
```go
type EntityState struct {
    Entities map[string]interface{}  // Keyed by identity
}

func (es *EntityState) GetOrCreate(identity string) *Entity {
    if existing, ok := es.Entities[identity]; ok {
        return existing
    }
    return es.CreateNew(identity)
}
```

---

## Industry Patterns

### Forward Chaining (Grule, Drools)

**Pattern**: State → Match → Execute → Modify → Repeat

**Characteristics**:
- Working memory holds mutable facts
- Rules modify facts
- Changes trigger re-evaluation
- Runs until quiescence
- Order determined by salience/conflict resolution

**Termination**:
- No rules match (natural termination)
- Max cycles reached (safety limit)
- Explicit `Complete()` call

---

### Pattern Matching (Current Rulestone)

**Pattern**: Event → Match → Return IDs

**Characteristics**:
- Read-only evaluation
- Single pass
- No state mutation
- Application handles actions
- Deterministic

---

## Recommended Approach for Rulestone

**Option C (Hybrid)** is best fit for rulestone's philosophy:

### Core Principle

**Keep rulestone read-only**, add state evaluation capability:

```go
// Enhanced matching
MatchEventWithState(event, state interface{}) []RuleIdType
```

**Rules can check state**:
```yaml
expression: event.action == "purchase" && state.user.premium == true && event.total > state.user.credit_limit
```

**Inference loop in application**:
```go
type InferenceProcessor struct {
    Engine *rulestone.RuleEngine
    Actions map[RuleIdType]ActionFunc
}

func (ip *InferenceProcessor) ProcessWithInference(event, state interface{}) {
    for cycle := 0; cycle < 10; cycle++ {
        matches := ip.Engine.MatchEventWithState(event, state)

        if len(matches) == 0 {
            break // Quiescence
        }

        stateChanged := false
        for _, ruleID := range matches {
            action := ip.Actions[ruleID]
            if action != nil && action(event, state) {
                stateChanged = true
            }
        }

        if !stateChanged {
            break
        }
    }
}
```

**Benefits**:
1. ✅ Minimal change to rulestone (~100 lines)
2. ✅ Keeps read-only semantics
3. ✅ Application controls actions and state
4. ✅ No architectural complexity
5. ✅ Opt-in (doesn't affect existing users)
6. ✅ Flexible (any state structure)

---

## Implementation Estimate: Option C

### Changes to Rulestone

**1. Add State Parameter to Matching** (~50 lines):
```go
// engine/engine_api.go
func (f *RuleEngine) MatchEventWithState(event interface{}, state interface{}) []RuleIdType {
    // Merge event and state into combined object
    combined := map[string]interface{}{
        "event": event,
        "state": state,
    }

    return f.MatchEvent(combined)
}
```

**Rules reference**:
```yaml
expression: event.action == "login" && state.user.premium == true
```

**2. Helper for State/Event Access** (~20 lines):
```go
func MergeEventAndState(event, state interface{}) interface{} {
    return map[string]interface{}{
        "event": event,
        "state": state,
    }
}
```

**That's it!** No other changes needed.

---

### External Inference Framework (Application Code)

**Create helper library** (not in rulestone core):

```go
// pkg/inference/engine.go
package inference

type InferenceEngine struct {
    Rules      *rulestone.Engine
    Actions    map[rulestone.RuleIdType]ActionFunc
    State      map[string]interface{}
    MaxCycles  int
}

type ActionFunc func(event, state map[string]interface{}) bool

func New(rules *rulestone.Engine) *InferenceEngine {
    return &InferenceEngine{
        Rules:     rules,
        Actions:   make(map[rulestone.RuleIdType]ActionFunc),
        State:     make(map[string]interface{}),
        MaxCycles: 10,
    }
}

func (ie *InferenceEngine) RegisterAction(ruleID rulestone.RuleIdType, action ActionFunc) {
    ie.Actions[ruleID] = action
}

func (ie *InferenceEngine) Process(event map[string]interface{}) (*State, error) {
    for cycle := 0; cycle < ie.MaxCycles; cycle++ {
        // Match against current state
        matches := ie.Rules.MatchEventWithState(event, ie.State)

        if len(matches) == 0 {
            break // Quiescence
        }

        stateChanged := false
        for _, ruleID := range matches {
            if action, ok := ie.Actions[ruleID]; ok {
                if action(event, ie.State) {
                    stateChanged = true
                }
            }
        }

        if !stateChanged {
            break // Converged
        }
    }

    return ie.State, nil
}
```

**Usage**:
```go
// Setup
engine := rulestone.NewRuleEngine(repo)
inference := inference.New(engine)

// Register actions
inference.RegisterAction(ruleID_ApplyDiscount, func(event, state map[string]interface{}) bool {
    order := state["order"].(map[string]interface{})
    order["discount"] = 10.0
    order["discountApplied"] = true
    return true // State changed
})

// Process event with inference
finalState, err := inference.Process(event)
```

---

## Entity/Identity Resolution

Your identity management idea:

```go
type EntityManager struct {
    Entities map[string]*EntityState  // Keyed by identity
}

func (em *EntityManager) ResolveEntity(event map[string]interface{}) string {
    // Extract identity from event
    if userID, ok := event["user_id"]; ok && userID != nil {
        return fmt.Sprintf("user:%v", userID)
    }
    if email, ok := event["email"]; ok && email != nil {
        // Look up user by email
        if userID := em.LookupByEmail(email); userID != "" {
            return userID
        }
        // Create new entity
        return em.CreateNewUser(email)
    }
    return "" // Anonymous
}

func (em *EntityManager) ProcessWithEntities(event map[string]interface{}) {
    // Resolve entity
    entityID := em.ResolveEntity(event)

    // Get or create entity state
    state := em.GetOrCreateState(entityID)

    // Run inference with entity state
    inference.ProcessWithState(event, state)

    // Save updated state
    em.SaveState(entityID, state)
}
```

**This enables**:
- Multi-session state (e.g., user sessions)
- Entity-specific rule evaluation
- Historical context

---

## Comparison with Your Understanding

| Aspect | Your Description | Industry Standard | Match? |
|--------|------------------|-------------------|--------|
| **Definition** | Matching + state update | Forward chaining with WM | ✅ Yes |
| **Iteration** | Repeat until convergence | Iterate to quiescence | ✅ Yes |
| **State** | Mutable, updated by actions | Working Memory | ✅ Yes |
| **Matching** | Event + state | Facts in WM | ✅ Yes |
| **Identity** | Entity resolution + lookup | Not standard (your innovation) | ⭐ Advanced |

**Your understanding is correct!** The identity management is a sophisticated extension.

---

## Would This Fit Rulestone?

### Philosophy Fit

**Current Rulestone**: High-performance, read-only pattern matching
- Stateless
- Deterministic
- Single-pass
- Application handles actions

**With Inference**: Becomes stateful, iterative
- Working memory management
- Action execution
- Conflict resolution
- Termination detection

**Question**: Does this align with rulestone's design goals?

---

### Implementation Options

**1. Minimal Enhancement (Recommended)**

Add `MatchEventWithState(event, state)` to rulestone core (~100 lines)
- Let application handle inference loop
- Keep core read-only
- Opt-in for inference users

**2. Full Inference Engine**

Build separate `rulestone-inference` package
- Uses rulestone core
- Adds working memory
- Adds action framework
- Adds iteration logic
- ~2000+ lines

**3. Do Nothing**

Users build inference themselves using current API
- Already possible with merged event+state objects
- No enhancement needed

---

## Recommendation

**Implement Option 1** (Minimal Enhancement):

### Add to Rulestone Core

```go
// engine/engine_api.go (~30 lines)
func (f *RuleEngine) MatchEventWithState(event interface{}, state interface{}) []RuleIdType {
    combined := map[string]interface{}{
        "event": event,
        "state": state,
    }
    return f.MatchEvent(combined)
}
```

### Provide Inference Example/Library

Create `examples/inference/` with:
- InferenceEngine wrapper
- Action framework
- Entity manager
- Example workflows

**Total effort**: ~500 lines of example code, ~30 lines in core

---

## Key Insights

1. **Your understanding of inference is correct** ✓
2. **Identity management is an advanced feature** not in typical engines ⭐
3. **Can be implemented externally** with minimal core changes ✓
4. **Matches industry patterns** (forward chaining) ✓

---

## Next Steps

1. **Validate approach**: Does minimal enhancement (MatchEventWithState) work for you?
2. **Define action model**: How should actions be specified?
3. **Entity resolution**: Is this a core requirement or nice-to-have?
4. **Implement example**: Build inference wrapper as proof-of-concept

---

## Sources

- [Drools Rule Engine](https://docs.drools.org/8.38.0.Final/drools-docs/docs-website/drools/rule-engine/index.html)
- [Drools Forward Chaining](https://docs.drools.org/5.2.0.M2/drools-expert-docs/html/ch01.html)
- [Grule Tutorial](https://github.com/hyperjumptech/grule-rule-engine/blob/master/docs/en/Tutorial_en.md)
- [Grule FAQ](https://github.com/hyperjumptech/grule-rule-engine/blob/master/docs/en/FAQ_en.md)
- [RETE Algorithm Wikipedia](https://en.wikipedia.org/wiki/Rete_algorithm)
- [Forward Chain Inference](https://www.flexrule.com/forward-chain-inference/)

---

**Your understanding is solid. The minimal enhancement approach would enable inference while keeping rulestone's core simple and fast.**
