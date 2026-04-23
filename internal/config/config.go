// Package config models the JVM tuning profile: persistence on disk
// (configs/*.json), the "active" pointer in HKCU, and auto-generation
// from detected hardware.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	"stalart-wrapper/internal/sysinfo"
)

const registryPath = `Software\StalartJvmWrapper`
const fallbackPreset = "balanced"

// ErrNotFound is returned when a config file does not exist on disk.
var ErrNotFound = errors.New("config not found")

type Config struct {
	HeapSizeGB  int  `json:"heap_size_gb"`
	PreTouch    bool `json:"pre_touch"`
	MetaspaceMB int  `json:"metaspace_mb"`

	ZAllocationSpikeTolerance float64 `json:"z_allocation_spike_tolerance"`
	ZCollectionIntervalSec    int     `json:"z_collection_interval_sec"`
	ZFragmentationLimit       int     `json:"z_fragmentation_limit"`

	ParallelGCThreads int `json:"parallel_gc_threads"`
	ConcGCThreads     int `json:"conc_gc_threads"`

	ReservedCodeCacheSizeMB int     `json:"reserved_code_cache_size_mb"`
	MaxInlineLevel          int     `json:"max_inline_level"`
	FreqInlineSize          int     `json:"freq_inline_size"`
	InlineSmallCode         int     `json:"inline_small_code"`
	MaxNodeLimit            int     `json:"max_node_limit"`
	NodeLimitFudgeFactor    int     `json:"node_limit_fudge_factor"`
	CompileThresholdScaling float64 `json:"compile_threshold_scaling"`

	UseLargePages        bool `json:"use_large_pages"`
	UseThreadPriorities  bool `json:"use_thread_priorities"`
	ThreadPriorityPolicy int  `json:"thread_priority_policy"`
	AutoBoxCacheMax      int  `json:"auto_box_cache_max"`
}

// Dir returns the configs directory next to the executable.
// Falls back to ./configs if the executable path can't be resolved.
func Dir() string {
	self, err := os.Executable()
	if err != nil {
		return filepath.Join(".", "configs")
	}
	exeDir := filepath.Dir(self)
	exeConfigs := filepath.Join(exeDir, "configs")

	// Dev/build layout: binaries live in build/, canonical presets are in
	// project-root configs/. Prefer root configs to avoid duplicating runtime
	// state under build/configs.
	if strings.EqualFold(filepath.Base(exeDir), "build") {
		rootConfigs := filepath.Join(filepath.Dir(exeDir), "configs")
		if st, statErr := os.Stat(rootConfigs); statErr == nil && st.IsDir() {
			return rootConfigs
		}
	}
	return exeConfigs
}

// Ensure makes sure the configs directory and a fallback config exist,
// and that an active config is selected in the registry.
func Ensure(sys sysinfo.Info) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create configs dir: %w", err)
	}

	for name, cfg := range Presets(sys) {
		path := filepath.Join(dir, name+".json")
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		if err := cfg.Save(name); err != nil {
			return fmt.Errorf("save %s config: %w", name, err)
		}
	}

	entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return fmt.Errorf("scan configs dir: %w", err)
	}
	if len(entries) == 0 {
		if err := Generate(sys).Save(fallbackPreset); err != nil {
			return fmt.Errorf("save %s config: %w", fallbackPreset, err)
		}
	}

	if ActiveName() == "" {
		if err := SetActive(RecommendPreset(sys)); err != nil {
			return fmt.Errorf("set active config: %w", err)
		}
	}
	return nil
}

// RecommendPreset returns the best preset name for current hardware.
func RecommendPreset(sys sysinfo.Info) string {
	switch {
	case sys.TotalGB() < 12 || sys.CPUThreads <= 4:
		return "compat"
	case sys.TotalGB() >= 32 && sys.CPUThreads >= 12 && sys.HasBigCache():
		return "ultra"
	case sys.TotalGB() >= 24 && sys.CPUThreads >= 10:
		return "performance"
	default:
		return "balanced"
	}
}

// Save writes the config to configs/<name>.json.
func (c Config) Save(name string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create configs dir: %w", err)
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Load reads configs/<name>.json.
func Load(name string) (Config, error) {
	path := filepath.Join(Dir(), name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

// LoadActive reads the config currently selected in the registry.
// Falls back to the balanced preset when no selection has been made
// or the selected profile no longer exists on disk.
func LoadActive() (cfg Config, loadedName string, err error) {
	requested := ActiveName()
	if requested == "" {
		requested = fallbackPreset
	}
	cfg, err = Load(requested)
	if errors.Is(err, ErrNotFound) {
		if requested != fallbackPreset {
			if fallbackCfg, fallbackErr := Load(fallbackPreset); fallbackErr == nil {
				return fallbackCfg, fallbackPreset, nil
			}
		}
		if fallbackCfg, fallbackErr := Load("default"); fallbackErr == nil {
			return fallbackCfg, "default", nil
		}
	}
	return cfg, requested, err
}

// ActiveExists reports whether the currently selected active config
// name has a corresponding file on disk.
func ActiveExists() bool {
	name := ActiveName()
	if name == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(Dir(), name+".json"))
	return err == nil
}

// List returns the names (without .json) of every config on disk.
func List() ([]string, error) {
	entries, err := filepath.Glob(filepath.Join(Dir(), "*.json"))
	if err != nil {
		return nil, fmt.Errorf("scan configs: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		base := filepath.Base(e)
		names = append(names, strings.TrimSuffix(base, ".json"))
	}
	return names, nil
}

// SetActive records the active config name in HKCU.
func SetActive(name string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, registryPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry: %w", err)
	}
	defer key.Close()
	if err := key.SetStringValue("ActiveConfig", name); err != nil {
		return fmt.Errorf("set ActiveConfig: %w", err)
	}
	return nil
}

// ActiveName reads the active config name from HKCU, empty string if unset.
func ActiveName() string {
	key, err := registry.OpenKey(registry.CURRENT_USER, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	val, _, err := key.GetStringValue("ActiveConfig")
	if err != nil {
		return ""
	}
	return val
}
