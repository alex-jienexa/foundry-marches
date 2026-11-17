# Архитектура — навигационная сводка

Модульный монолит Go: API и worker — один бинарь с разными командами; отдельный Discord-бот, который ходит только в API. PostgreSQL — состояние кампании и очереди; relay имеет отдельную служебную SQLite. SolidJS UI через Caddy. Foundry удалённый; WebDAV уже существует.

- [Полная архитектура и модули](../../docs/architecture/overview.md), [components](../../docs/architecture/components.md).
- [Foundry](../../docs/architecture/foundry.md): Foundry владеет листом, приложение — историей/намерениями; абсолютные записи, предусловия, operation_id, read-back, `external_adjustment`, fallback Foundry Bridge.
- `integration_jobs` — истина, River — исполнитель; `event_outbox` — durable события, NOTIFY только будит. Никакого Redis по умолчанию.
- [Obsidian/Codex](../../docs/architecture/obsidian.md): БД-посредник, rev/hash, ETag/If-Match, конфликты без затирания, сначала Codex read-only.
- [Экспедиции](../../docs/architecture/expeditions.md), [фракции](../../docs/architecture/factions.md), [автоматы](../../docs/architecture/states.md).
- [Диаграммы](../../docs/README.md#диаграммы-проектное-решение-не-запущенная-система).

Полные правила видимости, SSRF, OAuth/PKCE/CSRF, журналирования и транзакционности — в overview, не в этой сводке. Ни один процесс ещё не реализован. [Неопределённости](../../docs/development/source-gaps.md).
