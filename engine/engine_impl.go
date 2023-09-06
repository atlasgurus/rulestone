package engine

import (
	"bytes"
	"fmt"
	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/immutable"
	"github.com/atlasgurus/rulestone/objectmap"
	"github.com/atlasgurus/rulestone/types"
	"github.com/zyedidia/generic/hashmap"
	"github.com/zyedidia/generic/hashset"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RepoInterface interface {
	Register(f *InternalRule)
}

type GeneralRuleRecord struct {
	definition *InternalRule
	id         uint
}

// MapScalar Implement MapperConfig interface
func (repo *RuleEngineRepo) MapScalar(v interface{}) interface{} {
	return condition.NewInterfaceOperand(v, repo.ctx)
}

func (repo *RuleEngineRepo) GetAppCtx() *types.AppContext {
	return repo.ctx
}

func NewRuleEngineRepo() *RuleEngineRepo {
	ctx := types.NewAppContext()
	return &RuleEngineRepo{ctx: ctx, ruleApi: NewRuleApi(ctx)}
}

type CatEvaluatorKind int8

type CatEvaluator interface {
	Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) (bool, error)
	GetCategory() types.Category
	immutable.SetElement
}

type EvalCategoryRec struct {
	Cat      types.Category
	Eval     condition.Operand
	AttrKeys []string
}

func (v *EvalCategoryRec) GetHash() uint64 {
	return uint64(v.Cat)
}

func (v *EvalCategoryRec) Equals(element immutable.SetElement) bool {
	// Cryptographic hash can be used for equality check
	return v.GetHash() == element.(*EvalCategoryRec).GetHash()
}

func (v *EvalCategoryRec) Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
	return v.Eval.Evaluate(event, frames)
}

func (v *EvalCategoryRec) GetCategory() types.Category {
	// Cryptographic hash can be used for equality check
	return v.Cat
}

// CompareCondRepo contains mappings between filter Compare Conditions and attributes assigned to them.
// Identical Compare Conditions will be mapped to the same category.
type CompareCondRepo struct {
	AttributeToCompareCondRecord map[string]*hashset.Set[*EvalCategoryRec]
	CondToCompareCondRecord      *hashmap.Map[condition.Condition, *EvalCategoryRec]
	CondToCategoryMap            *hashmap.Map[condition.Condition, *hashmap.Map[condition.Operand, []condition.Operand]]
	CondToStringMatcher          *hashmap.Map[condition.Condition, *StringMatcher]
	EvalCategoryRecs             []*EvalCategoryRec
	RuleRepo                     condition.RuleRepo
	ObjectAttributeMapper        *objectmap.ObjectAttributeMapper
	CondFactory                  *condition.Factory
	ctx                          *types.AppContext
}

func (repo *CompareCondRepo) NewEvalCategoryRec(eval condition.Operand) *EvalCategoryRec {
	result := &EvalCategoryRec{
		Cat:  types.Category(len(repo.EvalCategoryRecs) + 1),
		Eval: eval,
	}
	repo.EvalCategoryRecs = append(repo.EvalCategoryRecs, result)
	return result
}

func (repo *CompareCondRepo) DiscardEvalCategoryRec(evalCategoryRec *EvalCategoryRec) {
	// Only need to support removing the most recently added evalCategoryRec
	if repo.EvalCategoryRecs[len(repo.EvalCategoryRecs)-1].Cat != evalCategoryRec.Cat {
		panic("Should not happen")
	}
	// Remove the last one
	repo.EvalCategoryRecs = repo.EvalCategoryRecs[:len(repo.EvalCategoryRecs)-1]
}

// ConvertToCategoryCondition this has to be called from the root condition or and/or/not boolean operator
func (repo *CompareCondRepo) ConvertToCategoryCondition(c condition.Condition, parentScope *ForEachScope) condition.Condition {
	var result condition.Condition
	switch c.GetKind() {
	case condition.AndCondKind:
		result = repo.CondFactory.NewAndCond(types.MapSlice(
			c.GetOperands(), func(c condition.Condition) condition.Condition {
				return repo.ConvertToCategoryCondition(c, parentScope)
			})...)
	case condition.OrCondKind:
		result = repo.CondFactory.NewOrCond(types.MapSlice(
			c.GetOperands(), func(c condition.Condition) condition.Condition {
				return repo.ConvertToCategoryCondition(c, parentScope)
			})...)
	case condition.NotCondKind:
		result = repo.CondFactory.NewNotCond(repo.ConvertToCategoryCondition(c.GetOperands()[0], parentScope))
	case condition.CategoryCondKind:
		panic("CategoryCondKind not expected")
	case condition.CompareCondKind:
		result = repo.processCompareCondition(c.(*condition.CompareCondition), parentScope)
	case condition.ExprCondKind:
		result = repo.processExprCondition(c.(*condition.ExprCondition), parentScope)
	default:
		panic("should not happen")
	}
	return result
}

func (repo *CompareCondRepo) registerCatEvaluatorForAddress(attrAddress []int, catEvaluator *EvalCategoryRec) {
	if catEvaluator != nil {
		addrMatchKey := objectmap.AddressMatchKey(attrAddress)
		var condRecords *hashset.Set[*EvalCategoryRec]
		var ok bool
		if condRecords, ok = repo.AttributeToCompareCondRecord[addrMatchKey]; !ok {
			condRecords = types.NewHashSet[*EvalCategoryRec]()
			repo.AttributeToCompareCondRecord[addrMatchKey] = condRecords
		}
		condRecords.Put(catEvaluator)
		catEvaluator.AttrKeys = append(catEvaluator.AttrKeys, addrMatchKey)
	}
}

func (repo *CompareCondRepo) unregisterCatEvaluator(catEvaluator *EvalCategoryRec) {
	if catEvaluator != nil {
		for _, addrMatchKey := range catEvaluator.AttrKeys {
			var condRecords *hashset.Set[*EvalCategoryRec]
			var ok bool
			if condRecords, ok = repo.AttributeToCompareCondRecord[addrMatchKey]; ok {
				condRecords.Remove(catEvaluator)
				if condRecords.Size() == 0 {
					delete(repo.AttributeToCompareCondRecord, addrMatchKey)
				}
			}
		}
	}
}

func (repo *CompareCondRepo) genEvalForLogicalOp(
	op token.Token,
	xEval condition.Operand,
	yEval condition.Operand) condition.Operand {

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			X := xEval.Evaluate(event, frames).Convert(condition.BooleanOperandKind)
			if X.GetKind() == condition.ErrorOperandKind {
				return X
			}
			Y := yEval.Evaluate(event, frames).Convert(condition.BooleanOperandKind)
			if Y.GetKind() == condition.ErrorOperandKind {
				return Y
			}

			switch op {
			case token.LAND:
				return condition.NewBooleanOperand(bool(X.(condition.BooleanOperand)) && bool(Y.(condition.BooleanOperand)))
			case token.LOR:
				return condition.NewBooleanOperand(bool(X.(condition.BooleanOperand)) || bool(Y.(condition.BooleanOperand)))
			default:
				panic("should not get here")
			}
		}, xEval, yEval)
}

func (repo *CompareCondRepo) genEvalForCompareOperands(
	compOp condition.CompareOp,
	xEval condition.Operand,
	yEval condition.Operand) condition.Operand {

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			X := xEval.Evaluate(event, frames)
			xKind := X.GetKind()
			Y := yEval.Evaluate(event, frames)
			yKind := Y.GetKind()
			if xKind == condition.ErrorOperandKind {
				return X
			}
			if yKind == condition.ErrorOperandKind {
				return Y
			}

			// Convert toward the higher kind, e.g. int -> float -> bool -> string
			X, Y = condition.ReconcileOperands(X, Y)

			switch compOp {
			case condition.CompareEqualOp:
				return condition.NewBooleanOperand(X.Equals(Y))
			case condition.CompareNotEqualOp:
				return condition.NewBooleanOperand(!X.Equals(Y))
			case condition.CompareGreaterOp:
				return condition.NewBooleanOperand(X.Greater(Y))
			case condition.CompareGreaterOrEqualOp:
				return condition.NewBooleanOperand(!Y.Greater(X))
			case condition.CompareLessOp:
				return condition.NewBooleanOperand(Y.Greater(X))
			case condition.CompareLessOrEqualOp:
				return condition.NewBooleanOperand(!X.Greater(Y))
			default:
				panic("Not implemented")
			}
		}, xEval, yEval)
}

