# Разработка — порядок и состояние

Проект находится в активной реализации. Полный [план](../../docs/development/roadmap.md), [контрольные точки](../../docs/development/milestones.md), [workflow](../../docs/development/workflow.md), [фактические проверки](../../docs/verification.md).

## Текущее состояние

- TASK-002 (этап 0) — `in_progress`: self-hosted Relay 3.4.1 и Foundry read-only sync подтверждены; Production обнаруживает 18 Actors. На изолированном тестовом Relay 3.4.1 подтверждены headless login Foundry 14.367 (POST /join), загрузка игрового UI и регистрация одного клиента под Puffy. Полный headless lifecycle, SSE, production scopes и usage/quota ещё не закрыты; runbook замены Relay подготовлен, production migration не выполнялась.
- TASK-003 (этап 1) — `in_progress`: backend bootstrap, PostgreSQL/Compose, health endpoint и CI существуют; Discord OAuth, sessions, RBAC, audit, migrations, backup/restore не завершены.
- TASK-004 (этап 2) — `in_progress`: Relay adapter, PF2e decoder/calculator, persistence и sync реализованы частично. Все 11 Actor пока `snapshot_only`; следующий bounded slice — явная привязка к существующему персонажу.
- TASK-005…TASK-013 — backlog; этапы 3–11 не начаты.

Ни один из этапов 0–2 не объявлять завершённым, пока не выполнен весь scope и acceptance criteria.

## Порядок реализации

Внутри этапа по возможности: миграции/DB tests → API/curl → bot → UI. Технические задачи разрешается переставлять при соблюдении зависимостей и без потери исходных продуктовых требований. Перед параллельной работой определить непересекающиеся owned files.

## Критические правила

- Не изменять Foundry Actor: текущая интеграция read-only.
- Не создавать application characters и не назначать владельцев автоматически.
- WorldID приложения — UUID; не заменять Relay world identifier.
- Не выполнять destructive DB operations и не удалять Docker volumes.
- Не добавлять production deployment или CD без отдельного одобрения владельца.
- CI workflow уже существует: documentation, database (PostgreSQL 15/16) и backend jobs. Удалённые workflow runs для текущей ветки на момент проверки отсутствовали; не заявлять успешный CI run без ссылки/результата.
- Не помещать секреты в репозиторий, логи, документы или чат.

Текущие blockers и действия — только в `state/current.yaml` и единственном `state/HANDOFF.md`. Вопросы владельца: [open-questions](../../docs/development/open-questions.md); отсутствующее ТЗ: [source-gaps](../../docs/development/source-gaps.md).
