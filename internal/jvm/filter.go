package jvm

import "strings"

var exactRemove = map[string]struct{}{
	"-XX:-PrintCommandLineFlags": {},

	// GC selector — we always inject +UseZGC
	"-XX:+UseG1GC":           {},
	"-XX:-UseG1GC":           {},
	"-XX:+UseZGC":            {},
	"-XX:-UseZGC":            {},
	"-XX:+UseShenandoahGC":   {},
	"-XX:-UseShenandoahGC":   {},
	"-XX:+UseParallelGC":     {},
	"-XX:+UseSerialGC":       {},
	"-XX:+UseCompressedOops": {},
	"-XX:-UseCompressedOops": {},

	"-XX:+PerfDisableSharedMem":  {},
	"-XX:-PerfDisableSharedMem":  {},
	"-XX:+AlwaysPreTouch":        {},
	"-XX:-AlwaysPreTouch":        {},
	"-XX:+DisableExplicitGC":     {},
	"-XX:-DisableExplicitGC":     {},
	"-XX:+ZProactive":            {},
	"-XX:-ZProactive":            {},
	"-XX:+ZUncommit":             {},
	"-XX:-ZUncommit":             {},
	"-XX:+UseNUMA":               {},
	"-XX:-UseNUMA":               {},
	"-XX:+UseThreadPriorities":   {},
	"-XX:-UseThreadPriorities":   {},
	"-XX:+UseLargePages":         {},
	"-XX:-UseLargePages":         {},
	"-XX:+ParallelRefProcEnabled": {},
	"-XX:-ParallelRefProcEnabled": {},

	// JDK9-era flags removed from JVM; cause fatal startup errors on JDK 25
	"-XX:+UseConcMarkSweepGC":      {},
	"-XX:-UseConcMarkSweepGC":      {},
	"-XX:+UseParNewGC":             {},
	"-XX:-UseParNewGC":             {},
	"-XX:+UseBiasedLocking":        {},
	"-XX:-UseBiasedLocking":        {},
	"-XX:+AggressiveOpts":          {},
	"-XX:-AggressiveOpts":          {},
	"-XX:+UseFastAccessorMethods":  {},
	"-XX:-UseFastAccessorMethods":  {},
	"-XX:+UnlockCommercialFeatures": {},
	"-XX:-UnlockCommercialFeatures": {},
	"-XX:+FlightRecorder":          {},
	"-XX:-FlightRecorder":          {},

	// Flags with no effect on JDK 25
	"-XX:+UseStringDeduplication":        {},
	"-XX:-UseStringDeduplication":        {},
	"-XX:+UseDynamicNumberOfGCThreads":   {},
	"-XX:-UseDynamicNumberOfGCThreads":   {},
	"-XX:+AlwaysActAsServerClassMachine": {},
	"-XX:-AlwaysActAsServerClassMachine": {},
	"-XX:+UseXMMForArrayCopy":            {},
	"-XX:-UseXMMForArrayCopy":            {},
	"-XX:+UseFPUForSpilling":             {},
	"-XX:-UseFPUForSpilling":             {},
	"-XX:-DontCompileHugeMethods":        {},
	"-XX:+DontCompileHugeMethods":        {},
	"-XX:+G1UseAdaptiveIHOP":             {},
	"-XX:-G1UseAdaptiveIHOP":             {},
	"-XX:+UnlockExperimentalVMOptions":   {},
	"-XX:-UnlockExperimentalVMOptions":   {},
	"-XX:+UnlockDiagnosticVMOptions":     {},
	"-XX:-UnlockDiagnosticVMOptions":     {},
	"-XX:+UseCounterDecay":               {},
	"-XX:-UseCounterDecay":               {},
}

