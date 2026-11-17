---
id: TASK-007
title: 5. Wiki, Obsidian, Campaign Codex
status: backlog
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-17'
estimate: null
parent: null
depends_on:
- TASK-006
requirement_ids:
- FM-REQ-006
files_owned:
- src/
- tests/
- docs/
- .github/workflows/ci.yml
labels:
- stage-5
- planned
---

# 5. Wiki, Obsidian, Campaign Codex

## Objective

Реализовать полный этап 5 из [дорожной карты](../../docs/development/roadmap.md), не сокращая его до каркаса.

## Scope

Wiki с видимостью и версиями; WebDAV-клиент; журналы Foundry на чтение; затем запись

## Sources and preconditions

- [FM-REQ-006](../requirements/FM-REQ-006.md), [полный раздел](../../docs/architecture/obsidian.md), [SQL](../../docs/database/schema.sql).
- [Нерешённые вопросы](../../docs/development/open-questions.md), [пробелы исходников](../../docs/development/source-gaps.md).
- Внешние доступы/изменения в Foundry/WebDAV и деплой требуют отдельного разрешения. Для спайка нужны тестовые Actor/мир и заметки; текущий импорт таких разрешений не даёт.
- Пути files_owned для будущего кода предварительные; перед параллельной работой разбить на дочерние задачи с непересекающимся владением файлами.

## Acceptance criteria

- [ ] Состав этапа выполнен полностью, либо оставшаяся часть явно блокирует его закрытие.
- [ ] Проверки: `gm_only`-страница недоступна игроку (404); устаревший `rev` создаёт конфликт; правка в Obsidian появляется в Wiki
- [ ] Готово, когда: сценарий I из ТЗ; Codex → Wiki → Obsidian без потерь
- [ ] API/бот/UI, миграции, тесты и документация согласованы в порядке исходного плана, где применимо.
- [ ] Открытые продуктовые решения не подменены предположениями; подтверждения записаны.

## Verification

Привести команды, версии и фактические результаты. Точные команды runtime будут определены при реализации; не копировать ещё несуществующий Compose/build как выполненный.

## Progress log

- 2026-11-17 — Создана из полного этапа v0.3. Реализация не начата.
