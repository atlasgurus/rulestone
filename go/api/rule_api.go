package api

import (
	"encoding/json"
	"errors"
	c "github.com/rulestone/condition"
	"github.com/rulestone/types"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
)

type RuleApi struct {
	ctx *types.AppContext
}

type RuleDefinition struct {
	Name      string
	Metadata  map[string]interface{}
	Condition c.Condition
}

type Condition struct {
	// Case 1. Compare Field against Value.  Verify that Field1, Field2, Operation, Arg(s), ForEach are empty
	Field string `json:"field,omitempty"`
	// Operator: comparison operator =, !=, >
	Operator string      `json:"operator,omitempty"`
	Value    interface{} `json:"value,omitempty"`

	// Case 2. Compare two fields.  Verify that Value, Field, Operator, Args, Arg, ForEach are empty
	Field1 string `json:"field1,omitempty"`
	Field2 string `json:"field2,omitempty"`

	// Case 3. Boolean operation on Args. Verify that Value, Field*, Operator, Arg(s), ForEach are empty
	// Operation: logical operation AND, OR
	Operation string       `json:"operation,omitempty"`
	Args      []*Condition `json:"args,omitempty"`

	// Case 4. Boolean NOT operation on Arg. Verify that Value, Field*, Operator, Args, ForEach are empty
	// Operation: logical operation NOT
	Arg *Condition `json:"arg,omitempty"`

	// Case 5,6. ForEach Verify that Value, Field*, Operation, Operator, Arg(s) are empty
	ForAll  *ForEach `json:"for_all,omitempty"`
	ForSome *ForEach `json:"for_some,omitempty"`
	Comment string   `json:"comment,omitempty"`

	// Case 7. symbolic non-json boolean expression, e.g. {"expression":"2*(10+foo)>25"}
	Expression string `json:"expression,omitempty"`
}

type ForEach struct {
	Path      string     `json:"path"`
	Element   string     `json:"element"`
	Condition *Condition `json:"condition,omitempty"`
	ForAll    *ForEach   `json:"for_all,omitempty"`
	ForSome   *ForEach   `json:"for_some,omitempty"`
	Comment   string     `json:"comment,omitempty"`
}

type Rule struct {
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Condition  *Condition             `json:"condition,omitempty"`
	ForAll     *ForEach               `json:"for_all,omitempty"`
	ForSome    *ForEach               `json:"for_some,omitempty"`
	Expression string                 `json:"expression,omitempty"`
}

func (api *RuleApi) ReadRule(r io.Reader, fileType string) (*Rule, error) {
	var result Rule

	switch strings.ToLower(fileType) {
	case "json":
		decoder := json.NewDecoder(r)
		if err := decoder.Decode(&result); err != nil {
			return nil, api.ctx.Errorf("error parsing JSON:%s", err)
		}
	case "yaml", "yml":
		decoder := yaml.NewDecoder(r)
		if err := decoder.Decode(&result); err != nil {
			return nil, api.ctx.Errorf("error parsing YAML:%s", err)
		}
	default:
		return nil, api.ctx.Errorf("unsupported file type:%s", fileType)
	}

	return &result, nil
}

