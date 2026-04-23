package config

import "stalart-wrapper/internal/sysinfo"

// Generate produces a ZGC-optimised Config for the given hardware.
//
// Heap size, GC thread counts, and JIT inlining limits are derived from
// detected RAM and CPU topology. ZGC allocation spike tolerance is scaled
// with available cache bandwidth: X3D-class parts can absorb larger spikes
// without GC stalls because their hot working set fits in L3.
func Generate(sys sysinfo.Info) Config {
	heap := sizeHeap(sys.TotalGB())
	parallel, concurrent := gcThreads(sys.CPUThreads)
	jit := jitProfile(sys)

	spikeTolerance := 5.0
	fragLimit := 15
	if sys.HasBigCache() {
		// X3D-class parts handle higher allocation spikes well: large L3
		// absorbs the burst before ZGC's concurrent phase catches up.
		spikeTolerance = 10.0
		fragLimit = 10
		if sys.CPUThreads >= 16 {
			concurrent++
		}
	}

	return Config{
		HeapSizeGB:  int(heap),
		PreTouch:    sys.TotalGB() >= 16,
		MetaspaceMB: 512,

		ZAllocationSpikeTolerance: spikeTolerance,
		ZCollectionIntervalSec:    0,
		ZFragmentationLimit:       fragLimit,

		ParallelGCThreads: parallel,
		ConcGCThreads:     concurrent,

		ReservedCodeCacheSizeMB: 512,
		MaxInlineLevel:          jit.maxInlineLevel,
		FreqInlineSize:          jit.freqInlineSize,
		InlineSmallCode:         jit.inlineSmallCode,
		MaxNodeLimit:            jit.maxNodeLimit,
		NodeLimitFudgeFactor:    8000,
		CompileThresholdScaling: 0.65,

		UseLargePages:        sys.LargePages,
		UseThreadPriorities:  true,
		ThreadPriorityPolicy: 1,
		AutoBoxCacheMax:      8192,
	}
}

// Presets returns named tuning profiles derived from current hardware.
func Presets(sys sysinfo.Info) map[string]Config {
	balanced := Generate(sys)

	compat := balanced
	compat.PreTouch = false
	compat.ZAllocationSpikeTolerance = 2.0
	compat.ZFragmentationLimit = 20
	compat.ParallelGCThreads, compat.ConcGCThreads = 4, 2
	compat.MetaspaceMB = 384
	compat.ReservedCodeCacheSizeMB = 256
	if compat.HeapSizeGB > 6 {
		compat.HeapSizeGB = 6
	}
	compat.CompileThresholdScaling = 0.6

	performance := balanced
	performance.PreTouch = sys.TotalGB() >= 12
	performance.ZAllocationSpikeTolerance = 7.0
	performance.ZFragmentationLimit = 10
	performance.ParallelGCThreads = clamp(balanced.ParallelGCThreads+1, 4, 10)
	performance.ConcGCThreads = clamp(balanced.ConcGCThreads+1, 2, 5)
	performance.CompileThresholdScaling = 0.7

	ultra := performance
	ultra.PreTouch = sys.TotalGB() >= 16
	ultra.ZAllocationSpikeTolerance = 10.0
	ultra.ZFragmentationLimit = 10
	ultra.MetaspaceMB = 640
	ultra.ParallelGCThreads = clamp(performance.ParallelGCThreads+1, 4, 10)
	ultra.ConcGCThreads = clamp(performance.ConcGCThreads+1, 2, 5)
	ultra.MaxInlineLevel = clamp(ultra.MaxInlineLevel+2, 15, 24)
	ultra.FreqInlineSize = ultra.FreqInlineSize + 100
	ultra.InlineSmallCode = ultra.InlineSmallCode + 500
	ultra.MaxNodeLimit = ultra.MaxNodeLimit + 30000
	ultra.CompileThresholdScaling = 0.75

	return map[string]Config{
		"balanced":    balanced,
		"compat":      compat,
		"performance": performance,
		"ultra":       ultra,
	}
}

type jitLimits struct {
	maxInlineLevel  int
	freqInlineSize  int
	inlineSmallCode int
	maxNodeLimit    int
}

func jitProfile(sys sysinfo.Info) jitLimits {
	if sys.HasBigCache() {
		return jitLimits{
			maxInlineLevel:  22,
			freqInlineSize:  800,
			inlineSmallCode: 6500,
			maxNodeLimit:    360000,
		}
	}
	return jitLimits{
		maxInlineLevel:  18,
		freqInlineSize:  600,
		inlineSmallCode: 4500,
		maxNodeLimit:    280000,
	}
}

// sizeHeap picks a heap between 2 and 8 GB based on total RAM.
// ZGC scales well with larger heaps, but the game's live set stays
// ~2–3 GB regardless — capping at 8 GB avoids wasted reservation.
func sizeHeap(totalGB uint64) uint64 {
	switch {
	case totalGB >= 24:
		return 8
	case totalGB >= 16:
		return 6
	case totalGB >= 12:
		return 5
	case totalGB >= 8:
		return 4
	case totalGB >= 6:
		return 3
	default:
		return 2
	}
}

// gcThreads derives STW and concurrent worker counts from logical thread count.
// Parallel workers run during STW (game paused, siblings free for GC work).
// Concurrent workers share CPU with the game — kept at ~half of parallel.
func gcThreads(threads int) (parallel, concurrent int) {
	parallel = clamp(threads-2, 2, 10)
	concurrent = clamp(parallel/2, 1, 5)
	return
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
