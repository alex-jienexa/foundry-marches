---
id: TASK-003
title: 1. Основа
status: ready
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-17'
estimate: null
parent: null
depends_on:
- TASK-001
requirement_ids:
- FM-REQ-002
files_owned:
- src/
- tests/
- docs/
- .github/workflows/ci.yml
labels:
- stage-1
- planned
---

# 1. Основа

## Objective

Реализовать полный этап 1 из [дорожной карты](../../docs/development/roadmap.md), не сокращая его до каркаса.

## Scope

Go-каркас, PostgreSQL 15+, миграции, Discord OAuth, роли и разрешения, сервисный токен бота, аудит, Docker Compose, резервные копии

## Sources and preconditions

- [FM-REQ-002](../requirements/FM-REQ-002.md), [полный раздел](../../docs/architecture/overview.md), [SQL](../../docs/database/schema.sql).
- [Нерешённые вопросы](../../docs/development/open-questions.md), [пробелы исходников](../../docs/development/source-gaps.md).
- Внешние доступы/изменения в Foundry/WebDAV и деплой требуют отдельного разрешения. Для спайка нужны тестовые Actor/мир и заметки; текущий импорт таких разрешений не даёт.
- Пути files_owned для будущего кода предварительные; перед параллельной работой разбить на дочерние задачи с непересекающимся владением файлами.

## Acceptance criteria

- [ ] Состав этапа выполнен полностью, либо оставшаяся часть явно блокирует его закрытие.
- [ ] Проверки: `curl $API/readyz`; запрос игрока к `POST /api/settings` даёт 403; CI применяет `schema.sql` и `schema_tests.sql` на PG 15 и 16 и запускает скрипт гонок \`schema\_race\_tests.sh\`; восстановление копии в пустую БД
- [ ] Готово, когда: вход через Discord, управление пользователями, чистая установка одной командой
- [ ] API/бот/UI, миграции, тесты и документация согласованы в порядке исходного плана, где применимо.
- [ ] Открытые продуктовые решения не подменены предположениями; подтверждения записаны.

## Verification

Привести команды, версии и фактические результаты. Точные команды runtime будут определены при реализации; не копировать ещё несуществующий Compose/build как выполненный.

## Progress log

- 2026-11-17 — Создана из полного этапа v0.3. Реализация не начата.
