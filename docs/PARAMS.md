# Параметры конфигурации (`configs/*.json`)

Ниже перечислены основные поля пресетов и их смысл.

## Базовые

- `heap_size_gb` — размер heap (`-Xms/-Xmx`).
- `metaspace_mb` — `Metaspace` лимит.
- `pre_touch` — включает `AlwaysPreTouch`.

## ZGC

- `z_allocation_spike_tolerance` — допуск всплесков аллокаций.
- `z_collection_interval_sec` — интервал принудительного цикла ZGC (0 = авто).
- `z_fragmentation_limit` — порог фрагментации для более агрессивной уборки.

## Потоки и компиляция

- `parallel_gc_threads`, `conc_gc_threads` — число GC-потоков.
- `reserved_code_cache_size_mb` — размер code cache.
- `compile_threshold_scaling` — агрессивность JIT-компиляции.

## Прочие

- `use_string_deduplication` — дедупликация строк.
- `use_large_pages` — large pages (требует корректной системной настройки).

## Рекомендации

- Для повседневного использования оставляйте `stable` как базовый профиль.
- Меняйте 1-2 параметра за раз и сравнивайте через benchmark.
- Используйте реальные игровые прогоны, а не только короткие запуски.
