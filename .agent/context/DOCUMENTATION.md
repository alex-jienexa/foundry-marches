# Индекс контекста и полных источников

Сводки в этой папке экономят чтение при старте, но **не заменяют спецификацию**.

1. [PROJECT](PROJECT.md), [ARCHITECTURE](ARCHITECTURE.md), [DATABASE](DATABASE.md), [DEVELOPMENT](DEVELOPMENT.md).
2. **[Главный индекс всех разделов docs](../../docs/README.md)** — полные разделы исходника без сокращений.
3. [Исполняемый SQL](../../docs/database/schema.sql) → [таблицы](../../docs/database/tables.md) → [прочие SQL-объекты](../../docs/database/objects.md).
4. [Дорожная карта 0–11](../../docs/development/roadmap.md) → [этапные критерии](../../docs/development/milestones.md) → [task index](../tasks/README.md).
5. [Требования и трассировка](../requirements/README.md) → [ADR](../decisions/README.md).
6. [Десять вопросов](../../docs/development/open-questions.md) и [пробелы/несоответствия](../../docs/development/source-gaps.md).
7. [Оригиналы + SHA-256 + покрытие](../../docs/reference/README.md).
8. [Анализ репозитория](../../docs/repository-analysis.md) и [реальные проверки](../../docs/verification.md).

**Приоритет:** исполняемые DB-правила — SQL; продуктовый смысл — полная документация; неразрешённый конфликт — вопрос, не молчаливая правка. DBML/ER/сводки производны. `docs/reference/originals/` — исторические входные файлы, их не менять при разработке. `.pad/template-base/` — upstream baseline, не контекст текущего продукта.
