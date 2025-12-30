# Phase 2: Rule Validation and Compilation Architecture

## Overview

Separate rule validation/compilation from rule loading to support operational workflows where rules need to be validated before deployment and loaded quickly at runtime.

## Current Issues

1. **No validation during load**: `RegisterRulesFromFile` accepts invalid rules without error
2. **All validation happens at engine creation**: Errors surface late in the workflow
3. **No pre-compilation**: Rules are parsed and optimized every time an engine is created
4. **No atomic rule updates**: Cannot validate → load → use pattern safely

## Requirements

### Functional Requirements

1. **Separate validation step**: Validate rules before loading into repository
2. **Full AST validation**: Parse expressions, check functions, operators, syntax
3. **Compile step**: Pre-compute optimized structures that can be loaded quickly
4. **Fast loading**: Load pre-compiled rules without re-parsing or re-optimizing
5. **Clear error messages**: Validation errors should pinpoint exact issue in rule file

### Non-Requirements

- ❌ Backward compatibility with current API (breaking changes allowed)
- ❌ Transactional rollback support
- ❌ Runtime rule modification

## Proposed Architecture

### Three-Phase Model

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Source    │  Compile│  Compiled   │  Load   │   Runtime   │
│   Rules     │────────>│   Rules     │────────>│   Engine    │
│ (YAML/JSON) │         │  (Binary)   │         │             │
└─────────────┘         └─────────────┘         └─────────────┘
      │                                               │
      │                 Validate                      │
      └───────────────────────────────────────────────┘
```

### Phase 1: Validate
- Parse YAML/JSON structure
- Parse expression AST
- Validate functions exist
- Validate operators are supported
- Check for syntax errors
- **Output**: Validation report with errors

### Phase 2: Compile
- All validation from Phase 1
- Build category optimization structures
- Build string matchers
- Compute expression hashes
- Generate evaluator closures
- **Output**: Compiled rule file (binary format)

### Phase 3: Load
- Deserialize compiled rule structures
- Register in repository
- Create engine (no parsing/optimization needed)
- **Output**: Ready-to-use RuleEngine

## API Design

### Option A: Explicit Compile/Load (Recommended)

```go
// Validate rules without compilation
func ValidateRulesFile(path string) error

// Compile rules to binary format
func CompileRulesFile(sourcePath, compiledPath string) error

// Load pre-compiled rules (fast)
func (repo *RuleEngineRepo) LoadCompiledRules(path string) ([]uint, error)

// Create engine from pre-compiled rules (no optimization)
func NewRuleEngineFromCompiled(repo *RuleEngineRepo) (*RuleEngine, error)

// Legacy: compile and load in one step (for development)
func (repo *RuleEngineRepo) RegisterRulesFromFile(path string) ([]uint, error) {
    // Internally calls: validate → compile → load
}
```

**Workflow 1: Development (fast iteration)**
```go
repo := NewRuleEngineRepo()
_, err := repo.RegisterRulesFromFile("rules.yaml")  // Compile on-the-fly
engine, err := NewRuleEngine(repo)
```

**Workflow 2: Production (pre-compiled)**
```bash
# Build step (CI/CD pipeline)
rulestone-compile rules.yaml -o rules.compiled

# Validation step (deployment gate)
rulestone-validate rules.yaml
```

```go
// Runtime (fast load)
repo := NewRuleEngineRepo()
_, err := repo.LoadCompiledRules("rules.compiled")  // No parsing!
engine, err := NewRuleEngineFromCompiled(repo)       // No optimization!
```

**Workflow 3: Hot reload**
```go
// Validate before reloading
if err := ValidateRulesFile("new-rules.yaml"); err != nil {
    log.Error("Invalid rules: %v", err)
    return  // Keep using old engine
}

// Compile to temp file
tmpCompiled := "/tmp/rules-" + uuid.New() + ".compiled"
if err := CompileRulesFile("new-rules.yaml", tmpCompiled); err != nil {
    log.Error("Compilation failed: %v", err)
    return
}

// Load into new repo
newRepo := NewRuleEngineRepo()
newRepo.LoadCompiledRules(tmpCompiled)
newEngine, _ := NewRuleEngineFromCompiled(newRepo)

// Atomic swap
atomic.StorePointer(&currentEngine, unsafe.Pointer(newEngine))
```

### Option B: Compile-to-memory (Alternative)

```go
// Compile rules in memory (no file I/O)
type CompiledRules struct {
    // Internal optimized structures
}

func CompileRules(reader io.Reader) (*CompiledRules, error)

func (repo *RuleEngineRepo) LoadCompiledRules(compiled *CompiledRules) ([]uint, error)
```

## Compiled Rule Format

### Requirements
- Fast deserialization (avoid reflection)
- Version compatibility (detect incompatible formats)
- Human-readable debug info (optional)

### Proposed Format Options

**Option 1: Binary with Protocol Buffers**
```protobuf
message CompiledRuleSet {
    uint32 format_version = 1;
    repeated CompiledRule rules = 2;
    CategoryOptimizations optimizations = 3;
}

message CompiledRule {
    uint32 id = 1;
    map<string, string> metadata = 2;
    bytes condition_ast = 3;  // Serialized condition tree
    repeated CategoryEvaluator evaluators = 4;
}
```

**Option 2: Custom Binary Format**
```
Header:
  Magic: 0x52554C45 ("RULE")
  Version: uint16
  RuleCount: uint32

Per Rule:
  RuleID: uint32
  MetadataSize: uint32
  Metadata: JSON
  ConditionSize: uint32
  Condition: Serialized AST
  EvaluatorCount: uint32
  Evaluators: [...]
