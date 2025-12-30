# Phase 2: Rule Validation and Testing Architecture (REVISED)

## Overview

Add optional validation and built-in testing to support operational workflows where rules need to be validated before deployment, without requiring a separate compiled format.

## Current Issues

1. **No validation during load**: `RegisterRulesFromFile` accepts invalid rules without error
2. **All validation happens at engine creation**: Errors surface late in the workflow
3. **File-based API**: Cannot load rules from database, S3, HTTP requests
4. **No built-in testing**: No way to verify rules behave as expected
5. **Validation incompleteness**: Even if we validate expressions, optimization phase can still fail

## Key Insight: Compiled Format Not Viable

**Problem**: Go closures (evaluator functions) cannot be serialized
- Engine builds closures during optimization
- These capture context and cannot be marshaled
- Any "compiled" format would still need to rebuild closures
- Result: No performance benefit from pre-compilation

**Solution**: Validate fully, but don't serialize. Allow validation to be skipped on trusted inputs.

## Requirements

### Functional Requirements

1. **Optional validation during load**: Load with or without validation
2. **Full validation**: Parse expressions, check functions, operators, run tests
3. **Reader-based API**: Load from io.Reader (database, S3, HTTP, etc.)
4. **Built-in rule testing**: Rules specify test cases in metadata
5. **Clear error messages**: Validation errors pinpoint exact issue

### Non-Requirements

- ❌ Backward compatibility with current API (breaking changes allowed)
- ❌ Transactional rollback support
- ❌ Compiled binary format (not viable due to closure serialization)
- ❌ Runtime rule modification

## Proposed Architecture

### Load with Optional Validation

```
┌─────────────┐         ┌─────────────────┐         ┌─────────────┐
│   Source    │         │  Repository     │         │   Engine    │
│   Rules     │  Load   │  (in-memory)    │  Build  │  (ready to  │
│ (YAML/JSON/ │────────>│                 │────────>│   match)    │
│  io.Reader) │         │                 │         │             │
└─────────────┘         └─────────────────┘         └─────────────┘
      │                         │                         │
      │    Validate (optional)  │                         │
      └─────────────────────────┴─────────────────────────┘
                                │
                         Run Tests (optional)
```

### Validation Modes

**Full Validation** (default for untrusted sources):
1. Parse YAML/JSON structure
2. Parse expression AST (full syntax validation)
3. Build engine and run optimization (catch all errors)
4. Execute test cases if present in metadata
5. **Output**: Validated repository + test results

**Skip Validation** (for trusted sources):
1. Parse YAML/JSON structure
2. Store expression strings without parsing
3. Build engine (errors are fatal, not graceful)
4. Skip test execution
5. **Output**: Repository (faster, but fails hard on errors)

**Note**: Even "skip validation" must parse and optimize when creating the engine. The benefit is:
- Skipping redundant validation in CI → production pipeline
- Treating validation errors as fatal (fail fast) vs. returning detailed reports

## API Design

### New API (io.Reader-based with optional validation)

```go
// LoadOptions controls validation and testing behavior
type LoadOptions struct {
    Validate   bool  // If true, validate expressions during load (default: true)
    RunTests   bool  // If true, execute test cases in metadata (default: true)
    FileFormat string // "yaml", "json", or "" for auto-detect
}

// Load rules from any source (file, database, S3, HTTP, etc.)
func (repo *RuleEngineRepo) LoadRules(reader io.Reader, opts LoadOptions) (*LoadResult, error)

// Convenience wrapper for files
func (repo *RuleEngineRepo) LoadRulesFromFile(path string, opts LoadOptions) (*LoadResult, error)

// Result includes validation and test results
type LoadResult struct {
    RuleIDs        []uint
    ValidationOK   bool
    TestResults    []TestResult
    Errors         []error
}

type TestResult struct {
    RuleID      string
    TestName    string
    Passed      bool
    Expected    bool
    Actual      bool
    Event       map[string]interface{}
    Error       error
}
```

**Workflow 1: Development (full validation + testing)**
```go
repo := NewRuleEngineRepo()
result, err := repo.LoadRulesFromFile("rules.yaml", LoadOptions{
    Validate: true,
    RunTests: true,
})
if err != nil {
    log.Fatal("Load failed: %v", err)
}
if !result.ValidationOK {
    log.Fatal("Validation failed")
}
for _, tr := range result.TestResults {
    if !tr.Passed {
        log.Error("Test failed: %s - %s", tr.RuleID, tr.TestName)
    }
}

engine, err := NewRuleEngine(repo)
```

