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

// preprocessExpression handles reserved Go keywords in expressions
// by replacing them with valid identifiers before parsing.
// This allows users to use intuitive function names like "if" without
// conflicting with Go's syntax.
func preprocessExpression(expr string) string {
	// Replace "if(" with "ifFunc(" to avoid Go parser keyword conflict
	// Use regex to only replace when followed by opening parenthesis
	result := regexp.MustCompile(`\bif\s*\(`).ReplaceAllString(expr, "ifFunc(")
	return result
}

type RepoInterface interface {
	Register(f *InternalRule)
}

type GeneralRuleRecord struct {
	definition *InternalRule
	id         uint
}

// GetHash returns the cryptographic hash of the rule's compiled condition.
func (r *GeneralRuleRecord) GetHash() uint64 {
	return r.definition.GetHash()
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
	return &RuleEngineRepo{
		ctx:      ctx,
		ruleApi:  NewRuleApi(ctx),
		Optimize: false, // Default to non-optimized mode, set explicitly if needed
	}
}

type CatEvaluatorKind int8

type CatEvaluator interface {
	Evaluate(event *objectmap.ObjectAttributeMap, frames []interface{}) (bool, error)
	GetCategory() types.Category
	immutable.SetElement
}

type EvalCategoryRec struct {
	Cat                      types.Category
	Eval                     condition.Operand
	AttrKeys                 []string
	IsUndefinedEqualityCheck bool   // true if this category checks "field == undefined"
	FieldPath                string // field path for undefined checks (e.g., "age", "user.name")
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
	// Categories that must be evaluated even if their attributes aren't in the event
	// (e.g., null checks like "field == null", constant expressions like "1 == 1")
	AlwaysEvaluateCategories     *hashset.Set[*EvalCategoryRec]
	// Set of categories that check "field == undefined" (for efficient DefaultCatList)
	UndefinedEqualityCategories  *hashset.Set[types.Category]
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
				// AND logic with three-valued semantics:
				// false && anything → false (short circuit)
				// true && undefined → undefined
				// undefined && false → false
				// undefined && true → undefined
				if X.GetKind() == condition.BooleanOperandKind && !bool(X.(condition.BooleanOperand)) {
					return condition.NewBooleanOperand(false)
				}
				if Y.GetKind() == condition.BooleanOperandKind && !bool(Y.(condition.BooleanOperand)) {
					return condition.NewBooleanOperand(false)
				}
				if X.GetKind() == condition.UndefinedOperandKind || Y.GetKind() == condition.UndefinedOperandKind {
					return condition.NewUndefinedOperand(nil)
				}
				return condition.NewBooleanOperand(bool(X.(condition.BooleanOperand)) && bool(Y.(condition.BooleanOperand)))
			case token.LOR:
				// OR logic with three-valued semantics:
				// true || anything → true (short circuit)
				// false || undefined → undefined
				// undefined || true → true
				// undefined || false → undefined
				if X.GetKind() == condition.BooleanOperandKind && bool(X.(condition.BooleanOperand)) {
					return condition.NewBooleanOperand(true)
				}
				if Y.GetKind() == condition.BooleanOperandKind && bool(Y.(condition.BooleanOperand)) {
					return condition.NewBooleanOperand(true)
				}
				if X.GetKind() == condition.UndefinedOperandKind || Y.GetKind() == condition.UndefinedOperandKind {
					return condition.NewUndefinedOperand(nil)
				}
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
			// Temporary debug for null inequality issue
			// if compOp == condition.CompareNotEqualOp {
			// 	fmt.Printf("DEBUG != Compare START: X=%v (kind=%v), Y=%v (kind=%v)\n", X, xKind, Y, yKind)
			// }
			if xKind == condition.ErrorOperandKind {
				return X
			}
			if yKind == condition.ErrorOperandKind {
				return Y
			}

			// Special handling for undefined comparisons (THREE-VALUED LOGIC)
			// Check undefined BEFORE null handling
			if xKind == condition.UndefinedOperandKind && yKind == condition.UndefinedOperandKind {
				// Both undefined: undefined == undefined → true, undefined != undefined → false
				switch compOp {
				case condition.CompareEqualOp:
					return condition.NewBooleanOperand(true)
				case condition.CompareNotEqualOp:
					return condition.NewBooleanOperand(false)
				default:
					// Ordering operations with undefined → undefined
					return condition.NewUndefinedOperand(nil)
				}
			}

			if xKind == condition.UndefinedOperandKind || yKind == condition.UndefinedOperandKind {
				// One operand is undefined - all comparisons return undefined (three-valued logic)
				// This is the breaking change: undefined != value returns undefined (not true)
				// When cast to boolean, undefined becomes false
				return condition.NewUndefinedOperand(nil)
			}

			// DEBUG
			if false {  // Disabled debug
				fmt.Printf("[DEBUG] Comparison: X=%v (kind=%v), Y=%v (kind=%v), op=%v\n",
					X, xKind, Y, yKind, compOp)
			}

			// Special handling for null comparisons
			// Null is a VALUE (different from undefined which means "no value")
			// null == value → false (unless both null)
			// null != value → true (unless both null)
			// null > value → false (null is not orderable)
			if xKind == condition.NullOperandKind || yKind == condition.NullOperandKind {
				bothNull := xKind == condition.NullOperandKind && yKind == condition.NullOperandKind
				switch compOp {
				case condition.CompareEqualOp:
					return condition.NewBooleanOperand(bothNull)
				case condition.CompareNotEqualOp:
					return condition.NewBooleanOperand(!bothNull)
				case condition.CompareGreaterOp, condition.CompareGreaterOrEqualOp,
					condition.CompareLessOp, condition.CompareLessOrEqualOp:
					// Null is not orderable, all ordering comparisons return false
					return condition.NewBooleanOperand(false)
				}
			}

			// Convert toward the higher kind, e.g. int -> float -> bool -> string
			X, Y = condition.ReconcileOperands(X, Y)

			// Check for errors after reconciliation
			if X.GetKind() == condition.ErrorOperandKind {
				return X
			}
			if Y.GetKind() == condition.ErrorOperandKind {
				return Y
			}

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
		}, xEval, yEval, repo.CondFactory.NewIntOperand(int64(compOp)))
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
		category := condition.NewIntOperand(int64(scope.Evaluator.GetCategory()))

		// Add the constant itself
		categoryList, _ := categoryMap.Get(constOperand)
		categoryMap.Put(constOperand, append(categoryList, category))

		// Also add type-converted versions for numeric/string/time interoperability
		constKind := constOperand.GetKind()
		if constKind == condition.StringOperandKind {
			// Try adding int and float versions of string constants
			intVersion := constOperand.Convert(condition.IntOperandKind)
			if intVersion.GetKind() != condition.ErrorOperandKind {
				intList, _ := categoryMap.Get(intVersion)
				categoryMap.Put(intVersion, append(intList, category))
			}
			floatVersion := constOperand.Convert(condition.FloatOperandKind)
			if floatVersion.GetKind() != condition.ErrorOperandKind {
				floatList, _ := categoryMap.Get(floatVersion)
				categoryMap.Put(floatVersion, append(floatList, category))
			}
			// Try adding time version of string constants
			timeVersion := constOperand.Convert(condition.TimeOperandKind)
			if timeVersion.GetKind() != condition.ErrorOperandKind {
				timeList, _ := categoryMap.Get(timeVersion)
				categoryMap.Put(timeVersion, append(timeList, category))
			}
		} else if constKind == condition.IntOperandKind || constKind == condition.FloatOperandKind {
			// Try adding string version of numeric constants
			stringVersion := constOperand.Convert(condition.StringOperandKind)
			stringList, _ := categoryMap.Get(stringVersion)
			categoryMap.Put(stringVersion, append(stringList, category))
			// Note: We don't convert numbers to time here because any number can be
			// interpreted as a Unix timestamp, leading to false positive matches
		} else if constKind == condition.TimeOperandKind {
			// Try adding string version of time constants
			stringVersion := constOperand.Convert(condition.StringOperandKind)
			stringList, _ := categoryMap.Get(stringVersion)
			categoryMap.Put(stringVersion, append(stringList, category))
			// Try adding int version of time constants (for Unix nano timestamps)
			intVersion := constOperand.Convert(condition.IntOperandKind)
			if intVersion.GetKind() != condition.ErrorOperandKind {
				intList, _ := categoryMap.Get(intVersion)
				categoryMap.Put(intVersion, append(intList, category))
			}
		}
		// Note: Boolean types are intentionally excluded
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
			}

			// Try type conversions for numeric/string/time comparisons (but NOT boolean)
			if xKind == condition.StringOperandKind {
				// Try converting string to int
				intX := X.Convert(condition.IntOperandKind)
				if intX.GetKind() != condition.ErrorOperandKind {
					if catList, k := categoryMap.Get(intX); k {
						return condition.NewListOperand(catList)
					}
				}
				// Try converting string to float
				floatX := X.Convert(condition.FloatOperandKind)
				if floatX.GetKind() != condition.ErrorOperandKind {
					if catList, k := categoryMap.Get(floatX); k {
						return condition.NewListOperand(catList)
					}
				}
				// Try converting string to time
				timeX := X.Convert(condition.TimeOperandKind)
				if timeX.GetKind() != condition.ErrorOperandKind {
					if catList, k := categoryMap.Get(timeX); k {
						return condition.NewListOperand(catList)
					}
				}
			} else if xKind == condition.IntOperandKind || xKind == condition.FloatOperandKind {
				// Try converting number to string (but NOT to/from boolean)
				stringX := X.Convert(condition.StringOperandKind)
				if catList, k := categoryMap.Get(stringX); k {
					return condition.NewListOperand(catList)
				}
				// Note: We don't try converting numbers to time here because any number
				// can be interpreted as a Unix timestamp, leading to false positive matches
			} else if xKind == condition.TimeOperandKind {
				// Try converting time to string
				stringX := X.Convert(condition.StringOperandKind)
				if catList, k := categoryMap.Get(stringX); k {
					return condition.NewListOperand(catList)
				}
				// Try converting time to int (Unix nano timestamp)
				intX := X.Convert(condition.IntOperandKind)
				if intX.GetKind() != condition.ErrorOperandKind {
					if catList, k := categoryMap.Get(intX); k {
						return condition.NewListOperand(catList)
					}
				}
			}
			// Note: Boolean types are intentionally excluded from cross-type conversions

			return condition.IntConst0
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
	// Preprocess expression to handle reserved keywords
	expr := preprocessExpression(exprCondition.Expr)

	// Convert the expression to an AST node tree
	node, err := parser.ParseExpr(expr)

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
		// Check if this is an undefined equality check first (before const optimization)
		isUndefinedEqualityCheck := compareCond.CompareOp == condition.CompareEqualOp &&
			(compareCond.LeftOperand.GetKind() == condition.UndefinedOperandKind ||
				compareCond.RightOperand.GetKind() == condition.UndefinedOperandKind)

		// Special case equal compare against a constant that can be done via a hash lookup
		// BUT: Skip hash lookup optimization for undefined checks (they need special DefaultCatList handling)
		if compareCond.CompareOp == condition.CompareEqualOp &&
			(compareCond.LeftOperand.IsConst() || compareCond.RightOperand.IsConst()) &&
			!isUndefinedEqualityCheck {
			eval = repo.processCompareEqualToConstCondition(compareCond, scope)
		} else {
			eval = repo.genEvalForCompareCondition(compareCond, scope)
		}
		if eval != nil && eval.GetKind() == condition.ErrorOperandKind {
			return condition.NewErrorCondition(eval.(condition.ErrorOperand))
		}
		evalCatRec.Eval = eval

		repo.CondToCompareCondRecord.Put(compareCond, evalCatRec)

		// Detect and mark undefined-equality checks (field == undefined)
		// These will be added to DefaultCatList for efficient processing
		if eval != nil && compareCond.CompareOp == condition.CompareEqualOp {
			isUndefinedCheck := compareCond.LeftOperand.GetKind() == condition.UndefinedOperandKind ||
				compareCond.RightOperand.GetKind() == condition.UndefinedOperandKind
			if isUndefinedCheck {
				evalCatRec.IsUndefinedEqualityCheck = true
				// Extract field path from the non-undefined operand
				if compareCond.LeftOperand.GetKind() == condition.UndefinedOperandKind && len(evalCatRec.AttrKeys) > 0 {
					evalCatRec.FieldPath = evalCatRec.AttrKeys[0]
				} else if compareCond.RightOperand.GetKind() == condition.UndefinedOperandKind && len(evalCatRec.AttrKeys) > 0 {
					evalCatRec.FieldPath = evalCatRec.AttrKeys[0]
				}
				// Add to UndefinedEqualityCategories set for DefaultCatList building
				repo.UndefinedEqualityCategories.Put(evalCatRec.GetCategory())
			}
		}

		// Register categories that must always be evaluated:
		// 1. Undefined checks: comparisons where one operand is undefined (field == undefined)
		// 2. Null checks: comparisons where one operand is null (field == null, field != null)
		// 3. Constant expressions: comparisons with no event dependencies (1 == 1, true)
		// Note: Only register if eval is not nil (processCompareEqualToConstCondition can return nil for duplicates)
		if eval != nil {
			isUndefinedCheck := compareCond.LeftOperand.GetKind() == condition.UndefinedOperandKind ||
				compareCond.RightOperand.GetKind() == condition.UndefinedOperandKind
			isNullCheck := compareCond.LeftOperand.GetKind() == condition.NullOperandKind ||
				compareCond.RightOperand.GetKind() == condition.NullOperandKind
			hasNoEventDependencies := len(evalCatRec.AttrKeys) == 0

			if isUndefinedCheck || isNullCheck || hasNoEventDependencies {
				repo.AlwaysEvaluateCategories.Put(evalCatRec)
			}
		}
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
			hasUndefined := false
			for _, eval := range condEvaluators {
				result := eval.Func(event, frames)
				if result.GetKind() == condition.ErrorOperandKind {
					return result
				}
				// false short-circuits (undefined && false → false)
				if result.GetKind() == condition.BooleanOperandKind && !result.(condition.BooleanOperand) {
					return condition.NewBooleanOperand(false)
				}
				// Track undefined (doesn't short-circuit)
				if result.GetKind() == condition.UndefinedOperandKind {
					hasUndefined = true
				}
			}
			// If any was undefined and none were false → undefined
			if hasUndefined {
				return condition.NewUndefinedOperand(nil)
			}
			// All were true
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
			hasUndefined := false
			for _, eval := range condEvaluators {
				result := eval.Func(event, frames)
				if result.GetKind() == condition.ErrorOperandKind {
					return result
				}
				// true short-circuits (undefined || true → true)
				if result.GetKind() == condition.BooleanOperandKind && result.(condition.BooleanOperand) {
					return condition.NewBooleanOperand(true)
				}
				// Track undefined (doesn't short-circuit)
				if result.GetKind() == condition.UndefinedOperandKind {
					hasUndefined = true
				}
			}
			// If any was undefined and none were true → undefined
			if hasUndefined {
				return condition.NewUndefinedOperand(nil)
			}
			// All were false
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
			}
			// !(undefined) → undefined (three-valued logic)
			if result.GetKind() == condition.UndefinedOperandKind {
				return result
			}
			// !(null) → true (null is falsey)
			if result.GetKind() == condition.NullOperandKind {
				return condition.NewBooleanOperand(true)
			}
			// Must be BooleanOperand at this point
			if result.GetKind() != condition.BooleanOperandKind {
				return condition.NewErrorOperand(fmt.Errorf("NOT operator expects boolean, got %v", result.GetKind()))
			}
			return condition.NewBooleanOperand(!bool(result.(condition.BooleanOperand)))
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
					// Array is missing/null - return false (rule doesn't apply to missing arrays)
					return condition.NewBooleanOperand(false)
				}

				// Empty array - return true (vacuous truth: "all elements" in empty set satisfy any condition)
				if numElements == 0 {
					return condition.NewBooleanOperand(true)
				}

				// Non-empty array - evaluate condition for each element
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
					}
					// Handle undefined: propagate it (forAll with undefined element → undefined)
					if result.GetKind() == condition.UndefinedOperandKind {
						break
					}
					// Handle null as falsey
					if result.GetKind() == condition.NullOperandKind {
						result = condition.NewBooleanOperand(false)
						break
					}
					// Handle false boolean
					if result.GetKind() == condition.BooleanOperandKind && !result.(condition.BooleanOperand) {
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
		// Add to AlwaysEvaluateCategories so it runs for empty arrays
		// (just like field == null runs for missing fields)
		repo.AlwaysEvaluateCategories.Put(evalCatRec)
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
					}
					// Handle undefined: propagate it (forSome with undefined element → keep searching)
					if result.GetKind() == condition.UndefinedOperandKind {
						continue
					}
					// Handle null as falsey (continue searching)
					if result.GetKind() == condition.NullOperandKind {
						continue
					}
					// Handle true boolean (found a match!)
					if result.GetKind() == condition.BooleanOperandKind && result.(condition.BooleanOperand) {
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
		case "forAll", "all":
			return negateIfTrue(repo.processForAllFunc(n, scope), negate)
		case "forSome", "any":
			return negateIfTrue(repo.processForSomeFunc(n, scope), negate)
		default:
			// Functions that return operands (if, abs, min, max, etc.) cannot be used
			// as standalone boolean conditions. They must be used in comparisons or arithmetic.
			// For example: "if(premium, 100, 50) > threshold" is valid
			// But: "if(premium, 100, 50)" as a standalone expression is invalid
			return condition.NewErrorCondition(fmt.Errorf("function '%s' cannot be used as boolean condition - must be used in comparison or arithmetic expression", funcName))
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

func (repo *CompareCondRepo) funcLength(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("wrong number of arguments for length() function"))
	}

	// Evaluate the path argument (should be a string like "items")
	pathOperand := repo.evalAstNode(n.Args[0], scope)
	if pathOperand.GetKind() == condition.ErrorOperandKind {
		return pathOperand
	}

	if pathOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("length() only supports string path"))
	}

	path := string(pathOperand.(condition.StringOperand))

	// Get the array address for the path
	arrayAddress, err := getAttributePathAddress(path+"[]", scope)
	if err != nil {
		return condition.NewErrorOperand(err)
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
			if err != nil {
				// Array missing - return undefined (distinct from explicit null)
				// This ensures length("items") != 0 doesn't match when items is missing
				return condition.NewUndefinedOperand(nil)
			}
			return condition.NewIntOperand(int64(numElements))
		}, pathOperand) // pathOperand in Args for proper hash
}