// processCompareEqualToConstCondition: Special case equal compare against a constant that can be done via a hash lookup
func (repo *CompareCondRepo) processCompareEqualToConstCondition(
	compareCond *condition.CompareCondition, scope *ForEachScope) condition.Operand {

	var constOperand condition.Operand
	var varOperand condition.Operand
	if compareCond.LeftOperand.IsConst() {
		constOperand = compareCond.LeftOperand
		varOperand = compareCond.RightOperand
	} else if compareCond.RightOperand.IsConst() {
		constOperand = compareCond.RightOperand
		varOperand = compareCond.LeftOperand
	} else {
		panic("one of the operands must be a constant")
	}

	return repo.processEvalForIsInConstantList(varOperand, []condition.Operand{constOperand}, scope)
}

// processEvalForIsInConstantList
func (repo *CompareCondRepo) processEvalForIsInConstantList(
	varOperand condition.Operand, consOperandList []condition.Operand, scope *ForEachScope) condition.Operand {
	varOperand = repo.evalOperandAccess(repo.evalOperandAddress(varOperand, scope), scope)
	if varOperand.GetKind() == condition.ErrorOperandKind {
		return varOperand
	}

	// Create a dummy compare operation ignoring the consOperandList value and look it up
	dummyCondition := condition.NewCompareCond(condition.CompareEqualOp, varOperand, condition.NewIntOperand(0))
	categoryMap, seenCond := repo.CondToCategoryMap.Get(dummyCondition)
	if !seenCond {
		categoryMap = types.NewHashMap[condition.Operand, []condition.Operand]()
		repo.CondToCategoryMap.Put(dummyCondition, categoryMap)
	}

	// Create an entry in the categoryMap for each of the consOperandList
	for _, constOperand := range consOperandList {
		categoryList, _ := categoryMap.Get(constOperand)
		categoryMap.Put(
			constOperand,
			append(categoryList, condition.NewIntOperand(int64(scope.Evaluator.GetCategory()))))
	}

	if seenCond {
		// We have seen a condition identical to this one except for the const compare equal operand.
		// Unregister the duplicate address to evaluator mappings that may have been created.
		repo.unregisterCatEvaluator(scope.Evaluator)
		return nil
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			X := varOperand.Evaluate(event, frames)
			xKind := X.GetKind()
			if xKind == condition.ErrorOperandKind {
				return X
			}
			catList, k := categoryMap.Get(X)
			if k {
				return condition.NewListOperand(catList)
			} else {
				return condition.IntConst0
			}
		}, varOperand)
}

func (repo *CompareCondRepo) processEvalForContains(
	varOperand condition.Operand, stringsToMatch []string, scope *ForEachScope) condition.Operand {
	varOperand = repo.evalOperandAccess(repo.evalOperandAddress(varOperand, scope), scope)
	if varOperand.GetKind() == condition.ErrorOperandKind {
		return varOperand
	}

	// Create a dummy compare operation ignoring the stringsToMatch value and look it up
	dummyCondition := condition.NewCompareCond(condition.CompareContainsOp, varOperand, condition.NewIntOperand(0))
	stringMatcher, seenCond := repo.CondToStringMatcher.Get(dummyCondition)
	if !seenCond {
		stringMatcher = NewStringMatcher()
		repo.CondToStringMatcher.Put(dummyCondition, stringMatcher)
	}

	// Create an entry in the stringMatcher for each of the stringsToMatch
	for _, constOperand := range stringsToMatch {
		stringMatcher.AddPattern(constOperand, condition.NewIntOperand(int64(scope.Evaluator.GetCategory())))
	}

	if seenCond {
		// We have seen a condition identical to this one except for the const compare equal operand.
		// Unregister the duplicate address to evaluator mappings that may have been created.
		repo.unregisterCatEvaluator(scope.Evaluator)
		return nil
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			X := varOperand.Evaluate(event, frames).Convert(condition.StringOperandKind)
			xKind := X.GetKind()
			if xKind == condition.ErrorOperandKind {
				return X
			}
			catList := stringMatcher.Match(string(X.(condition.StringOperand)))
			if len(catList) > 0 {
				return condition.NewListOperand(catList)
			} else {
				return condition.IntConst0
			}
		}, varOperand)
}

type timeRange struct {
	start time.Time
	end   time.Time
}

func (repo *CompareCondRepo) evalIsInConstantListWithDateRange(
	valOperand condition.Operand,
	dateOperand condition.Operand,
	consOperandList []condition.Operand) condition.Operand {

	valueMap := types.NewHashMap[condition.Operand, timeRange]()

	for i := 0; i < len(consOperandList); i += 3 {
		end := i + 3

		// prevent exceeding slice bounds
		if end > len(consOperandList) {
			return condition.NewErrorOperand(repo.ctx.LogError(fmt.Errorf("invalid date range")))
		}

		value := consOperandList[i]
		date1 := consOperandList[i+1].Convert(condition.TimeOperandKind)
		if date1.GetHash() == condition.ErrorOperandKind {
			return date1
		}
		date2 := consOperandList[i+2].Convert(condition.TimeOperandKind)
		if date2.GetHash() == condition.ErrorOperandKind {
			return date2
		}
		valueMap.Put(value, timeRange{time.Time(date1.(condition.TimeOperand)), time.Time(date2.(condition.TimeOperand))})
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			X := valOperand.Evaluate(event, frames)
			xKind := X.GetKind()
			if xKind == condition.NullOperandKind {
				// Special case for TMSIS. May want to have a more general solution.
				return condition.NewBooleanOperand(true)
			}
			if xKind == condition.ErrorOperandKind {
				return X
			}
			Y := dateOperand.Evaluate(event, frames)
			yKind := Y.GetKind()
			if yKind == condition.ErrorOperandKind {
				return Y
			}
			var date condition.Operand
			if yKind != condition.NullOperandKind {
				date = Y.Convert(condition.TimeOperandKind)
				if date.GetKind() != condition.TimeOperandKind {
					return condition.NewErrorOperand(repo.ctx.LogError(fmt.Errorf("invalid date range")))
				}
			}
			dateRange, k := valueMap.Get(X)
			if k {
				if yKind == condition.NullOperandKind {
					// Special case for TMSIS. Missing date is considered to be in the range.
					return condition.NewBooleanOperand(true)
				}
				return condition.NewBooleanOperand(
					!(dateRange.start.After(time.Time(date.(condition.TimeOperand))) ||
						dateRange.end.Before(time.Time(date.(condition.TimeOperand)))))
			} else {
				return condition.NewBooleanOperand(false)
			}
		}, valOperand, dateOperand)
}

func (repo *CompareCondRepo) genEvalForCompareCondition(
	compareCond *condition.CompareCondition, scope *ForEachScope) condition.Operand {

	lOperand := repo.evalOperandAccess(repo.evalOperandAddress(compareCond.LeftOperand, scope), scope)
	if lOperand.GetKind() == condition.ErrorOperandKind {
		return lOperand
	}

	rOperand := repo.evalOperandAccess(repo.evalOperandAddress(compareCond.RightOperand, scope), scope)
	if rOperand.GetKind() == condition.ErrorOperandKind {
		return rOperand
	}
	return repo.genEvalForCompareOperands(compareCond.CompareOp, lOperand, rOperand)
}

func (repo *CompareCondRepo) genEvalForExprCondition(
	exprCondition *condition.ExprCondition, scope *ForEachScope) condition.Operand {
	// Convert the expression to an AST node tree
	node, err := parser.ParseExpr(exprCondition.Expr)

	if err != nil {
		return condition.NewErrorOperand(repo.ctx.LogError(err))
	}
	return repo.evalAstNode(node, scope)
}

