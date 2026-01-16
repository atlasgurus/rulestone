package cateng

import (
	"fmt"
	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/immutable"
	"github.com/atlasgurus/rulestone/types"
	"github.com/zyedidia/generic/hashmap"
	"sort"
)

type CatSetMask struct {
	Mask   types.Mask
	Index1 types.Category
}

// SliceMap is a slice that behaves like a map with positive integer keys.
type SliceMap[T any] []T

// Set sets a value at an arbitrary index in the slice,
// automatically expanding the slice if needed.
func (s SliceMap[T]) Set(index int, value T) SliceMap[T] {
	if index < 0 {
		panic("")
	}

	// If the index is within the current length of the slice, just set the value.
	if index < len(s) {
		s[index] = value
		return s
	}

	// If the index is within the current capacity of the slice, expand the length and set the value.
	if index < cap(s) {
		result := s[0 : index+1]
		result[index] = value
		return result
	}

	// If the index is larger than the current capacity, create a new slice and copy values.
	newSlice := make([]T, index+1)
	copy(newSlice, s)
	newSlice[index] = value
	return newSlice
}

type CatSetMaskArray struct {
	//val map[types.Category][]*CatSetMask
	val0 SliceMap[[]*CatSetMask]
	val1 SliceMap[[]*CatSetMask]
}

func (array *CatSetMaskArray) Get(index types.Category) []*CatSetMask {
	if index < types.MaxCategory {
		if int(index) < len(array.val0) {
			return array.val0[index]
		} else {
			return nil
		}
	} else {
		index -= types.MaxCategory
		if int(index) < len(array.val1) {
			return array.val1[index]
		} else {
			return nil
		}
	}
}

func (array *CatSetMaskArray) Set(index types.Category, masks []*CatSetMask) {
	if index < types.MaxCategory {
		array.val0 = array.val0.Set(int(index), masks)
	} else {
		index -= types.MaxCategory
		array.val1 = array.val1.Set(int(index), masks)
	}
}

type RuleFilterRec struct {
	RuleIndex condition.RuleIndexType
	RuleId    condition.RuleIdType
}

type CatSetFilter struct {
	CatSetIndex1 types.Category
	RuleSet      []*RuleFilterRec
	CatSetMasks  []*CatSetMask
}

type FilterTables struct {
	CatToCatSetMask   *CatSetMaskArray
	CatSetFilters     []*CatSetFilter
	BuilderMetrics    BuilderMetrics
	RuleRecs          []*condition.RuleRec
	NegCats           map[types.Category]types.Category
	DefaultCategories map[types.Category]int
	DefaultCatList    []types.Category
}

type CatFilterSetType struct {
	CatSetIndex1 types.Category
	AndSet       types.AndOrSet
}

type CatSetRec struct {
	CatFilterSet CatFilterSetType
	RuleSet      []*RuleFilterRec
	CatSetMasks  []*CatSetMask
}

type CatFilter struct {
	AndSet types.AndOrSet
}

type BuilderMetrics struct {
	OrSetsRemoved    uint
	AndSetsRemoved   uint
	OrSetsInlined    uint
	AndOrSetsInlined uint
	AndOrSetsGCed    uint
}

type Options struct {
	OrOptimizationFreqThreshold  uint
	AndOptimizationFreqThreshold uint
	Verbose                      bool
}

type FilterBuilder struct {
	RuleRepo      *condition.RuleRepo
	CatFilterSets []*CatSetRec
	CatSetMap     *hashmap.Map[types.AndOrSet, *CatSetRec]
	NegCats       map[types.Category]types.Category
	RuleRecs      []*condition.RuleRec
	options       *Options
	metrics       BuilderMetrics
}

func (fb *FilterBuilder) registerAndSet(andSet types.AndOrSet) *CatSetRec {
	result, ok := fb.CatSetMap.Get(andSet)
	if !ok {
		result = &CatSetRec{CatFilterSet: CatFilterSetType{types.Category(len(fb.CatFilterSets) + 1), andSet}, RuleSet: []*RuleFilterRec{}, CatSetMasks: []*CatSetMask{}}
		fb.CatFilterSets = append(fb.CatFilterSets, result)
		fb.CatSetMap.Put(andSet, result)
	}
	return result
}