// count(array_path, element_name, condition) - Count elements matching condition
func (repo *CompareCondRepo) funcCount(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 3 {
		return condition.NewErrorOperand(fmt.Errorf("count() requires 3 arguments: array_path, element_name, condition"))
	}

	// Parse arguments (same as forSome)
	pathOperand := repo.evalAstNode(n.Args[0], scope)
	if pathOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("count() first argument must be string path"))
	}
	arrayPath := string(pathOperand.(condition.StringOperand))

	elemOperand := repo.evalAstNode(n.Args[1], scope)
	if elemOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("count() element name must be string"))
	}
	elementName := string(elemOperand.(condition.StringOperand))

	// Get condition expression
	condExpr := n.Args[2]

	// Setup array iteration (like forSome)
	arrayAddress, err := getAttributePathAddress(arrayPath+"[]", scope)
	if err != nil {
		return condition.NewErrorOperand(err)
	}

	// Create scope for iteration
	scopeForPath, expandedPath := expandPath(arrayPath+"[]", scope)
	addr, err := scopeForPath.AttrDictRec.AttributePathToAddress(expandedPath)
	if err != nil {
		return condition.NewErrorOperand(err)
	}
	newDictRec := scopeForPath.AttrDictRec.AddressToDictionaryRec(addr)
	newPath := scopeForPath.AttrDictRec.AddressToFullPath(addr)

	newScope := &ForEachScope{
		Path:         newPath,
		Element:      elementName,
		NestingLevel: scope.NestingLevel + 1,
		ParentScope:  scope,
		AttrDictRec:  newDictRec,
	}

	// Evaluate condition in new scope
	condOperand := repo.evalAstNode(condExpr, newScope)
	if condOperand.GetKind() == condition.ErrorOperandKind {
		return condOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
			if err != nil {
				// Array missing - return undefined
				return condition.NewUndefinedOperand(nil)
			}

			count := 0
			parentsFrame := frames[arrayAddress.ParentParameterIndex]
			currentAddressLen := len(arrayAddress.Address)
			currentAddress := make([]int, currentAddressLen+1)
			copy(currentAddress, arrayAddress.Address)

			for i := 0; i < numElements; i++ {
				currentAddress[currentAddressLen] = i
				newFrame := objectmap.GetNestedAttributeByAddress(parentsFrame, currentAddress)

				if newFrame == nil {
					continue
				}

				frames[newScope.NestingLevel] = newFrame

				result := condOperand.Evaluate(event, frames)

				// Count if true
				boolResult := result.Convert(condition.BooleanOperandKind)
				if boolResult.GetKind() == condition.BooleanOperandKind &&
					bool(boolResult.(condition.BooleanOperand)) {
					count++
				}
			}

			return condition.NewIntOperand(int64(count))
		}, pathOperand, elemOperand)
}