func (repo *CompareCondRepo) processCompareCondition(
	compareCond *condition.CompareCondition, scope *ForEachScope) condition.Condition {
	if scope.ParentScope != nil {
		panic("must be called from root scope")
	}

	oldEvalCondRec, ok := repo.CondToCompareCondRecord.Get(compareCond)
	evalCatRec := scope.Evaluator
	if ok {
		if evalCatRec != nil {
			// We have seen another condition identical to this one.  Use its category instead of the new one.
			repo.DiscardEvalCategoryRec(evalCatRec)

			// Remove address to the new eval record registration too.
			repo.unregisterCatEvaluator(evalCatRec)
			scope.Evaluator = nil
		}
		evalCatRec = oldEvalCondRec
	} else {
		if evalCatRec == nil {
			evalCatRec = repo.NewEvalCategoryRec(nil)
			// Capture evaluator record so that we can register nested attribute access against it.
			scope.Evaluator = evalCatRec
			defer scope.ResetEvaluator()
		}

		var eval condition.Operand
		// Special case equal compare against a constant that can be done via a hash lookup
		if compareCond.CompareOp == condition.CompareEqualOp &&
			(compareCond.LeftOperand.IsConst() || compareCond.RightOperand.IsConst()) {
			eval = repo.processCompareEqualToConstCondition(compareCond, scope)
		} else {
			eval = repo.genEvalForCompareCondition(compareCond, scope)
		}
		if eval != nil && eval.GetKind() == condition.ErrorOperandKind {
			return condition.NewErrorCondition(eval.(condition.ErrorOperand))
		}
		evalCatRec.Eval = eval

		repo.CondToCompareCondRecord.Put(compareCond, evalCatRec)
	}
	return condition.NewCategoryCond(evalCatRec.GetCategory())
}

// ForEachScope keeps track of the local scope data.
// Each for_all, for_some or for_each filter element starts a new scope with its index element
// and path pointing to the data attribute over which the element iterates in this scope.
// The scope's element is available from all the nested scopes both during the filter build time
// as the parentScope parameter and at runtime via array of attribute addresses, one for each
// ancestor scope.
type ForEachScope struct {
	// Path, e.g. $.members or $member.children.
	// Each path consist of the outer path reference (e.g. $ for root or $member for path to
	// the member element) and the path to a nested array element.  The address of the denoted
	// element is therefore a concatenation of the address of the parent element's path
	// and the address of the path of the current element.
	Path         string
	Element      string
	NestingLevel int
	ParentScope  *ForEachScope
	Evaluator    *EvalCategoryRec
	AttrDictRec  *objectmap.AttrDictionaryRec
}

func (scope *ForEachScope) ResetEvaluator() {
	scope.Evaluator = nil
}

type EvalCondFunc func(event *objectmap.ObjectAttributeMap, frames []interface{}) (bool, error)

func (repo *CompareCondRepo) genEvalForAndCondition(
	cond *condition.AndCond, parentScope *ForEachScope) condition.Operand {
	var condEvaluators []*condition.ExprOperand
	for _, c := range cond.Operands {
		eval := repo.genEvalForCondition(c, parentScope)
		if eval.GetKind() == condition.ErrorOperandKind {
			return eval
		}
		condEvaluators = append(condEvaluators, eval.(*condition.ExprOperand))
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			for _, eval := range condEvaluators {
				result := eval.Func(event, frames)
				if result.GetKind() == condition.ErrorOperandKind {
					return result
				} else if !result.(condition.BooleanOperand) {
					return condition.NewBooleanOperand(false)
				}
			}
			// Return true unless at least one is false
			return condition.NewBooleanOperand(true)
		}, types.MapSlice(condEvaluators, func(o *condition.ExprOperand) condition.Operand { return o })...)
}

func (repo *CompareCondRepo) genEvalForOrCondition(
	cond *condition.OrCond, parentScope *ForEachScope) condition.Operand {
	var condEvaluators []*condition.ExprOperand
	for _, c := range cond.Operands {
		eval := repo.genEvalForCondition(c, parentScope)
		if eval.GetKind() == condition.ErrorOperandKind {
			return eval
		}
		condEvaluators = append(condEvaluators, eval.(*condition.ExprOperand))
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			for _, eval := range condEvaluators {
				result := eval.Func(event, frames)
				if result.GetKind() == condition.ErrorOperandKind {
					return result
				} else if result.(condition.BooleanOperand) {
					return condition.NewBooleanOperand(true)
				}
			}
			// Return true unless at least one is false
			return condition.NewBooleanOperand(false)
		}, types.MapSlice(condEvaluators, func(o *condition.ExprOperand) condition.Operand { return o })...)
}

func (repo *CompareCondRepo) genEvalForNotCondition(
	cond *condition.NotCond, parentScope *ForEachScope) condition.Operand {
	eval := repo.genEvalForCondition(cond.Operand, parentScope)
	if eval.GetKind() == condition.ErrorOperandKind {
		return eval
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			result := eval.Evaluate(event, frames)
			if result.GetKind() == condition.ErrorOperandKind {
				return result
			} else {
				return condition.NewBooleanOperand(!bool(result.(condition.BooleanOperand)))
			}
		}, eval)
}

func (repo *CompareCondRepo) genEvalForCondition(
	c condition.Condition, parentScope *ForEachScope) condition.Operand {
	switch c.GetKind() {
	case condition.AndCondKind:
		return repo.genEvalForAndCondition(c.(*condition.AndCond), parentScope)
	case condition.OrCondKind:
		return repo.genEvalForOrCondition(c.(*condition.OrCond), parentScope)
	case condition.NotCondKind:
		return repo.genEvalForNotCondition(c.(*condition.NotCond), parentScope)
	case condition.CategoryCondKind:
		panic("CategoryCondKind not expected")
	case condition.CompareCondKind:
		return repo.genEvalForCompareCondition(c.(*condition.CompareCondition), parentScope)
	case condition.ExprCondKind:
		return repo.genEvalForExprCondition(c.(*condition.ExprCondition), parentScope)
	default:
		panic(fmt.Sprintf("should not happen %v", c.GetKind()))
	}
}

func (repo *CompareCondRepo) setupEvalForEach(parentScope *ForEachScope, element string, path string) (
	*objectmap.AttributeAddress, *ForEachScope, error) {
	// arrayAddress points to the array
	if arrayAddress, err := getAttributePathAddress(path+"[]", parentScope); err != nil {
		return nil, nil, err
	} else {
		arrPath := path + "[]"
		scope, ePath := expandPath(arrPath, parentScope)
		addr, err := scope.AttrDictRec.AttributePathToAddress(ePath)
		if err != nil {
			return nil, nil, err
		}
		newDictRec := scope.AttrDictRec.AddressToDictionaryRec(addr)

		newPath := scope.AttrDictRec.AddressToFullPath(addr)
		if arrayAddress.ParentParameterIndex != scope.NestingLevel {
			panic("why?")
		}
		newScope := &ForEachScope{
			// ARRAY_ELEMENT issue
			Path:         newPath, // add explicit indexing to the path
			Element:      element,
			NestingLevel: parentScope.NestingLevel + 1,
			ParentScope:  parentScope,
			AttrDictRec:  newDictRec}

		return &objectmap.AttributeAddress{
			Address:              addr,
			Path:                 newPath,
			ParentParameterIndex: scope.NestingLevel,
			FullAddress:          scope.AttrDictRec.AddressToFullAddress(addr)}, newScope, nil
	}
}

func (repo *CompareCondRepo) genEvalForAllCondition(
	path string, element string, cond condition.Condition, parentScope *ForEachScope) condition.Operand {

	arrayAddress, newScope, err := repo.setupEvalForEach(parentScope, element, path)
	if err != nil {
		return condition.NewErrorOperand(err)
	} else {
		nestingLevel := newScope.NestingLevel

		eval := repo.genEvalForCondition(cond, newScope)
		if eval.GetKind() == condition.ErrorOperandKind {
			return eval
		}

		return repo.CondFactory.NewExprOperand(
			func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
				numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
				if err != nil {
					return condition.NewErrorOperand(err)
				}

				parentsFrame := frames[arrayAddress.ParentParameterIndex]
				currentAddressLen := len(arrayAddress.Address)
				currentAddress := types.GetIntSlice()
				currentAddress = append(currentAddress, arrayAddress.Address...)
				currentAddress = append(currentAddress, 0)
				var result condition.Operand = condition.NewBooleanOperand(true)
				for i := 0; i < numElements; i++ {
					currentAddress[currentAddressLen] = i
					newFrame := objectmap.GetNestedAttributeByAddress(parentsFrame, currentAddress)
					if newFrame == nil {
						// TODO: record diagnostics somewhere that attribute is not available
						// we don't want to do for all, maybe have a metric of how many times the attribute
						// could not be accessed
						result = condition.NewBooleanOperand(false)
						break
					}
					frames[nestingLevel] = newFrame
					result = eval.Evaluate(event, frames)
					if result.GetKind() == condition.ErrorOperandKind {
						break
					} else if !result.(condition.BooleanOperand) {
						break
					}
				}
				// Return true unless at least one is false
				types.PutIntSlice(currentAddress)
				return result
			}, eval)
	}
}

