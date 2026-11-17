---
id: TASK-012
title: 10. Торговое предприятие
status: backlog
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-17'
estimate: null
parent: null
depends_on:
- TASK-011
requirement_ids:
- FM-REQ-011
files_owned:
- src/
- tests/
- docs/
- .github/workflows/ci.yml
labels:
- stage-10
- planned
---

# 10. Торговое предприятие

## Objective

Реализовать полный этап 10 из [дорожной карты](../../docs/development/roadmap.md), не сокращая его до каркаса.

## Scope

карточка, склад, персонал, события, действия, правила автоматики

## Sources and preconditions

- [FM-REQ-011](../requirements/FM-REQ-011.md), [полный раздел](../../docs/architecture/overview.md), [SQL](../../docs/database/schema.sql).
- [Нерешённые вопросы](../../docs/development/open-questions.md), [пробелы исходников](../../docs/development/source-gaps.md).
- Внешние доступы/изменения в Foundry/WebDAV и деплой требуют отдельного разрешения. Для спайка нужны тестовые Actor/мир и заметки; текущий импорт таких разрешений не даёт.
- Пути files_owned для будущего кода предварительные; перед параллельной работой разбить на дочерние задачи с непересекающимся владением файлами.

## Acceptance criteria

- [ ] Состав этапа выполнен полностью, либо оставшаяся часть явно блокирует его закрытие.
- [ ] Проверки: правило с `requires_gm_approval = false` отвергается БД
- [ ] Готово, когда: сценарий H из ТЗ
- [ ] API/бот/UI, миграции, тесты и документация согласованы в порядке исходного плана, где применимо.
- [ ] Открытые продуктовые решения не подменены предположениями; подтверждения записаны.

## Verification

Привести команды, версии и фактические результаты. Точные команды runtime будут определены при реализации; не копировать ещё несуществующий Compose/build как выполненный.

## Progress log

- 2026-11-17 — Создана из полного этапа v0.3. Реализация не начата.