**Workflow 2: CI/CD validation**
```go
// Step 1: Validate and test in CI
f, _ := os.Open("rules.yaml")
defer f.Close()

tmpRepo := NewRuleEngineRepo()
result, err := tmpRepo.LoadRules(f, LoadOptions{Validate: true, RunTests: true})
if err != nil || !result.ValidationOK {
    os.Exit(1)  // Fail CI build
}

// Step 2: Deploy rules.yaml to production servers
// (validated in CI, so can skip validation in prod for speed)
```

**Workflow 3: Production (skip validation for speed)**
```go
// Load from S3, database, etc.
rulesJSON := fetchFromDatabase()
reader := strings.NewReader(rulesJSON)

repo := NewRuleEngineRepo()
result, err := repo.LoadRules(reader, LoadOptions{
    Validate:   false,  // Already validated in CI
    RunTests:   false,  // Skip tests in production
    FileFormat: "json",
})
if err != nil {
    log.Fatal("Load failed: %v", err)  // Fatal errors only
}

engine, err := NewRuleEngine(repo)
```

**Workflow 4: Hot reload**
```go
// Fetch new rules from HTTP POST
newRules := fetchFromHTTPRequest()

// Validate in temporary repo first
tmpRepo := NewRuleEngineRepo()
result, err := tmpRepo.LoadRules(
    bytes.NewReader(newRules),
    LoadOptions{Validate: true, RunTests: true},
)
if err != nil || !result.ValidationOK {
    return fmt.Errorf("invalid rules: %v", err)
}

// Build new engine
newEngine, err := NewRuleEngine(tmpRepo)
if err != nil {
    return fmt.Errorf("engine build failed: %v", err)
}

// Atomic swap
atomic.StorePointer(&currentEngine, unsafe.Pointer(newEngine))
```

## Rule Format with Built-in Tests

### YAML Format (Recommended)

```yaml
- metadata:
    id: user-eligibility-check
    description: Check if user is eligible for premium features
    tags: [premium, user, eligibility]

  expression: user.age >= 18 && user.country == "US" && user.subscription == "premium"

  tests:
    - name: eligible user
      event:
        user:
          age: 25
          country: US
          subscription: premium
      expect: true

    - name: underage user
      event:
        user:
          age: 17
          country: US
          subscription: premium
      expect: false

    - name: non-US user
      event:
        user:
          age: 25
          country: CA
          subscription: premium
      expect: false

    - name: free tier user
      event:
        user:
          age: 25
          country: US
          subscription: free
      expect: false

- metadata:
    id: high-value-transaction
    description: Detect high-value transactions

  expression: transaction.amount > 1000 && transaction.currency == "USD"

  tests:
    - name: high value USD
      event:
        transaction:
          amount: 1500
          currency: USD
      expect: true

    - name: low value USD
      event:
        transaction:
          amount: 500
          currency: USD
      expect: false

    - name: high value EUR
      event:
        transaction:
          amount: 1500
          currency: EUR
      expect: false
```

### JSON Format (Alternative)

```json
[
  {
    "metadata": {
      "id": "user-eligibility-check",
      "description": "Check if user is eligible for premium features"
    },
    "expression": "user.age >= 18 && user.country == \"US\"",
    "tests": [
      {
        "name": "eligible user",
        "event": {
          "user": {
            "age": 25,
            "country": "US"
          }
        },
        "expect": true
      },
      {
        "name": "underage user",
        "event": {
          "user": {
            "age": 17,
            "country": "US"
          }
        },
        "expect": false
      }
    ]
  }
]
```

### Test Execution Behavior

1. **During load with `RunTests: true`**:
   - Execute each test case against the rule
   - Compare actual result with expected result
   - Collect failures in `LoadResult.TestResults`
   - Loading succeeds even if tests fail (non-blocking)

2. **Test result reporting**:
   - Each test result includes: rule ID, test name, pass/fail, expected, actual
   - Failed tests are logged but don't prevent loading
   - CI/CD can check `result.TestResults` and fail build if any test fails

3. **Benefits of built-in tests**:
   - **Self-documenting**: Rules show expected behavior
   - **Data-driven testing**: No Go code needed for most tests
   - **Version control**: Test cases travel with rules
   - **Regression detection**: Changes to engine caught immediately

## Implementation Plan

### Phase 2a: Core API Changes (2-3 days)

**Goal**: Replace file-based API with io.Reader-based API + LoadOptions

