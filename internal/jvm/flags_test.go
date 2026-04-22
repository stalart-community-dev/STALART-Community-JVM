package jvm

import (
	"strings"
	"testing"

	"stalart-wrapper/internal/config"
)

func TestFlagsConservativeJDK2501Profile(t *testing.T) {
	cfg := config.Config{
		HeapSizeGB:              8,
		MetaspaceMB:             512,
		MaxGCPauseMillis:        35,
		ParallelGCThreads:       8,
		ConcGCThreads:           2,
		SoftRefLRUPolicyMSPerMB: 50,
		ReservedCodeCacheSizeMB: 512,
	}
	flags := Flags(cfg)
	joined := " " + strings.Join(flags, " ") + " "

	mustContain := []string{
		"-Xmx8g",
		"-Xms2g",
		"-XX:+UseG1GC",
		"-XX:MaxGCPauseMillis=35",
		"-XX:ParallelGCThreads=8",
		"-XX:ConcGCThreads=2",
		"-XX:ReservedCodeCacheSize=512m",
	}
	for _, f := range mustContain {
		if !strings.Contains(joined, " "+f+" ") {
			t.Fatalf("expected %q in flags: %v", f, flags)
		}
	}

	mustNotContain := []string{
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:G1HeapRegionSize=",
		"-XX:+G1UseAdaptiveIHOP",
	}
	for _, f := range mustNotContain {
		if strings.Contains(joined, f) {
			t.Fatalf("did not expect %q in conservative flags: %v", f, flags)
		}
	}
}

func TestClientCompatPropsJDK25(t *testing.T) {
	props := ClientCompatProps()
	joined := " " + strings.Join(props, " ") + " "
	mustContain := []string{
		"-Dio.netty.jfr.enabled=false",
		"-Dio.netty.noUnsafe=false",
		"-Dio.netty.tryReflectionSetAccessible=true",
		"-Djdk.attach.allowAttachSelf=true",
		"--sun-misc-unsafe-memory-access=allow",
	}
	for _, p := range mustContain {
		if !strings.Contains(joined, " "+p+" ") {
			t.Fatalf("expected %q in props: %v", p, props)
		}
	}
}

func TestJava25SafeFlags(t *testing.T) {
	cfg := config.Config{
		MaxGCPauseMillis:               50,
		ParallelGCThreads:              10,
		ConcGCThreads:                  5,
		InitiatingHeapOccupancyPercent: 24,
		G1NewSizePercent:               16,
		G1MaxNewSizePercent:            65,
		G1ReservePercent:               15,
		G1HeapWastePercent:             5,
		G1MixedGCCountTarget:           4,
		G1MixedGCLiveThresholdPercent:  90,
		SurvivorRatio:                  32,
		MaxTenuringThreshold:           1,
		UseStringDeduplication:         true,
	}
	flags := Java25SafeFlags(cfg)
	joined := " " + strings.Join(flags, " ") + " "
	mustContain := []string{
		"-XX:+UseG1GC",
		"-XX:MaxGCPauseMillis=50",
		"-XX:ParallelGCThreads=10",
		"-XX:ConcGCThreads=5",
		"-XX:InitiatingHeapOccupancyPercent=24",
		"-XX:G1NewSizePercent=16",
		"-XX:G1MaxNewSizePercent=65",
		"-XX:G1ReservePercent=15",
		"-XX:G1HeapWastePercent=5",
		"-XX:G1MixedGCCountTarget=4",
		"-XX:G1MixedGCLiveThresholdPercent=90",
		"-XX:SurvivorRatio=32",
		"-XX:MaxTenuringThreshold=1",
		"-XX:+ParallelRefProcEnabled",
		"-XX:+DisableExplicitGC",
		"-XX:+PerfDisableSharedMem",
		"-Djdk.nio.maxCachedBufferSize=262144",
		"-XX:+UseStringDeduplication",
	}
	for _, f := range mustContain {
		if !strings.Contains(joined, " "+f+" ") {
			t.Fatalf("expected %q in java25 safe flags: %v", f, flags)
		}
	}
}
