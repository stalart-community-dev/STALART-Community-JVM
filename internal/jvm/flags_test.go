package jvm

import (
	"strings"
	"testing"

	"stalart-wrapper/internal/config"
)

func TestFlagsZGC(t *testing.T) {
	cfg := config.Config{
		HeapSizeGB:                8,
		MetaspaceMB:               512,
		ZAllocationSpikeTolerance: 5.0,
		ZFragmentationLimit:       15,
		ParallelGCThreads:         8,
		ConcGCThreads:             4,
		ReservedCodeCacheSizeMB:   512,
	}
	flags := Flags(cfg)
	joined := " " + strings.Join(flags, " ") + " "

	mustContain := []string{
		"-Xmx8g",
		"-Xms8g",
		"-XX:SoftMaxHeapSize=7g",
		"-XX:+UseZGC",
		"-XX:ConcGCThreads=4",
		"-XX:ParallelGCThreads=8",
		"-XX:ZFragmentationLimit=15",
		"-XX:ZAllocationSpikeTolerance=5.0",
		"-XX:ReservedCodeCacheSize=512m",
		"-XX:+ZProactive",
		"-XX:+DisableExplicitGC",
	}
	for _, f := range mustContain {
		if !strings.Contains(joined, " "+f+" ") {
			t.Fatalf("expected %q in flags: %v", f, flags)
		}
	}

	mustNotContain := []string{
		"-XX:+UseG1GC",
		"-XX:MaxGCPauseMillis=",
		"-XX:G1HeapRegionSize=",
		"-XX:+ParallelRefProcEnabled",
	}
	for _, f := range mustNotContain {
		if strings.Contains(joined, f) {
			t.Fatalf("did not expect %q in ZGC flags: %v", f, flags)
		}
	}
}

func TestFlagsSoftMaxHeapFloor(t *testing.T) {
	cfg := config.Config{HeapSizeGB: 2, MetaspaceMB: 256}
	flags := Flags(cfg)
	joined := " " + strings.Join(flags, " ") + " "
	// SoftMaxHeapSize must not go below HeapSizeGB when heap is already small
	if !strings.Contains(joined, " -XX:SoftMaxHeapSize=2g ") {
		t.Fatalf("expected SoftMaxHeapSize=2g for 2GB heap: %v", flags)
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
