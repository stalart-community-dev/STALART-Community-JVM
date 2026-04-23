# Optimal Config Selection

## Goal

Choose the best JVM preset for a specific PC without built-in runtime metric collection.

## Baseline strategy

1. Apply `Apply Recommended Config` / `cli.exe --autotune`.
2. Use `stable` as the default baseline.
3. If needed, compare `stable` against a custom JSON profile.

## Manual validation

Validate presets manually:
- same map/route scenario;
- same graphics settings;
- at least 10-15 minutes per preset;
- compare real frametime smoothness and stutter behavior.

## Measurement hygiene

- same map/route scenario for each run;
- same graphics settings;
- close heavy background processes;
- avoid very short sessions;
- lock selected result via `Select Config`.
