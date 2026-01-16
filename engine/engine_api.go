package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/atlasgurus/rulestone/cateng"
	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/objectmap"
	"github.com/atlasgurus/rulestone/types"
	"github.com/zyedidia/generic/hashmap"
	"github.com/zyedidia/generic/hashset"
	"gopkg.in/yaml.v3"
)

type ExternalRule struct {
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Expression string                 `json:"expression"`
	Tests      []TestCase             `json:"tests,omitempty" yaml:"tests,omitempty"`
}

// TestCase defines a test case for a rule
type TestCase struct {
	Name   string                 `json:"name" yaml:"name"`
	Event  map[string]interface{} `json:"event" yaml:"event"`
	Expect bool                   `json:"expect" yaml:"expect"`
}

// ruleInfo tracks rule information for testing
type ruleInfo struct {
	internalID uint
	externalID string
	tests      []TestCase
}

// LoadOption is a functional option for configuring rule loading
type LoadOption func(*loadConfig)

// loadConfig holds the configuration for rule loading
type loadConfig struct {
	validate   bool
	runTests   bool
	fileFormat string
	optimize   bool
}

// defaultLoadConfig returns the default configuration for rule loading
func defaultLoadConfig() *loadConfig {
	return &loadConfig{
		validate:   true,
		runTests:   false,
		fileFormat: "",
		optimize:   false, // Default: no optimization
	}
}

// WithValidate enables or disables expression validation during load
func WithValidate(validate bool) LoadOption {
	return func(c *loadConfig) {
		c.validate = validate
	}
}

// WithRunTests enables or disables test execution during load
func WithRunTests(runTests bool) LoadOption {
	return func(c *loadConfig) {
		c.runTests = runTests
	}
}

// WithFileFormat specifies the file format ("yaml", "json", or "" for auto-detect)
func WithFileFormat(format string) LoadOption {
	return func(c *loadConfig) {
		c.fileFormat = format
	}
}

// WithOptimize enables or disables category engine optimizations
func WithOptimize(optimize bool) LoadOption {
	return func(c *loadConfig) {
		c.optimize = optimize
	}
}

// TestResult contains the result of executing a single test case
type TestResult struct {
	RuleID   string                 // Rule ID from metadata
	TestName string                 // Test case name
	Passed   bool                   // Whether test passed
	Expected bool                   // Expected result
	Actual   bool                   // Actual result
	Event    map[string]interface{} // Test event data
	Error    error                  // Error if test execution failed
}

// LoadResult contains the result of loading rules
type LoadResult struct {
	RuleIDs      []uint       // IDs of loaded rules
	ValidationOK bool         // True if all rules validated successfully
	TestResults  []TestResult // Results from test execution
	Errors       []error      // Validation or test errors
}

// TestSummary contains statistics about test execution
type TestSummary struct {
	Total  int // Total number of tests
	Passed int // Number of tests that passed
	Failed int // Number of tests that failed
	Errors int // Number of tests that had execution errors
}

// GetTestSummary returns statistics about test execution
func (r *LoadResult) GetTestSummary() TestSummary {
	summary := TestSummary{Total: len(r.TestResults)}
	for _, tr := range r.TestResults {
		if tr.Error != nil {
			summary.Errors++
		} else if tr.Passed {
			summary.Passed++
		} else {
			summary.Failed++
		}
	}
	return summary
}

// GetFailedTests returns only the tests that failed
func (r *LoadResult) GetFailedTests() []TestResult {
	failed := make([]TestResult, 0)
	for _, tr := range r.TestResults {
		if !tr.Passed || tr.Error != nil {
			failed = append(failed, tr)
		}
	}
	return failed
}

// FormatTestResult returns a human-readable string for a single test result
func (tr *TestResult) FormatTestResult() string {
	if tr.Error != nil {
		return fmt.Sprintf("[ERROR] %s - %s: %v", tr.RuleID, tr.TestName, tr.Error)
	}
	status := "PASS"
	if !tr.Passed {
		status = "FAIL"
	}
	return fmt.Sprintf("[%s] %s - %s (expected: %v, actual: %v)", status, tr.RuleID, tr.TestName, tr.Expected, tr.Actual)
}