- [ ] Add `LoadOptions` struct (Validate, RunTests, FileFormat)
- [ ] Add `LoadResult` struct (RuleIDs, ValidationOK, TestResults, Errors)
- [ ] Add `TestResult` struct
- [ ] Implement `LoadRules(io.Reader, LoadOptions) (*LoadResult, error)`
- [ ] Implement `LoadRulesFromFile(path, LoadOptions) (*LoadResult, error)`
- [ ] Add test metadata parsing from YAML/JSON
- [ ] Update existing tests to use new API

**Deliverables**:
- New io.Reader-based API
- Backward compatible wrapper (optional)
- Updated unit tests

### Phase 2b: Validation Logic (2-3 days)

**Goal**: Extract and enhance validation logic

- [ ] Extract validation from `NewRuleEngine` into separate function
- [ ] Make validation optional via `LoadOptions.Validate`
- [ ] Add detailed error messages with:
  - Rule ID
  - Line number (if available)
  - Expression causing error
  - Specific error reason
- [ ] Handle validation errors gracefully vs. fatally
- [ ] Update error_validation_test.go

**Deliverables**:
- Optional validation during load
- Detailed validation error reporting
- All validation tests passing

### Phase 2c: Built-in Test Execution (2-3 days)

**Goal**: Execute test cases defined in rule metadata

- [ ] Parse `tests` field from rule metadata
- [ ] Implement test execution logic:
  - Create temporary engine for each rule
  - Run each test event
  - Compare actual vs. expected result
- [ ] Collect test results in `LoadResult.TestResults`
- [ ] Add test execution flag in `LoadOptions.RunTests`
- [ ] Add test result formatting/reporting

**Deliverables**:
- Test execution during load
- Test result reporting
- Example rules with tests

### Phase 2d: Data-Driven Test Migration (1-2 days)

**Goal**: Convert hardcoded Go tests to data-driven tests in rule files

- [ ] Create `tests/data/` directory for rule files with tests
- [ ] Convert existing test cases to YAML format with test metadata
- [ ] Create single Go test runner that:
  - Loads rule files from tests/data/
  - Executes built-in tests
  - Reports results
- [ ] Remove redundant Go test code
- [ ] Keep only engine-level and infrastructure tests in Go

**Test categories to convert**:
- Boolean literal tests → data/boolean_literals.yaml
- Type conversion tests → data/type_conversion.yaml
- Null handling tests → data/null_handling.yaml
- Quantifier tests → data/quantifiers.yaml
- Expression complexity tests → data/expressions.yaml

**Deliverables**:
- Data-driven test suite
- Simpler, more maintainable tests
- Reduced Go test code

### Phase 2e: Documentation and Examples (1-2 days)

- [ ] Update README with new API
- [ ] Document rule format with tests section
- [ ] Add workflow examples:
  - Development workflow
  - CI/CD validation
  - Production deployment
  - Hot reload
- [ ] Update ARCHITECTURE.md
- [ ] Add example rules with comprehensive tests
- [ ] Create migration guide

**Deliverables**:
- Complete documentation
- Working examples for all workflows
- Migration guide from Phase 1 API

## Success Criteria

### Functionality
- ✅ Rules can be loaded from any io.Reader (files, databases, HTTP, etc.)
- ✅ Validation is optional and controlled by LoadOptions
- ✅ Built-in tests execute automatically during load
- ✅ Test results are reported in LoadResult

### Usability
- ✅ Clear error messages for validation failures
- ✅ Test failures show rule ID, test name, expected vs. actual
- ✅ Examples for all common workflows (dev, CI/CD, prod, hot reload)
- ✅ Rules are self-documenting with inline tests

### Code Quality
- ✅ All existing tests pass or migrated to data-driven format
- ✅ Data-driven tests cover all major functionality
- ✅ Reduced Go test code (simpler maintenance)
- ✅ Rule files with tests serve as examples

## Testing Strategy

### Unit Tests (Go)
- LoadOptions parsing and defaults
- LoadResult structure and error handling
- Test metadata parsing from YAML/JSON
- Test execution logic (pass/fail detection)
- Validation with/without errors
- io.Reader compatibility (strings, files, bytes, etc.)

### Data-Driven Tests (YAML)
- Boolean literals and operators
- Type conversions (string↔number↔boolean)
- Null handling and missing fields
- Quantifiers (forAll, forSome)
- Complex expressions and nesting
- String functions (regexpMatch, containsAny)
- Arithmetic and comparison operations

### Integration Tests (Go)
- Load rules → validate → execute tests → create engine → match events
- Hot reload scenario with validation
- Load from multiple sources (file, string, HTTP mock)
- Large rule sets (1000+ rules with tests)
- Error recovery and graceful degradation