```

**Option 3: JSON with Precomputed Structures** (Easiest to implement first)
```json
{
  "version": "1.0",
  "rules": [
    {
      "id": 1,
      "metadata": {"id": "rule-1"},
      "expression": "a == 1 && b > 10",
      "ast": {
        "type": "AND",
        "operands": [...]
      },
      "categories": [1, 5, 7],
      "attributes": ["a", "b"]
    }
  ],
  "optimizations": {
    "categoryMap": {...},
    "stringMatchers": {...}
  }
}
```

## Implementation Plan

### Phase 2a: Validation Infrastructure (1-2 days)
- [ ] Create `ValidateRulesFile(path string) error`
- [ ] Extract validation logic from `NewRuleEngine` into reusable function
- [ ] Add detailed error messages with line numbers
- [ ] Add validation tests
- [ ] Update error_validation_test.go to use new API

**Deliverables**:
- Standalone validation function
- Comprehensive validation error messages
- All validation tests passing

### Phase 2b: Compilation to JSON (2-3 days)
- [ ] Design compiled JSON schema
- [ ] Create `CompiledRuleSet` struct
- [ ] Implement `CompileRulesFile(source, dest string) error`
- [ ] Serialize:
  - Parsed AST
  - Category mappings
  - String matcher patterns
  - Attribute addresses
- [ ] Add compilation tests
- [ ] Add CLI tool: `rulestone-compile`

**Deliverables**:
- Compiled rule format v1 (JSON)
- Compilation function
- CLI tool for offline compilation

### Phase 2c: Fast Loading (2-3 days)
- [ ] Create `LoadCompiledRules(path string) ([]uint, error)`
- [ ] Deserialize compiled JSON
- [ ] Reconstruct engine structures without re-parsing
- [ ] Create `NewRuleEngineFromCompiled(repo) (*RuleEngine, error)`
- [ ] Benchmark: compare compiled load vs. source load
- [ ] Add loading tests

**Deliverables**:
- Fast loading from compiled format
- Performance benchmarks
- Updated documentation

### Phase 2d: Binary Format Optimization (Optional, 2-3 days)
- [ ] Design binary format or use protobuf
- [ ] Implement binary serialization
- [ ] Implement binary deserialization
- [ ] Benchmark: compare binary vs. JSON
- [ ] Add format version checking

**Deliverables**:
- Binary compiled format
- Performance improvements over JSON

### Phase 2e: Integration and Documentation (1-2 days)
- [ ] Update README with new workflow
- [ ] Add examples for each workflow
- [ ] Update ARCHITECTURE.md
- [ ] Add migration guide from Phase 1
- [ ] Update API documentation

**Deliverables**:
- Complete documentation
- Working examples
- Migration guide

## Success Criteria

### Performance
- ✅ Compiled load is 10x faster than source load
- ✅ Validation without compilation is 5x faster than engine creation

### Usability
- ✅ Clear error messages for validation failures
- ✅ Simple CLI tools for validation and compilation
- ✅ Examples for all common workflows

### Reliability
- ✅ All existing tests pass
- ✅ New tests for validation, compilation, loading
- ✅ Format versioning prevents incompatible loads

## Testing Strategy

### Unit Tests
- Validation function with invalid rules
- Compilation with various rule types
- Loading from compiled format
- Format version mismatch handling

### Integration Tests
- End-to-end: source → compile → load → match
- Hot reload scenario
- Large rule sets (1000+ rules)

### Performance Tests
- Benchmark compilation time vs. engine creation time
- Benchmark load time: compiled vs. source
- Benchmark match performance: compiled vs. source (should be identical)

## Migration Path

### For Library Users

**Before (Phase 1)**:
```go
repo := engine.NewRuleEngineRepo()
repo.RegisterRulesFromFile("rules.yaml")
eng, _ := engine.NewRuleEngine(repo)
```

**After (Phase 2) - Development**:
```go
// Same API, internally compiles
repo := engine.NewRuleEngineRepo()
repo.RegisterRulesFromFile("rules.yaml")
eng, _ := engine.NewRuleEngine(repo)
```

**After (Phase 2) - Production**:
```go
// Pre-compile in CI/CD
// $ rulestone-compile rules.yaml -o rules.compiled

// Fast load at runtime
repo := engine.NewRuleEngineRepo()
repo.LoadCompiledRules("rules.compiled")
eng, _ := engine.NewRuleEngineFromCompiled(repo)
```

## Open Questions

1. **Closure serialization**: How to serialize Go closures (evaluator functions)?
   - Option A: Re-build closures from AST during load
   - Option B: Use code generation to create static evaluators
   - Option C: Keep AST, interpret at runtime (slower)

2. **Format stability**: How to handle format changes across versions?
   - Include version in compiled file
   - Reject incompatible versions
   - Support migration tools?

3. **Incremental compilation**: Should we support compiling individual rules?
   - Useful for dynamic rule addition
   - Complicates API

4. **Compression**: Should compiled files be compressed?
   - Smaller files
   - Slower load (but probably still faster than parsing)

## Decision Log

### 2025-12-30: Initial Design
- **Decision**: Use JSON format initially for simplicity
- **Rationale**: Easier to debug, human-readable, can optimize to binary later
- **Alternative considered**: Binary format with protobuf - more complex to implement

### 2025-12-30: API Design
- **Decision**: Separate validate/compile/load functions
- **Rationale**: Clear separation of concerns, supports multiple workflows
- **Alternative considered**: Single API with options - less flexible

### 2025-12-30: Backward Compatibility
- **Decision**: Breaking changes allowed
- **Rationale**: Project in early phase, clean API more important
- **Alternative considered**: Keep old API with deprecation - adds maintenance burden