// FormatTestSummary returns a human-readable summary of test results
func (s *TestSummary) FormatTestSummary() string {
	if s.Total == 0 {
		return "No tests executed"
	}
	return fmt.Sprintf("Tests: %d total, %d passed, %d failed, %d errors", s.Total, s.Passed, s.Failed, s.Errors)
}

type RuleApi struct {
	ctx *types.AppContext
}

type InternalRule struct {
	Metadata  map[string]interface{}
	Condition condition.Condition
}

// GetHash returns the cryptographic hash of the compiled rule condition.
// The hash uniquely identifies the rule's semantic content and is computed
// recursively from the entire condition tree structure.
func (rule *InternalRule) GetHash() uint64 {
	return rule.Condition.GetHash()
}

func externalToInternalRule(rule *ExternalRule) (*InternalRule, error) {
	cond := condition.NewExprCondition(rule.Expression)
	if cond.GetKind() == condition.ErrorCondKind {
		return nil, cond.(*condition.ErrorCondition).Err
	}
	return &InternalRule{
		Metadata:  rule.Metadata,
		Condition: cond}, nil
}

func (api *RuleApi) ReadRules(r io.Reader, fileType string) ([]InternalRule, error) {
	var result []ExternalRule

	switch strings.ToLower(fileType) {
	case "json":
		decoder := json.NewDecoder(r)
		if err := decoder.Decode(&result); err != nil {
			return nil, api.ctx.Errorf("error parsing JSON: %s", err)
		}
	case "yaml", "yml":
		decoder := yaml.NewDecoder(r)
		if err := decoder.Decode(&result); err != nil {
			return nil, api.ctx.Errorf("error parsing YAML: %s", err)
		}
	default:
		return nil, api.ctx.Errorf("unsupported file type: %s", fileType)
	}

	internalRules := make([]InternalRule, len(result))
	for i, rule := range result {
		internalRule, err := externalToInternalRule(&rule)
		if err != nil {
			return nil, err
		}
		internalRules[i] = *internalRule
	}
	return internalRules, nil
}

func NewRuleApi(ctx *types.AppContext) *RuleApi {
	return &RuleApi{ctx: ctx}
}

type RuleEngineRepo struct {
	Rules    []*GeneralRuleRecord
	ctx      *types.AppContext
	ruleApi  *RuleApi
	Optimize bool // If true, apply category engine optimizations (default: true)
}

func (repo *RuleEngineRepo) Register(f *InternalRule) uint {
	result := uint(len(repo.Rules))
	repo.Rules = append(repo.Rules, &GeneralRuleRecord{f, result})
	return result
}

func (repo *RuleEngineRepo) RegisterRuleFromString(rule string, format string) (uint, error) {
	r := strings.NewReader(rule)
	rules, err := repo.ruleApi.ReadRules(r, format)
	if err != nil {
		return math.MaxUint, err
	}
	return repo.Register(&rules[0]), nil
}

func (repo *RuleEngineRepo) RegisterRulesFromFile(path string) ([]uint, error) {
	f, err := os.Open(path)
	if err != nil {
		return []uint{}, err
	}
	defer f.Close()

	fileType := filepath.Ext(path)
	fileType = fileType[1:] // Remove the dot from the extension

	rules, err := repo.ruleApi.ReadRules(f, fileType)
	if err != nil {
		return []uint{}, err
	}
	ruleIds := make([]uint, 0)
	for i := range rules {
		ruleId := repo.Register(&rules[i])
		ruleIds = append(ruleIds, ruleId)
	}
	return ruleIds, nil
}

