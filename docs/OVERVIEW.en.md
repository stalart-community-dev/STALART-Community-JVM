# Architecture and Runtime Flow

## Components

- `cmd/cli` — interactive install/uninstall and profile selection UI.
- `cmd/service` — IFEO debugger, real entrypoint for game Java launch.
- `internal/config` — profile loading/generation.
- `internal/jvm` — JVM flag building/filtering.

## Launch flow

1. `cli.exe --install` registers IFEO interception.
2. Game starts Java runtime.
3. Windows launches `service.exe`.
4. `service.exe`:
   - detects game launch vs bootstrap;
   - loads active profile;
   - injects compatible flags;
   - starts child process and waits for exit.
5. On exit:
   - writes `wrapper.log`.

## Log safety

- raw JVM args are never written;
- user paths are redacted to `<user>`;
- `wrapper.log` size is bounded.

## Auto-tune behavior

- on first setup, `stable` is created and activated;
- `cli.exe --autotune` sets `stable` as active;
- there are no additional built-in presets: deviations are done via custom JSON.