// sum(array_path, element_name, expression) - Sum values
func (repo *CompareCondRepo) funcSum(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 3 {
		return condition.NewErrorOperand(fmt.Errorf("sum() requires 3 arguments: array_path, element_name, expression"))
	}

	pathOperand := repo.evalAstNode(n.Args[0], scope)
	if pathOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("sum() first argument must be string path"))
	}
	arrayPath := string(pathOperand.(condition.StringOperand))

	elemOperand := repo.evalAstNode(n.Args[1], scope)
	if elemOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("sum() element name must be string"))
	}
	elementName := string(elemOperand.(condition.StringOperand))

	exprAst := n.Args[2]

	arrayAddress, err := getAttributePathAddress(arrayPath+"[]", scope)
	if err != nil {
		return condition.NewErrorOperand(err)
	}

	scopeForPath, expandedPath := expandPath(arrayPath+"[]", scope)
	addr, err := scopeForPath.AttrDictRec.AttributePathToAddress(expandedPath)
	if err != nil {
		return condition.NewErrorOperand(err)
	}
	newDictRec := scopeForPath.AttrDictRec.AddressToDictionaryRec(addr)
	newPath := scopeForPath.AttrDictRec.AddressToFullPath(addr)

	newScope := &ForEachScope{
		Path:         newPath,
		Element:      elementName,
		NestingLevel: scope.NestingLevel + 1,
		ParentScope:  scope,
		AttrDictRec:  newDictRec,
	}

	exprOperand := repo.evalAstNode(exprAst, newScope)
	if exprOperand.GetKind() == condition.ErrorOperandKind {
		return exprOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
			if err != nil {
				return condition.NewUndefinedOperand(nil)
			}

			sum := 0.0
			parentsFrame := frames[arrayAddress.ParentParameterIndex]
			currentAddressLen := len(arrayAddress.Address)
			currentAddress := make([]int, currentAddressLen+1)
			copy(currentAddress, arrayAddress.Address)

			for i := 0; i < numElements; i++ {
				currentAddress[currentAddressLen] = i
				newFrame := objectmap.GetNestedAttributeByAddress(parentsFrame, currentAddress)

				if newFrame == nil {
					continue
				}

				frames[newScope.NestingLevel] = newFrame

				result := exprOperand.Evaluate(event, frames)

				if result.GetKind() == condition.UndefinedOperandKind ||
					result.GetKind() == condition.NullOperandKind {
					continue
				}

				numeric := result.Convert(condition.FloatOperandKind)
				if numeric.GetKind() != condition.ErrorOperandKind {
					sum += float64(numeric.(condition.FloatOperand))
				}
			}

			return condition.NewFloatOperand(sum)
		}, pathOperand, elemOperand)
}