func (repo *CompareCondRepo) processForAllCondition(
	path string, element string, cond condition.Condition, parentScope *ForEachScope) condition.Condition {
	dummyCond := condition.NewAndCond(condition.NewExprCondition("forAll"), condition.NewExprCondition(path), condition.NewExprCondition(element), cond)
	evalCatRec, ok := repo.CondToCompareCondRecord.Get(dummyCond)
	if !ok {
		eval := repo.genEvalForAllCondition(path, element, cond, parentScope)

		if eval.GetKind() == condition.ErrorOperandKind {
			return condition.NewErrorCondition(eval.(condition.ErrorOperand))
		}
		evalCatRec = repo.NewEvalCategoryRec(eval)
		repo.CondToCompareCondRecord.Put(dummyCond, evalCatRec)
		// ARRAY_ELEMENT issue
		if arrayAddress, err := getAttributePathAddress(path+"[]", parentScope); err != nil {
			panic("should not happen: failed the check that passed earlier")
		} else {
			repo.registerCatEvaluatorForAddress(arrayAddress.FullAddress, evalCatRec)
		}
	}
	return condition.NewCategoryCond(evalCatRec.GetCategory())
}

func (repo *CompareCondRepo) genEvalForSomeCondition(
	path string, element string, cond condition.Condition, parentScope *ForEachScope) condition.Operand {
	// ARRAY_ELEMENT issue
	arrayAddress, newScope, err := repo.setupEvalForEach(parentScope, element, path)
	if err != nil {
		return condition.NewErrorOperand(err)
	} else {
		nestingLevel := newScope.NestingLevel
		eval := repo.genEvalForCondition(cond, newScope)
		if eval.GetKind() == condition.ErrorOperandKind {
			return eval
		}

		return repo.CondFactory.NewExprOperand(
			func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
				numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
				if err != nil {
					return condition.NewErrorOperand(err)
				}

				parentsFrame := frames[arrayAddress.ParentParameterIndex]
				currentAddressLen := len(arrayAddress.Address)
				currentAddress := types.GetIntSlice()
				currentAddress = append(currentAddress, arrayAddress.Address...)
				currentAddress = append(currentAddress, 0)
				var result condition.Operand = condition.NewBooleanOperand(false)
				for i := 0; i < numElements; i++ {
					currentAddress[currentAddressLen] = i
					newFrame := objectmap.GetNestedAttributeByAddress(parentsFrame, currentAddress)
					if newFrame == nil {
						// TODO: record diagnostics somewhere that attribute is not available
						// we don't want to do for all, maybe have a metric of how many times the attribute
						// could not be accessed
						break
					}
					frames[nestingLevel] = newFrame
					result = eval.Evaluate(event, frames)
					if result.GetKind() == condition.ErrorOperandKind {
						break
					} else if result.(condition.BooleanOperand) {
						break
					}
				}
				// Return true unless at least one is false
				types.PutIntSlice(currentAddress)

				return result
			}, eval)
	}
}

func (repo *CompareCondRepo) processForSomeCondition(
	path string, element string, cond condition.Condition, parentScope *ForEachScope) condition.Condition {
	dummyCond := condition.NewAndCond(condition.NewExprCondition("forSome"), condition.NewExprCondition(path), condition.NewExprCondition(element), cond)
	evalCatRec, ok := repo.CondToCompareCondRecord.Get(dummyCond)
	if !ok {
		eval := repo.genEvalForSomeCondition(path, element, cond, parentScope)
		if eval.GetKind() == condition.ErrorOperandKind {
			return condition.NewErrorCondition(eval.(condition.ErrorOperand))
		}

		evalCatRec = repo.NewEvalCategoryRec(eval)
		repo.CondToCompareCondRecord.Put(dummyCond, evalCatRec)
		// ARRAY_ELEMENT issue
		if arrayAddress, err := getAttributePathAddress(path+"[]", parentScope); err != nil {
			//if elementAddress, err := getAttributePathAddress(repo, cond.Path, parentScope); err != nil {
			panic("should not happen: failed the check that passed earlier")
		} else {
			repo.registerCatEvaluatorForAddress(arrayAddress.FullAddress, evalCatRec)
		}
	}
	return condition.NewCategoryCond(evalCatRec.GetCategory())
}

func negateIfTrue(cond condition.Condition, negate bool) condition.Condition {
	if negate {
		return condition.NewNotCond(cond)
	} else {
		return cond
	}
}

func (repo *CompareCondRepo) processCondNode(node ast.Node, negate bool, scope *ForEachScope) condition.Condition {
	switch n := node.(type) {
	case *ast.CallExpr:
		funcName := n.Fun.(*ast.Ident).Name
		switch funcName {
		case "regexpMatch":
			return negateIfTrue(repo.processBoolFunc(funcRegexpMatch, n, scope), negate)
		case "hasValue":
			return negateIfTrue(repo.processBoolFunc(funcHasValue, n, scope), negate)
		case "isEqualToAnyWithDate":
			return negateIfTrue(repo.processBoolFunc(funcIsEqualToAnyWithDate, n, scope), negate)
		case "isEqualToAny":
			return negateIfTrue(repo.processIsEqualToAny(n, scope), negate)
		case "containsAny":
			return negateIfTrue(repo.processContains(n, scope), negate)
		case "forAll":
			return negateIfTrue(repo.processForAllFunc(n, scope), negate)
		case "forSome":
			return negateIfTrue(repo.processForSomeFunc(n, scope), negate)
		default:
			return condition.NewErrorCondition(fmt.Errorf("unsupported function: %s", funcName))
		}
	case *ast.BinaryExpr:
		if negate {
			switch n.Op {
			case token.LAND:
				return repo.CondFactory.NewOrCond(repo.processCondNode(n.X, true, scope), repo.processCondNode(n.Y, true, scope))
			case token.LOR:
				return repo.CondFactory.NewAndCond(repo.processCondNode(n.X, true, scope), repo.processCondNode(n.Y, true, scope))
			case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
				return repo.processCompareBinaryExpr(n, negate, scope)
			default:
				return condition.NewErrorCondition(
					repo.ctx.Errorf("unsupported operator: %s", n.Op.String()))
			}
		} else {
			switch n.Op {
			case token.LAND:
				return repo.CondFactory.NewAndCond(repo.processCondNode(n.X, false, scope), repo.processCondNode(n.Y, false, scope))
			case token.LOR:
				return repo.CondFactory.NewOrCond(repo.processCondNode(n.X, false, scope), repo.processCondNode(n.Y, false, scope))
			case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
				return repo.processCompareBinaryExpr(n, negate, scope)
			default:
				return condition.NewErrorCondition(
					repo.ctx.Errorf("unsupported operator: %s", n.Op.String()))
			}
		}
	case *ast.UnaryExpr:
		switch n.Op {
		case token.NOT:
			return repo.processCondNode(n.X, !negate, scope)
		default:
			return condition.NewErrorCondition(
				repo.ctx.Errorf("unsupported operator: %s", n.Op.String()))
		}
	case *ast.ParenExpr:
		return repo.processCondNode(n.X, negate, scope)
	default:
		return condition.NewErrorCondition(
			repo.ctx.Errorf("unsupported node type: %v", node))
	}
}

type boolFuncT func(repo *CompareCondRepo, n *ast.CallExpr, scope *ForEachScope) condition.Operand

func (repo *CompareCondRepo) processBoolFunc(boolFunc boolFuncT, n *ast.CallExpr, scope *ForEachScope) condition.Condition {
	evalCatRec := repo.NewEvalCategoryRec(nil)
	if scope.Evaluator != nil {
		panic("Should not happen")
	}
	// Set evaluator record so that we can register nested attribute addresses access against it.
	scope.Evaluator = evalCatRec
	defer scope.ResetEvaluator()

	resultOperand := boolFunc(repo, n, scope)
	if resultOperand.GetKind() == condition.ErrorOperandKind {
		return condition.NewErrorCondition(resultOperand.(condition.ErrorOperand))
	}

	return repo.processCompareCondition(condition.NewCompareCond(condition.CompareEqualOp, resultOperand, condition.NewBooleanOperand(true)), scope)
}

