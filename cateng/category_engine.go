package cateng

import (
	"fmt"
	"sync/atomic"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
)

type Metrics struct {
	NumBitMaskChecks    uint64
	NumMaskArrayLookups uint64
	NumBitMaskMatches   uint64
	Comment             string
	NumMismatches       uint
}

type CategoryEngine struct {
	ruleRepo     *condition.RuleRepo
	FilterTables FilterTables
	Metrics      Metrics
}

func NewCategoryEngine(repo *condition.RuleRepo, options *Options) *CategoryEngine {
	var result CategoryEngine

	result.ruleRepo = repo
	result.FilterTables = BuildFilterTables(repo, options)
	if options == nil {
		result.Metrics.Comment = "non optimized"
	} else {
		result.Metrics.Comment = "optimized"
	}
	return &result
}

func applyCatSetMasks(csmList []*CatSetMask, matchMaskArray []types.Mask, result *[]condition.RuleIdType, f *CategoryEngine) {
	for _, csm := range csmList {
		v := matchMaskArray[csm.Index1-1]
		atomic.AddUint64(&f.Metrics.NumMaskArrayLookups, 1)
		if v != -1 {
			newV := v | csm.Mask
			matchMaskArray[csm.Index1-1] = newV
			atomic.AddUint64(&f.Metrics.NumBitMaskChecks, 1)
			if newV == -1 {
				// We got a match.
				catSetFilter := f.FilterTables.CatSetFilters[csm.Index1-1]

				atomic.AddUint64(&f.Metrics.NumBitMaskMatches, 1)

				// Process the synthetic categories from the set.
				if len(catSetFilter.CatSetMasks) > 0 {
					applyCatSetMasks(catSetFilter.CatSetMasks, matchMaskArray, result, f)
				}
				for _, cfr := range catSetFilter.RuleSet {
					*result = append(*result, cfr.RuleId)
				}
			}
		}
	}
}

func (f *CategoryEngine) MatchEvent(cats []types.Category) []condition.RuleIdType {
	matchMaskArray := make([]types.Mask, len(f.FilterTables.NegCats)+len(f.FilterTables.CatSetFilters))
	result := make([]condition.RuleIdType, 0, 100)

	defaultCatMap := make([]bool, len(f.FilterTables.DefaultCategories))

	catToCatSetMask := f.FilterTables.CatToCatSetMask
	for _, cat := range cats {
		if i, ok := f.FilterTables.DefaultCategories[cat]; ok {
			// The condition evaluated to false for the category with the default value of true.
			// Take note of that to avoid including this category in the list later in this function.
			defaultCatMap[i] = true
		}
		csml := catToCatSetMask.Get(cat)
		if csml != nil {
			applyCatSetMasks(csml, matchMaskArray, &result, f)
		}
	}

	// Now process default categories
	for i, cat := range f.FilterTables.DefaultCatList {
		if !defaultCatMap[i] {
			negCat, found := f.FilterTables.NegCats[cat]
			if !found {
				panic("negCat must exist for this")
			}
			csml := catToCatSetMask.Get(negCat)
			if csml != nil {
				applyCatSetMasks(csml, matchMaskArray, &result, f)
			}
		}
	}
	return result
}

func (f *CategoryEngine) PrintMetrics() {
	fmt.Printf("%s NumMaskArrayLookups: %d\n", f.Metrics.Comment, f.Metrics.NumMaskArrayLookups)
	fmt.Printf("%s NumBitMaskChecks:    %d\n", f.Metrics.Comment, f.Metrics.NumBitMaskChecks)
	fmt.Printf("%s NumBitMaskMatches:   %d\n", f.Metrics.Comment, f.Metrics.NumBitMaskMatches)
}
