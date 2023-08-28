package condition

import (
	"github.com/rulestone/immutable"
	"github.com/rulestone/types"
)

func AndOrSetToCondition(set types.AndOrSet) Condition {
	return NewAndCond(types.MapSlice(
		set.ToSlice(), func(orSet immutable.Set[types.Category]) Condition {
			return NewOrCond(types.MapSlice(
				orSet.ToSlice(), func(cat types.Category) Condition {
					return NewCategoryCond(cat)
				})...)
		})...)
}

func CategoryArraysToCondition(cats [][]types.Category) Condition {
	return NewAndCond(types.MapSlice(
		cats, func(orSet []types.Category) Condition {
			return NewOrCond(types.MapSlice[types.Category, Condition](
				orSet, func(cat types.Category) Condition {
					return NewCategoryCond(cat)
				})...)
		})...)
}
