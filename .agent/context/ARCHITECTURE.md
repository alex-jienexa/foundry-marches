# Архитектура — навигационная сводка

Модульный монолит Go: API и worker — один бинарь с разными командами; отдельный Discord-бот, который ходит только в API. PostgreSQL — состояние кампании и очереди; Relay имеет отдельную служебную SQLite. SolidJS UI через Caddy. Foundry удалённый; WebDAV уже существует.

## Foundry boundary

[ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md) — обязательная граница: Foundry является единственным источником истины для Actor/PF2e. Relay integration read-only. Единственное исключение — GM-authorized add/remove Item для существующего Actor с audit, idempotency и read-back. Запрещены XP/level/coin/attribute/ownership/любые произвольные Actor writes. Полная спецификация: [Foundry architecture](../../docs/architecture/foundry.md).

- [Полная архитектура и модули](../../docs/architecture/overview.md), [components](../../docs/architecture/components.md).
- [Foundry](../../docs/architecture/foundry.md): Actor snapshots и PF2e данные читаются из Foundry; приложение владеет только собственными campaign workflows/state.
- `integration_jobs` — состояние задач интеграции; worker — исполнитель; `event_outbox` — durable события, NOTIFY только будит. Никакого Redis по умолчанию.
- [Obsidian/Codex](../../docs/architecture/obsidian.md): БД-посредник, rev/hash, ETag/If-Match, конфликты без затирания; Foundry journals остаются read-only.
- [Экспедиции](../../docs/architecture/expeditions.md), [фракции](../../docs/architecture/factions.md), [автоматы](../../docs/architecture/states.md).

Полные правила видимости, SSRF, OAuth/PKCE/CSRF, журналирования и транзакционности — в overview. Проверять фактическую реализацию по коду и тестам; сводка не доказывает завершение модулей. [Неопределённости](../../docs/development/source-gaps.md).
