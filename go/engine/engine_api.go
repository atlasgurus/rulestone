package engine

import (
	"github.com/rulestone/api"
	"github.com/rulestone/cateng"
	"github.com/rulestone/condition"
	"github.com/rulestone/objectmap"
	"github.com/rulestone/types"
	"github.com/zyedidia/generic/hashmap"
	"github.com/zyedidia/generic/hashset"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type RuleEngineRepo struct {
	Rules []*GeneralRuleRecord
	ctx     *types.AppContext
	ruleApi *api.RuleApi
}

func (repo *RuleEngineRepo) Register(f *api.RuleDefinition) uint {
	result := uint(len(repo.Rules))
	repo.Rules = append(repo.Rules, &GeneralRuleRecord{f, result})
	return result
}

func (repo *RuleEngineRepo) RegisterRule(rule *api.Rule) (uint, error) {
	rd, err := repo.ruleApi.RuleToRuleDefinition(rule)
	if err != nil {
		return math.MaxUint, err
	}
	return repo.Register(rd), nil
}

func (repo *RuleEngineRepo) RegisterRuleFromString(rule string, format string) (uint, error) {
	r := strings.NewReader(rule)
	rd, err := repo.ruleApi.ReadRule(r, format)
	if err != nil {
		return math.MaxUint, err
	}
	return repo.RegisterRule(rd)
}

func (repo *RuleEngineRepo) RegisterRuleFromFile(path string) (uint, error) {
	f, err := os.Open(path)
	if err != nil {
		return math.MaxUint, err
	}
	defer f.Close()

	fileType := filepath.Ext(path)
	fileType = fileType[1:] // Remove the dot from the extension

	rule, err := repo.ruleApi.ReadRule(f, fileType)
	if err != nil {
		return math.MaxUint, err
	}
	return repo.RegisterRule(rule)
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
	result.CondToStringMatcher.Each(func(key condition.Condition, value *StringMatcher) {value.Build()})

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

func (f *RuleEngine) GetRuleDefinition(ruleId int) *api.RuleDefinition {
	if ruleId >= 0 && ruleId < len(f.repo.Rules) {
		return f.repo.Rules[ruleId].definition
	} else {
		return nil
	}
}