func (fb *FilterBuilder) registerNegativeCat(cat types.Category) types.Category {
	negCat, ok := fb.NegCats[cat]
	if !ok {
		negCat = cat + types.MaxCategory
		fb.NegCats[cat] = negCat
	}
	return negCat
}

func (fb *FilterBuilder) replaceFilterSet(set types.AndOrSet, with types.AndOrSet) bool {
	if !set.Equals(with) {
		csr, ok := fb.CatSetMap.Get(set)
		if !ok {
			panic("Unregistered set.")
		}
		newCsr :=
			&CatSetRec{
				CatFilterSet: CatFilterSetType{
					CatSetIndex1: csr.CatFilterSet.CatSetIndex1,
					AndSet:       with},
				RuleSet:     csr.RuleSet,
				CatSetMasks: csr.CatSetMasks}
		fb.CatFilterSets[newCsr.CatFilterSet.CatSetIndex1-1] = newCsr
		fb.CatSetMap.Remove(set)
		fb.CatSetMap.Put(with, newCsr)
		return true
	}
	return false
}

func (fb *FilterBuilder) unregisterFilterSet(set types.AndOrSet) {
	csr, ok := fb.CatSetMap.Get(set)
	if !ok {
		panic("Unregistered set.")
	}
	fb.CatFilterSets[csr.CatFilterSet.CatSetIndex1-1] = nil
	fb.CatSetMap.Remove(set)
}

func (fb *FilterBuilder) buildCatSetFilterForAndOrSets(set *types.AndOrSet) {
	catFilter := CatFilter{*set}
	catSetRec := fb.registerAndSet(catFilter.AndSet)
	catSetRec.RuleSet = append(catSetRec.RuleSet,
		&RuleFilterRec{1, 0})
}

func (fb *FilterBuilder) buildCatSetFiltersForAndOrSets(andOrSets []*types.AndOrSet) {
	for _, fs := range andOrSets {
		fb.buildCatSetFilterForAndOrSets(fs)
	}
}

func collectCategories(c condition.Condition) []types.Category {
	switch c.GetKind() {
	case condition.CategoryCondKind:
		return []types.Category{c.(*condition.CategoryCond).Cat}
	default:
		var result []types.Category
		for _, v := range c.GetOperands() {
			result = append(result, collectCategories(v)...)
		}
		return result
	}
}

func (fb *FilterBuilder) buildCatSetFilters(repo *condition.RuleRepo) {
	fb.RuleRecs = computeRuleRecs(repo.Rules)
	for _, c := range fb.RuleRecs {
		fb.buildCatSetFilter(c.Rule, c.RuleIndex)
	}
}

func computeRuleRecs(rules []*condition.Rule) []*condition.RuleRec {
	var nextIndex condition.RuleIndexType = -1
	var result []*condition.RuleRec
	for _, rule := range rules {
		nextIndex += 1
		result = append(result, &condition.RuleRec{Rule: rule, RuleIndex: nextIndex})
	}
	return result
}

func (fb *FilterBuilder) computeCatFilter(cond condition.Condition) CatFilter {
	switch cond.GetKind() {
	case condition.CategoryCondKind:
		cat := cond.(*condition.CategoryCond).Cat
		//if cat < 0 {
		//panic("Invalid category (< 0)")
		//}
		s1 := immutable.Of[types.Category](cat)
		s2 := immutable.Of[immutable.Set[types.Category]](*s1)
		return CatFilter{types.AndOrSet(*s2)}
	case condition.AndCondKind:
		return fb.processAndOp(cond)
	case condition.OrCondKind:
		return fb.processOrOp(cond)
	case condition.NotCondKind:
		return fb.processNotOp(cond.GetOperands()[0])
	}
	return CatFilter{}
}