func (repo *CompareCondRepo) processIsEqualToAny(n *ast.CallExpr, scope *ForEachScope) condition.Condition {
	evalCatRec := repo.NewEvalCategoryRec(nil)
	if scope.Evaluator != nil {
		panic("Should not happen")
	}
	// Set evaluator record so that we can register nested attribute addresses access against it.
	scope.Evaluator = evalCatRec
	defer scope.ResetEvaluator()

	if len(n.Args) < 2 {
		return condition.NewErrorCondition(fmt.Errorf("wrong number of arguments for isEqualToAny() function"))
	}

	argOperands := types.MapSlice(n.Args, func(o ast.Expr) condition.Operand { return repo.evalAstNode(o, scope) })
	firstErrorOperand := types.FindFirstInSlice(
		argOperands, func(o condition.Operand) bool { return o.GetKind() == condition.ErrorOperandKind })
	if firstErrorOperand != nil {
		return condition.NewErrorCondition((*firstErrorOperand).(*condition.ErrorOperand).Err)
	}

	constOperands := types.FilterSlice(argOperands[1:], func(o condition.Operand) bool { return o.IsConst() })

	// See if we are comparing against const values. TODO: test this.
	if argOperands[0].IsConst() {
		for _, argOperand := range constOperands {
			if argOperands[0].Equals(argOperand) {
				// DefaultToTrue by apply NOT to a category that should never trigger
				return condition.NewNotCond(condition.NewCategoryCond(evalCatRec.GetCategory()))
			}
		}
		if len(constOperands) == len(argOperands)-1 {
			// All operands are constant and do not match
			return condition.NewCategoryCond(evalCatRec.GetCategory())
		}
	}

	if len(constOperands) != len(argOperands)-1 {
		// Not all operands in the match list are constants
		return condition.NewErrorCondition(fmt.Errorf("isEqualToAny() only supports constant match list"))
	}

	eval := repo.processEvalForIsInConstantList(argOperands[0], argOperands[1:], scope)

	if eval != nil && eval.GetKind() == condition.ErrorOperandKind {
		return condition.NewErrorCondition(eval.(condition.ErrorOperand))
	}
	evalCatRec.Eval = eval

	return condition.NewCategoryCond(evalCatRec.GetCategory())
}

func (repo *CompareCondRepo) processForEachFunc(n *ast.CallExpr, kind string, scope *ForEachScope) condition.Condition {
	pathOperand, elementOperand, exprCond, err := repo.setupForEachOperands(n, scope)
	if err != nil {
		return condition.NewErrorCondition(err)
	}

	switch kind {
	case "all":
		return repo.processForAllCondition(
			string(pathOperand.(condition.StringOperand)),
			string(elementOperand.(condition.StringOperand)),
			exprCond,
			scope)
	case "some":
		return repo.processForSomeCondition(
			string(pathOperand.(condition.StringOperand)),
			string(elementOperand.(condition.StringOperand)),
			exprCond,
			scope)

	default:
		panic(fmt.Sprintf("Unknown kind %s", kind))
	}
}

func (repo *CompareCondRepo) setupForEachOperands(n *ast.CallExpr, scope *ForEachScope) (
	condition.Operand, condition.Operand, condition.Condition, error) {
	if len(n.Args) != 3 {
		return nil, nil, nil, fmt.Errorf("wrong number of arguments for forAll() function")
	}

	pathOperand := repo.evalAstNode(n.Args[0], scope)
	if pathOperand.GetKind() == condition.ErrorOperandKind {
		return nil, nil, nil, pathOperand.(condition.ErrorOperand).Err
	}

	if pathOperand.GetKind() != condition.StringOperandKind {
		return nil, nil, nil, fmt.Errorf("forAll() only supports string path")
	}

	elementOperand := repo.evalAstNode(n.Args[1], scope)
	if elementOperand.GetKind() == condition.ErrorOperandKind {
		return nil, nil, nil, elementOperand.(condition.ErrorOperand).Err
	}

	if elementOperand.GetKind() != condition.StringOperandKind {
		return nil, nil, nil, fmt.Errorf("forAll() only supports string element operand")
	}

	// We can't process the expression here because we have not evaluated the parent scope yet, which is defined by
	// the path and element operands. We will process the expression in the child scope.
	// First convert the ast expression back to string.
	// Then construct a ForAllCond with the string expression and pass it back to ConvertToCategoryCondition.
	var buf bytes.Buffer
	err := printer.Fprint(&buf, token.NewFileSet(), n.Args[2])
	if err != nil {
		return nil, nil, nil, err
	}
	exprCond := condition.NewExprCondition(buf.String())
	return pathOperand, elementOperand, exprCond, nil
}

func (repo *CompareCondRepo) processForAllFunc(n *ast.CallExpr, scope *ForEachScope) condition.Condition {
	pathOperand, elementOperand, exprCond, err := repo.setupForEachOperands(n, scope)
	if err != nil {
		return condition.NewErrorCondition(err)
	}
	return repo.processForAllCondition(
		string(pathOperand.(condition.StringOperand)),
		string(elementOperand.(condition.StringOperand)),
		exprCond, scope)
}

func (repo *CompareCondRepo) processForSomeFunc(n *ast.CallExpr, scope *ForEachScope) condition.Condition {
	pathOperand, elementOperand, exprCond, err := repo.setupForEachOperands(n, scope)
	if err != nil {
		return condition.NewErrorCondition(err)
	}
	return repo.processForSomeCondition(
		string(pathOperand.(condition.StringOperand)),
		string(elementOperand.(condition.StringOperand)),
		exprCond, scope)
}

func (repo *CompareCondRepo) funcForSome(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	pathOperand, elementOperand, exprCond, err := repo.setupForEachOperands(n, scope)
	if err != nil {
		return condition.NewErrorOperand(err)
	}
	return repo.genEvalForSomeCondition(
		string(pathOperand.(condition.StringOperand)),
		string(elementOperand.(condition.StringOperand)),
		exprCond,
		scope)
}

func (repo *CompareCondRepo) funcForAll(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	pathOperand, elementOperand, exprCond, err := repo.setupForEachOperands(n, scope)
	if err != nil {
		return condition.NewErrorOperand(err)
	}
	return repo.genEvalForAllCondition(
		string(pathOperand.(condition.StringOperand)),
		string(elementOperand.(condition.StringOperand)),
		exprCond,
		scope)
}

func (repo *CompareCondRepo) processContains(n *ast.CallExpr, scope *ForEachScope) condition.Condition {
	evalCatRec := repo.NewEvalCategoryRec(nil)
	if scope.Evaluator != nil {
		panic("Should not happen")
	}
	// Set evaluator record so that we can register nested attribute addresses access against it.
	scope.Evaluator = evalCatRec
	defer scope.ResetEvaluator()

	if len(n.Args) < 2 {
		return condition.NewErrorCondition(fmt.Errorf("wrong number of arguments for containsAny() function"))
	}

	argOperands := types.MapSlice(n.Args,
		func(o ast.Expr) condition.Operand { return repo.evalAstNode(o, scope) })
	firstErrorOperand := types.FindFirstInSlice(
		argOperands, func(o condition.Operand) bool { return o.GetKind() == condition.ErrorOperandKind })
	if firstErrorOperand != nil {
		return condition.NewErrorCondition((*firstErrorOperand).(*condition.ErrorOperand).Err)
	}

	constOperands := types.FilterSlice(argOperands[1:], func(o condition.Operand) bool { return o.IsConst() })
	if len(constOperands) != len(argOperands)-1 {
		// Not all operands in the match list are constants
		return condition.NewErrorCondition(fmt.Errorf("containsAny() only supports constant string match list"))
	}
	constStringOperands := types.MapSlice(constOperands,
		func(o condition.Operand) string {
			return string(o.Convert(condition.StringOperandKind).(condition.StringOperand))
		})
	if len(constStringOperands) != len(constOperands) {
		// Not all operands in the match list are strings
		return condition.NewErrorCondition(fmt.Errorf("containsAny() only supports constant string match list"))
	}

	eval := repo.processEvalForContains(argOperands[0], constStringOperands, scope)

	if eval != nil && eval.GetKind() == condition.ErrorOperandKind {
		return condition.NewErrorCondition(eval.(condition.ErrorOperand))
	}
	evalCatRec.Eval = eval

	return condition.NewCategoryCond(evalCatRec.GetCategory())
}