// avg(array_path, element_name, expression) - Average values
func (repo *CompareCondRepo) funcAvg(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 3 {
		return condition.NewErrorOperand(fmt.Errorf("avg() requires 3 arguments: array_path, element_name, expression"))
	}

	pathOperand := repo.evalAstNode(n.Args[0], scope)
	if pathOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("avg() first argument must be string path"))
	}
	arrayPath := string(pathOperand.(condition.StringOperand))

	elemOperand := repo.evalAstNode(n.Args[1], scope)
	if elemOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("avg() element name must be string"))
	}
	elementName := string(elemOperand.(condition.StringOperand))

	exprAst := n.Args[2]

	arrayAddress, err := getAttributePathAddress(arrayPath+"[]", scope)
	if err != nil {
		return condition.NewErrorOperand(err)
	}

	scopeForPath, expandedPath := expandPath(arrayPath+"[]", scope)
	addr, err := scopeForPath.AttrDictRec.AttributePathToAddress(expandedPath)
	if err != nil {
		return condition.NewErrorOperand(err)
	}
	newDictRec := scopeForPath.AttrDictRec.AddressToDictionaryRec(addr)
	newPath := scopeForPath.AttrDictRec.AddressToFullPath(addr)

	newScope := &ForEachScope{
		Path:         newPath,
		Element:      elementName,
		NestingLevel: scope.NestingLevel + 1,
		ParentScope:  scope,
		AttrDictRec:  newDictRec,
	}

	exprOperand := repo.evalAstNode(exprAst, newScope)
	if exprOperand.GetKind() == condition.ErrorOperandKind {
		return exprOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
			if err != nil {
				return condition.NewUndefinedOperand(nil)
			}

			if numElements == 0 {
				return condition.NewUndefinedOperand(nil) // Can't average empty
			}

			sum := 0.0
			count := 0
			parentsFrame := frames[arrayAddress.ParentParameterIndex]
			currentAddressLen := len(arrayAddress.Address)
			currentAddress := make([]int, currentAddressLen+1)
			copy(currentAddress, arrayAddress.Address)

			for i := 0; i < numElements; i++ {
				currentAddress[currentAddressLen] = i
				newFrame := objectmap.GetNestedAttributeByAddress(parentsFrame, currentAddress)

				if newFrame == nil {
					continue
				}

				frames[newScope.NestingLevel] = newFrame

				result := exprOperand.Evaluate(event, frames)

				if result.GetKind() == condition.UndefinedOperandKind ||
					result.GetKind() == condition.NullOperandKind {
					continue
				}

				numeric := result.Convert(condition.FloatOperandKind)
				if numeric.GetKind() != condition.ErrorOperandKind {
					sum += float64(numeric.(condition.FloatOperand))
					count++
				}
			}

			if count == 0 {
				return condition.NewUndefinedOperand(nil)
			}

			return condition.NewFloatOperand(sum / float64(count))
		}, pathOperand, elemOperand)
}

// Duration functions for time arithmetic
// These convert numeric values to nanoseconds for use with time operations

