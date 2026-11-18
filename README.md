# FoundryMarches

Русскоязычное приложение для West Marches-кампании PF2e: Foundry VTT, Obsidian/WebDAV, Campaign Codex и Discord. Целевой стек: Go, PostgreSQL 15+, SolidJS/TypeScript.

## С чего начать

- Человеку: [индекс документации](docs/README.md), [план 0–11](docs/development/roadmap.md), [вопросы](docs/development/open-questions.md).
- Агенту: [AGENTS.md](AGENTS.md) → [PAD](pad.yaml) → [.agent/README.md](.agent/README.md) → [контекст и источники](.agent/context/DOCUMENTATION.md).
- БД: [полный SQL](docs/database/schema.sql), [каталог](docs/database/tables.md), [ограничения DBML/ER](docs/database/README.md), [тесты](tests/database/README.md).
- Foundry REST API и запуск через Docker: [инструкция интеграции](docs/integrations/foundry-world-sync-api.md).
- Что изменилось в шаблоне: [анализ репозитория](docs/repository-analysis.md).
- Как проверить: [результаты и команды](docs/verification.md).
- Оригиналы без потерь: [происхождение и хэши](docs/reference/README.md).

## Быстрый запуск API и PostgreSQL

Требуются Docker Engine и Docker Compose plugin.

```sh
cp .env.example .env
# Заполнить .env: сгенерировать пароли и указать Relay Client ID/API key.
docker compose up -d --build
docker compose ps
curl -fsS http://127.0.0.1:8080/healthz
```

Compose автоматически инициализирует PostgreSQL схемой из `docs/database/schema.sql`, создаёт ограниченную роль приложения и начальную запись Foundry World. Данные хранятся в именованном volume `postgres_data`. Внимание: `docker compose down -v` удаляет этот volume и все данные в нём.

Полная инструкция, переменные окружения, запуск синхронизации и ручной запуск без Docker описаны в [документации API](docs/integrations/foundry-world-sync-api.md).

## Быстрая проверка артефактов

```bash
python3 -m pip install -r requirements-docs.txt
python3 scripts/validate_artifacts.py
python3 scripts/generate_database_docs.py --check
```

## PAD

This project uses [PAD Template](https://github.com/alex-jienexa/pad_template) for persistent AI-assisted development context. See `.agent/` for project requirements, tasks, decisions, state, and agent protocols.
