---
id: FM-REQ-002
title: 1. Основа
type: functional
status: proposed
priority: must
created_at: '2026-11-17'
updated_at: '2026-11-17'
verification: test
owner: project-owner
related_tasks:
- TASK-003
---

# 1. Основа

## Statement

Реализовать весь состав этапа без сокращения: Go-каркас, PostgreSQL 15+, миграции, Discord OAuth, роли и разрешения, сервисный токен бота, аудит, Docker Compose, резервные копии

## Rationale and source

Полный [план этапов](../../docs/development/roadmap.md); [профильный полный раздел](../../docs/architecture/overview.md). Ссылки на сценарии исходного ТЗ сохранены, но само ТЗ отсутствует — не достраивать его по предположениям. [Открытые вопросы](../../docs/development/open-questions.md) решает project-owner перед зависимой реализацией.

## Acceptance criteria

- [ ] Проверки: `curl $API/readyz`; запрос игрока к `POST /api/settings` даёт 403; CI применяет `schema.sql` и `schema_tests.sql` на PG 15 и 16 и запускает скрипт гонок \`schema\_race\_tests.sh\`; восстановление копии в пустую БД
- [ ] Итог: вход через Discord, управление пользователями, чистая установка одной командой
- [ ] Для всех операций выполняются серверные permissions/CanSee, журналы/инварианты и безопасность из полных разделов.
- [ ] DB → API/curl → bot → UI, если применимо этапу; источники истины, контекст и тесты обновлены.

## Verification

Фактические команды и результаты фиксируются в задаче; текст плана не служит доказательством реализации.
