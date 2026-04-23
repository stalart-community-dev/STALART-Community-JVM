# Configuration profile

The JDK 25 branch uses a single primary profile: `stable`.

## What this means

- default configuration file: `configs/stable.json`;
- active profile is stored at `HKCU\\Software\\StalartJvmWrapper`;
- `cli.exe --autotune` sets `stable` as the active profile;
- `Reset Config` recreates `stable` with default values.

## When custom JSON profiles are needed

You can still add custom profiles manually into `configs/` and switch them via `Select Config`.
For production usage, `stable` remains the baseline.

## Where to inspect runtime behavior

- `logs/wrapper.log`
