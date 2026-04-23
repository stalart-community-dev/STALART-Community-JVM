# Configuration Parameters (`configs/*.json`)

Key preset fields and what they control.

## Core

- `heap_size_gb` — heap size (`-Xms/-Xmx`).
- `metaspace_mb` — metaspace limit.
- `pre_touch` — enables `AlwaysPreTouch`.

## ZGC

- `z_allocation_spike_tolerance` — tolerance for allocation bursts.
- `z_collection_interval_sec` — forced ZGC cycle interval (0 = automatic).
- `z_fragmentation_limit` — fragmentation threshold for more aggressive cleanup.

## Threads and JIT

- `parallel_gc_threads`, `conc_gc_threads` — GC thread counts.
- `reserved_code_cache_size_mb` — code cache size.
- `compile_threshold_scaling` — JIT aggressiveness.

## Other

- `use_string_deduplication` — string dedup.
- `use_large_pages` — large pages mode (requires correct OS setup).

## Recommendations

- Keep `stable` as the default daily-use profile.
- Change 1-2 parameters at a time and compare by benchmark.
- Prefer real gameplay runs over short startup-only runs.
