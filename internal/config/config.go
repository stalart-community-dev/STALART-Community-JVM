// Package config models the JVM tuning profile: persistence on disk
// (configs/stable.json), the "active" pointer in HKCU, and the universal
// default profile.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const registryPath = `Software\StalartJvmWrapper`
const fallbackPreset = "stable"

// ErrNotFound is returned when a config file does not exist on disk.
var ErrNotFound = errors.New("config not found")

type Config struct {
	HeapSizeGB  int  `json:"heap_size_gb"`
	PreTouch    bool `json:"pre_touch"`
	MetaspaceMB int  `json:"metaspace_mb"`

	ZAllocationSpikeTolerance float64 `json:"z_allocation_spike_tolerance"`
	ZCollectionIntervalSec    int     `json:"z_collection_interval_sec"`
	ZFragmentationLimit       int     `json:"z_fragmentation_limit"`

	// Zero means let ZGC auto-detect from hardware.
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

// DefaultConfig returns the universal stable profile. Values are conservative
// enough for any modern machine with 8+ GB RAM.
func DefaultConfig() Config {
	return Config{
		HeapSizeGB:  6,
		PreTouch:    true,
		MetaspaceMB: 512,

		ZAllocationSpikeTolerance: 5.0,
		ZCollectionIntervalSec:    0,
		ZFragmentationLimit:       15,

		// 0 = ZGC auto-detects optimal thread counts from CPU topology.
		ParallelGCThreads: 0,
		ConcGCThreads:     0,

		ReservedCodeCacheSizeMB: 512,
		MaxInlineLevel:          18,
		FreqInlineSize:          600,
		InlineSmallCode:         4500,
		MaxNodeLimit:            280000,
		NodeLimitFudgeFactor:    8000,
		CompileThresholdScaling: 0.65,

		UseThreadPriorities:  true,
		ThreadPriorityPolicy: 1,
		AutoBoxCacheMax:      8192,
	}
}

// Dir returns the configs directory next to the executable.
func Dir() string {
	self, err := os.Executable()
	if err != nil {
		return filepath.Join(".", "configs")
	}
	exeDir := filepath.Dir(self)
	exeConfigs := filepath.Join(exeDir, "configs")

	if strings.EqualFold(filepath.Base(exeDir), "build") {
		rootConfigs := filepath.Join(filepath.Dir(exeDir), "configs")
		if st, statErr := os.Stat(rootConfigs); statErr == nil && st.IsDir() {
			return rootConfigs
		}
	}
	return exeConfigs
}

// Ensure creates the configs directory and stable.json if they do not exist,
// then selects stable as the active config when no selection has been made.
func Ensure() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create configs dir: %w", err)
	}

	path := filepath.Join(dir, "stable.json")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := DefaultConfig().Save("stable"); err != nil {
			return fmt.Errorf("save stable config: %w", err)
		}
	}

	if ActiveName() == "" {
		if err := SetActive("stable"); err != nil {
			return fmt.Errorf("set active config: %w", err)
		}
	}
	return nil
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
// Falls back to stable when no selection has been made or the profile is missing.
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
	}
	return cfg, requested, err
}

// ActiveExists reports whether the active config file exists on disk.
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