func (api *RuleApi) RuleToRuleDefinition(rule *Rule) (*RuleDefinition, error) {
	var result RuleDefinition
	firstError := api.ctx.NumErrors()

	if rule.Condition != nil {
		if rule.ForAll != nil {
			return nil, api.ctx.Errorf("for_all is not expected here")
		}
		if rule.ForSome != nil {
			return nil, api.ctx.Errorf("for_some is not expected here")
		}
		if rule.Expression != "" {
			return nil, api.ctx.Errorf("expression is not expected here")
		}
		result = RuleDefinition{
			Name:      "",
			Metadata:  rule.Metadata,
			Condition: api.mapRuleCondition(rule.Condition)}
	} else if rule.ForAll != nil {
		if rule.Condition != nil {
			return nil, api.ctx.Errorf("condition is not expected here")
		}
		if rule.ForSome != nil {
			return nil, api.ctx.Errorf("for_some is not expected here")
		}
		if rule.Expression != "" {
			return nil, api.ctx.Errorf("expression is not expected here")
		}
		result = RuleDefinition{
			Name:      "",
			Metadata:  rule.Metadata,
			Condition: api.mapForEachCondition(rule.ForAll, true)}
	} else if rule.ForSome != nil {
		if rule.Condition != nil {
			return nil, api.ctx.Errorf("condition is not expected here")
		}
		if rule.ForAll != nil {
			return nil, api.ctx.Errorf("for_all is not expected here")
		}
		if rule.Expression != "" {
			return nil, api.ctx.Errorf("expression is not expected here")
		}
		result = RuleDefinition{
			Name:      "",
			Metadata:  rule.Metadata,
			Condition: api.mapForEachCondition(rule.ForSome, false)}
	} else if rule.Expression != "" {
		if rule.Condition != nil {
			return nil, api.ctx.Errorf("condition is not expected here")
		}
		if rule.ForAll != nil {
			return nil, api.ctx.Errorf("for_all is not expected here")
		}
		if rule.ForSome != nil {
			return nil, api.ctx.Errorf("for_some is not expected here")
		}
		result = RuleDefinition{
			Name:      "",
			Metadata:  rule.Metadata,
			Condition: api.mapRuleExpression(rule.Expression)}
	}

	if api.ctx.NumErrors() > firstError {
		// Return the first error if errors found
		return nil, api.ctx.GetError(firstError)
	}
	return &result, nil
}

type CondKind int8

const (
	InvalidCondKind            CondKind = 0
	CompareFieldKind                    = 1
	CompareFieldsKind                   = 2
	LogicalOperationKind                = 3
	UnaryLogicalOperationKind           = 4
	ForAllConditionKind                 = 5
	ForSomeConditionKind                = 6
	ForExpressionConditionKind          = 7
)

func (api *RuleApi) stringToCompareOp(s string) (c.CompareOp, error) {
	switch s {
	case "=":
		return c.CompareEqualOp, nil
	case ">":
		return c.CompareGreaterOp, nil
	case "<":
		return c.CompareLessOp, nil
	case "<=":
		return c.CompareLessOrEqualOp, nil
	case ">=":
		return c.CompareGreaterOrEqualOp, nil
	case "!=":
		return c.CompareNotEqualOp, nil
	default:
		return c.CompareInvalidOp, api.ctx.Errorf("invalid compare operation %s", s)
	}
}

func (api *RuleApi) mapRuleCondition(cond *Condition) c.Condition {
	kind, _ := api.getConditionKind(cond)
	switch kind {
	case InvalidCondKind:
		return c.NewErrorCondition(errors.New("InvalidCondKind"))
	case CompareFieldKind:
		if compareOp, err := api.stringToCompareOp(cond.Operator); err != nil {
			return c.NewErrorCondition(err)
		} else {
			valueOperand := c.NewInterfaceOperand(cond.Value, api.ctx)
			if valueOperand.GetKind() == c.ErrorOperandKind {
				return c.NewErrorCondition(valueOperand.(c.ErrorOperand))
			}
			return c.NewCompareCond(compareOp,
				c.NewAttributeOperand(cond.Field),
				valueOperand)
		}
	case CompareFieldsKind:
		if compareOp, err := api.stringToCompareOp(cond.Operator); err != nil {
			return c.NewErrorCondition(err)
		} else {
			return c.NewCompareCond(compareOp,
				c.NewAttributeOperand(cond.Field1),
				c.NewAttributeOperand(cond.Field2))
		}
	case LogicalOperationKind:
		switch cond.Operation {
		case "AND":
			return c.NewAndCond(types.MapSlice(cond.Args, api.mapRuleCondition)...)
		case "OR":
			return c.NewOrCond(types.MapSlice(cond.Args, api.mapRuleCondition)...)
		default:
			panic("invalid operation")
		}
	case UnaryLogicalOperationKind:
		switch cond.Operation {
		case "NOT":
			return c.NewNotCond(api.mapRuleCondition(cond.Arg))
		default:
			panic("invalid operation")
		}
	case ForAllConditionKind:
		return api.mapForEachCondition(cond.ForAll, true)
	case ForSomeConditionKind:
		return api.mapForEachCondition(cond.ForSome, false)
	case ForExpressionConditionKind:
		return api.mapRuleExpression(
			strings.ReplaceAll(cond.Expression, "\n", ""))
	default:
		panic("should not get here")
	}
}