// LoadRules loads rules from an io.Reader with optional configuration
func (repo *RuleEngineRepo) LoadRules(reader io.Reader, opts ...LoadOption) (*LoadResult, error) {
	// Apply functional options
	config := defaultLoadConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Set optimization mode on repo
	repo.Optimize = config.optimize

	result := &LoadResult{
		RuleIDs:      make([]uint, 0),
		ValidationOK: true,
		TestResults:  make([]TestResult, 0),
		Errors:       make([]error, 0),
	}

	// Parse rules from reader
	var externalRules []ExternalRule
	switch strings.ToLower(config.fileFormat) {
	case "json":
		decoder := json.NewDecoder(reader)
		if err := decoder.Decode(&externalRules); err != nil {
			return nil, repo.ctx.Errorf("error parsing JSON: %s", err)
		}
	case "yaml", "yml", "":
		// Default to YAML if not specified
		decoder := yaml.NewDecoder(reader)
		if err := decoder.Decode(&externalRules); err != nil {
			return nil, repo.ctx.Errorf("error parsing YAML: %s", err)
		}
	default:
		return nil, repo.ctx.Errorf("unsupported file format: %s", config.fileFormat)
	}

	// Track rule ID mapping for testing
	ruleInfos := make([]ruleInfo, 0, len(externalRules))

	// Process each rule
	for ruleIndex, extRule := range externalRules {
		// Get rule ID from metadata (if available)
		ruleID := ""
		if id, ok := extRule.Metadata["id"]; ok {
			if idStr, ok := id.(string); ok {
				ruleID = idStr
			}
		}

		// Create a descriptor for error messages
		ruleDescriptor := ""
		if ruleID != "" {
			ruleDescriptor = ruleID
		} else {
			ruleDescriptor = fmt.Sprintf("rule at index %d", ruleIndex)
		}

		// Check for missing expression
		if extRule.Expression == "" {
			if config.validate {
				result.ValidationOK = false
				result.Errors = append(result.Errors, repo.ctx.Errorf("%s: missing or empty expression", ruleDescriptor))
				continue // Skip this rule
			} else {
				return nil, repo.ctx.Errorf("%s: missing or empty expression", ruleDescriptor)
			}
		}

		// Convert to internal rule (includes expression parsing)
		var internalRule *InternalRule
		var err error

		if config.validate {
			// Validate by parsing expression
			internalRule, err = externalToInternalRule(&extRule)
			if err != nil {
				result.ValidationOK = false
				result.Errors = append(result.Errors, repo.ctx.Errorf("%s: validation failed: %v", ruleDescriptor, err))
				continue // Skip this rule
			}
		} else {
			// Skip validation, just store expression
			internalRule, err = externalToInternalRule(&extRule)
			if err != nil {
				// Even without validation, basic parsing errors are fatal
				return nil, repo.ctx.Errorf("%s: failed to parse: %v", ruleDescriptor, err)
			}
		}

		// Register the rule
		ruleInternalID := repo.Register(internalRule)
		result.RuleIDs = append(result.RuleIDs, ruleInternalID)

		// Save test info for later execution
		if config.runTests && len(extRule.Tests) > 0 {
			ruleInfos = append(ruleInfos, ruleInfo{
				internalID: ruleInternalID,
				externalID: ruleID,
				tests:      extRule.Tests,
			})
		}
	}

	// Validate by attempting to create engine if requested
	if config.validate {
		_, err := NewRuleEngine(repo)
		if err != nil {
			result.ValidationOK = false
			result.Errors = append(result.Errors, repo.ctx.Errorf("engine creation failed: %v", err))
		}
	}

	// Run tests if requested (create engine once for all tests)
	if config.runTests && len(ruleInfos) > 0 {
		testResults := repo.runAllTests(ruleInfos)
		result.TestResults = append(result.TestResults, testResults...)
	}

	return result, nil
}

// LoadRulesFromFile is a convenience wrapper for LoadRules that loads from a file
func (repo *RuleEngineRepo) LoadRulesFromFile(path string, opts ...LoadOption) (*LoadResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Auto-detect file format from extension if not specified
	config := defaultLoadConfig()
	for _, opt := range opts {
		opt(config)
	}

	if config.fileFormat == "" {
		ext := filepath.Ext(path)
		if len(ext) > 0 {
			// Add file format option based on extension
			opts = append(opts, WithFileFormat(ext[1:]))
		}
	}

	return repo.LoadRules(f, opts...)
}