func (repo *CompareCondRepo) processCompareBinaryExpr(n *ast.BinaryExpr, negate bool, scope *ForEachScope) condition.Condition {
	var compareOp condition.CompareOp
	if negate {
		negate = false
		switch n.Op {
		case token.EQL:
			compareOp = condition.CompareEqualOp
			negate = true
		case token.LSS:
			compareOp = condition.CompareGreaterOrEqualOp
		case token.GTR:
			compareOp = condition.CompareLessOrEqualOp
		case token.NEQ:
			compareOp = condition.CompareEqualOp
		case token.LEQ:
			compareOp = condition.CompareGreaterOp
		case token.GEQ:
			compareOp = condition.CompareLessOp
		default:
			return condition.NewErrorCondition(
				repo.ctx.Errorf("unsupported operator: %s", n.Op.String()))
		}
	} else {
		switch n.Op {
		case token.EQL:
			compareOp = condition.CompareEqualOp
		case token.LSS:
			compareOp = condition.CompareLessOp
		case token.GTR:
			compareOp = condition.CompareGreaterOp
		case token.NEQ:
			negate = !negate
			compareOp = condition.CompareEqualOp
		case token.LEQ:
			compareOp = condition.CompareLessOrEqualOp
		case token.GEQ:
			compareOp = condition.CompareGreaterOrEqualOp
		default:
			return condition.NewErrorCondition(
				repo.ctx.Errorf("unsupported operator: %s", n.Op.String()))
		}
	}

	evalCatRec := repo.NewEvalCategoryRec(nil)
	if scope.Evaluator != nil {
		panic("Should not happen")
	}
	// Set evaluator record so that we can register nested attribute addresses access against it.
	scope.Evaluator = evalCatRec
	defer scope.ResetEvaluator()

	xOperand := repo.evalAstNode(n.X, scope)
	if xOperand.GetKind() == condition.ErrorOperandKind {
		return condition.NewErrorCondition(xOperand.(condition.ErrorOperand))
	}

	yOperand := repo.evalAstNode(n.Y, scope)
	if yOperand.GetKind() == condition.ErrorOperandKind {
		return condition.NewErrorCondition(yOperand.(condition.ErrorOperand))
	}

	if negate {
		return condition.NewNotCond(repo.processCompareCondition(condition.NewCompareCond(compareOp, xOperand, yOperand), scope))
	} else {
		return repo.processCompareCondition(condition.NewCompareCond(compareOp, xOperand, yOperand), scope)
	}
}

func (repo *CompareCondRepo) processExprCondition(exprCondition *condition.ExprCondition, scope *ForEachScope) condition.Condition {
	if scope.ParentScope != nil {
		panic("must be called from root scope")
	}

	// Convert the expression to an AST node tree
	node, err := parser.ParseExpr(exprCondition.Expr)

	if err != nil {
		return condition.NewErrorCondition(repo.ctx.LogError(err))
	}

	// Process the AST node tree representation.
	// We will first recursively decent through logical operations to the nodes that
	// would evaluate to category conditions.
	return repo.processCondNode(node, false, scope)
}

// preprocessAstExpr: convert ast expression to condition.Operand
func (repo *CompareCondRepo) preprocessAstExpr(node ast.Expr, scope *ForEachScope) condition.Operand {
	switch n := node.(type) {
	case *ast.BasicLit:
		switch n.Kind {
		case token.INT, token.FLOAT:
			val, err := strconv.ParseFloat(n.Value, 64)
			if err != nil {
				return condition.NewErrorOperand(err)
			}
			return repo.CondFactory.NewFloatOperand(val)
		case token.STRING:
			unquotedStr, err := strconv.Unquote(n.Value)
			if err != nil {
				return condition.NewErrorOperand(fmt.Errorf("unable to unquote \"%s\"", n.Value))
			}
			return repo.CondFactory.NewStringOperand(unquotedStr)
		}
	case *ast.Ident:
		return repo.CondFactory.NewSelOperand(nil, n.Name)
	case *ast.SelectorExpr:
		x := repo.preprocessAstExpr(n.X, scope)

		switch x.GetKind() {
		case condition.ErrorOperandKind:
			return x
		case condition.SelOperandKind:
			return repo.CondFactory.NewSelOperand(
				x.(*condition.SelOperand).Base,
				x.(*condition.SelOperand).Selector+"."+n.Sel.Name)
		case condition.IndexOperandKind, condition.AddressOperandKind:
			return repo.CondFactory.NewSelOperand(x, n.Sel.Name)
		default:
			panic("should not get here")
		}
	case *ast.IndexExpr:
		x := repo.preprocessAstExpr(n.X, scope)

		i := repo.preprocessAstExpr(n.Index, scope)
		if i.GetKind() == condition.ErrorOperandKind {
			return i
		}

		switch x.GetKind() {
		case condition.ErrorOperandKind:
			return x
		case condition.SelOperandKind:
			return repo.CondFactory.NewIndexOperand(
				repo.CondFactory.NewSelOperand(
					x.(*condition.SelOperand).Base,
					x.(*condition.SelOperand).Selector+"[]"), i)
		case condition.IndexOperandKind:
			return repo.CondFactory.NewIndexOperand(repo.CondFactory.NewSelOperand(x, "[]"), i)
		default:
			panic("should not get here")
		}
	case *ast.ParenExpr:
		return repo.evalAstNode(n.X, scope)
	case *ast.CallExpr:
		funcName := n.Fun.(*ast.Ident).Name
		switch funcName {
		case "date":
			return repo.convertToType(n, scope, condition.TimeOperandKind)
		case "string":
			return repo.convertToType(n, scope, condition.StringOperandKind)
		case "int":
			return repo.convertToType(n, scope, condition.IntOperandKind)
		case "float":
			return repo.convertToType(n, scope, condition.FloatOperandKind)
		case "regexpMatch":
			return funcRegexpMatch(repo, n, scope)
		case "hasValue":
			return funcHasValue(repo, n, scope)
		case "isEqualToAnyWithDate":
			return funcIsEqualToAnyWithDate(repo, n, scope)
		case "isEqualToAny":
			return repo.funcIsEqualToAny(n, scope)
		case "forAll":
			return repo.funcForAll(n, scope)
		case "forSome":
			return repo.funcForSome(n, scope)
		case "sqrt":
			argOperand := repo.evalAstNode(n.Args[0], scope)
			if argOperand.GetKind() == condition.ErrorOperandKind {
				return argOperand
			}
			return repo.CondFactory.NewExprOperand(
				func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
					arg := argOperand.Evaluate(event, frames)
					if arg.GetKind() == condition.ErrorOperandKind {
						return arg
					}
					arg = arg.Convert(condition.FloatOperandKind)
					return condition.NewFloatOperand(math.Sqrt(float64(arg.(condition.FloatOperand))))
				}, argOperand, condition.StringOperand(funcName)) // funcName as hash seed to avoid cache collisions
		default:
			return condition.NewErrorOperand(fmt.Errorf("unsupported function: %s", funcName))
		}
	case *ast.BinaryExpr:
		xOperand := repo.evalAstNode(n.X, scope)
		if xOperand.GetKind() == condition.ErrorOperandKind {
			return xOperand
		}
		yOperand := repo.evalAstNode(n.Y, scope)
		if yOperand.GetKind() == condition.ErrorOperandKind {
			return yOperand
		}

		switch n.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO:
			return repo.CondFactory.NewExprOperand(
				func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
					xVal := xOperand.Evaluate(event, frames).Convert(condition.FloatOperandKind)
					if xVal.GetKind() == condition.ErrorOperandKind {
						return xVal
					}
					lv := float64(xVal.(condition.FloatOperand))

					yVal := yOperand.Evaluate(event, frames).Convert(condition.FloatOperandKind)
					if yVal.GetKind() == condition.ErrorOperandKind {
						return yVal
					}
					rv := float64(yVal.(condition.FloatOperand))

					switch n.Op {
					case token.ADD:
						return condition.NewFloatOperand(lv + rv)
					case token.SUB:
						return condition.NewFloatOperand(lv - rv)
					case token.MUL:
						return condition.NewFloatOperand(lv * rv)
					case token.QUO:
						return condition.NewFloatOperand(lv / rv)
					default:
						return condition.NewErrorOperand(fmt.Errorf("unsupported operator: %s", n.Op.String()))
					}
				}, xOperand, yOperand)
		case token.EQL:
			return repo.genEvalForCompareOperands(condition.CompareEqualOp, xOperand, yOperand)
		case token.LSS:
			return repo.genEvalForCompareOperands(condition.CompareLessOp, xOperand, yOperand)
		case token.GTR:
			return repo.genEvalForCompareOperands(condition.CompareGreaterOp, xOperand, yOperand)
		case token.NEQ:
			return repo.genEvalForCompareOperands(condition.CompareNotEqualOp, xOperand, yOperand)
		case token.LEQ:
			return repo.genEvalForCompareOperands(condition.CompareLessOrEqualOp, xOperand, yOperand)
		case token.GEQ:
			return repo.genEvalForCompareOperands(condition.CompareGreaterOrEqualOp, xOperand, yOperand)
		case token.LAND:
			return repo.genEvalForLogicalOp(token.LAND, xOperand, yOperand)
		case token.LOR:
			return repo.genEvalForLogicalOp(token.LOR, xOperand, yOperand)
		default:
			return condition.NewErrorOperand(fmt.Errorf("unsupported operator: %s", n.Op.String()))
		}
	case *ast.UnaryExpr:
		switch n.Op {
		case token.NOT:
			xOperand := repo.evalAstNode(n.X, scope)
			if xOperand.GetKind() == condition.ErrorOperandKind {
				return xOperand
			}
			return repo.genEvalForCompareOperands(
				condition.CompareNotEqualOp, xOperand, condition.NewBooleanOperand(true))
		default:
			return condition.NewErrorOperand(
				repo.ctx.Errorf("unsupported operator: %s", n.Op.String()))
		}
	}

	return repo.CondFactory.NewErrorOperand(fmt.Errorf("unsupported node type: %T", node))
}