func (fb *FilterBuilder) buildCatSetFilter(rule *condition.Rule, ruleIndex condition.RuleIndexType) {
	catFilter := fb.computeCatFilter(rule.Cond)
	catSetRec := fb.registerAndSet(catFilter.AndSet)
	catSetRec.RuleSet = append(
		catSetRec.RuleSet, &RuleFilterRec{RuleIndex: ruleIndex, RuleId: rule.RuleId})
}

func (fb *FilterBuilder) processAndOp(cond condition.Condition) CatFilter {
	if len(cond.GetOperands()) > 0 {
		operands := types.MapSlice(cond.GetOperands(), fb.computeCatFilter)
		return types.Reduce(operands[1:], fb.andFilters, operands[0])
	} else {
		return CatFilter{AndSet: types.EmptyAndOrSet}
	}
}

func (fb *FilterBuilder) processOrOp(cond condition.Condition) CatFilter {
	if len(cond.GetOperands()) > 0 {
		operands := types.MapSlice(cond.GetOperands(), fb.computeCatFilter)
		return types.Reduce(operands[1:], fb.orFilters, operands[0])
	} else {
		return CatFilter{AndSet: types.EmptyAndOrSet}
	}
}

func (fb *FilterBuilder) processNotOp(cond condition.Condition) CatFilter {
	switch cond.GetKind() {
	case condition.CategoryCondKind:
		cat := cond.(*condition.CategoryCond).Cat
		if cat < 0 {
			panic("Invalid category (< 0)")
		}
		// Substitute the negation of the category with its negative category
		negCat := fb.registerNegativeCat(cat)
		return fb.computeCatFilter(condition.NewCategoryCond(negCat))
	case condition.AndCondKind:
		return fb.computeCatFilter(
			condition.NewOrCond(
				types.MapSlice(cond.GetOperands(), func(a condition.Condition) condition.Condition {
					return condition.NewNotCond(a)
				})...))
	case condition.OrCondKind:
		return fb.computeCatFilter(
			condition.NewAndCond(
				types.MapSlice(cond.GetOperands(), func(a condition.Condition) condition.Condition {
					return condition.NewNotCond(a)
				})...))
	case condition.NotCondKind:
		return fb.computeCatFilter(cond)
	default:
		panic("Unexpected condition kind")
	}
}

func (fb *FilterBuilder) andFilters(a CatFilter, b CatFilter) CatFilter {
	andSet := a.AndSet.Union(b.AndSet)
	if andSet.Size() > 64 {
		longAndOrSet := andSet.ToSlice()
		part1AndOrSet := immutable.Of(longAndOrSet[:64]...)
		part1CatSetRec := fb.registerAndSet(types.AndOrSet(*part1AndOrSet))
		part2AndOrSet := immutable.Of(
			append(longAndOrSet[64:], *immutable.Of(-part1CatSetRec.CatFilterSet.CatSetIndex1))...)
		return CatFilter{AndSet: types.AndOrSet(*part2AndOrSet)}
	} else {
		return CatFilter{AndSet: andSet}
	}
}

func (fb *FilterBuilder) orFilters(a CatFilter, b CatFilter) CatFilter {
	aSlice := a.AndSet.ToSlice()
	bSlice := b.AndSet.ToSlice()
	if len(aSlice) == 0 {
		return CatFilter{AndSet: b.AndSet}
	} else if len(bSlice) == 0 {
		return CatFilter{AndSet: a.AndSet}
	} else if len(aSlice) == 1 && len(bSlice) == 1 {
		s := immutable.Of(*(immutable.Of(append(aSlice[0].ToSlice(), bSlice[0].ToSlice()...)...)))
		return CatFilter{AndSet: types.AndOrSet(*s)}
	} else {
		aCatSetRec := fb.registerAndSet(a.AndSet)
		bCatSetRec := fb.registerAndSet(b.AndSet)
		s := immutable.Of(*(immutable.Of(
			-aCatSetRec.CatFilterSet.CatSetIndex1,
			-bCatSetRec.CatFilterSet.CatSetIndex1)))
		return CatFilter{AndSet: types.AndOrSet(*s)}
	}
}

