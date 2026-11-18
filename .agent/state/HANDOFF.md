# Передача контекста FoundryMarches

## Текущее состояние на 2026-11-18

Рабочая ветка: `feat/task-004-character-linking`. Ветка основана на `main` merge commit `618bf862dd070e3688709cc319cf55b042e96df0`; не слита в `main`. Владелец самостоятельно выполнит локальную проверку и создаст PR. Не создавать PR и не выполнять merge.

Последний известный HEAD на момент обновления: `e156bdf1a455b5f54d3cf5dc0487c1c431763734`. Перед дальнейшими изменениями обязательно перепроверить branch HEAD.

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

- Текущая feature branch: `feat/task-004-character-linking`, создана от main merge commit `618bf862dd070e3688709cc319cf55b042e96df0`.
- Добавлен `FoundryRepository.LinkActor` в `src/backend/internal/characters/foundry_link.go`: транзакционная явная связь уже обнаруженного Actor типа `character` с существующим application character; проверки world/наличия/конфликтов; idempotent повтор; append-only audit. Owner и `owner_confirmed_at` не меняются.
- Добавлены unit coverage для SQLSTATE helper и opt-in PostgreSQL integration tests на success/idempotency/duplicate links/NPC rejection.
- Добавлен Discord-only auth boundary: Gateway bot передаёт Discord user/guild/member role IDs, backend проверяет guild allow-list и зарегистрированную учётную запись; bot-to-backend защищён отдельным `DISCORD_BOT_API_TOKEN`, не равным Foundry sync token.
- Добавлены registration, hybrid RBAC (`user_roles` + `discord_role_map`), admin bootstrap через `DISCORD_ADMIN_USER_IDS`, internal endpoints для list/preview/link, Discord confirmation button и commands `/register`, `/foundry-actors`, `/characters`, `/character-link`, `/role-grant`, `/role-map`.
- Добавлена migration `docs/database/migrations/0002_discord_bot_auth.sql`, Dockerfile и Compose service для бота, инструкция `docs/integrations/discord-bot.md` и базовые unit tests для service-token middleware/request decoder.
- Runtime/compile/test verification этих новых файлов ещё не выполнены. Migration должна быть применена к существующей БД вручную; не удалять volume.

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
- TASK-003 / этап 1 — `in_progress`: bootstrap, PostgreSQL/Compose, health и CI есть; Discord bot identity/RBAC/migration реализованы в bot-only варианте; backups/restore, полный audit coverage, runtime verification и web sessions (если появятся) остаются pending.
- TASK-004 / этап 2 — `in_progress`: decoder, adapter, persistence, world sync, explicit Actor link repository и Discord confirmation flow есть; approval/rejection, ownership confirmation, limits/NPC и полная обработка job/retry не реализованы.
- TASK-005…TASK-013 — backlog.

## Следующий рекомендуемый шаг

1. Запустить `gofmt`, `go test ./...`, `go vet ./...` из `src/backend` и PostgreSQL integration suite с изолированной БД `*_test`; проверить/исправить найденные ошибки.
2. Проверить PR/main workflow status после merge #1 отдельно от локального сообщения владельца об успешных тестах.
3. Спроектировать user identity/session/RBAC для API (TASK-003 dependency) и только после этого добавить `POST /api/characters/link` с permission `foundry.manage`, без использования sync token как пользовательского credential.
4. Затем реализовать approval/rejection и character review history; сохранить DB slot trigger и добавить API-level 409 mapping для slot limit.
5. Отдельно проработать integration job/retry/awaiting_sync lifecycle. Не включать XP write в Foundry без нового архитектурного решения и разрешения владельца.

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

## Discord bot setup

Подробный сценарий создания Discord application/bot, получения Application ID и Guild ID, настройки `.env`, применения migration, запуска Compose и проверки персонажа Андрея: `docs/integrations/discord-bot.md`. Секреты не передавать в чат и не коммитить.

## Документы

- Runtime результаты: `docs/verification.md`.
- API и Compose инструкция: `docs/integrations/foundry-world-sync-api.md`.
- Глобальная дорожная карта: `docs/development/roadmap.md`.
- Контрольные точки: `docs/development/milestones.md`.
- Вопросы и пробелы: `docs/development/open-questions.md`, `docs/development/source-gaps.md`.
- Текущая машина состояния: `state/current.yaml`.