func (repo *CompareCondRepo) convertToType(n *ast.CallExpr, scope *ForEachScope, operandKind condition.OperandKind) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("wrong number of arguments for float conversion"))
	}
	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}
	if argOperand.IsConst() {
		return argOperand.Convert(operandKind)
	}
	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)
			if arg.GetKind() == condition.ErrorOperandKind {
				return arg
			}
			return arg.Convert(operandKind)
		}, argOperand, condition.IntOperand(operandKind)) // operandKind as hash seed to avoid cache collisions
}

func funcIsEqualToAnyWithDate(repo *CompareCondRepo, n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) < 5 {
		return condition.NewErrorOperand(fmt.Errorf("wrong number of arguments for isEqualtoAnyWithDate() function"))
	}

	if len(n.Args)%3 != 2 {
		return condition.NewErrorOperand(fmt.Errorf("wrong number of arguments for isEqualtoAnyWithDate() function"))
	}

	argOperands := types.MapSlice(n.Args, func(o ast.Expr) condition.Operand { return repo.evalAstNode(o, scope) })
	firstErrorOperand := types.FindFirstInSlice(
		argOperands, func(o condition.Operand) bool { return o.GetKind() == condition.ErrorOperandKind })
	if firstErrorOperand != nil {
		return condition.NewErrorOperand((*firstErrorOperand).(*condition.ErrorOperand).Err)
	}

	constOperands := types.FilterSlice(argOperands[2:], func(o condition.Operand) bool { return o.IsConst() })

	if len(constOperands) != len(argOperands)-2 {
		// Not all operands in the match list are constants
		return condition.NewErrorOperand(fmt.Errorf("isEqualToAny() only supports constant match list"))
	}

	return repo.evalIsInConstantListWithDateRange(argOperands[0], argOperands[1], argOperands[2:])
}

func funcHasValue(repo *CompareCondRepo, n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("wrong number of arguments for hasValue() function"))
	}
	// In most places we call evalAstNode, but it also calls repo.evalOperandAccess in addition to the calls below.
	// In this case we don't want to call evalAstNode, because we also want to check if we are dealing with
	// addressable operand here.
	argOperand := repo.evalOperandAddress(repo.preprocessAstExpr(n.Args[0], scope), scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}

	if argOperand.IsConst() {
		// A constant always exists
		return condition.NewBooleanOperand(true)
	}

	if argOperand.GetKind() != condition.AddressOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("argument to hasValue() function must be an addressable expression"))
	}

	// Now that we made sure that we got the AddressOperandKind above we can do the last step and evaluate the value.
	// Normally this would be done for us automatically in
	argOperand = repo.evalOperandAccess(argOperand, scope)
	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)
			kind := arg.GetKind()
			if kind == condition.ErrorOperandKind {
				return arg
			}
			return condition.NewBooleanOperand(kind != condition.NullOperandKind)
		}, argOperand) // operandKind as hash seed to avoid cache collisions
}

func funcRegexpMatch(repo *CompareCondRepo, n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 2 {
		return condition.NewErrorOperand(fmt.Errorf("wrong number of arguments for regexpMatch() function"))
	}
	patternOperand := repo.evalAstNode(n.Args[0], scope)

	if patternOperand.GetKind() == condition.ErrorOperandKind {
		return patternOperand
	}

	if !patternOperand.IsConst() {
		return condition.NewErrorOperand(
			fmt.Errorf("the first operand of regexpMatch() must be a constant string pattern"))
	}

	if patternOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(
			fmt.Errorf("the first operand of regexpMatch() must be a constant string pattern"))
	}

	patternString := string(patternOperand.(condition.StringOperand))
	re, err := regexp.Compile(patternString)
	if err != nil {
		return condition.NewErrorOperand(
			fmt.Errorf("invalid pattern:\"%s\" passed to regexpMatch()", patternString))
	}

	argOperand := repo.evalAstNode(n.Args[1], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)
			kind := arg.GetKind()
			if kind == condition.ErrorOperandKind {
				return arg
			}
			argString := string(arg.Convert(condition.StringOperandKind).(condition.StringOperand))
			result := condition.NewBooleanOperand(re.MatchString(argString))
			return result
		}, argOperand) // operandKind as hash seed to avoid cache collisions
}

func (repo *CompareCondRepo) funcIsEqualToAny(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) < 2 {
		return condition.NewErrorOperand(fmt.Errorf("wrong number of arguments for isEqualToAny() function"))
	}

	argOperands := types.MapSlice(n.Args, func(o ast.Expr) condition.Operand { return repo.evalAstNode(o, scope) })
	firstErrorOperand := types.FindFirstInSlice(
		argOperands, func(o condition.Operand) bool { return o.GetKind() == condition.ErrorOperandKind })
	if firstErrorOperand != nil {
		return *firstErrorOperand
	}

	constOperands := types.FilterSlice(argOperands[1:], func(o condition.Operand) bool { return o.IsConst() })

	// See if we are comparing against const values
	if argOperands[0].IsConst() {
		for _, argOperand := range constOperands {
			if argOperands[0].Equals(argOperand) {
				return condition.NewBooleanOperand(true)
			}
		}
		if len(constOperands) == len(argOperands)-1 {
			// All operands are constant and do not match
			return condition.NewBooleanOperand(false)
		}
	}

	// TODO: below is not implemented yet
	argOperand := repo.evalOperandAccess(argOperands[0], scope)
	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)
			kind := arg.GetKind()
			if kind == condition.ErrorOperandKind {
				return arg
			}
			return condition.NewBooleanOperand(kind != condition.NullOperandKind)
		}, argOperand) // operandKind as hash seed to avoid cache collisions
}

func findElementScope(element string, parentScope *ForEachScope) *ForEachScope {
	// ASSUME: rootScope.element == "" and will match $.something path
	for scope := parentScope; scope != nil; scope = scope.ParentScope {
		if element == scope.Element || scope.NestingLevel == 0 {
			return scope
		}
	}
	return nil
}

// expandPath will take a path like parent.some.attribute, split it into parent and some.attribute components,
// lookup the parent element in the parent scopes
// It will return:
//   - the ancestor scope defining the referenced element
//   - child path or full path if the ancestor is the root scope
func expandPath(path string, parentScope *ForEachScope) (*ForEachScope, string) {
	// 1. Given a path like $parent.some.attribute, split it into parent and some.attribute components
	parts := strings.SplitN(path, ".", 2)
	// 2. Lookup the parent in the parent scopes and get the nesting NestingLevel of the scope that defines the parent
	scope := findElementScope(parts[0], parentScope)
	if scope.NestingLevel == 0 {
		return scope, path
	} else {
		return scope, parts[1]
	}
}