func (repo *CompareCondRepo) funcDays(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("days() requires exactly one argument"))
	}

	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}


	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)

			// Convert to numeric
			numericArg := arg.Convert(condition.FloatOperandKind)
			if numericArg.GetKind() == condition.ErrorOperandKind {
				return numericArg
			}

			// Calculate nanoseconds: days * 24 * 60 * 60 * 1e9
			days := float64(numericArg.(condition.FloatOperand))
			nanos := int64(days * 24 * 60 * 60 * 1e9)
			return condition.NewIntOperand(nanos)
		}, argOperand)
}

func (repo *CompareCondRepo) funcHours(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("hours() requires exactly one argument"))
	}

	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}


	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)

			// Convert to numeric
			numericArg := arg.Convert(condition.FloatOperandKind)
			if numericArg.GetKind() == condition.ErrorOperandKind {
				return numericArg
			}

			// Calculate nanoseconds: hours * 60 * 60 * 1e9
			hours := float64(numericArg.(condition.FloatOperand))
			nanos := int64(hours * 60 * 60 * 1e9)
			return condition.NewIntOperand(nanos)
		}, argOperand)
}

func (repo *CompareCondRepo) funcMinutes(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("minutes() requires exactly one argument"))
	}

	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}


	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)

			// Convert to numeric
			numericArg := arg.Convert(condition.FloatOperandKind)
			if numericArg.GetKind() == condition.ErrorOperandKind {
				return numericArg
			}

			// Calculate nanoseconds: minutes * 60 * 1e9
			minutes := float64(numericArg.(condition.FloatOperand))
			nanos := int64(minutes * 60 * 1e9)
			return condition.NewIntOperand(nanos)
		}, argOperand)
}

func (repo *CompareCondRepo) funcSeconds(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("seconds() requires exactly one argument"))
	}

	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}


	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)

			// Convert to numeric
			numericArg := arg.Convert(condition.FloatOperandKind)
			if numericArg.GetKind() == condition.ErrorOperandKind {
				return numericArg
			}

			// Calculate nanoseconds: seconds * 1e9
			secs := float64(numericArg.(condition.FloatOperand))
			nanos := int64(secs * 1e9)
			return condition.NewIntOperand(nanos)
		}, argOperand)
}

// Ternary/conditional operator: if(condition, true_value, false_value)
func (repo *CompareCondRepo) funcIf(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 3 {
		return condition.NewErrorOperand(fmt.Errorf("if() requires exactly three arguments: condition, true_value, false_value"))
	}

	condOperand := repo.evalAstNode(n.Args[0], scope)
	if condOperand.GetKind() == condition.ErrorOperandKind {
		return condOperand
	}

	trueOperand := repo.evalAstNode(n.Args[1], scope)
	if trueOperand.GetKind() == condition.ErrorOperandKind {
		return trueOperand
	}

	falseOperand := repo.evalAstNode(n.Args[2], scope)
	if falseOperand.GetKind() == condition.ErrorOperandKind {
		return falseOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			cond := condOperand.Evaluate(event, frames)

			// Handle undefined condition - return undefined
			if cond.GetKind() == condition.UndefinedOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Convert to boolean
			boolCond := cond.Convert(condition.BooleanOperandKind)
			if boolCond.GetKind() == condition.ErrorOperandKind {
				return boolCond
			}

			if bool(boolCond.(condition.BooleanOperand)) {
				return trueOperand.Evaluate(event, frames)
			}
			return falseOperand.Evaluate(event, frames)
		}, condOperand, trueOperand, falseOperand)
}

// Math functions

// abs(x) - Absolute value
func (repo *CompareCondRepo) funcAbs(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("abs() requires exactly one argument"))
	}

	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)

			// Handle undefined - math on undefined returns undefined
			if arg.GetKind() == condition.UndefinedOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Handle null
			if arg.GetKind() == condition.NullOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Convert to numeric
			numericArg := arg.Convert(condition.FloatOperandKind)
			if numericArg.GetKind() == condition.ErrorOperandKind {
				return numericArg
			}

			val := float64(numericArg.(condition.FloatOperand))
			return condition.NewFloatOperand(math.Abs(val))
		}, argOperand, condition.StringOperand("abs"))
}

// ceil(x) - Ceiling (round up)
func (repo *CompareCondRepo) funcCeil(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("ceil() requires exactly one argument"))
	}

	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)

			// Handle undefined
			if arg.GetKind() == condition.UndefinedOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Handle null
			if arg.GetKind() == condition.NullOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Convert to numeric
			numericArg := arg.Convert(condition.FloatOperandKind)
			if numericArg.GetKind() == condition.ErrorOperandKind {
				return numericArg
			}

			val := float64(numericArg.(condition.FloatOperand))
			return condition.NewFloatOperand(math.Ceil(val))
		}, argOperand, condition.StringOperand("ceil"))
}

// floor(x) - Floor (round down)
func (repo *CompareCondRepo) funcFloor(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 1 {
		return condition.NewErrorOperand(fmt.Errorf("floor() requires exactly one argument"))
	}

	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)

			// Handle undefined
			if arg.GetKind() == condition.UndefinedOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Handle null
			if arg.GetKind() == condition.NullOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Convert to numeric
			numericArg := arg.Convert(condition.FloatOperandKind)
			if numericArg.GetKind() == condition.ErrorOperandKind {
				return numericArg
			}

			val := float64(numericArg.(condition.FloatOperand))
			return condition.NewFloatOperand(math.Floor(val))
		}, argOperand, condition.StringOperand("floor"))
}

// round(x) or round(x, digits) - Round to n decimal places
func (repo *CompareCondRepo) funcRound(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) < 1 || len(n.Args) > 2 {
		return condition.NewErrorOperand(fmt.Errorf("round() requires one or two arguments: round(x) or round(x, digits)"))
	}

	argOperand := repo.evalAstNode(n.Args[0], scope)
	if argOperand.GetKind() == condition.ErrorOperandKind {
		return argOperand
	}

	var digitsOperand condition.Operand
	if len(n.Args) == 2 {
		digitsOperand = repo.evalAstNode(n.Args[1], scope)
		if digitsOperand.GetKind() == condition.ErrorOperandKind {
			return digitsOperand
		}
	}

	// Build args list for hash - only include non-nil operands
	hashArgs := []condition.Operand{argOperand, condition.StringOperand("round")}
	if digitsOperand != nil {
		hashArgs = append(hashArgs, digitsOperand)
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			arg := argOperand.Evaluate(event, frames)

			// Handle undefined
			if arg.GetKind() == condition.UndefinedOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Handle null
			if arg.GetKind() == condition.NullOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Convert to numeric
			numericArg := arg.Convert(condition.FloatOperandKind)
			if numericArg.GetKind() == condition.ErrorOperandKind {
				return numericArg
			}

			val := float64(numericArg.(condition.FloatOperand))

			// Get digits (default to 0 for whole number rounding)
			digits := int64(0)
			if digitsOperand != nil {
				digitsVal := digitsOperand.Evaluate(event, frames)
				if digitsVal.GetKind() == condition.UndefinedOperandKind {
					return condition.NewUndefinedOperand(nil)
				}
				digitsNumeric := digitsVal.Convert(condition.IntOperandKind)
				if digitsNumeric.GetKind() == condition.ErrorOperandKind {
					return digitsNumeric
				}
				digits = int64(digitsNumeric.(condition.IntOperand))
			}

			// Round to specified decimal places
			shift := math.Pow(10, float64(digits))
			rounded := math.Round(val*shift) / shift
			return condition.NewFloatOperand(rounded)
		}, hashArgs...)
}