### Workflow Tests (Go)
- CI/CD workflow: validate in CI, skip in prod
- Development workflow: full validation + tests
- Production workflow: skip validation for speed
- Hot reload: validate → load → atomic swap

## Migration Path

### For Library Users

**Phase 1 API (Current)**:
```go
repo := engine.NewRuleEngineRepo()
repo.RegisterRulesFromFile("rules.yaml")  // No validation until engine creation
eng, _ := engine.NewRuleEngine(repo)      // Errors surface here
```

**Phase 2 API (New) - With Validation**:
```go
repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRulesFromFile("rules.yaml", LoadOptions{
    Validate: true,  // Validate during load
    RunTests: true,  // Execute built-in tests
})
if err != nil || !result.ValidationOK {
    log.Fatal("Validation failed")
}
eng, _ := engine.NewRuleEngine(repo)  // Should succeed
```

**Phase 2 API (New) - Production (Skip Validation)**:
```go
repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRulesFromFile("rules.yaml", LoadOptions{
    Validate: false,  // Already validated in CI
    RunTests: false,  // Skip tests in production
})
if err != nil {
    log.Fatal("Load failed: %v", err)
}
eng, _ := engine.NewRuleEngine(repo)
```

**Phase 2 API (New) - From Database/S3**:
```go
// Fetch rules from any source
rulesData := fetchFromDatabase()
reader := bytes.NewReader(rulesData)

repo := engine.NewRuleEngineRepo()
result, err := repo.LoadRules(reader, LoadOptions{
    Validate:   true,
    RunTests:   true,
    FileFormat: "json",
})
```

### Breaking Changes

1. **New API required**: `LoadRules` and `LoadRulesFromFile` replace `RegisterRulesFromFile`
2. **LoadOptions required**: Must specify validation/testing preferences
3. **LoadResult returned**: Instead of just `[]uint`, get full result with tests
4. **Test metadata**: Rules can now include `tests` field (optional but recommended)

### Deprecation Strategy

**Option 1**: Keep old API, mark deprecated
```go
// Deprecated: Use LoadRulesFromFile with LoadOptions
func (repo *RuleEngineRepo) RegisterRulesFromFile(path string) ([]uint, error) {
    result, err := repo.LoadRulesFromFile(path, LoadOptions{
        Validate: true,
        RunTests: false,  // Old API didn't run tests
    })
    return result.RuleIDs, err
}
```

**Option 2**: Remove old API (clean break)
- Document migration in MIGRATION.md
- Provide examples for all use cases
- Version bump: v1.x → v2.0

## Decision Log

### 2025-12-30: Compiled Format Rejected
- **Decision**: Do NOT create compiled binary format
- **Rationale**: Go closures cannot be serialized; any "compiled" format would still need to rebuild closures from AST, providing no performance benefit
- **Alternative considered**: Compiled format with closure serialization - not viable in Go

### 2025-12-30: Validation Strategy
- **Decision**: Make validation optional via LoadOptions, don't separate into standalone function
- **Rationale**:
  - Standalone validation incomplete (doesn't catch optimization errors)
  - Better to validate fully during load or skip entirely for trusted sources
  - CI/CD can validate, production can skip for speed
- **Alternative considered**: Separate ValidateRulesFile function - doesn't provide complete validation

### 2025-12-30: io.Reader-based API
- **Decision**: Replace file-based API with io.Reader-based API
- **Rationale**:
  - Supports loading from databases, S3, HTTP requests, memory
  - More flexible for diverse deployment scenarios
  - Standard Go pattern (like json.Decoder, yaml.Decoder)
- **Alternative considered**: Keep file-based API - too restrictive

### 2025-12-30: Built-in Test Cases
- **Decision**: Add `tests` field to rule metadata (YAML/JSON)
- **Rationale**:
  - Self-documenting rules
  - Version control for tests alongside rules
  - Enables data-driven testing, reduces Go test code
  - CI/CD can validate behavior, not just syntax
  - Rules become unit-testable
- **Alternative considered**: Keep tests separate in Go - harder to maintain, tests can drift from rules

### 2025-12-30: Data-Driven Test Migration
- **Decision**: Convert most Go tests to data-driven YAML tests
- **Rationale**:
  - Simpler to write and maintain
  - Non-developers can contribute tests
  - Tests serve as examples
  - Reduces code volume
- **Alternative considered**: Keep all Go tests - more code to maintain

### 2025-12-30: Backward Compatibility
- **Decision**: Breaking changes allowed (v2.0)
- **Rationale**: Project in early phase, clean API more important than compatibility
- **Alternative considered**: Deprecate old API - adds maintenance burden
