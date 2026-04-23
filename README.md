# Stalart JVM Wrapper

[![eng](https://img.shields.io/badge/lang-English-blue)](README.en.md)
[![ru](https://img.shields.io/badge/lang-Russian-blue)](README.md)

Неофициальная утилита для оптимизации запуска JVM в Stalart на Windows (ветка JDK 25).

## Возможности

- устанавливает IFEO-перехват для `java.exe` / `javaw.exe` через `service.exe`;
- применяет **единый стабильный профиль** `configs/stable.json` для игрового запуска;
- фильтрует конфликтующие флаги лаунчера и подставляет безопасные JVM-параметры;
- пишет технический лог запуска в `logs/wrapper.log`.

## Как работает

1. Пользователь запускает `cli.exe` и выбирает `Install`.
2. Windows регистрирует `service.exe` как IFEO debugger для Java-процесса.
3. При старте игры Windows сначала запускает `service.exe`.
4. `service.exe` проверяет сценарий запуска, применяет `stable`-конфиг и стартует JVM.
5. После завершения процесса пишет код выхода и диагностику в лог.

Подробная схема: [docs/OVERVIEW.md](./docs/OVERVIEW.md).

## Быстрый старт

1. Распакуйте релиз в `jvm_wrapper` рядом с лаунчером Stalart.
2. Запустите `cli.exe` от имени администратора.
3. Выберите `Install`.
4. Запустите игру.

## Конфигурация

- активный профиль хранится в `HKCU\\Software\\StalartJvmWrapper`;
- основной профиль: `configs/stable.json`;
- в CLI доступны:
  - `Select Config` — выбрать активный JSON (если добавлены кастомные профили),
  - `Reset Config` — пересоздать `stable.json` значениями по умолчанию,
  - `cli.exe --autotune` — выставить активным `stable`.

Параметры JSON: [docs/PARAMS.md](./docs/PARAMS.md).

## Troubleshooting

См. [docs/TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md).

## Сборка

```bash
go build -trimpath -ldflags="-s -w" -o build/cli.exe ./cmd/cli
go build -trimpath -ldflags="-s -w" -o build/service.exe ./cmd/service
```

## Credits

- [SilentBless](https://github.com/SilentBless) — оригинальная идея и ранняя архитектура JVM-wrapper.
- [stalart-community-dev](https://github.com/stalart-community-dev/stalart-community-jvm) — развитие ветки JDK 25.

## Дисклеймер

- Утилита не аффилирована с разработчиками игры.
- Используйте на свой риск.
- По вопросам утилиты используйте Issues репозитория.
