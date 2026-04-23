// Package jvm turns config.Config into JVM flags and merges them with
// the launcher argv, stripping conflicting -X/-XX entries first.
//
// Emitted options target HotSpot in JDK 25 with ZGC (Generational ZGC
// is the default since JDK 21). Conservative set: heap, GC, metaspace,
// code cache, NIO, and C2 JIT only.
package jvm

import (
	"fmt"

	"stalart-wrapper/internal/config"
)

// ClientCompatProps returns -D system properties for legacy Forge/FML
// (Netty, LaunchWrapper) on JDK 21+. Applied whenever the bundled
// javaw is matched, even when heap/GC tuning is skipped.
func ClientCompatProps() []string {
	return []string{
		// Netty 4.2 on JDK 25+: Forge/LaunchWrapper class loaders cannot
		// resolve jdk.jfr.FlightRecorder — disable JFR hooks before Netty clinit.
		"-Dio.netty.jfr.enabled=false",
		// Netty defaults io.netty.noUnsafe=true on Java 25+; legacy Forge/FML
		// still expects the Unsafe-based fast path for direct buffer lifecycle.
		"-Dio.netty.noUnsafe=false",
		"-Dio.netty.tryReflectionSetAccessible=true",
		"-Djdk.attach.allowAttachSelf=true",
		// JDK 24+ (JEP 498): legacy LaunchWrapper / natives expect misc.Unsafe
		// without hard failure during early class init.
		"--sun-misc-unsafe-memory-access=allow",
	}
}

// Flags renders the tuning profile as a list of -X / -XX flags for ZGC.
func Flags(cfg config.Config) []string {
	cc := cfg.ReservedCodeCacheSizeMB
	if cc == 0 {
		cc = 512
	}

	// SoftMaxHeapSize guides ZGC's proactive collection cadence: it tries to
	// keep live data under this threshold, leaving 1 GB headroom for allocation
	// spikes within the committed heap before triggering a full cycle.
	// Below 3 GB there is no room to subtract, so soft = hard limit.
	softMax := cfg.HeapSizeGB - 1
	if softMax < 2 {
		softMax = cfg.HeapSizeGB
	}

	spikeTolerance := cfg.ZAllocationSpikeTolerance
	if spikeTolerance == 0 {
		spikeTolerance = 5.0
	}
	fragLimit := cfg.ZFragmentationLimit
	if fragLimit == 0 {
		fragLimit = 15
	}

	flags := []string{
		fmt.Sprintf("-Xmx%dg", cfg.HeapSizeGB),
		fmt.Sprintf("-Xms%dg", cfg.HeapSizeGB),
		fmt.Sprintf("-XX:SoftMaxHeapSize=%dg", softMax),

		fmt.Sprintf("-XX:MetaspaceSize=%dm", cfg.MetaspaceMB),
		fmt.Sprintf("-XX:MaxMetaspaceSize=%dm", cfg.MetaspaceMB),

		"-XX:+UseZGC",
		fmt.Sprintf("-XX:ZFragmentationLimit=%d", fragLimit),
		fmt.Sprintf("-XX:ZAllocationSpikeTolerance=%.1f", spikeTolerance),

		"-XX:+ZProactive",
		"-XX:+DisableExplicitGC",
		"-XX:+PerfDisableSharedMem",

		fmt.Sprintf("-XX:ReservedCodeCacheSize=%dm", cc),
		"-Djdk.nio.maxCachedBufferSize=262144",
	}

	if cfg.ConcGCThreads > 0 {
		flags = append(flags, fmt.Sprintf("-XX:ConcGCThreads=%d", cfg.ConcGCThreads))
	}
	if cfg.ParallelGCThreads > 0 {
		flags = append(flags, fmt.Sprintf("-XX:ParallelGCThreads=%d", cfg.ParallelGCThreads))
	}
	if cfg.ZCollectionIntervalSec > 0 {
		flags = append(flags, fmt.Sprintf("-XX:ZCollectionInterval=%d", cfg.ZCollectionIntervalSec))
	}
	if cfg.PreTouch {
		flags = append(flags, "-XX:+AlwaysPreTouch")
	}
	if cfg.UseLargePages {
		flags = append(flags, "-XX:+UseLargePages")
	}
	if cfg.UseThreadPriorities {
		flags = append(flags,
			"-XX:+UseThreadPriorities",
			fmt.Sprintf("-XX:ThreadPriorityPolicy=%d", cfg.ThreadPriorityPolicy),
		)
	}
	if cfg.AutoBoxCacheMax > 0 {
		flags = append(flags, fmt.Sprintf("-XX:AutoBoxCacheMax=%d", cfg.AutoBoxCacheMax))
	}
	if cfg.MaxInlineLevel > 0 {
		flags = append(flags, fmt.Sprintf("-XX:MaxInlineLevel=%d", cfg.MaxInlineLevel))
	}
	if cfg.FreqInlineSize > 0 {
		flags = append(flags, fmt.Sprintf("-XX:FreqInlineSize=%d", cfg.FreqInlineSize))
	}
	if cfg.InlineSmallCode > 0 {
		flags = append(flags, fmt.Sprintf("-XX:InlineSmallCode=%d", cfg.InlineSmallCode))
	}
	if cfg.MaxNodeLimit > 0 {
		flags = append(flags, fmt.Sprintf("-XX:MaxNodeLimit=%d", cfg.MaxNodeLimit))
	}
	if cfg.NodeLimitFudgeFactor > 0 {
		flags = append(flags, fmt.Sprintf("-XX:NodeLimitFudgeFactor=%d", cfg.NodeLimitFudgeFactor))
	}
	if cfg.CompileThresholdScaling > 0 {
		flags = append(flags, fmt.Sprintf("-XX:CompileThresholdScaling=%.2f", cfg.CompileThresholdScaling))
	}

	return flags
}
