# Передача контекста FoundryMarches

## Состояние на 2026-11-25

Рабочая ветка: `feat/conventional-commits`. Перед следующими изменениями обязательно проверить точный HEAD и повторно получить актуальные SHA файлов. Ветка создана от `main` commit `8a69f78fb2a1bafe44fb8203a3a5e82656a0d5eb`. Не создавать PR и не выполнять merge: владелец делает это самостоятельно.

## Действующее решение по Foundry

[ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md) принято владельцем 2026-11-25.

- Foundry — единственный источник истины для Actor sheet data и PF2e mechanics.
- Relay integration read-only.
- Приложение не пишет XP, level, coins/currency, attributes, abilities, HP, AC, skills, class, ancestry, background, ownership/permissions, Actor flags или любые произвольные Actor fields.
- Единственное исключение — GM-authorized add/remove Item на существующем Actor (equipment, abilities и другие Items). Это отдельная узкая задача TASK-019.
- Item mutation требует server-side GM authorization, stable Item identifiers, dedicated least-privilege credential, operation ID/idempotency, audit, read-after-write verification и безопасного поведения при timeout/outage.
- Generic Relay write proxy запрещён. Read credentials остаются read-only; не выдавать write scope до завершения TASK-019 security review и owner approval.
- Ручные изменения в Foundry поступают в приложение через sync; приложение не перезаписывает их.
- App-owned campaign state хранится в PostgreSQL и не даёт приложению права записи в Foundry.

## Подтверждённый Production

- Production развёрнут в `/opt/foundry-marches/`, Compose project `foundry-marches-prod`.
- Отдельные API, Discord bot и PostgreSQL 18 Alpine; новая независимая Production БД, без переноса Dev данных.
- API доступен на loopback `127.0.0.1:8081`; PostgreSQL не опубликован на host port.
- Отдельные network и volume; API/Bot/PostgreSQL имеют memory limits и container hardening.
- Образы опубликованы в GHCR с immutable tag `sha-8a69f78`.
- Discord bot подключён к production guild.
- `/integration-status`: PostgreSQL, Foundry Relay и Foundry доступны; Foundry обнаруживает 18 Actor.
- Фоновая Foundry sync включена с интервалом 5 минут.
- `GET /livez` и `GET /readyz` вернули HTTP 200 `ok`.
- Владелец подтвердил работоспособность бота; пользовательская обратная связь ещё ожидается.
- Backup script включён в deployment bundle, но успешный backup/restore drill ещё не подтверждён.

### Изоляция и безопасность

Не трогать существующий Compose project `jienexyz`, его PostgreSQL volume/данные, Caddy, Redis, `catalog-container` и `rosochkinsad.ru`. Не удалять volumes и не выполнять destructive DB operations. Не просить/не публиковать значения `.env.prod`, токенов и ключей. Не менять firewall или Caddy без отдельного запроса.

Сервер Ubuntu 24.04, 1 vCPU, около 1.9 GiB RAM и 19 GiB root disk; свободное место после установки образов было около 4.7 GB. Наблюдать RAM/OOM и диск.

## Актуальные задачи

Индекс: [tasks](../tasks/README.md); дорожная карта: [roadmap](../../docs/development/roadmap.md).

1. **TASK-014 — Production stabilization:** feedback, backup schedule, isolated restore drill, RAM/OOM/disk/log monitoring. Самый высокий приоритет.
2. **TASK-015 — Foundry Item search:** read-only Actor/Compendium search, filters, pagination and outage behavior.
3. **TASK-016 — NPC Discord webhooks:** GM-only, dynamic NPC name/avatar, channel allow-list, secure credentials and audit.
4. **TASK-004 — Characters/Foundry:** finish approval/rejection, ownership confirmation, slots, NPC lifecycle and sync retry/recovery.
5. **TASK-017 — Release automation:** Conventional Commit release notes, SemVer validation and `v0.2.0-beta`.
6. **TASK-018 — Deployment gate:** validate host, immutable image update, health checks and rollback. Automatic CD stays disabled until TASK-014/TASK-018 pass and owner explicitly approves.
7. **TASK-019 — GM Item add/remove:** sole Foundry write exception, only after security review and narrow adapter design.
8. Continue all product stages 0–11; vertical tasks do not close the full stages.

## Engineering process

- Conventional Commits 1.0.0 is mandatory; see `.agent/protocols/COMMIT_CONVENTIONS.md`.
- No PR/merge unless explicitly requested.
- Do not claim tests/build/CI passed without actual output or confirmed CI.
- Work only in `src/` for application code; docs and PAD remain outside `src/`.
- Before GitHub edits, verify branch HEAD and fetch current file SHA. Use separate Conventional Commit per file update per current project workflow.

## Known implementation constraints

- Preserve raw Actor snapshots, unknown fields, nulls and empty objects.
- Distinguish real `Actor.items[]` from Compendium references in `system.items`.
- Never use localized Item names as identifiers; use IDs, slugs, UUID and compendiumSource.
- Do not execute arbitrary PF2e Rule Elements.
- Do not create Actors automatically or infer ownership.
- Do not use Foundry sync token as user authentication.
- Never enable generic Relay write scopes or expose write credential to players.
- Never disturb unrelated host services or Production volumes.
