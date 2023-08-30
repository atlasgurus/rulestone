package condition

import (
	"github.com/atlasgurus/rulestone/types"
)

type RuleIdType uint32

type Rule struct {
	RuleId RuleIdType
	Cond   Condition
}

func NewRule(RuleId RuleIdType, Cond Condition) *Rule {
	return &Rule{RuleId: RuleId, Cond: Cond}
}

type RuleIndexType int32

type RuleRec struct {
	Rule      *Rule
	RuleIndex RuleIndexType
}

type RuleRepo struct {
	Rules []*Rule
}

func NewRuleRepo(rules []*Rule) *RuleRepo {
	return &RuleRepo{Rules: rules}
}

func (repo *RuleRepo) Register(rule *Rule) {
	repo.Rules = append(repo.Rules, rule)
}

func AndOrTablesToRuleRepo(tables [][][]types.Category) *RuleRepo {
	var rules []*Rule
	for ruleIndex, andList := range tables {
		var orConds []Condition

		for _, orList := range andList {
			var catConds []Condition
			for _, cat := range orList {
				catConds = append(catConds, NewCategoryCond(cat))
			}
			orConds = append(orConds, NewOrCond(catConds...))
		}
		rules = append(rules, &Rule{
			RuleId: RuleIdType(ruleIndex),
			Cond:   NewAndCond(orConds...),
		})
	}
	return &RuleRepo{Rules: rules}
}