func NewFilterBuilder(repo *condition.RuleRepo, options *Options) *FilterBuilder {
	if options == nil {
		options = &Options{}
	}
	result := FilterBuilder{RuleRepo: repo, CatFilterSets: []*CatSetRec{},
		CatSetMap: types.NewHashMap[types.AndOrSet, *CatSetRec](),
		options:   options,
		NegCats:   make(map[types.Category]types.Category)}
	result.buildCatSetFilters(repo)
	return &result
}

func (fb *FilterBuilder) optimize() {
	if fb.options.OrOptimizationFreqThreshold > 0 {
		for {
			removeCount := fb.optimiseOrSets()
			if removeCount <= 0 {
				break
			}
			fb.metrics.OrSetsRemoved += removeCount
		}
		if fb.options.Verbose {
			fmt.Printf("optimizedOrSets removed %d\n", fb.metrics.OrSetsRemoved)
		}
	}
	if fb.options.AndOptimizationFreqThreshold > 0 {
		for {
			removeCount := fb.optimiseAndSets()
			if removeCount <= 0 {
				break
			}
			fb.metrics.AndSetsRemoved += removeCount
		}
		if fb.options.Verbose {
			fmt.Printf("optimizedAndOrSets removed %d\n", fb.metrics.AndSetsRemoved)
		}
	}
	fb.inlineOrSets()
	if fb.options.Verbose {
		fmt.Printf("OrSetsInlined %d\n", fb.metrics.OrSetsInlined)
	}
	fb.inlineAndOrSets()
	if fb.options.Verbose {
		fmt.Printf("AndOrSetsInlined %d\n", fb.metrics.AndOrSetsInlined)
	}

	for i := 0; i < 10; i++ {
		fb.gcAndOrSets()
	}
	if fb.options.Verbose {
		fmt.Printf("AndOrSetsGCed %d\n", fb.metrics.AndOrSetsGCed)
	}
}

type catPair struct {
	cat1 types.Category
	cat2 types.Category
}

type pairFreq struct {
	cp   catPair
	freq uint
}

func (fb *FilterBuilder) optimiseOrSets() uint {
	catSetFreq := fb.computeOrSetFreq()
	countOfSetsRemoved := uint(0)

	for _, csr := range fb.CatFilterSets {
		if csr != nil {
			var newAndSet []immutable.Set[types.Category]
			csr.CatFilterSet.AndSet.Each(func(s immutable.Set[types.Category]) {
				catsFreq := sortCatSetFreq(s, catSetFreq)

				newOrSet := s.ToHashSet()
				for _, pf := range catsFreq {
					if pf.freq > fb.options.OrOptimizationFreqThreshold &&
						newOrSet.Has(pf.cp.cat1) &&
						newOrSet.Has(pf.cp.cat2) {
						s := immutable.Of(*(immutable.Of(pf.cp.cat1, pf.cp.cat2)))
						newSetRec := fb.registerAndSet(types.AndOrSet(*s))
						if newSetRec.CatFilterSet.CatSetIndex1 != csr.CatFilterSet.CatSetIndex1 {
							newOrSet.Remove(pf.cp.cat1)
							newOrSet.Remove(pf.cp.cat2)
							newOrSet.Put(-newSetRec.CatFilterSet.CatSetIndex1)
							countOfSetsRemoved++
						}
						break
					}
				}

				newAndSet = append(newAndSet, *immutable.FromHashSet(&newOrSet))
			})
			fb.replaceFilterSet(csr.CatFilterSet.AndSet, types.AndOrSet(*immutable.Of(newAndSet...)))
		}
	}
	return countOfSetsRemoved
}