// LoadRulesFromString is a convenience wrapper for LoadRules that loads from a string
func (repo *RuleEngineRepo) LoadRulesFromString(content string, opts ...LoadOption) (*LoadResult, error) {
	reader := strings.NewReader(content)
	return repo.LoadRules(reader, opts...)
}

// runAllTests executes test cases for all rules and returns test results
func (repo *RuleEngineRepo) runAllTests(ruleInfos []ruleInfo) []TestResult {
	results := make([]TestResult, 0)

	// Create engine to test all rules
	// We need to do this because rules aren't ready to match until engine is created
	tempEngine, err := NewRuleEngine(repo)
	if err != nil {
		// If engine creation fails, all tests fail
		for _, ruleInfo := range ruleInfos {
			for _, test := range ruleInfo.tests {
				results = append(results, TestResult{
					RuleID:   ruleInfo.externalID,
					TestName: test.Name,
					Passed:   false,
					Expected: test.Expect,
					Actual:   false,
					Event:    test.Event,
					Error:    repo.ctx.Errorf("failed to create engine: %v", err),
				})
			}
		}
		return results
	}

	// Run test cases for each rule
	for _, ruleInfo := range ruleInfos {
		for _, test := range ruleInfo.tests {
			matches := tempEngine.MatchEvent(test.Event)

			// Check if this specific rule matched
			ruleMatched := false
			for _, matchedRuleID := range matches {
				if uint(matchedRuleID) == ruleInfo.internalID {
					ruleMatched = true
					break
				}
			}

			passed := (ruleMatched == test.Expect)

			results = append(results, TestResult{
				RuleID:   ruleInfo.externalID,
				TestName: test.Name,
				Passed:   passed,
				Expected: test.Expect,
				Actual:   ruleMatched,
				Event:    test.Event,
				Error:    nil,
			})
		}
	}

	return results
}

func RuleEngineRepoToCompareCondRepo(repo *RuleEngineRepo) (*CompareCondRepo, error) {
	result := CompareCondRepo{
		CondToCompareCondRecord:      types.NewHashMap[condition.Condition, *EvalCategoryRec](),
		CondToCategoryMap:            types.NewHashMap[condition.Condition, *hashmap.Map[condition.Operand, []condition.Operand]](),
		CondToStringMatcher:          types.NewHashMap[condition.Condition, *StringMatcher](),
		AttributeToCompareCondRecord: make(map[string]*hashset.Set[*EvalCategoryRec]),
		AlwaysEvaluateCategories:     types.NewHashSet[*EvalCategoryRec](),
		UndefinedEqualityCategories:  types.NewHashSet[types.Category](),
		ObjectAttributeMapper:        objectmap.NewObjectAttributeMapper(repo),
		CondFactory:                  condition.NewFactory(),
		ctx:                          repo.ctx,
	}

	rootScope := &ForEachScope{
		Path:         "",
		Element:      "", // Will match $.something reference
		NestingLevel: 0,
		ParentScope:  nil,
		AttrDictRec:  result.ObjectAttributeMapper.RootDictRec}

	for id, f := range repo.Rules {
		cond := result.ConvertToCategoryCondition(f.definition.Condition, rootScope)
		if cond.GetKind() == condition.ErrorCondKind {
			return nil, cond.(*condition.ErrorCondition).Err
		}
		result.RuleRepo.Register(condition.NewRule(condition.RuleIdType(id), cond))
	}

	// Pass UndefinedEqualityCategories to RuleRepo for DefaultCatList building
	result.RuleRepo.UndefinedEqualityCategories = result.UndefinedEqualityCategories

	// Build the string matchers
	result.CondToStringMatcher.Each(func(key condition.Condition, value *StringMatcher) { value.Build() })

	return &result, nil
}

type RuleEngineMetrics struct {
	NumCatEvals uint64
}

type RuleEngine struct {
	repo         *RuleEngineRepo
	catEngine    *cateng.CategoryEngine
	compCondRepo *CompareCondRepo
	Metrics      RuleEngineMetrics
}

