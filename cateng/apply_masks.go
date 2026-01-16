package cateng

import (
	"sync/atomic"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
	"github.com/klauspost/cpuid/v2"
)

// CPU feature detection
var (
	hasAVX2 = cpuid.CPU.Supports(cpuid.AVX2)
	hasAVX  = cpuid.CPU.Supports(cpuid.AVX)
)

// applyCatSetMasksOptimized is the entry point that dispatches to the best implementation
func applyCatSetMasksOptimized(csmList []*CatSetMask, matchMaskArray []types.Mask, result *[]condition.RuleIdType, f *CategoryEngine) {
	// For small lists, scalar is faster due to setup overhead
	if len(csmList) < 8 {
		applyCatSetMasksScalar(csmList, matchMaskArray, result, f)
		return
	}

	// Try SIMD implementations based on CPU capabilities
	if hasAVX2 || hasAVX {
		// Architecture-specific SIMD implementations in:
		// - apply_masks_amd64.go (AVX2 for x86_64)
		// - apply_masks_arm64.go (NEON for ARM64)
		applyCatSetMasksSIMD(csmList, matchMaskArray, result, f)
		return
	}

	// Fallback to scalar implementation
	applyCatSetMasksScalar(csmList, matchMaskArray, result, f)
}

// applyCatSetMasksSIMD is implemented in architecture-specific files:
// - apply_masks_amd64.go (with AVX2 support)
// - apply_masks_arm64.go (with NEON support)

// applyCatSetMasksScalar is the original scalar implementation (fallback)
func applyCatSetMasksScalar(csmList []*CatSetMask, matchMaskArray []types.Mask, result *[]condition.RuleIdType, f *CategoryEngine) {
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
					applyCatSetMasksOptimized(catSetFilter.CatSetMasks, matchMaskArray, result, f)
				}
				for _, cfr := range catSetFilter.RuleSet {
					*result = append(*result, cfr.RuleId)
				}
			}
		}
	}
}

// processMatch handles the match logic extracted for reuse
func processMatch(idx int, f *CategoryEngine, matchMaskArray []types.Mask, result *[]condition.RuleIdType) {
	catSetFilter := f.FilterTables.CatSetFilters[idx]

	atomic.AddUint64(&f.Metrics.NumBitMaskMatches, 1)

	// Process synthetic categories
	if len(catSetFilter.CatSetMasks) > 0 {
		applyCatSetMasksOptimized(catSetFilter.CatSetMasks, matchMaskArray, result, f)
	}

	// Collect matching rules
	for _, cfr := range catSetFilter.RuleSet {
		*result = append(*result, cfr.RuleId)
	}
}