func (fb *FilterBuilder) computeSyntheticCatFreq() *hashmap.Map[types.Category, uint] {
	result := types.NewHashMap[types.Category, uint]()
	for _, csr := range fb.CatFilterSets {
		if csr != nil {
			csr.CatFilterSet.AndSet.Each(func(s immutable.Set[types.Category]) {
				s.Each(func(c types.Category) {
					if c < 0 {
						count, _ := result.Get(c)
						result.Put(c, count+1)
					}
				})
			})
		}
	}
	return result
}

func (fb *FilterBuilder) inlineAndOrSets() {
	catSetFreq := fb.computeSyntheticCatFreq()

	for _, csr := range fb.CatFilterSets {
		if csr != nil {
			andOrSet := csr.CatFilterSet.AndSet
			newAndOrSet := fb.inlineAndOrSet(andOrSet, catSetFreq)
			fb.replaceFilterSet(andOrSet, types.AndOrSet(*immutable.Of(newAndOrSet...)))
		}
	}
}

func (fb *FilterBuilder) gcAndOrSets() {
	catSetFreq := fb.computeSyntheticCatFreq()

	for index1, csr := range fb.CatFilterSets {
		if csr != nil && len(csr.RuleSet) == 0 {
			freq, _ := catSetFreq.Get(types.Category(-index1 - 1))
			if freq == 0 {
				fb.metrics.AndOrSetsGCed++
				fb.unregisterFilterSet(csr.CatFilterSet.AndSet)
			}
		}
	}
}

func (fb *FilterBuilder) inlineOrSets() {
	catSetFreq := fb.computeSyntheticCatFreq()

	for _, csr := range fb.CatFilterSets {
		if csr != nil {
			andOrSet := csr.CatFilterSet.AndSet
			var newS []immutable.Set[types.Category]
			andOrSet.Each(func(s immutable.Set[types.Category]) {
				newS = append(newS, *immutable.Of(fb.inlineOrSet(s, catSetFreq)...))
			})
			fb.replaceFilterSet(andOrSet, types.AndOrSet(*immutable.Of(newS...)))
		}
	}
}

func sortCatSetFreq(s immutable.Set[types.Category], catSetFreq *hashmap.Map[immutable.Set[types.Category], uint]) []pairFreq {
	cats := s.ToSlice()
	var catsFreq []pairFreq
	for i, c1 := range cats {
		for _, c2 := range cats[i+1:] {
			cp := immutable.Of(c1, c2)
			freq, _ := catSetFreq.Get(*cp)
			catsFreq = append(catsFreq, pairFreq{cp: catPair{cat1: c1, cat2: c2}, freq: freq})
		}
	}
	sort.Slice(catsFreq, func(i, j int) bool {
		return catsFreq[i].freq > catsFreq[j].freq
	})
	return catsFreq
}

type orPair struct {
	orSet1 immutable.Set[types.Category]
	orSet2 immutable.Set[types.Category]
}

type orPairFreq struct {
	cp   orPair
	freq uint
}

func (fb *FilterBuilder) optimiseAndSets() uint {
	orSetFreq := fb.computeAndSetFreq()
	countOfSetsRemoved := uint(0)

	for _, csr := range fb.CatFilterSets {
		if csr != nil {
			orFreq := sortOrSetFreq(csr, orSetFreq)

			newAndSet := csr.CatFilterSet.AndSet.ToHashSet()

			for _, pf := range orFreq {
				if pf.freq > fb.options.AndOptimizationFreqThreshold &&
					newAndSet.Has(pf.cp.orSet1) &&
					newAndSet.Has(pf.cp.orSet2) {
					s := immutable.Of(pf.cp.orSet1, pf.cp.orSet2)
					newSetRec := fb.registerAndSet(types.AndOrSet(*s))
					if newSetRec.CatFilterSet.CatSetIndex1 != csr.CatFilterSet.CatSetIndex1 {
						newAndSet.Remove(pf.cp.orSet1)
						newAndSet.Remove(pf.cp.orSet2)
						newAndSet.Put(*immutable.Of(-newSetRec.CatFilterSet.CatSetIndex1))
						countOfSetsRemoved++
					}
					break
				}
			}

			fb.replaceFilterSet(csr.CatFilterSet.AndSet, types.AndOrSet(*immutable.FromHashSet(&newAndSet)))
		}
	}
	//fmt.Printf("optimizedAndSets removed %d\n", countOfSetsRemoved)
	return countOfSetsRemoved
}

