package engine

import (
	"encoding/json"
	"fmt"
	"github.com/atlasgurus/rulestone/cateng"
	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/objectmap"
	"github.com/atlasgurus/rulestone/types"
	"github.com/zyedidia/generic/hashmap"
	"github.com/zyedidia/generic/hashset"
	"gopkg.in/yaml.v3"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
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

// LoadOptions controls rule loading behavior
type LoadOptions struct {
	Validate   bool   // If true, validate expressions during load (default: true)
	RunTests   bool   // If true, execute test cases (default: true)
	FileFormat string // "yaml", "json", or "" for auto-detect from file extension
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
	Rules   []*GeneralRuleRecord
	ctx     *types.AppContext
	ruleApi *RuleApi
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

// LoadRules loads rules from an io.Reader with optional validation and testing
func (repo *RuleEngineRepo) LoadRules(reader io.Reader, opts LoadOptions) (*LoadResult, error) {
	result := &LoadResult{
		RuleIDs:      make([]uint, 0),
		ValidationOK: true,
		TestResults:  make([]TestResult, 0),
		Errors:       make([]error, 0),
	}

	// Parse rules from reader
	var externalRules []ExternalRule
	switch strings.ToLower(opts.FileFormat) {
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
		return nil, repo.ctx.Errorf("unsupported file format: %s", opts.FileFormat)
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
			if opts.Validate {
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

		if opts.Validate {
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
		if opts.RunTests && len(extRule.Tests) > 0 {
			ruleInfos = append(ruleInfos, ruleInfo{
				internalID: ruleInternalID,
				externalID: ruleID,
				tests:      extRule.Tests,
			})
		}
	}

	// Validate by attempting to create engine if requested
	if opts.Validate {
		_, err := NewRuleEngine(repo)
		if err != nil {
			result.ValidationOK = false
			result.Errors = append(result.Errors, repo.ctx.Errorf("engine creation failed: %v", err))
		}
	}

	// Run tests if requested (create engine once for all tests)
	if opts.RunTests && len(ruleInfos) > 0 {
		testResults := repo.runAllTests(ruleInfos)
		result.TestResults = append(result.TestResults, testResults...)
	}

	return result, nil
}

// LoadRulesFromFile is a convenience wrapper for LoadRules that loads from a file
func (repo *RuleEngineRepo) LoadRulesFromFile(path string, opts LoadOptions) (*LoadResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Auto-detect file format from extension if not specified
	if opts.FileFormat == "" {
		ext := filepath.Ext(path)
		if len(ext) > 0 {
			opts.FileFormat = ext[1:] // Remove the dot
		}
	}

	return repo.LoadRules(f, opts)
}

// LoadRulesFromString is a convenience wrapper for LoadRules that loads from a string
func (repo *RuleEngineRepo) LoadRulesFromString(content string, opts LoadOptions) (*LoadResult, error) {
	reader := strings.NewReader(content)
	return repo.LoadRules(reader, opts)
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
	catEngine := cateng.NewCategoryEngine(&compCondRepo.RuleRepo, &cateng.Options{
		// TODO implement option passing
		OrOptimizationFreqThreshold:  0,
		AndOptimizationFreqThreshold: 1,
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
		f.Metrics.NumCatEvals++
		result := catEvaluator.Evaluate(event, FrameStack[:])
		switch r := result.(type) {
		case condition.ErrorOperand:
			// TODO: find a way to report errors
			// can't report every error, have to aggregate errors and report periodic statistics
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
		default:
			panic("should not get here")
		}
	})
	f.compCondRepo.ObjectAttributeMapper.FreeObjects()
	return f.catEngine.MatchEvent(eventCategories)
}

func (f *RuleEngine) GetRuleDefinition(ruleId uint) *InternalRule {
	if ruleId >= 0 && int(ruleId) >= 0 && int(ruleId) < len(f.repo.Rules) {
		return f.repo.Rules[ruleId].definition
	} else {
		return nil
	}
}