func NewRuleEngine(repo *RuleEngineRepo) (*RuleEngine, error) {
	compCondRepo, err := RuleEngineRepoToCompareCondRepo(repo)
	if err != nil {
		return nil, err
	}

	// Set optimization thresholds based on Optimize flag
	var orThreshold, andThreshold uint
	if repo.Optimize {
		// Optimized mode: use default values
		orThreshold = 0  // OR optimization disabled by default
		andThreshold = 1 // AND optimization enabled with threshold 1
	} else {
		// Non-optimized mode: disable all optimizations
		orThreshold = 0
		andThreshold = 0
	}

	catEngine := cateng.NewCategoryEngine(&compCondRepo.RuleRepo, &cateng.Options{
		OrOptimizationFreqThreshold:  orThreshold,
		AndOptimizationFreqThreshold: andThreshold,
		Verbose:                      false,
	})

	return &RuleEngine{repo: repo, catEngine: catEngine, compCondRepo: compCondRepo}, nil
}

func (f *RuleEngine) MatchEvent(v interface{}) []condition.RuleIdType {
	matchingCompareCondRecords := types.NewHashSet[*EvalCategoryRec]()
	event := f.compCondRepo.ObjectAttributeMapper.MapObject(v,
		// Callback for each attribute of interest found in the mapped event
		func(addr []int) {
			addrMatchId := objectmap.AddressMatchKey(addr)
			catEvaluators, ok := f.compCondRepo.AttributeToCompareCondRecord[addrMatchId]
			if ok {
				catEvaluators.Each(
					func(catEvaluator *EvalCategoryRec) {
						matchingCompareCondRecords.Put(catEvaluator)
					})
			}
		})

	// Also evaluate categories that must always run (e.g., null checks, constant expressions, forAll)
	f.compCondRepo.AlwaysEvaluateCategories.Each(func(catEvaluator *EvalCategoryRec) {
		matchingCompareCondRecords.Put(catEvaluator)
	})

	var eventCategories []types.Category
	var FrameStack = [20]interface{}{event.Values}
	matchingCompareCondRecords.Each(func(catEvaluator *EvalCategoryRec) {
		atomic.AddUint64(&f.Metrics.NumCatEvals, 1)
		result := catEvaluator.Evaluate(event, FrameStack[:])
		switch r := result.(type) {
		case condition.ErrorOperand:
			// TODO: find a way to report errors
			// can't report every error, have to aggregate errors and report periodic statistics
		case condition.UndefinedOperand:
			// Undefined results don't add to eventCategories (not applicable)
			// This represents "the rule doesn't apply" due to missing fields
		case condition.BooleanOperand:
			cat := catEvaluator.GetCategory()
			if r {
				eventCategories = append(eventCategories, cat)
			}
		case *condition.ListOperand:
			for _, c := range r.List {
				cat := types.Category(c.(condition.IntOperand))
				eventCategories = append(eventCategories, cat)
			}
		case condition.IntOperand:
			if r != 0 {
				eventCategories = append(eventCategories, types.Category(r))
			}
		case condition.FloatOperand:
			// Treat non-zero floats as truthy for category matching
			if r != 0.0 {
				cat := catEvaluator.GetCategory()
				eventCategories = append(eventCategories, cat)
			}
		case condition.StringOperand:
			// Treat non-empty strings as truthy for category matching
			if len(r) > 0 {
				cat := catEvaluator.GetCategory()
				eventCategories = append(eventCategories, cat)
			}
		case condition.TimeOperand:
			// Treat non-zero times as truthy for category matching
			if !time.Time(r).IsZero() {
				cat := catEvaluator.GetCategory()
				eventCategories = append(eventCategories, cat)
			}
		case condition.NullOperand:
			// Null operands are falsy - don't add category
			// (do nothing)
		default:
			panic(fmt.Sprintf("Unexpected operand type in category evaluation: %T", result))
		}
	})
	f.compCondRepo.ObjectAttributeMapper.FreeObject(event)
	return f.catEngine.MatchEvent(eventCategories)
}

func (f *RuleEngine) GetRuleDefinition(ruleId uint) *InternalRule {
	if ruleId >= 0 && int(ruleId) >= 0 && int(ruleId) < len(f.repo.Rules) {
		return f.repo.Rules[ruleId].definition
	} else {
		return nil
	}
}
