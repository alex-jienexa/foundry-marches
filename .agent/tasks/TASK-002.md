---
id: TASK-002
title: 0. Спайк (1–2 дня)
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
- FM-REQ-001
files_owned:
- docs/architecture/decisions/spike-0-results.md
- .agent/decisions/
labels:
- stage-0
- planned
---

# 0. Спайк (1–2 дня)

## Objective

Реализовать полный этап 0 из [дорожной карты](../../docs/development/roadmap.md), не сокращая его до каркаса.

## Scope

relay у вас на хосте; чтение листа PF2e; монеты как предметы; запись XP с флагом операции; повышение уровня тестовому актору; хуки SSE; безголовая сессия; лимит self-hosted relay; ETag и `If-Match` на вашем WebDAV; выгрузка журналов Codex

## Sources and preconditions

- [FM-REQ-001](../requirements/FM-REQ-001.md), [полный раздел](../../docs/architecture/foundry.md), [SQL](../../docs/database/schema.sql).
- [Нерешённые вопросы](../../docs/development/open-questions.md), [пробелы исходников](../../docs/development/source-gaps.md).
- Внешние доступы/изменения в Foundry/WebDAV и деплой требуют отдельного разрешения. Для спайка нужны тестовые Actor/мир и заметки; текущий импорт таких разрешений не даёт.
- Пути files_owned для будущего кода предварительные; перед параллельной работой разбить на дочерние задачи с непересекающимся владением файлами.

## Acceptance criteria

- [ ] Состав этапа выполнен полностью, либо оставшаяся часть явно блокирует его закрытие.
- [ ] Проверки: пункты из 2.6, 2.9, 3.2, 3.6; для каждого — результат и вывод
- [ ] Готово, когда: есть решение «relay или собственный модуль» и образцы данных Codex
- [ ] API/бот/UI, миграции, тесты и документация согласованы в порядке исходного плана, где применимо.
- [ ] Открытые продуктовые решения не подменены предположениями; подтверждения записаны.

## Verification

Привести команды, версии и фактические результаты. Точные команды runtime будут определены при реализации; не копировать ещё несуществующий Compose/build как выполненный.

## Progress log

- 2026-11-17 — Создана из полного этапа v0.3. Реализация не начата.
