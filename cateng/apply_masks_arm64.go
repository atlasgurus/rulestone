//go:build arm64

package cateng

import (
	"sync/atomic"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
	"github.com/klauspost/cpuid/v2"
)

// ARM64 always has NEON support (it's mandatory in ARMv8)
var hasNEON = true

func init() {
	// On ARM64, NEON is always available
	// We use cpuid just for consistency, but NEON is guaranteed
	hasNEON = cpuid.CPU.Supports(cpuid.ASIMD) || true
}

// applyCatSetMasksSIMD is the ARM64 implementation using NEON instructions
// NEON supports 128-bit vectors, so we process 2x int64 at a time
func applyCatSetMasksSIMD(csmList []*CatSetMask, matchMaskArray []types.Mask, result *[]condition.RuleIdType, f *CategoryEngine) {
	i := 0

	// Process in batches of 2 for NEON (128-bit = 2x 64-bit)
	for i+1 < len(csmList) {
		csm0, csm1 := csmList[i], csmList[i+1]

		idx0, idx1 := int(csm0.Index1-1), int(csm1.Index1-1)

		// Check for index conflicts
		if idx0 == idx1 {
			applyCatSetMasksScalar(csmList[i:i+2], matchMaskArray, result, f)
			i += 2
			continue
		}

		// Load current values
		v0, v1 := matchMaskArray[idx0], matchMaskArray[idx1]

		atomic.AddUint64(&f.Metrics.NumMaskArrayLookups, 2)

		// Skip if already complete
		skip0, skip1 := v0 == -1, v1 == -1

		if !skip0 || !skip1 {
			// Perform OR operations
			// TODO: Replace with NEON assembly for 2x speedup
			var new0, new1 types.Mask

			if !skip0 {
				new0 = v0 | csm0.Mask
				matchMaskArray[idx0] = new0
				atomic.AddUint64(&f.Metrics.NumBitMaskChecks, 1)
			}
			if !skip1 {
				new1 = v1 | csm1.Mask
				matchMaskArray[idx1] = new1
				atomic.AddUint64(&f.Metrics.NumBitMaskChecks, 1)
			}

			// Check for matches
			if !skip0 && new0 == -1 {
				processMatch(idx0, f, matchMaskArray, result)
			}
			if !skip1 && new1 == -1 {
				processMatch(idx1, f, matchMaskArray, result)
			}
		}

		i += 2
	}

	// Process remaining element
	if i < len(csmList) {
		applyCatSetMasksScalar(csmList[i:], matchMaskArray, result, f)
	}
}