// min(a, b, ...) - Minimum of multiple values
func (repo *CompareCondRepo) funcMin(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) < 2 {
		return condition.NewErrorOperand(fmt.Errorf("min() requires at least two arguments"))
	}

	argOperands := make([]condition.Operand, len(n.Args))
	for i, arg := range n.Args {
		argOperands[i] = repo.evalAstNode(arg, scope)
		if argOperands[i].GetKind() == condition.ErrorOperandKind {
			return argOperands[i]
		}
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			minVal := math.Inf(1) // Positive infinity
			hasValue := false

			for _, argOp := range argOperands {
				arg := argOp.Evaluate(event, frames)

				// Handle undefined - skip this value
				if arg.GetKind() == condition.UndefinedOperandKind {
					continue
				}

				// Handle null - skip this value
				if arg.GetKind() == condition.NullOperandKind {
					continue
				}

				// Convert to numeric
				numericArg := arg.Convert(condition.FloatOperandKind)
				if numericArg.GetKind() == condition.ErrorOperandKind {
					return numericArg
				}

				val := float64(numericArg.(condition.FloatOperand))
				if !hasValue || val < minVal {
					minVal = val
					hasValue = true
				}
			}

			// If all values were undefined/null, return undefined
			if !hasValue {
				return condition.NewUndefinedOperand(nil)
			}

			return condition.NewFloatOperand(minVal)
		}, append(argOperands, condition.StringOperand("min"))...)
}

// max(a, b, ...) - Maximum of multiple values
func (repo *CompareCondRepo) funcMax(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) < 2 {
		return condition.NewErrorOperand(fmt.Errorf("max() requires at least two arguments"))
	}

	argOperands := make([]condition.Operand, len(n.Args))
	for i, arg := range n.Args {
		argOperands[i] = repo.evalAstNode(arg, scope)
		if argOperands[i].GetKind() == condition.ErrorOperandKind {
			return argOperands[i]
		}
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			maxVal := math.Inf(-1) // Negative infinity
			hasValue := false

			for _, argOp := range argOperands {
				arg := argOp.Evaluate(event, frames)

				// Handle undefined - skip this value
				if arg.GetKind() == condition.UndefinedOperandKind {
					continue
				}

				// Handle null - skip this value
				if arg.GetKind() == condition.NullOperandKind {
					continue
				}

				// Convert to numeric
				numericArg := arg.Convert(condition.FloatOperandKind)
				if numericArg.GetKind() == condition.ErrorOperandKind {
					return numericArg
				}

				val := float64(numericArg.(condition.FloatOperand))
				if !hasValue || val > maxVal {
					maxVal = val
					hasValue = true
				}
			}

			// If all values were undefined/null, return undefined
			if !hasValue {
				return condition.NewUndefinedOperand(nil)
			}

			return condition.NewFloatOperand(maxVal)
		}, append(argOperands, condition.StringOperand("max"))...)
}

// pow(base, exponent) - Power/exponentiation
func (repo *CompareCondRepo) funcPow(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 2 {
		return condition.NewErrorOperand(fmt.Errorf("pow() requires exactly two arguments: base and exponent"))
	}

	baseOperand := repo.evalAstNode(n.Args[0], scope)
	if baseOperand.GetKind() == condition.ErrorOperandKind {
		return baseOperand
	}

	expOperand := repo.evalAstNode(n.Args[1], scope)
	if expOperand.GetKind() == condition.ErrorOperandKind {
		return expOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			base := baseOperand.Evaluate(event, frames)

			// Handle undefined
			if base.GetKind() == condition.UndefinedOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Handle null
			if base.GetKind() == condition.NullOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			exp := expOperand.Evaluate(event, frames)

			// Handle undefined
			if exp.GetKind() == condition.UndefinedOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Handle null
			if exp.GetKind() == condition.NullOperandKind {
				return condition.NewUndefinedOperand(nil)
			}

			// Convert to numeric
			numericBase := base.Convert(condition.FloatOperandKind)
			if numericBase.GetKind() == condition.ErrorOperandKind {
				return numericBase
			}

			numericExp := exp.Convert(condition.FloatOperandKind)
			if numericExp.GetKind() == condition.ErrorOperandKind {
				return numericExp
			}

			baseVal := float64(numericBase.(condition.FloatOperand))
			expVal := float64(numericExp.(condition.FloatOperand))

			result := math.Pow(baseVal, expVal)
			return condition.NewFloatOperand(result)
		}, baseOperand, expOperand, condition.StringOperand("pow"))
}

// Array aggregation functions (iterate array and aggregate values)

// minOf(array_path, element_name, expression) - Minimum value from array
func (repo *CompareCondRepo) funcMinOf(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 3 {
		return condition.NewErrorOperand(fmt.Errorf("minOf() requires 3 arguments: array_path, element_name, expression"))
	}

	pathOperand := repo.evalAstNode(n.Args[0], scope)
	if pathOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("minOf() first argument must be string path"))
	}
	arrayPath := string(pathOperand.(condition.StringOperand))

	elemOperand := repo.evalAstNode(n.Args[1], scope)
	if elemOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("minOf() element name must be string"))
	}
	elementName := string(elemOperand.(condition.StringOperand))

	exprAst := n.Args[2]

	arrayAddress, err := getAttributePathAddress(arrayPath+"[]", scope)
	if err != nil {
		return condition.NewErrorOperand(err)
	}

	scopeForPath, expandedPath := expandPath(arrayPath+"[]", scope)
	addr, err := scopeForPath.AttrDictRec.AttributePathToAddress(expandedPath)
	if err != nil {
		return condition.NewErrorOperand(err)
	}
	newDictRec := scopeForPath.AttrDictRec.AddressToDictionaryRec(addr)
	newPath := scopeForPath.AttrDictRec.AddressToFullPath(addr)

	newScope := &ForEachScope{
		Path:         newPath,
		Element:      elementName,
		NestingLevel: scope.NestingLevel + 1,
		ParentScope:  scope,
		AttrDictRec:  newDictRec,
	}

	exprOperand := repo.evalAstNode(exprAst, newScope)
	if exprOperand.GetKind() == condition.ErrorOperandKind {
		return exprOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
			if err != nil {
				return condition.NewUndefinedOperand(nil)
			}

			if numElements == 0 {
				return condition.NewUndefinedOperand(nil)
			}

			minVal := math.Inf(1)
			hasValue := false
			parentsFrame := frames[arrayAddress.ParentParameterIndex]
			currentAddressLen := len(arrayAddress.Address)
			currentAddress := make([]int, currentAddressLen+1)
			copy(currentAddress, arrayAddress.Address)

			for i := 0; i < numElements; i++ {
				currentAddress[currentAddressLen] = i
				newFrame := objectmap.GetNestedAttributeByAddress(parentsFrame, currentAddress)

				if newFrame == nil {
					continue
				}

				frames[newScope.NestingLevel] = newFrame

				result := exprOperand.Evaluate(event, frames)

				if result.GetKind() == condition.UndefinedOperandKind ||
					result.GetKind() == condition.NullOperandKind {
					continue
				}

				numeric := result.Convert(condition.FloatOperandKind)
				if numeric.GetKind() != condition.ErrorOperandKind {
					val := float64(numeric.(condition.FloatOperand))
					if !hasValue || val < minVal {
						minVal = val
						hasValue = true
					}
				}
			}

			if !hasValue {
				return condition.NewUndefinedOperand(nil)
			}

			return condition.NewFloatOperand(minVal)
		}, pathOperand, elemOperand)
}

