# Undefined Semantics Feature - Work in Progress

This folder contains research and design documents for implementing UndefinedOperand to distinguish missing fields from explicit null values.

## Status

**Research & Design Phase** - Implementation not started yet

## Documents

### 1. NULL_SEMANTICS_RESEARCH.md
Comprehensive industry research comparing how major rule engines and data systems handle missing vs null:
- SQL three-valued logic
- MongoDB's v8.0 breaking change
- OPA Rego's undefined semantics
- CEL, json-rules-engine, TypeScript/JavaScript
- Industry comparison matrix
- Lessons learned from MongoDB's migration

### 2. UNDEFINED_SEMANTICS_ANALYSIS.md
Deep analysis of whether we can eliminate DefaultCatList using three-valued logic:
- Comparison of binary logic vs three-valued logic
- Proof that undefined propagation eliminates need for DefaultCatList
- Industry alignment analysis
- Performance analysis

### 3. STRICT_MODE_DESIGN.md
Original research on implementing strict mode:
- Discovery of DefaultCatList mechanism
- Root cause analysis of negative category behavior
- Alternative approaches considered

### 4. IMPLEMENTATION_PLAN_UNDEFINED.md
**Final implementation strategy** with efficient two-set tracking:
- Complete semantics specification
- Three mechanisms working together (natural triggering, DefaultCatList, AlwaysEvaluateCategories)
- Track evaluated separately from fired using two sets
- DefaultCatList shrinks from 100+ to ~5-10 entries
- Implementation checklist with file locations
- Breaking changes and migration guide

## Key Decisions

1. ✅ Distinguish missing from null EVERYWHERE (no modes/flags)
2. ✅ Use three-valued logic with undefined propagation
3. ✅ Keep optimized DefaultCatList (small, for `field == undefined` only)
4. ✅ Track evaluated separately from fired (two-set approach)
5. ✅ Add `undefined` keyword to expression language
6. ✅ ~300 lines of code, ~150 tests, breaking changes acceptable

## Next Steps

1. Review implementation plan
2. Start implementation with golang-engineer agent
3. Comprehensive testing
4. Migration guide for users
5. Update main ARCHITECTURE.md documentation
