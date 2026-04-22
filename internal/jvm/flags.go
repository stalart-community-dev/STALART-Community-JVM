// Package jvm turns config.Config into JVM flags and merges them with
// the launcher argv, stripping conflicting -X/-XX entries first.
//
// Emitted options target HotSpot in JDK 25.0.1 with a conservative
// performance profile: heap/GC/metaspace/code cache/NIO only.
// Risky experimental or deep-tuning switches are intentionally omitted.
package jvm

import (
	"fmt"

	"stalart-wrapper/internal/config"
)

// ClientCompatProps returns -D system properties for legacy Forge/FML
// (Netty, LaunchWrapper) on JDK 21+. They are applied whenever the
// bundled javaw is matched, even when heap/GC tuning is skipped.
func ClientCompatProps() []string {
	return []string{
		// Netty 4.2 on JDK 25+: Forge/LaunchWrapper (Gravit) class loaders cannot
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

// Flags renders the tuning profile as a list of -X / -XX: flags.
func Flags(cfg config.Config) []string {
	cc := cfg.ReservedCodeCacheSizeMB
	if cc == 0 {
		cc = 512
	}

	// Keep initial heap modest: large Xms + AlwaysPreTouch commits physical
	// pages immediately and often causes "Could not create the JVM" on
	// 8–12 GB machines. HotSpot grows the heap toward -Xmx as needed.
	xms := cfg.HeapSizeGB
	if xms > 2 {
		xms = 2
	}
	flags := []string{
		fmt.Sprintf("-Xmx%dg", cfg.HeapSizeGB),
		fmt.Sprintf("-Xms%dg", xms),

		fmt.Sprintf("-XX:MetaspaceSize=%dm", cfg.MetaspaceMB),
		fmt.Sprintf("-XX:MaxMetaspaceSize=%dm", cfg.MetaspaceMB),

		"-XX:+UseG1GC",
		fmt.Sprintf("-XX:MaxGCPauseMillis=%d", cfg.MaxGCPauseMillis),

		fmt.Sprintf("-XX:ParallelGCThreads=%d", cfg.ParallelGCThreads),
		fmt.Sprintf("-XX:ConcGCThreads=%d", cfg.ConcGCThreads),

		"-XX:+ParallelRefProcEnabled",
		"-XX:+DisableExplicitGC",
		fmt.Sprintf("-XX:SoftRefLRUPolicyMSPerMB=%d", cfg.SoftRefLRUPolicyMSPerMB),

		// Do not force -XX:+DisableAttachMechanism: Gravit / agents may need
		// the JVM attach API (see jdk.attach.allowAttachSelf in launcher recipes).
		"-XX:+PerfDisableSharedMem",

		// JDK 21+: explicit NonNMethod/Profiled/NonProfiled splits can fall
		// below VM minimums after alignment → init failure. Only set total.
		fmt.Sprintf("-XX:ReservedCodeCacheSize=%dm", cc),

		"-Djdk.nio.maxCachedBufferSize=262144",
	}

	if cfg.PreTouch {
		flags = append(flags, "-XX:+AlwaysPreTouch")
	}
	if cfg.UseLargePages {
		flags = append(flags, "-XX:+UseLargePages")
	}
	if cfg.UseStringDeduplication {
		flags = append(flags, "-XX:+UseStringDeduplication")
	}

	return flags
}

// LightFlags returns a compatibility-first optimization profile that keeps
// launcher's own heap/module arguments intact. This is used as a second-stage
// fallback when full replacement flags fail early on Java 25.
func LightFlags(cfg config.Config) []string {
	flags := []string{
		"-XX:+UseG1GC",
		fmt.Sprintf("-XX:MaxGCPauseMillis=%d", cfg.MaxGCPauseMillis),
		fmt.Sprintf("-XX:ParallelGCThreads=%d", cfg.ParallelGCThreads),
		fmt.Sprintf("-XX:ConcGCThreads=%d", cfg.ConcGCThreads),
		"-XX:+ParallelRefProcEnabled",
		"-XX:+DisableExplicitGC",
		"-XX:+PerfDisableSharedMem",
		"-Djdk.nio.maxCachedBufferSize=262144",
	}
	if cfg.UseStringDeduplication {
		flags = append(flags, "-XX:+UseStringDeduplication")
	}
	return flags
}

// Java25SafeFlags returns a minimal optimization set for Java 25 that
// provides practical GC tuning while avoiding fragile deep-tuning options.
func Java25SafeFlags(cfg config.Config) []string {
	flags := []string{
		"-XX:+UseG1GC",
		fmt.Sprintf("-XX:MaxGCPauseMillis=%d", cfg.MaxGCPauseMillis),
		fmt.Sprintf("-XX:ParallelGCThreads=%d", cfg.ParallelGCThreads),
		fmt.Sprintf("-XX:ConcGCThreads=%d", cfg.ConcGCThreads),
		fmt.Sprintf("-XX:InitiatingHeapOccupancyPercent=%d", cfg.InitiatingHeapOccupancyPercent),
		fmt.Sprintf("-XX:G1NewSizePercent=%d", cfg.G1NewSizePercent),
		fmt.Sprintf("-XX:G1MaxNewSizePercent=%d", cfg.G1MaxNewSizePercent),
		fmt.Sprintf("-XX:G1ReservePercent=%d", cfg.G1ReservePercent),
		fmt.Sprintf("-XX:G1HeapWastePercent=%d", cfg.G1HeapWastePercent),
		fmt.Sprintf("-XX:G1MixedGCCountTarget=%d", cfg.G1MixedGCCountTarget),
		fmt.Sprintf("-XX:G1MixedGCLiveThresholdPercent=%d", cfg.G1MixedGCLiveThresholdPercent),
		fmt.Sprintf("-XX:SurvivorRatio=%d", cfg.SurvivorRatio),
		fmt.Sprintf("-XX:MaxTenuringThreshold=%d", cfg.MaxTenuringThreshold),
		"-XX:+ParallelRefProcEnabled",
		"-XX:+DisableExplicitGC",
		"-XX:+PerfDisableSharedMem",
		"-Djdk.nio.maxCachedBufferSize=262144",
	}
	if cfg.UseStringDeduplication {
		flags = append(flags, "-XX:+UseStringDeduplication")
	}
	return flags
}