// maxOf(array_path, element_name, expression) - Maximum value from array
func (repo *CompareCondRepo) funcMaxOf(n *ast.CallExpr, scope *ForEachScope) condition.Operand {
	if len(n.Args) != 3 {
		return condition.NewErrorOperand(fmt.Errorf("maxOf() requires 3 arguments: array_path, element_name, expression"))
	}

	pathOperand := repo.evalAstNode(n.Args[0], scope)
	if pathOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("maxOf() first argument must be string path"))
	}
	arrayPath := string(pathOperand.(condition.StringOperand))

	elemOperand := repo.evalAstNode(n.Args[1], scope)
	if elemOperand.GetKind() != condition.StringOperandKind {
		return condition.NewErrorOperand(fmt.Errorf("maxOf() element name must be string"))
	}
	elementName := string(elemOperand.(condition.StringOperand))

	exprAst := n.Args[2]

	arrayAddress, err := getAttributePathAddress(arrayPath+"[]", scope)
	if err != nil {
		return condition.NewErrorOperand(err)
	}

	scopeForPath, expandedPath := expandPath(arrayPath+"[]", scope)
	addr, err := scopeForPath.AttrDictRec.AttributePathToAddress(expandedPath)
	if err != nil {
		return condition.NewErrorOperand(err)
	}
	newDictRec := scopeForPath.AttrDictRec.AddressToDictionaryRec(addr)
	newPath := scopeForPath.AttrDictRec.AddressToFullPath(addr)

	newScope := &ForEachScope{
		Path:         newPath,
		Element:      elementName,
		NestingLevel: scope.NestingLevel + 1,
		ParentScope:  scope,
		AttrDictRec:  newDictRec,
	}

	exprOperand := repo.evalAstNode(exprAst, newScope)
	if exprOperand.GetKind() == condition.ErrorOperandKind {
		return exprOperand
	}

	return repo.CondFactory.NewExprOperand(
		func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
			numElements, err := event.GetNumElementsAtAddress(arrayAddress, frames)
			if err != nil {
				return condition.NewUndefinedOperand(nil)
			}

			if numElements == 0 {
				return condition.NewUndefinedOperand(nil)
			}

			maxVal := math.Inf(-1)
			hasValue := false
			parentsFrame := frames[arrayAddress.ParentParameterIndex]
			currentAddressLen := len(arrayAddress.Address)
			currentAddress := make([]int, currentAddressLen+1)
			copy(currentAddress, arrayAddress.Address)

			for i := 0; i < numElements; i++ {
				currentAddress[currentAddressLen] = i
				newFrame := objectmap.GetNestedAttributeByAddress(parentsFrame, currentAddress)

				if newFrame == nil {
					continue
				}

				frames[newScope.NestingLevel] = newFrame

				result := exprOperand.Evaluate(event, frames)

				if result.GetKind() == condition.UndefinedOperandKind ||
					result.GetKind() == condition.NullOperandKind {
					continue
				}

				numeric := result.Convert(condition.FloatOperandKind)
				if numeric.GetKind() != condition.ErrorOperandKind {
					val := float64(numeric.(condition.FloatOperand))
					if !hasValue || val > maxVal {
						maxVal = val
						hasValue = true
					}
				}
			}

			if !hasValue {
				return condition.NewUndefinedOperand(nil)
			}

			return condition.NewFloatOperand(maxVal)
		}, pathOperand, elemOperand)
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

	// Check if this is an undefined-related comparison
	isUndefinedCheck := xOperand.GetKind() == condition.UndefinedOperandKind ||
		yOperand.GetKind() == condition.UndefinedOperandKind

	if negate {
		// For undefined checks, use category-level NOT (DefaultCatList mechanism)
		// For regular negations, use evaluation-level NOT (undefined propagation)
		if isUndefinedCheck {
			// Category-level NOT for undefined checks
			return condition.NewNotCond(repo.processCompareCondition(condition.NewCompareCond(compareOp, xOperand, yOperand), scope))
		} else {
			// For regular negations: instead of NOT(field == value),
			// create direct CompareNotEqualOp comparison
			// This allows undefined propagation to work naturally
			var negatedOp condition.CompareOp
			switch compareOp {
			case condition.CompareEqualOp:
				negatedOp = condition.CompareNotEqualOp
			case condition.CompareLessOp:
				negatedOp = condition.CompareGreaterOrEqualOp
			case condition.CompareGreaterOp:
				negatedOp = condition.CompareLessOrEqualOp
			case condition.CompareLessOrEqualOp:
				negatedOp = condition.CompareGreaterOp
			case condition.CompareGreaterOrEqualOp:
				negatedOp = condition.CompareLessOp
			case condition.CompareNotEqualOp:
				negatedOp = condition.CompareEqualOp
			}
			return repo.processCompareCondition(condition.NewCompareCond(negatedOp, xOperand, yOperand), scope)
		}
	} else {
		return repo.processCompareCondition(condition.NewCompareCond(compareOp, xOperand, yOperand), scope)
	}
}

