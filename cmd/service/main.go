// Command service is the IFEO Debugger for javaw.exe / java.exe. Windows
// launches it as: service.exe <full path to java[w]> <jvm args...>. When
// the image matches the STALART bundled JVM (SHA-256), ZGC flags are merged;
// otherwise the process is started unchanged. The child is created with
// NtCreateUserProcess (IFEO skip), priorities are boosted, then the
// service waits until the child process exits.
//
// service.exe has no UI — cli.exe handles install/config.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"stalart-wrapper/internal/config"
	"stalart-wrapper/internal/jvm"
	"stalart-wrapper/internal/logging"
	"stalart-wrapper/internal/process"
	"stalart-wrapper/internal/sysinfo"
)

func main() {
	closeLog, err := logging.Setup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[log] %v\n", err)
	}
	defer closeLog()

	if len(os.Args) < 2 {
		slog.Error("service started without target executable")
		fmt.Fprintln(os.Stderr, "[service] missing target executable")
		os.Exit(1)
	}

	slog.Info("service startup", "args_count", len(os.Args)-1)
	os.Exit(launch(os.Args[1], os.Args[2:]))
}

// launch spawns the target javaw with optional JVM flag injection and
// returns the exit code to propagate to the OS.
func launch(exePath string, args []string) int {
	isJava25Runtime := strings.Contains(strings.ToLower(exePath), "java25-windows-x86-64")
	origArgs := append([]string(nil), args...)
	presetName := "passthrough"
	presetMode := "passthrough"

	sys := sysinfo.Detect()
	slog.Info("system detected",
		"cores", sys.CPUCores,
		"ram_gb", sys.TotalGB(),
		"free_ram_gb", sys.FreeGB(),
		"l3_mb", sys.L3CacheMB,
		"big_cache", sys.HasBigCache(),
		"large_pages", sys.LargePages,
	)

	if err := config.Ensure(); err != nil {
		slog.Warn("config ensure failed", "err", err)
		fmt.Fprintf(os.Stderr, "[config] %v\n", err)
	}

	tune, matchErr := jvm.MatchRuntime(exePath)
	if matchErr != nil {
		slog.Warn("JVM image match skipped", "err", matchErr)
	}

	switch {
	case !tune:
		slog.Info("JVM passthrough (not STALART bundled image or no reference JVM)")
	case !jvm.IsLikelyGameLaunch(args):
		slog.Info("bundled JVM bootstrap detected: passthrough without JVM injection", "arg_count", len(args))
	default:
		cfg, loadedName, cfgErr := config.LoadActive()
		switch {
		case cfgErr != nil:
			slog.Warn("config load failed, passthrough without JVM injection", "err", cfgErr)
		case cfg.HeapSizeGB == 0:
			slog.Warn("config has zero heap, passthrough without JVM injection", "name", loadedName)
		default:
			if requested := config.ActiveName(); requested != "" && requested != loadedName {
				slog.Warn("active config missing, fell back to default",
					"requested", requested,
					"loaded", loadedName,
				)
			}
			injected := append(jvm.ClientCompatProps(), jvm.Flags(cfg)...)
			args = jvm.FilterArgs(args, injected)
			presetName = loadedName
			presetMode = "zgc"
			slog.Info("config loaded",
				"name", loadedName,
				"heap_gb", cfg.HeapSizeGB,
				"metaspace_mb", cfg.MetaspaceMB,
				"parallel_gc", cfg.ParallelGCThreads,
				"conc_gc", cfg.ConcGCThreads,
				"spike_tolerance", cfg.ZAllocationSpikeTolerance,
				"large_pages", cfg.UseLargePages,
				"flags_count", len(injected),
			)
		}
	}

	runOnce := func(runExe string, runArgs []string, attempt string) (int, int64, error) {
		slog.Info("process starting",
			"attempt", attempt,
			"exe", logging.RedactPath(runExe),
			"arg_count", len(runArgs),
		)

		proc, err := process.Start(runExe, runArgs)
		if err != nil {
			slog.Error("process start failed", "attempt", attempt, "err", err)
			fmt.Fprintf(os.Stderr, "[process] %v\n", err)
			return 1, 0, err
		}
		defer proc.Close()
		slog.Info("process started", "attempt", attempt, "pid", proc.PID)

		if err := proc.Boost(); err != nil {
			slog.Warn("process boost partial", "attempt", attempt, "err", err)
			fmt.Fprintf(os.Stderr, "[boost] %v\n", err)
		}

		start := time.Now()
		code, err := proc.Wait()
		waitMs := time.Since(start).Milliseconds()
		slog.Info("process exit summary",
			"attempt", attempt,
			"preset_name", presetName,
			"preset_mode", presetMode,
			"wait_ms", waitMs,
			"exit_code", code,
		)
		if err != nil {
			slog.Error("process wait failed", "attempt", attempt, "err", err, "wait_ms", waitMs)
			fmt.Fprintf(os.Stderr, "[wait] %v\n", err)
			return 1, waitMs, err
		}
		return code, waitMs, nil
	}

	code, waitMs, err := runOnce(exePath, args, "primary")
	if err != nil {
		return 1
	}

	isFastUnstableExit := func(code int, waitMs int64) bool {
		signed := int32(uint32(code))
		return waitMs <= 5000 && signed == -123
	}

	bellsoftJavaPath := func(currentExe string) string {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		name := "javaw.exe"
		if strings.HasSuffix(strings.ToLower(currentExe), "\\java.exe") {
			name = "java.exe"
		}
		candidate := filepath.Join(home, "AppData", "Roaming", "GravitLauncherStore", "Java", "bellsoft-lts", "bin", name)
		if _, err := os.Stat(candidate); err != nil {
			return ""
		}
		return candidate
	}

	if isJava25Runtime && jvm.IsLikelyGameLaunch(origArgs) && isFastUnstableExit(code, waitMs) {
		slog.Warn("fast unstable Java 25 primary exit, retrying identical launch",
			"wait_ms", waitMs,
			"code", int32(uint32(code)),
		)
		code, waitMs, err = runOnce(exePath, args, "retry_passthrough")
		if err != nil {
			return 1
		}
		if isFastUnstableExit(code, waitMs) {
			if alt := bellsoftJavaPath(exePath); alt != "" {
				slog.Warn("fast unstable Java 25 retry exit, trying BellSoft runtime",
					"wait_ms", waitMs,
					"code", int32(uint32(code)),
					"alt_exe", logging.RedactPath(alt),
				)
				code, waitMs, err = runOnce(alt, args, "fallback_bellsoft")
				if err != nil {
					return 1
				}
			}
		}
	}

	u := uint32(code)
	slog.Info("service exit",
		"code", code,
		"code_i32", int32(u),
		"code_hex", fmt.Sprintf("0x%x", u),
		"low8_unsigned", u&0xFF,
		"low8_as_signed_int8", int(int8(u&0xFF)),
		"wait_ms", waitMs,
	)
	return code
}