func sortOrSetFreq(csr *CatSetRec, orSetFreq *hashmap.Map[immutable.Set[immutable.Set[types.Category]], uint]) []orPairFreq {
	var orFreq []orPairFreq
	orSets := csr.CatFilterSet.AndSet.ToSlice()
	for i, orSet1 := range orSets {
		for j := i + 1; j < len(orSets); j++ {
			orSet2 := orSets[j]
			cp := immutable.Of(orSet1, orSet2)
			freq, _ := orSetFreq.Get(*cp)
			orFreq = append(orFreq, orPairFreq{cp: orPair{orSet1: orSet1, orSet2: orSet2}, freq: freq})
		}
	}

	sort.Slice(orFreq, func(i, j int) bool {
		if orFreq[i].freq == orFreq[j].freq {
			return orFreq[i].cp.orSet1.GetHash()^orFreq[i].cp.orSet2.GetHash() >
				orFreq[j].cp.orSet1.GetHash()^orFreq[j].cp.orSet2.GetHash()
		} else {
			return orFreq[i].freq > orFreq[j].freq
		}
	})
	return orFreq
}

func (fb *FilterBuilder) computeOrSetFreq() *hashmap.Map[immutable.Set[types.Category], uint] {
	result := types.NewHashMap[immutable.Set[types.Category], uint]()
	for _, csr := range fb.CatFilterSets {
		if csr != nil {
			csr.CatFilterSet.AndSet.Each(func(s immutable.Set[types.Category]) {
				cats := s.ToSlice()
				for i, c1 := range cats {
					for j := i + 1; j < len(cats); j++ {
						c2 := cats[j]
						cp := immutable.Of(c1, c2)
						count, _ := result.Get(*cp)
						result.Put(*cp, count+1)
					}
				}
			})
		}
	}
	return result
}

func (fb *FilterBuilder) computeAndSetFreq() *hashmap.Map[immutable.Set[immutable.Set[types.Category]], uint] {
	result := types.NewHashMap[immutable.Set[immutable.Set[types.Category]], uint]()
	for _, csr := range fb.CatFilterSets {
		if csr != nil {
			orSets := csr.CatFilterSet.AndSet.ToSlice()
			for i, s1 := range orSets {
				for j := i + 1; j < len(orSets); j++ {
					s2 := orSets[j]
					cp := immutable.Of(s1, s2)
					count, _ := result.Get(*cp)
					result.Put(*cp, count+1)
				}
			}
		}
	}
	return result
}

func (fb *FilterBuilder) buildFilterTables() FilterTables {
	result := FilterTables{CatToCatSetMask: fb.computeCatToRules(),
		CatSetFilters: types.MapSlice(
			fb.CatFilterSets,
			func(csr *CatSetRec) *CatSetFilter {
				if csr == nil {
					return nil
				} else {
					return &CatSetFilter{
						CatSetIndex1: csr.CatFilterSet.CatSetIndex1,
						RuleSet:      csr.RuleSet,
						CatSetMasks:  csr.CatSetMasks}
				}
			})}

	result.BuilderMetrics = fb.metrics
	result.RuleRecs = fb.RuleRecs
	result.NegCats = fb.NegCats
	result.DefaultCategories = make(map[types.Category]int)

	// Add all negated categories to DefaultCatList
	// TODO: Optimize to only include necessary negations (undefined checks, hash-optimized comparisons)
	for cat := range fb.NegCats {
		result.DefaultCategories[cat] = len(result.DefaultCategories)
		result.DefaultCatList = append(result.DefaultCatList, cat)
	}

	return result
}

