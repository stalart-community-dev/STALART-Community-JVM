# Stalart JVM Wrapper

[![eng](https://img.shields.io/badge/lang-English-blue)](README.en.md)
[![ru](https://img.shields.io/badge/lang-Russian-blue)](README.md)

Unofficial utility for optimizing JVM startup in Stalart on Windows (JDK 25 branch).

## Features

- installs IFEO interception for `java.exe` / `javaw.exe` through `service.exe`;
- applies a **single stable profile** from `configs/stable.json`;
- filters conflicting launcher flags and injects safe JVM options;
- writes launch diagnostics to `logs/wrapper.log`.

## How it works

1. User runs `cli.exe` and selects `Install`.
2. Windows registers `service.exe` as IFEO debugger for Java processes.
3. On game launch, Windows starts `service.exe` first.
4. `service.exe` validates launch context, applies the `stable` profile, and starts JVM.
5. On exit it writes status and diagnostics to the log.

Detailed flow: [docs/OVERVIEW.en.md](./docs/OVERVIEW.en.md).

## Quick start

1. Extract release files into `jvm_wrapper` next to the Stalart launcher.
2. Run `cli.exe` as administrator.
3. Select `Install`.
4. Launch the game.

## Configuration

- active profile key: `HKCU\\Software\\StalartJvmWrapper`;
- default profile file: `configs/stable.json`;
- CLI actions:
  - `Select Config` to switch profile (if you added custom JSON profiles),
  - `Reset Config` to recreate `stable.json` defaults,
  - `cli.exe --autotune` to set `stable` as active profile.

JSON parameters: [docs/PARAMS.en.md](./docs/PARAMS.en.md).

## Troubleshooting

See [docs/TROUBLESHOOTING.en.md](./docs/TROUBLESHOOTING.en.md).

## Build

```bash
go build -trimpath -ldflags="-s -w" -o build/cli.exe ./cmd/cli
go build -trimpath -ldflags="-s -w" -o build/service.exe ./cmd/service
```

## Credits

- [SilentBless](https://github.com/SilentBless) — original idea and early JVM-wrapper architecture.
- [stalart-community-dev](https://github.com/stalart-community-dev/stalart-community-jvm) — ongoing JDK 25 branch development.

## Disclaimer

- This utility is not affiliated with the game developer.
- Use at your own risk.
- For utility-related issues, use repository Issues.
