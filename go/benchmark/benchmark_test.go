package benchmark

import (
	"fmt"
	"github.com/rulestone/cateng"
	"github.com/rulestone/condition"
	is "github.com/rulestone/immutable"
	"github.com/rulestone/types"
	"github.com/zyedidia/generic/hashmap"
	"github.com/zyedidia/generic/hashset"
	"math/rand"
	"runtime"
	"sort"
	"testing"
	"time"
)

const verbose = false

type expressionInference struct {
	haveSeenThisFS *hashset.Set[types.AndOrSet]
	filters        *hashset.Set[types.AndOrSet]
	complementFS   *hashmap.Map[types.AndOrSet, is.Set[types.Category]]
}

func (ei expressionInference) processFeatureSet(fs types.AndOrSet) {
	if !ei.haveSeenThisFS.Has(fs) {
		ei.haveSeenThisFS.Put(fs)
		ei.filters.Put(fs)

		for _, f := range fs.ToSlice() {
			cfs := fs.Difference(types.AndOrSet(*is.Of(f)))
			orSet, ok := ei.complementFS.Get(cfs)
			if ok {
				// Combine this set with the one that has the same complement
				newOrSet := orSet.Union(f)
				if !newOrSet.Equals(orSet) && !newOrSet.Equals(f) {
					// The combined set will have at least one more element than the original
					newAndOrSet := cfs.Union(types.AndOrSet(*is.Of(newOrSet)))
					oldAndOrSet := cfs.Union(types.AndOrSet(*is.Of(orSet)))
					ei.removeFilter(oldAndOrSet)
					ei.removeFilter(fs)
					ei.processFeatureSet(newAndOrSet)
				}
			} else {
				ei.complementFS.Put(cfs, f)
			}
		}
	}
}

func (ei expressionInference) removeFilter(fs types.AndOrSet) {
	ei.filters.Remove(fs)
	for _, f := range fs.ToSlice() {
		cfs := fs.Difference(types.AndOrSet(*is.Of(f)))
		_, ok := ei.complementFS.Get(cfs)
		if ok {
			ei.complementFS.Remove(cfs)
		}
	}
}

func printAndOrSet(set types.AndOrSet) {
	fmt.Printf("[")

	setSlice := set.ToSlice()
	sort.Slice(setSlice, func(i, j int) bool {
		aSlice := setSlice[i].ToSlice()
		bSlice := setSlice[j].ToSlice()
		return aSlice[0] < bSlice[0]
	})
	for _, v := range setSlice {
		fmt.Printf("[")
		for _, y := range v.ToSlice() {
			fmt.Printf("%v,", y)
		}
		fmt.Printf("]")
	}
	fmt.Printf("]\n")
}

func (ei expressionInference) printFilters(verbose bool) {
	fmt.Printf("num_filters = %v\n", ei.filters.Size())
	fmt.Printf("num_sets = %v\n", ei.haveSeenThisFS.Size())
	fmt.Printf("num_comp_feature_sets = %v\n", ei.complementFS.Size())
	if verbose {
		fmt.Printf("Rules: \n")
		ei.filters.Each(func(set types.AndOrSet) {
			fmt.Printf("%#v\n", set.ToSlices())
			//printAndOrSet(set)
		})
	}
}

type testExpressionInference struct {
	ei        expressionInference
	fSets     [][]types.Category
	ruleRepo  *condition.RuleRepo
	catFilter *cateng.CategoryEngine
	testNum   int
}

const MAX_SETS = 100 // 100
const MAX_CATS = 10  // 10

func (tei *testExpressionInference) generateTestCase() {
	tei.ei = expressionInference{haveSeenThisFS: types.NewHashSet[types.AndOrSet](),
		filters:      types.NewHashSet[types.AndOrSet](),
		complementFS: types.NewHashMap[types.AndOrSet, is.Set[types.Category]]()}
	tei.fSets = nil
	for i := 0; i < MAX_SETS; i++ {
		var newFs []types.Category
		for j := 0; j < MAX_CATS; j++ {
			newFs = append(newFs, types.Category(rand.Intn(MAX_CATS)+2))
		}
		tei.getAndProcessVariations(newFs)
	}
}

func (tei *testExpressionInference) getAndProcessVariations(fs []types.Category) {
	tei.ei.processFeatureSet(types.SliceToAndOrSet(fs))

	for i, v := range fs {
		if v%5 == 0 {
			for j := 0; j < MAX_SETS; j++ {
				fs[i] = types.Category(rand.Intn(200) + 1)
				andOrSet := types.SliceToAndOrSet(fs)
				tei.ei.processFeatureSet(andOrSet)
				tei.fSets = append(tei.fSets, types.SliceToSet(fs).ToSlice())
			}
		}
	}
}

func (tei *testExpressionInference) testFilters() {
	tei.testNum++

	fmt.Printf("\n\n========================================================\n")
	fmt.Printf("New test case %d\n", tei.testNum)
	fmt.Printf("========================================================\n")
	tei.generateTestCase()
	//tei.compileAndRun("non optimized", nil)
	for ot := uint(0); ot < 50; ot += 5 {
		for at := uint(1); at < 2; at += 5 {
			fmt.Printf("time with ot = %d; at = %d\n", ot, at)
			tei.compileAndRun("optimized", &cateng.Options{ot, at, true})
		}
	}
}

func (tei *testExpressionInference) compileAndRun(comment string, options *cateng.Options) {
	tei.compileTestCase(tei.ei.filters, options)
	runtime.GC()
	start := time.Now()
	for i := 0; i < 100; i++ {
		tei.runTestCase(tei.fSets)
	}
	duration := time.Since(start)
	fmt.Printf("Took(%v) %v\n", comment, duration)
	tei.catFilter.PrintMetrics()
}

func exactMatchUdf(ruleIndex condition.RuleIndexType, bits uint64) bool {
	panic("not implemented")
}

func (tei *testExpressionInference) compileTestCase(filters *hashset.Set[types.AndOrSet], options *cateng.Options) {
	var conditions []condition.Condition
	filters.Each(func(set types.AndOrSet) {
		conditions = append(conditions, condition.AndOrSetToCondition(set))
	})
	ruleNum := 0
	tei.ruleRepo = condition.NewRuleRepo(types.MapSlice(conditions, func(cond condition.Condition) *condition.Rule {
		// Python example has one rule with all conditions OR-ed together.
		ruleNum++
		//fmt.Printf("rule_%v => %#v\n", ruleNum, *cond)
		return &condition.Rule{RuleId: condition.RuleIdType(ruleNum), Cond: cond}
	}))
	tei.catFilter = cateng.NewCategoryEngine(tei.ruleRepo, options)
}

func (tei *testExpressionInference) runTestCase(fsSets [][]types.Category) {
	numMismatches := 0
	for _, v := range fsSets {
		matches := tei.catFilter.MatchEvent(v)
		//fmt.Printf("Try to match: %v => %v\n", v, matches)
		if len(matches) < 1 {
			fmt.Printf("Try to match: %v => %v\n", v, matches)
			fmt.Printf("mismatch!!!\n")
			numMismatches++
		}
	}
	if verbose {
		fmt.Printf("mismatched: %d out of %d\n", numMismatches, len(fsSets))
		tei.ei.printFilters(false)
	}
	if numMismatches > 0 {
		tei.ei.printFilters(true)
		panic("mismatch detected")
	}
}

func TestBenchmark(t *testing.T) {
	fei := testExpressionInference{}
	//for i := 0; i < 10; i++ {
	fei.testFilters()
	//}
}
