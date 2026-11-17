# Проверка импорта — 2026-11-17

## Что не проверено / не реализовано

Нет кода Go/SolidJS/бота, runtime Compose, goose-миграций и действующего OpenAPI. Не запускались production, live Foundry/PF2e/relay/WebDAV/Codex/Discord и remote GitHub Actions. Версии/тарифы/ссылки внешних интеграций из исходного плана отдельно не переисследовались. Отсутствующее ТЗ v1.0 не восстановлено. Положительные результаты ниже относятся к **переносу артефактов и baseline SQL**, не к готовому приложению.

## Реальное исполнение PostgreSQL

| Среда | Чистое применение полного schema.sql | Функциональные проверки | Проверки гонок | Код завершения |
|---|---|---|---|---|
| PostgreSQL 15.20 | PASS | 166 / 166 | 9 / 9 | 0 |
| PostgreSQL 16.16 | PASS | 166 / 166 | 9 / 9 | 0 |

Прогоны выполнены на двух независимых пустых локальных кластерах с `btree_gist`. SQL и оба исходных тестовых файла использованы **без изменений**. Последовательно выполнялись команды (из корня репозитория, PGHOST/PGPORT/PGUSER/PGDATABASE указывали отдельную тестовую БД):

```bash
psql -X -v ON_ERROR_STOP=1 -f docs/database/schema.sql
psql -X -v ON_ERROR_STOP=1 -f tests/database/schema_tests.sql
bash tests/database/schema_race_tests.sh
```

Логи сохранены полностью:

- PG15: [DDL](verification/pg15-schema.log), [166 тестов](verification/pg15-tests.log), [гонки](verification/pg15-races.log).
- PG16: [DDL](verification/pg16-schema.log), [166 тестов](verification/pg16-tests.log), [гонки](verification/pg16-races.log).
- [Сверка объектов](verification/postgresql-catalog-counts.txt): 115 таблиц, 1030 колонок, 231 физический FK, 118 non-internal триггеров. SQL содержит 51 свою функцию; расширение btree_gist дополнительно устанавливает свои функции.

Это 3 конкурентных сценария / 9 assertions, а не 9 различных сценариев: одобрение персонажей, покупка последнего товара, подтверждение последнего места. Не считать набор исчерпывающим доказательством всех бизнес-инвариантов и всех гонок.

## Целостность и воспроизводимость

```bash
python3 -m pip install -r requirements-docs.txt
python3 scripts/validate_artifacts.py
python3 scripts/generate_database_docs.py --check
bash -n tests/database/schema_race_tests.sh
python3 -m py_compile scripts/validate_artifacts.py scripts/generate_database_docs.py
```

PASS: четыре оригинала совпадают с размерами/SHA-256, все фрагменты Markdown без сокращений присутствуют в рабочих документах, baseline SQL/тесты побайтово равны оригиналам, все 12 этапов представлены задачами, metadata/dependencies PAD валидны, локальные Markdown-ссылки ведут к существующим файлам. Проверка ссылок подтверждает файлы, не содержимое якорей и не доступность внешних URL.

Проекции воспроизводятся без diff. По AST: 115 таблиц, 1030 колонок, 231 FK — числа совпадают с реально созданной БД. 272 SQL-оператора, в том числе 51 функция, 50 явных CREATE TRIGGER, 28 явных CREATE INDEX, 21 ALTER TABLE, DO, домен, extension и 4 seed INSERT. Индексы и триггеры, созданные неявно/динамически, не сводятся к этим исходным счётчикам.

## Разбор диаграмм

Node.js 22.23.2, зависимости закреплены в tools/docs/package-lock.json:

```bash
npm ci --prefix tools/docs --no-audit --no-fund
npm run check --prefix tools/docs
```

PASS: DBML-парсер принял 115 таблиц, 1030 колонок и 231 связь, Mermaid принял system, deployment, sequence и er-diagram. Это синтаксическая проверка; общая ER на 115 таблиц может быть слишком крупной для одного экрана. Для работы по таблицам предусмотрены tables.md/inventory.json. DBML/ER не заменяют SQL-инварианты.

## Перенос в Git

Пакет подготовлен относительно commit `3032d2da61c090bde83887f564f9487d1e99e27e`. Проверка `git apply --check` и фактическое применение patch выполнены на отдельной чистой копии этого snapshot; файлы сверены с подготовленным overlay. Коммит, push и deploy не выполнялись.
