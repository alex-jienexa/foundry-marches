# Передача контекста FoundryMarches

## Текущее состояние на 2026-11-18

Рабочая ветка: `feat/task-004-character-decoder`. Ветка не слита в `main`; владелец планирует открыть PR самостоятельно после проверки этих изменений. Не создавать PR и не выполнять merge.

Последний подтверждённый code HEAD перед документационным checkpoint: `4ca6fa87c4b7b40e25832f7ca15400bb39f5d713`. Документационные изменения после него фиксируются отдельными commit в этой же ветке.

## Подтверждённая инфраструктура

- Foundry VTT 14 build 367, PF2e 8.7.0.
- ThreeHats self-hosted Foundry REST API Relay 3.4.1; Pairing подтверждён.
- Relay: `https://relay.jiene.ru`; WebSocket: `wss://relay.jiene.ru/relay`.
- Caddy обслуживает HTTPS; Foundry доступен через `foundry.jiene.ru`.
- Docker Compose: PostgreSQL 18 Alpine и Go API.
- API опубликован на host loopback `127.0.0.1:8080`; PostgreSQL не публикует host port.
- `GET /healthz` вернул HTTP 200.
- Не трогать `catalog-container` и `rosochkinsad.ru`.
- Не удалять PostgreSQL volume; не выполнять destructive DB operations без отдельного разрешения.
- Секреты не копировать в Git, чат или документацию.

## Подтверждённый результат Foundry sync

После обновления PF2e владелец синхронизировал мир Foundry и повторил sync приложения. Успешно обнаружены и сохранены 11 Actor snapshots в `foundry_actors`.

- Все записи относятся к внутреннему UUID WorldID приложения `00000000-0000-4000-8000-000000000001`.
- У всех 11 записей `character_id IS NULL`; disposition — `snapshot_only`.
- Это ожидаемо: приложение не создаёт персонажей автоматически и не выводит ownership из косвенных признаков.
- Relay identifier `westmarches` не используется как WorldID приложения.
- Синхронизация read-only; Foundry Actor не изменялись.

## Код в рабочей ветке

- Go module расположен в `src/backend/`.
- Bootstrap REST API и `GET /healthz`.
- Relay HTTP adapter, world discovery/sync service.
- PF2e Actor DTO/decoder и частичный calculator.
- Persistence mapper/repository для characters, foundry_actors и inventory.
- Unit/HTTP tests и opt-in PostgreSQL integration tests.
- Docker Compose и Dockerfile.
- CI workflow с jobs: documentation, database (PostgreSQL 15/16) и backend (gofmt, go test, go vet).

Наличие CI workflow подтверждено, но GitHub Actions runs для этой ветки на момент проверки отсутствовали. Не заявлять успешный удалённый прогон до фактического результата после PR.

## Состояние этапов

- TASK-002 / этап 0 — `in_progress`: Relay и sync проверены; WebDAV ETag/If-Match, Codex exports, SSE/headless session и usage/quota остаются pending.
- TASK-003 / этап 1 — `in_progress`: bootstrap, PostgreSQL/Compose, health и CI есть; Discord OAuth, sessions, RBAC, audit, migrations, backup/restore не завершены.
- TASK-004 / этап 2 — `in_progress`: decoder, adapter, persistence и world sync есть; явная Actor binding, approval, limits/NPC и полная обработка job/retry не реализованы.
- TASK-005…TASK-013 — backlog.

## Следующий рекомендуемый шаг

1. После PR/merge владельцем и подтверждения актуального HEAD продолжить в рабочей ветке.
2. Спроектировать и реализовать явную привязку существующего application character к Foundry Actor:
   - Actor должен иметь type `character`;
   - Actor должен принадлежать настроенному world UUID;
   - Actor не должен быть связан с другим application character;
   - application character не должен быть связан с другим Actor;
   - операция должна требовать соответствующего разрешения и быть атомарной;
   - не создавать персонажей и не назначать владельцев автоматически.
3. Добавить API, DB constraints/transactions и tests для link/unlink policy; до изменения схемы изучить существующие constraints.
4. Затем approval и полноценный sync lifecycle.
5. Отдельно вернуться к оставшимся проверкам spike и к недостающим исходным требованиям.

## Архитектурные правила

- Foundry — авторитетный источник PF2e mechanics; текущая интеграция read-only.
- Не менять Foundry Actor.
- Сохранять raw snapshots и неизвестные поля; не выдумывать отсутствующие данные.
- Различать `Actor.items[]` и ссылки в `system.items` классов/ancestry/background.
- Использовать IDs/slugs/UUID/compendiumSource, не локализованные имена.
- Не исполнять произвольные Rule Elements; unsupported expressions/predicates отмечать явно.
- Не менять внутренний WorldID UUID на Relay identifier.
- Не делать автоматическое создание персонажей или inference ownership.
- Не реализовывать XP write в Foundry без отдельного решения и разрешения.
- Все исполняемые/тестируемые исходники размещать под `src/`; документация/PAD — вне `src/`.

## Документы

- Runtime результаты: `docs/verification.md`.
- API и Compose инструкция: `docs/integrations/foundry-world-sync-api.md`.
- Глобальная дорожная карта: `docs/development/roadmap.md`.
- Контрольные точки: `docs/development/milestones.md`.
- Вопросы и пробелы: `docs/development/open-questions.md`, `docs/development/source-gaps.md`.
- Текущая машина состояния: `state/current.yaml`.