func getAttributePathAddress(attrPath string, parentScope *ForEachScope) (*objectmap.AttributeAddress, error) {
	pScope, ePath := expandPath(attrPath, parentScope)
	addr, err := pScope.AttrDictRec.AttributePathToAddress(ePath)
	if err != nil {
		return nil, err
	}
	// TODO, should this be coming from the AttributePathToAddress?
	// Also the MatchId is wrong here.  It doesn't break things though, because
	// the nested matchid is not really used
	return &objectmap.AttributeAddress{
		Address:              addr,
		Path:                 ePath,
		ParentParameterIndex: pScope.NestingLevel,
		FullAddress:          pScope.AttrDictRec.AddressToFullAddress(addr)}, nil
}

func (repo *CompareCondRepo) evalAstNode(node ast.Expr, scope *ForEachScope) condition.Operand {
	return repo.evalOperandAccess(repo.evalOperandAddress(repo.preprocessAstExpr(node, scope), scope), scope)
}

func (repo *CompareCondRepo) evalOperandAccess(operand condition.Operand, scope *ForEachScope) condition.Operand {
	if operand.IsConst() {
		// This includes error operand
		return operand
	}

	if operand.GetKind() == condition.ExpressionOperandKind {
		return operand
	}

	// assert: this is an AddressOperand
	repo.registerCatEvaluatorForAddress(operand.(*condition.AddressOperand).FullAddress, scope.Evaluator)

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			address := operand.Evaluate(event, frames)
			if address.GetKind() == condition.ErrorOperandKind {
				return address
			}
			val := objectmap.GetNestedAttributeByAddress(
				frames[address.(*condition.AddressOperand).ParameterIndex], address.(*condition.AddressOperand).Address)
			if val == nil {
				return condition.NewNullOperand(address.(*condition.AddressOperand))
			}
			return val.(condition.Operand)
		}, operand)
}

func (repo *CompareCondRepo) evalOperandAddress(operand condition.Operand, scope *ForEachScope) condition.Operand {
	switch o := operand.(type) {
	case *condition.AttributeOperand:
		attrAddress, err := getAttributePathAddress(operand.(*condition.AttributeOperand).AttributePath, scope)
		if err != nil {
			return repo.CondFactory.NewErrorOperand(err)
		} else {
			return repo.CondFactory.NewAddressOperand(
				attrAddress.Address,
				attrAddress.FullAddress,
				attrAddress.ParentParameterIndex,
				nil)
		}
	case *condition.SelOperand:
		if o.Base == nil {
			pScope, ePath := expandPath(o.Selector, scope)
			addr, err := pScope.AttrDictRec.AttributePathToAddress(ePath)
			if err != nil {
				return condition.NewErrorOperand(err)
			}
			return repo.CondFactory.NewAddressOperand(addr, pScope.AttrDictRec.AddressToFullAddress(addr), pScope.NestingLevel, nil)
		}

		base := repo.evalOperandAddress(o.Base, scope)
		if base.GetKind() == condition.ErrorOperandKind {
			return base
		}
		switch b := base.(type) {
		case *condition.AddressOperand:
			dictRec := repo.ObjectAttributeMapper.RootDictRec.AddressToDictionaryRec(b.Address)
			addr, err := dictRec.AttributePathToAddress(o.Selector)
			if err != nil {
				return condition.NewErrorOperand(err)
			}
			attrAddress := objectmap.ExtendAddress(b.Address, addr...)
			fullAddress := objectmap.ExtendAddress(b.FullAddress, addr...)
			if b.ExprOperand == nil {
				// Base is a simple address with no dynamic computation.  Use it as immutable.
				return repo.CondFactory.NewAddressOperand(attrAddress, fullAddress, b.ParameterIndex, nil)
			} else {
				// Base address has to be computed and so does the selector
				return repo.CondFactory.NewAddressOperand(
					attrAddress,
					fullAddress,
					b.ParameterIndex,
					repo.CondFactory.NewExprOperand(
						func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
							baseAddress := b.Evaluate(event, frames)
							switch baseAddress.GetKind() {
							case condition.ErrorOperandKind:
								return baseAddress
							case condition.AddressOperandKind:
								attrAddress = append(baseAddress.(*condition.AddressOperand).Address, addr...)
								fullAddress = append(baseAddress.(*condition.AddressOperand).FullAddress, addr...)
								return condition.NewAddressOperand(attrAddress, fullAddress, b.ParameterIndex, nil)
							}
							panic("should not get here")
							//val := objectmap.GetNestedAttributeByAddress(frames[0], attrAddress)
							//return val.(condition.Operand)
						}, operand))
			}
		default:
			panic("should not get here")
		}
	case *condition.IndexOperand:
		if o.Base == nil {
			addr, err := repo.ObjectAttributeMapper.RootDictRec.AttributePathToAddress("")
			if err != nil {
				return condition.NewErrorOperand(err)
			}
			indexOperand := repo.evalOperandAddress(o.IndexExpr, scope)
			if indexOperand.GetKind() == condition.ErrorOperandKind {
				return indexOperand
			}
			if indexOperand.IsConst() {
				io := indexOperand.Convert(condition.IntOperandKind)
				if io.GetKind() == condition.ErrorOperandKind {
					return io
				}
				address := objectmap.ExtendAddress(addr, int(io.(condition.IntOperand)))
				return repo.CondFactory.NewAddressOperand(address, address, 0, nil)
			} else {
				// Use -1 for the index to indicate that it is not known until evaluation time
				staticAddress := objectmap.ExtendAddress(addr, -1)
				return repo.CondFactory.NewAddressOperand(
					staticAddress,
					staticAddress,
					0,
					repo.CondFactory.NewExprOperand(
						func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
							io := indexOperand.Evaluate(event, frames).Convert(condition.IntOperandKind)
							if io.GetKind() == condition.ErrorOperandKind {
								return io
							}
							address := objectmap.ExtendAddress(addr, int(io.(condition.IntOperand)))
							return condition.NewAddressOperand(address, address, 0, nil)
						}, operand))
			}
		} else {
			baseOperand := repo.evalOperandAddress(o.Base, scope)
			if baseOperand.GetKind() == condition.ErrorOperandKind {
				return baseOperand
			}
			indexOperand := repo.evalOperandAddress(o.IndexExpr, scope)
			if indexOperand.GetKind() == condition.ErrorOperandKind {
				return indexOperand
			}
			if baseOperand.(*condition.AddressOperand).ExprOperand == nil && indexOperand.IsConst() {
				io := indexOperand.Convert(condition.IntOperandKind)
				if io.GetKind() == condition.ErrorOperandKind {
					return io
				}
				address := objectmap.ExtendAddress(baseOperand.(*condition.AddressOperand).Address, int(io.(condition.IntOperand)))
				fullAddress := objectmap.ExtendAddress(baseOperand.(*condition.AddressOperand).FullAddress, int(io.(condition.IntOperand)))
				return repo.CondFactory.NewAddressOperand(address, fullAddress, baseOperand.(*condition.AddressOperand).ParameterIndex, nil)
			} else {
				// Use -1 for the index to indicate that it is not known until evaluation time
				staticAddress := objectmap.ExtendAddress(baseOperand.(*condition.AddressOperand).Address, -1)
				staticFullAddress := objectmap.ExtendAddress(baseOperand.(*condition.AddressOperand).FullAddress, -1)
				ioAccess := repo.evalOperandAccess(indexOperand, scope)
				return repo.CondFactory.NewAddressOperand(
					staticAddress,
					staticFullAddress,
					baseOperand.(*condition.AddressOperand).ParameterIndex,
					repo.CondFactory.NewExprOperand(func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
						bo := baseOperand.Evaluate(event, frames)
						if bo.GetKind() == condition.ErrorOperandKind {
							return bo
						}
						io := ioAccess.Evaluate(event, frames).Convert(condition.IntOperandKind)
						if io.GetKind() == condition.ErrorOperandKind {
							return io
						}
						iov := int(io.(condition.IntOperand))
						bov := bo.(*condition.AddressOperand)
						address := objectmap.ExtendAddress(bov.Address, iov)
						// fullAddress is not strictly needed in this case.  Use nil instead if you want to optimize.
						fullAddress := objectmap.ExtendAddress(bov.FullAddress, iov)
						return condition.NewAddressOperand(address, fullAddress, bov.ParameterIndex, nil)
					}, operand))
			}
		}
	default:
		return operand
	}
}
