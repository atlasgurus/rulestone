package engine

import (
	"encoding/json"
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

func RuleEngineRepoToCompareCondRepo(repo *RuleEngineRepo) (*CompareCondRepo, error) {
	result := CompareCondRepo{
		CondToCompareCondRecord:      types.NewHashMap[condition.Condition, *EvalCategoryRec](),
		CondToCategoryMap:            types.NewHashMap[condition.Condition, *hashmap.Map[condition.Operand, []condition.Operand]](),
		CondToStringMatcher:          types.NewHashMap[condition.Condition, *StringMatcher](),
		AttributeToCompareCondRecord: make(map[string]*hashset.Set[*EvalCategoryRec]),
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
		Verbose:                      true,
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