func (repo *CompareCondRepo) processExprCondition(exprCondition *condition.ExprCondition, scope *ForEachScope) condition.Condition {
	if scope.ParentScope != nil {
		panic("must be called from root scope")
	}

	// Preprocess expression to handle reserved keywords
	expr := preprocessExpression(exprCondition.Expr)

	// Convert the expression to an AST node tree
	node, err := parser.ParseExpr(expr)

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
		// Handle boolean literals
		if n.Name == "true" {
			return repo.CondFactory.NewBooleanOperand(true)
		}
		if n.Name == "false" {
			return repo.CondFactory.NewBooleanOperand(false)
		}
		// Handle undefined literal
		if n.Name == "undefined" {
			return condition.NewUndefinedOperand(nil)
		}
		// Handle null literal
		if n.Name == "null" {
			return condition.NewNullOperand(nil)
		}
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
		case "now":
			if len(n.Args) != 0 {
				return condition.NewErrorOperand(fmt.Errorf("now() function takes no arguments"))
			}
			// Return a function that evaluates to current time at runtime
			return repo.CondFactory.NewExprOperand(
				func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
					return condition.NewTimeOperand(time.Now())
				}, condition.StringOperand(funcName)) // funcName as hash seed
		case "regexpMatch":
			return funcRegexpMatch(repo, n, scope)
		case "hasValue":
			return funcHasValue(repo, n, scope)
		case "isEqualToAnyWithDate":
			return funcIsEqualToAnyWithDate(repo, n, scope)
		case "isEqualToAny":
			return repo.funcIsEqualToAny(n, scope)
		case "forAll", "all":
			return repo.funcForAll(n, scope)
		case "forSome", "any":
			return repo.funcForSome(n, scope)
		case "count":
			return repo.funcCount(n, scope)
		case "sum":
			return repo.funcSum(n, scope)
		case "avg":
			return repo.funcAvg(n, scope)
		case "minOf":
			return repo.funcMinOf(n, scope)
		case "maxOf":
			return repo.funcMaxOf(n, scope)
		case "length":
			return repo.funcLength(n, scope)
		case "days":
			return repo.funcDays(n, scope)
		case "hours":
			return repo.funcHours(n, scope)
		case "minutes":
			return repo.funcMinutes(n, scope)
		case "seconds":
			return repo.funcSeconds(n, scope)
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
		case "ifFunc": // "if" is preprocessed to "ifFunc" to avoid Go keyword conflict
			return repo.funcIf(n, scope)
		case "abs":
			return repo.funcAbs(n, scope)
		case "ceil":
			return repo.funcCeil(n, scope)
		case "floor":
			return repo.funcFloor(n, scope)
		case "round":
			return repo.funcRound(n, scope)
		case "min":
			return repo.funcMin(n, scope)
		case "max":
			return repo.funcMax(n, scope)
		case "pow":
			return repo.funcPow(n, scope)
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
			// Include operator token in args to distinguish a+b from a-b, a*b, a/b in hash
			opOperand := condition.NewStringOperand(n.Op.String())
			return repo.CondFactory.NewExprOperand(
				func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
					xVal := xOperand.Evaluate(event, frames)
					// Handle undefined values - arithmetic with undefined returns undefined
					if xVal.GetKind() == condition.UndefinedOperandKind {
						return condition.NewUndefinedOperand(nil)
					}
					// Handle null values - arithmetic with null returns null (not error)
					// Let comparison operators handle null according to their semantics
					if xVal.GetKind() == condition.NullOperandKind {
						return condition.NewNullOperand(nil)
					}
					xVal = xVal.Convert(condition.FloatOperandKind)
					if xVal.GetKind() == condition.ErrorOperandKind {
						return xVal
					}
					lv := float64(xVal.(condition.FloatOperand))

					yVal := yOperand.Evaluate(event, frames)
					// Handle undefined values - arithmetic with undefined returns undefined
					if yVal.GetKind() == condition.UndefinedOperandKind {
						return condition.NewUndefinedOperand(nil)
					}
					// Handle null values - arithmetic with null returns null (not error)
					if yVal.GetKind() == condition.NullOperandKind {
						return condition.NewNullOperand(nil)
					}
					yVal = yVal.Convert(condition.FloatOperandKind)
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
						if rv == 0 {
							return condition.NewErrorOperand(fmt.Errorf("division by zero"))
						}
						return condition.NewFloatOperand(lv / rv)
					default:
						return condition.NewErrorOperand(fmt.Errorf("unsupported operator: %s", n.Op.String()))
					}
				}, opOperand, xOperand, yOperand)
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
		case token.SUB:
			// Handle unary minus for negative numbers
			operand := repo.evalAstNode(n.X, scope)
			if operand.GetKind() == condition.ErrorOperandKind {
				return operand
			}
			// Negate the operand
			return repo.CondFactory.NewExprOperand(
				func(event *objectmap.ObjectAttributeMap, frames []interface{}) condition.Operand {
					val := operand.Evaluate(event, frames)
					if val.GetKind() == condition.ErrorOperandKind {
						return val
					}
					// Convert to numeric and negate
					numericVal := val.Convert(condition.FloatOperandKind)
					if numericVal.GetKind() == condition.ErrorOperandKind {
						return numericVal
					}
					return condition.NewFloatOperand(-float64(numericVal.(condition.FloatOperand)))
				}, operand)
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
			// hasValue returns true only if field exists with a non-null value
			// Both null and undefined are considered "no value"
			result := kind != condition.NullOperandKind && kind != condition.UndefinedOperandKind
			if false {  // Disabled debug
				fmt.Printf("[DEBUG hasValue] arg kind=%v, result=%v\n", kind, result)
			}
			return condition.NewBooleanOperand(result)
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
			// Handle null values - return false instead of panicking
			if kind == condition.NullOperandKind {
				return condition.NewBooleanOperand(false)
			}
			argString := string(arg.Convert(condition.StringOperandKind).(condition.StringOperand))
			result := condition.NewBooleanOperand(re.MatchString(argString))
			return result
		}, patternOperand, argOperand) // Include pattern in hash to distinguish different regexes
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
			if address.GetKind() == condition.UndefinedOperandKind {
				return address
			}
			addressOp := address.(*condition.AddressOperand)
			val := objectmap.GetNestedAttributeByAddress(
				frames[addressOp.ParameterIndex], addressOp.Address)
			if val == nil {
				// Distinguish missing field from explicit null
				// Get the field path from the FULL address (not scope-relative)
				fieldPath := event.DictRec.AddressToFullPath(addressOp.FullAddress)

				// Check if field exists in original event
				if event.FieldExistsInOriginalEvent(fieldPath) {
					// Field exists but is explicitly null
					return condition.NewNullOperand(addressOp)
				} else {
					// Field is missing (undefined)
					return condition.NewUndefinedOperand(addressOp)
				}
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
							case condition.UndefinedOperandKind:
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
						// Handle undefined (missing field used as array index)
						if io.GetKind() == condition.UndefinedOperandKind {
							return io // Propagate undefined
						}
						// Must be IntOperand at this point
						if io.GetKind() != condition.IntOperandKind {
							return condition.NewErrorOperand(fmt.Errorf("array index must be integer, got %v", io.GetKind()))
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