func (fb *FilterBuilder) computeCatToRules() *CatSetMaskArray {
	var result CatSetMaskArray

	setCatSetMask := func(csr *CatSetRec, mask types.Mask, cat types.Category) {
		catSetMask := CatSetMask{
			Index1: csr.CatFilterSet.CatSetIndex1,
			Mask:   mask,
		}

		if cat < 0 {
			catSetIndex1 := -cat
			catFilterSet := fb.CatFilterSets[catSetIndex1-1]
			catFilterSet.CatSetMasks = append(catFilterSet.CatSetMasks, &catSetMask)
		} else {
			result.Set(cat, append(result.Get(cat), &catSetMask))
		}
	}

	for _, csr := range fb.CatFilterSets {
		if csr != nil {
			bitMask := types.Mask(1)
			andSet := csr.CatFilterSet.AndSet.ToSlice()
			if len(andSet) > 0 {
				head := andSet[0]
				for _, orSet := range andSet[1:] {
					for _, ct := range orSet.ToSlice() {
						setCatSetMask(csr, bitMask, ct)
					}
					bitMask <<= 1
				}
				for _, ct := range head.ToSlice() {
					setCatSetMask(csr, -bitMask, ct)
				}
			}
		}
	}
	return &result
}

func (fb *FilterBuilder) inlineOrSet(
	orSet immutable.Set[types.Category],
	catSetFreq *hashmap.Map[types.Category, uint]) []types.Category {
	var result []types.Category
	orSet.Each(func(cat types.Category) {
		newS := []types.Category{cat}
		if cat < 0 {
			// E.g. AND(OR(-1, 2))
			// -1 == AND(OR(1))
			// -1's freq == 1
			// Replace the AndOrSet with AND(OR(1, 2))
			catCsr := fb.CatFilterSets[-cat-1]
			andOrSet := catCsr.CatFilterSet.AndSet
			if andOrSet.Size() == 1 {
				freq, _ := catSetFreq.Get(cat)
				inlineOrSet := andOrSet.ToSlice()[0]
				//if inlineOrSet.Size() < 3 || freq == 1 {
				if freq == 1 {
					newS = fb.inlineOrSet(inlineOrSet, catSetFreq)
					fb.metrics.OrSetsInlined++
				}
			}
		}
		result = append(result, newS...)
	})
	return result
}

func (fb *FilterBuilder) inlineAndOrSet(
	andOrSet types.AndOrSet,
	catSetFreq *hashmap.Map[types.Category, uint]) []immutable.Set[types.Category] {
	var result []immutable.Set[types.Category]
	andOrSet.Each(func(s immutable.Set[types.Category]) {
		newS := []immutable.Set[types.Category]{s}
		if s.Size() == 1 {
			cat := s.ToSlice()[0]
			if cat < 0 {
				// E.g. AND(OR(-1), OR(-2))
				// -1 == AND(OR(1), OR(2))
				// -1's freq == 1
				// Replace the AndOrSet with AND(OR(1), OR(2), OR(-2))
				catCsr := fb.CatFilterSets[-cat-1]
				// Make sure that the inlined set immutable.not referenced in a rule.
				// This can happen when rule condition immutable.included as AND predicate in another rule condition.
				if len(catCsr.RuleSet) == 0 {
					freq, ok := catSetFreq.Get(cat)
					if ok && freq == 1 {
						newS = fb.inlineAndOrSet(catCsr.CatFilterSet.AndSet, catSetFreq)
						fb.unregisterFilterSet(catCsr.CatFilterSet.AndSet)
						fb.metrics.AndOrSetsInlined++
					}
				}
			}
		}
		result = append(result, newS...)
	})
	return result
}

func BuildFilterTables(repo *condition.RuleRepo, options *Options) FilterTables {
	fb := NewFilterBuilder(repo, options)
	if fb.options.OrOptimizationFreqThreshold > 0 || fb.options.AndOptimizationFreqThreshold > 0 {
		fb.optimize()
	}
	return fb.buildFilterTables()
}