var prefixRemove = []string{
	// Heap
	"-Xms",
	"-Xmx",
	"-XX:SoftMaxHeapSize=",

	// Metaspace
	"-XX:MetaspaceSize=",
	"-XX:MaxMetaspaceSize=",

	// ZGC tuning (we inject our own)
	"-XX:ZAllocationSpikeTolerance=",
	"-XX:ZCollectionInterval=",
	"-XX:ZFragmentationLimit=",
	"-XX:ZUncommitDelay=",

	// G1 tuning — stripped when launcher still injects legacy G1 flags
	"-XX:MaxGCPauseMillis=",
	"-XX:G1HeapRegionSize=",
	"-XX:G1NewSizePercent=",
	"-XX:G1MaxNewSizePercent=",
	"-XX:G1ReservePercent=",
	"-XX:G1HeapWastePercent=",
	"-XX:G1MixedGCCountTarget=",
	"-XX:InitiatingHeapOccupancyPercent=",
	"-XX:G1MixedGCLiveThresholdPercent=",
	"-XX:G1RSetUpdatingPauseTimePercent=",
	"-XX:G1SATBBufferEnqueueingThresholdPercent=",
	"-XX:G1ConcRSHotCardLimit=",
	"-XX:G1ConcRefinementServiceIntervalMillis=",
	"-XX:GCTimeRatio=",
	"-XX:SurvivorRatio=",
	"-XX:MaxTenuringThreshold=",
	"-XX:SoftRefLRUPolicyMSPerMB=",

	// GC threads
	"-XX:ParallelGCThreads=",
	"-XX:ConcGCThreads=",

	// Code cache
	"-XX:ReservedCodeCacheSize=",
	"-XX:NonNMethodCodeHeapSize=",
	"-XX:ProfiledCodeHeapSize=",
	"-XX:NonProfiledCodeHeapSize=",

	// C2 JIT
	"-XX:MaxInlineLevel=",
	"-XX:FreqInlineSize=",
	"-XX:InlineSmallCode=",
	"-XX:MaxNodeLimit=",
	"-XX:NodeLimitFudgeFactor=",
	"-XX:NmethodSweepActivity=",
	"-XX:AllocatePrefetchStyle=",
	"-XX:CompileThresholdScaling=",

	// Misc
	"-XX:LargePageSizeInBytes=",
	"-XX:AutoBoxCacheMax=",
	"-XX:ThreadPriorityPolicy=",
	"-Dsun.reflect.inflationThreshold=",
	"-Djdk.reflect.useDirectMethodHandleOnly=",
	"-Dio.netty.jfr.enabled=",
	"-Dio.netty.noUnsafe=",
	"-Dio.netty.tryReflectionSetAccessible=",
	"-Djdk.attach.allowAttachSelf=",
	"--sun-misc-unsafe-memory-access=",

	// Removed in JDK 9+ (PermGen, illegal-access)
	"-XX:PermSize=",
	"-XX:MaxPermSize=",
	"--illegal-access=",
}

// jvmFlagTakesNextArg reports flags whose value is the following argv element.
func jvmFlagTakesNextArg(flag string) bool {
	switch flag {
	case "--add-opens", "--add-exports", "--add-reads", "--patch-module",
		"--upgrade-module-path", "--module-path", "--limit-modules", "--add-modules",
		"--module", "-m", "-p",
		"-javaagent", "-agentlib", "-agentpath":
		return true
	default:
		return false
	}
}

// splitArgs partitions the launcher's argv into JVM flags, the main class,
// and arguments passed to main().
func splitArgs(args []string) (jvm []string, mainClass string, app []string) {
	for i := 0; i < len(args); {
		a := args[i]
		if a == "-jar" {
			jvm = append(jvm, a)
			i++
			if i < len(args) {
				jvm = append(jvm, args[i])
				i++
			}
			return jvm, "", args[i:]
		}
		if a == "-classpath" || a == "-cp" {
			jvm = append(jvm, a)
			i++
			if i < len(args) {
				jvm = append(jvm, args[i])
			}
			i++
			continue
		}
		if jvmFlagTakesNextArg(a) {
			jvm = append(jvm, a)
			i++
			if i < len(args) {
				jvm = append(jvm, args[i])
			}
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			jvm = append(jvm, a)
			i++
			continue
		}
		mainClass = a
		app = args[i+1:]
		return
	}
	return
}

func shouldRemove(arg string) bool {
	if _, ok := exactRemove[arg]; ok {
		return true
	}
	for _, p := range prefixRemove {
		if strings.HasPrefix(arg, p) {
			return true
		}
	}
	return false
}

// IsLikelyGameLaunch reports whether argv looks like the actual Minecraft
// client process (not Gravit bootstrap / JavaAgent restart stage).
func IsLikelyGameLaunch(orig []string) bool {
	joined := strings.Join(orig, " ")
	return strings.Contains(joined, "net.minecraft.client.main") ||
		strings.Contains(joined, "net.minecraft.client.Main") ||
		strings.Contains(joined, "net.minecraft.launchwrapper.Launch") ||
		strings.Contains(joined, "cpw.mods") ||
		strings.Contains(joined, "GradleStart") ||
		strings.Contains(joined, "--gameDir") ||
		strings.Contains(joined, "--assetsDir") ||
		strings.Contains(joined, "--version")
}

// FilterArgs strips launcher-injected flags that conflict with ours
// (including legacy JDK9 flags fatal on JDK 25), then splices the
// generated flags back in, preserving the original main class and app args.
func FilterArgs(orig, injected []string) []string {
	jvmArgs, mainClass, app := splitArgs(orig)

	filtered := make([]string, 0, len(jvmArgs))
	for _, a := range jvmArgs {
		if !shouldRemove(a) {
			filtered = append(filtered, a)
		}
	}

	result := make([]string, 0, len(filtered)+len(injected)+1+len(app))
	result = append(result, filtered...)
	result = append(result, injected...)
	if mainClass != "" {
		result = append(result, mainClass)
	}
	return append(result, app...)
}

// InjectArgs keeps original JVM args and appends injected args before main class.
func InjectArgs(orig, injected []string) []string {
	jvmArgs, mainClass, app := splitArgs(orig)
	result := make([]string, 0, len(jvmArgs)+len(injected)+1+len(app))
	result = append(result, jvmArgs...)
	result = append(result, injected...)
	if mainClass != "" {
		result = append(result, mainClass)
	}
	return append(result, app...)
}