type FieldRequirement int8

const (
	Not FieldRequirement = 0
	Opt                  = 1
	Yes                  = 2
)

type ConditionValidationMap struct {
	Field      FieldRequirement
	Operator   FieldRequirement
	Value      FieldRequirement
	Field1     FieldRequirement
	Field2     FieldRequirement
	Operation  FieldRequirement
	Args       FieldRequirement
	Arg        FieldRequirement
	ForAll     FieldRequirement
	ForSome    FieldRequirement
	Expression FieldRequirement
	Comment    FieldRequirement
}

func (api *RuleApi) validateField(fieldName string, req FieldRequirement, present bool) error {
	if present {
		if req == Not {
			return api.ctx.Errorf("field %s not expected here", fieldName)
		}
	} else {
		if req == Yes {
			return api.ctx.Errorf("field %s is missing", fieldName)
		}
	}
	return nil
}

func (api *RuleApi) validateCondRecord(cond *Condition, validationMap *ConditionValidationMap) error {
	if err := api.validateField("Field", validationMap.Field, cond.Field != ""); err != nil {
		return err
	}
	if err := api.validateField("Field1", validationMap.Field1, cond.Field1 != ""); err != nil {
		return err
	}
	if err := api.validateField("Field2", validationMap.Field2, cond.Field2 != ""); err != nil {
		return err
	}
	if err := api.validateField("Operation", validationMap.Operation, cond.Operation != ""); err != nil {
		return err
	}
	if err := api.validateField("Operator", validationMap.Operator, cond.Operator != ""); err != nil {
		return err
	}
	if err := api.validateField("Value", validationMap.Value, cond.Value != nil); err != nil {
		return err
	}
	if err := api.validateField("Args", validationMap.Args, cond.Args != nil); err != nil {
		return err
	}
	if err := api.validateField("Arg", validationMap.Arg, cond.Arg != nil); err != nil {
		return err
	}
	if err := api.validateField("ForAll", validationMap.ForAll, cond.ForAll != nil); err != nil {
		return err
	}
	if err := api.validateField("ForSome", validationMap.ForSome, cond.ForSome != nil); err != nil {
		return err
	}
	if err := api.validateField("Comment", validationMap.Comment, cond.Comment != ""); err != nil {
		return err
	}
	return nil
}

