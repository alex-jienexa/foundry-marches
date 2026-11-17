# FoundryMarches — документация

Статус: **спецификация и проверенная SQL-модель; код приложения ещё не создан**. Импорт не является реализацией этапов 0–11.

## По роли и задаче

| Задача | Полный источник |
|---|---|
| Общая архитектура, стек, процессы, безопасность | [overview.md](architecture/overview.md) |
| Границы модулей | [components.md](architecture/components.md) |
| Foundry, владение данными, идемпотентность, Спайк 0 | [foundry.md](architecture/foundry.md) |
| Obsidian, WebDAV, Codex, конфликты | [obsidian.md](architecture/obsidian.md) |
| Риски исходного ТЗ | [review.md](architecture/review.md) |
| Конечные автоматы | [states.md](architecture/states.md) |
| Экспедиции | [expeditions.md](architecture/expeditions.md) |
| Фракции | [factions.md](architecture/factions.md) |
| Обоснование БД, аудит и разделение DB/application | [database/design.md](database/design.md) |
| Исполняемая схема | [database/schema.sql](database/schema.sql) |
| Все таблицы и DDL | [database/tables.md](database/tables.md) |
| DBML, ER и их ограничения | [database/README.md](database/README.md) |
| Миграции | [database/migrations.md](database/migrations.md) |
| Полный план 0–11 | [development/roadmap.md](development/roadmap.md) |
| Контрольные точки | [development/milestones.md](development/milestones.md) |
| Рабочий процесс | [development/workflow.md](development/workflow.md) |
| Нерешённые вопросы владельцу | [development/open-questions.md](development/open-questions.md) |
| Доступность ТЗ и расхождения источников | [development/source-gaps.md](development/source-gaps.md) |
| API: что определено, а что нет | [api/README.md](api/README.md) |
| Оригиналы, хэши и карта переноса | [reference/README.md](reference/README.md) |
| Анализ PAD и внесённые изменения | [repository-analysis.md](repository-analysis.md) |
| Реальные результаты проверок | [verification.md](verification.md) |

## Диаграммы (проектное решение, не запущенная система)

[Системная](architecture/diagrams/system.mmd) · [Развёртывание](architecture/diagrams/deployment.mmd) · [Операция Foundry](architecture/diagrams/sequence.mmd) · [ER](database/er-diagram.mmd).

## Минимальная проверка

```bash
python3 -m pip install -r requirements-docs.txt
python3 scripts/validate_artifacts.py
python3 scripts/generate_database_docs.py --check
```

Для реального выполнения DDL и SQL-тестов нужен PostgreSQL 15 или 16 с `btree_gist`; [команды](../tests/database/README.md). Команд сборки Go/SolidJS, Docker Compose, мигратора и бота пока нет — это будущая реализация.
