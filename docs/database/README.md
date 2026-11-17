# База данных

- **`schema.sql` — единственный полный исполняемый источник структуры и DB-инвариантов.** PostgreSQL 15+, расширение `btree_gist`.
- `schema.dbml` — генерируемая модель всех таблиц, колонок, PK, FK, обычных UNIQUE и значений по умолчанию. PostgreSQL-специфичные ограничения, динамические триггеры, функции и seed-данные остаются в SQL; не восстанавливать SQL из DBML.
- `er-diagram.mmd` — все таблицы и все физические FK, без приписывания связей полиморфным `*_type/id` и JSONB. Составной FK — одно ребро. Для читаемости показаны только ключевые колонки. Кардинальность many на стороне дочерней таблицы консервативная: отдельные UNIQUE могут сужать её до one.
- `tables.md` — полный каталог: каждая колонка, SQL `CREATE TABLE`, поздние `ALTER`, индексы и явные триггеры по таблице.
- `objects.md` — полный DDL остальных объектов (функции, домен, extension, динамические триггеры и seeds).
- `inventory.json` — машинный каталог; `design.md` — полное обоснование и аудит исходного автора.
- [migrations.md](migrations.md) — правила будущего внедрения; сейчас goose-миграций нет.

## Обновление производных

```bash
python3 -m pip install -r requirements-docs.txt
python3 scripts/generate_database_docs.py
python3 scripts/generate_database_docs.py --check
```

Генерация читает AST PostgreSQL, включая FK из поздних ALTER. Она не исполняет PL/pgSQL и не заменяет прогон на реальной БД. Рабочие SQL-тесты: [tests/database](../../tests/database/README.md).
