//go:build amd64

package cateng

import (
	"sync/atomic"

	"github.com/atlasgurus/rulestone/condition"
	"github.com/atlasgurus/rulestone/types"
)

// applyCatSetMasksSIMD is the AMD64 implementation using AVX2 instructions
// AVX2 supports 256-bit vectors, so we process 4x int64 at a time
func applyCatSetMasksSIMD(csmList []*CatSetMask, matchMaskArray []types.Mask, result *[]condition.RuleIdType, f *CategoryEngine) {
	i := 0

	// Process in batches of 4 for AVX2 (256-bit = 4x 64-bit)
	for i+3 < len(csmList) {
		csm0, csm1, csm2, csm3 := csmList[i], csmList[i+1], csmList[i+2], csmList[i+3]

		idx0, idx1, idx2, idx3 := int(csm0.Index1-1), int(csm1.Index1-1), int(csm2.Index1-1), int(csm3.Index1-1)

		// Check for index conflicts
		if idx0 == idx1 || idx0 == idx2 || idx0 == idx3 ||
			idx1 == idx2 || idx1 == idx3 || idx2 == idx3 {
			applyCatSetMasksScalar(csmList[i:i+4], matchMaskArray, result, f)
			i += 4
			continue
		}

		// Load current values
		v0, v1, v2, v3 := matchMaskArray[idx0], matchMaskArray[idx1], matchMaskArray[idx2], matchMaskArray[idx3]

		atomic.AddUint64(&f.Metrics.NumMaskArrayLookups, 4)

		// Skip if already complete
		skip0, skip1, skip2, skip3 := v0 == -1, v1 == -1, v2 == -1, v3 == -1

		if !skip0 || !skip1 || !skip2 || !skip3 {
			// Perform OR operations
			// TODO: Replace with AVX2 assembly for 4x speedup
			var new0, new1, new2, new3 types.Mask

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
			if !skip2 {
				new2 = v2 | csm2.Mask
				matchMaskArray[idx2] = new2
				atomic.AddUint64(&f.Metrics.NumBitMaskChecks, 1)
			}
			if !skip3 {
				new3 = v3 | csm3.Mask
				matchMaskArray[idx3] = new3
				atomic.AddUint64(&f.Metrics.NumBitMaskChecks, 1)
			}

			// Check for matches
			if !skip0 && new0 == -1 {
				processMatch(idx0, f, matchMaskArray, result)
			}
			if !skip1 && new1 == -1 {
				processMatch(idx1, f, matchMaskArray, result)
			}
			if !skip2 && new2 == -1 {
				processMatch(idx2, f, matchMaskArray, result)
			}
			if !skip3 && new3 == -1 {
				processMatch(idx3, f, matchMaskArray, result)
			}
		}

		i += 4
	}

	// Process remaining elements
	if i < len(csmList) {
		applyCatSetMasksScalar(csmList[i:], matchMaskArray, result, f)
	}
}
