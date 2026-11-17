# FoundryMarches

Русскоязычное приложение для West Marches-кампании PF2e: Foundry VTT, Obsidian/WebDAV, Campaign Codex и Discord. Цель — полный проект из приложенного архитектурного плана v0.3, без урезания до MVP.

**Сейчас:** PAD, полная документация, SQL baseline PostgreSQL (115 таблиц), функциональные и конкурентные тесты. **Ещё нет:** приложения, UI, бота, Compose, goose-миграций и действующего API. Целевой стек: Go, PostgreSQL 15+, SolidJS/TypeScript.

## С чего начать

- Человеку: [индекс документации](docs/README.md), [план 0–11](docs/development/roadmap.md), [вопросы](docs/development/open-questions.md).
- Агенту: [AGENTS.md](AGENTS.md) → [PAD](pad.yaml) → [.agent/README.md](.agent/README.md) → [контекст и источники](.agent/context/DOCUMENTATION.md).
- БД: [полный SQL](docs/database/schema.sql), [каталог](docs/database/tables.md), [ограничения DBML/ER](docs/database/README.md), [тесты](tests/database/README.md).
- Что изменилось в шаблоне: [анализ репозитория](docs/repository-analysis.md).
- Как проверить: [результаты и команды](docs/verification.md).
- Оригиналы без потерь: [происхождение и хэши](docs/reference/README.md).

## Быстрая проверка артефактов

```bash
python3 -m pip install -r requirements-docs.txt
python3 scripts/validate_artifacts.py
python3 scripts/generate_database_docs.py --check
```

Приложение пока не собирается: каркас и команды запуска создаются в TASK-003. Не запускать SQL-тесты на данных кампании.

## PAD

This project uses [PAD Template](https://github.com/alex-jienexa/pad_template) for persistent AI-assisted development context. See `.agent/` for project requirements, tasks, decisions, state, and agent protocols.
