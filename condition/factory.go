package condition

import (
	"github.com/atlasgurus/rulestone/types"
	"github.com/zyedidia/generic/hashmap"
)

type Factory struct {
	OperandCache   *hashmap.Map[Operand, Operand]
	ConditionCache *hashmap.Map[Condition, Condition]
}

func NewFactory() *Factory {
	return &Factory{
		OperandCache:   types.NewHashMap[Operand, Operand](),
		ConditionCache: types.NewHashMap[Condition, Condition](),
	}
}

func (factory *Factory) CacheOperand(operand Operand) Operand {
	result, ok := factory.OperandCache.Get(operand)
	if !ok {
		factory.OperandCache.Put(operand, operand)
		result = operand
	}
	return result
}

func (factory *Factory) NewIntOperand(val int64) Operand {
	return factory.CacheOperand(NewIntOperand(val))
}

func (factory *Factory) NewFloatOperand(val float64) Operand {
	return factory.CacheOperand(NewFloatOperand(val))
}

func (factory *Factory) NewStringOperand(val string) Operand {
	return factory.CacheOperand(NewStringOperand(val))
}

func (factory *Factory) NewBooleanOperand(val bool) Operand {
	return factory.CacheOperand(NewBooleanOperand(val))
}

func (factory *Factory) NewAttributeOperand(val string) Operand {
	return factory.CacheOperand(NewAttributeOperand(val))
}

func (factory *Factory) NewAddressOperand(address []int, fullAddress []int, parameterIndex int, exprOperand *ExprOperand) Operand {
	if exprOperand == nil {
		return factory.CacheOperand(NewAddressOperand(address, fullAddress, parameterIndex, exprOperand))
	} else {
		return NewAddressOperand(address, fullAddress, parameterIndex, exprOperand)
	}
}

func (factory *Factory) NewSelOperand(base Operand, selector string) Operand {
	return factory.CacheOperand(NewSelOperand(base, selector))
}

func (factory *Factory) NewIndexOperand(base Operand, indexExpr Operand) Operand {
	return factory.CacheOperand(NewIndexOperand(base, indexExpr))
}

func (factory *Factory) NewErrorOperand(val error) Operand {
	return factory.CacheOperand(NewErrorOperand(val))
}

func (factory *Factory) NewExprOperand(f EvalOperandFunc, args ...Operand) *ExprOperand {
	return factory.CacheOperand(NewExprOperand(f, args...)).(*ExprOperand)
}

func (factory *Factory) CacheCondition(cond Condition) Condition {
	result, ok := factory.ConditionCache.Get(cond)
	if !ok {
		factory.ConditionCache.Put(cond, cond)
		result = cond
	}
	return result
}

func (factory *Factory) NewCompareCond(op CompareOp, l Operand, r Operand) Condition {
	return factory.CacheCondition(NewCompareCond(op, l, r))
}

func (factory *Factory) NewAndCond(cond ...Condition) Condition {
	return factory.CacheCondition(NewAndCond(cond...))
}

func (factory *Factory) NewOrCond(cond ...Condition) Condition {
	return factory.CacheCondition(NewOrCond(cond...))
}

func (factory *Factory) NewNotCond(cond Condition) Condition {
	return factory.CacheCondition(NewNotCond(cond))
}
