# Исходный код приложения

Весь исполняемый и тестируемый код приложения размещается только в `src/`.

```text
src/
├── backend/       # Go-модуль: API, worker, доменные и интеграционные пакеты
├── bot/           # отдельный процесс Discord-бота, если он выделен из backend
└── frontend/      # SolidJS + TypeScript UI
```

Backend является самостоятельным Go-модулем в `src/backend/go.mod`. Go-пакеты приложения размещаются под `src/backend/internal/`. Тесты реализации хранятся рядом с пакетами. Тестовые fixtures и проектная документация остаются в корневом `docs/`, а не копируются в исходный код.

Структура процессов и границы модулей описаны в [docs/architecture/components.md](../docs/architecture/components.md).