func (api *RuleApi) getConditionKind(cond *Condition) (CondKind, error) {
	if cond.Field != "" {
		// Case 1. Compare Field against Value.  Verify that Field1, Field2, Operation, Arg(s), ForEach are empty
		if err := api.validateCondRecord(
			cond, &ConditionValidationMap{Field: Yes, Value: Yes, Operator: Yes, Comment: Opt}); err != nil {
			return InvalidCondKind, err
		}
		return CompareFieldKind, nil
	}
	if cond.Field1 != "" {
		// Case 2. Compare two fields.  Verify that Value, Field, Operator, Args, Arg, ForEach are empty
		if err := api.validateCondRecord(
			cond, &ConditionValidationMap{Field1: Yes, Field2: Yes, Operator: Yes, Comment: Opt}); err != nil {
			return InvalidCondKind, err
		}
		return CompareFieldsKind, nil
	}
	if cond.Args != nil {
		// Case 3. Boolean operation on Args. Verify that Value, Field*, Operator, Arg(s), ForEach are empty
		if err := api.validateCondRecord(
			cond, &ConditionValidationMap{Args: Yes, Operation: Yes, Comment: Opt}); err != nil {
			return InvalidCondKind, err
		}
		return LogicalOperationKind, nil
	}
	if cond.Arg != nil {
		// Case 4. Boolean NOT operation on Arg. Verify that Value, Field*, Operator, Args, ForEach are empty
		if err := api.validateCondRecord(
			cond, &ConditionValidationMap{Arg: Yes, Operation: Yes, Comment: Opt}); err != nil {
			return InvalidCondKind, err
		}
		return UnaryLogicalOperationKind, nil
	}
	if cond.ForAll != nil {
		// Case 5. ForAll Verify that Value, Field*, Operation, Operator, Arg(s), ForSome are empty
		if err := api.validateCondRecord(
			cond, &ConditionValidationMap{ForAll: Yes, Comment: Opt}); err != nil {
			return InvalidCondKind, err
		}
		return ForAllConditionKind, nil
	}

	if cond.ForSome != nil {
		// Case 6. ForSome Verify that Value, Field*, Operation, Operator, Arg(s), ForAll are empty
		if err := api.validateCondRecord(
			cond, &ConditionValidationMap{ForSome: Yes, Comment: Opt}); err != nil {
			return InvalidCondKind, err
		}
		return ForSomeConditionKind, nil
	}

	if cond.Expression != "" {
		// Case 7. ForSome Verify that Value, Field*, Operation, Operator, Arg(s), ForAll are empty
		if err := api.validateCondRecord(
			cond, &ConditionValidationMap{Expression: Yes, Comment: Opt}); err != nil {
			return InvalidCondKind, err
		}
		return ForExpressionConditionKind, nil
	}

	return InvalidCondKind, api.ctx.Errorf("invalid condition encountered")
}

func (api *RuleApi) mapForEachCondition(each *ForEach, all bool) c.Condition {
	if each.Condition != nil {
		if each.ForAll != nil {
			return c.NewErrorCondition(api.ctx.NewError("for_all is not expected here"))
		}
		if each.ForSome != nil {
			return c.NewErrorCondition(api.ctx.NewError("for_some is not expected here"))
		}
		if all {
			return c.NewForAllCond(each.Element, each.Path, api.mapRuleCondition(each.Condition))
		} else {
			return c.NewForSomeCond(each.Element, each.Path, api.mapRuleCondition(each.Condition))
		}
	} else if each.ForAll != nil {
		if each.Condition != nil {
			return c.NewErrorCondition(api.ctx.NewError("condition is not expected here"))
		}
		if each.ForSome != nil {
			return c.NewErrorCondition(api.ctx.NewError("for_some is not expected here"))
		}
		if all {
			return c.NewForAllCond(each.Element, each.Path, api.mapForEachCondition(each.ForAll, true))
		} else {
			return c.NewForSomeCond(each.Element, each.Path, api.mapForEachCondition(each.ForAll, true))
		}
	} else if each.ForSome != nil {
		if each.Condition != nil {
			return c.NewErrorCondition(api.ctx.NewError("condition is not expected here"))
		}
		if each.ForAll != nil {
			return c.NewErrorCondition(api.ctx.NewError("for_all is not expected here"))
		}
		if all {
			return c.NewForAllCond(each.Element, each.Path, api.mapForEachCondition(each.ForSome, false))
		} else {
			return c.NewForSomeCond(each.Element, each.Path, api.mapForEachCondition(each.ForSome, false))
		}
	}
	return c.NewErrorCondition(api.ctx.NewError("invalid for_each condition"))
}

func (api *RuleApi) mapRuleExpression(expr string) c.Condition {
	return c.NewExprCondition(expr)
}

func NewRuleApi(ctx *types.AppContext) *RuleApi {
	return &RuleApi{ctx: ctx}
}
