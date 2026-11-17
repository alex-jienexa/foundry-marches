# База данных — навигационная сводка

**Авторитетный DDL:** [docs/database/schema.sql](../../docs/database/schema.sql), PostgreSQL 15+, `btree_gist`; 115 таблиц. Это baseline на чистую БД, не набор goose-миграций.

- [Все таблицы, колонки, ограничения, индексы и явные триггеры](../../docs/database/tables.md).
- [Функции, динамические триггеры, seeds](../../docs/database/objects.md).
- [Полное обоснование и аудит v0.3](../../docs/database/design.md).
- [DBML, ER, JSON и ограничения проекций](../../docs/database/README.md).
- [Миграционная политика](../../docs/database/migrations.md).
- [166 функциональных + 9 конкурентных проверок](../../tests/database/README.md), [результаты PG15/16](../../docs/verification.md).

Критические инварианты: неизменяемые журналы/сторно, balanced ledger, Foundry-owned зеркала, автоматы, лимиты и блокировки заявок, sealed calendar, составные FK карты/экспедиции, GM-подтверждение сюжетных действий. DB не заменяет session identity, permissions/CanSee, выбор кандидата, календарные преобразования и travel math приложения. `SET LOCAL app.*` устанавливает доверенный backend в транзакции.

Менять SQL → тесты → регенерировать проекции → `--check`. Оригиналы не редактировать, не делать round-trip из DBML. Не запускать fixtures и race-тесты на реальной кампании.
