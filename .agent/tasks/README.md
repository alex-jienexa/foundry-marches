# Tasks

Tasks are bounded execution units. A task must be small enough to review and have a clear completion contract.

## Naming and lifecycle

- File name: `TASK-NNN-short-name.md`; IDs use the `TASK-NNN` form.
- IDs are stable and never reused.
- Lifecycle: `backlog`, `ready`, `in_progress`, `blocked`, `review`, `done`, `cancelled`.
- `blocked` can be entered from any non-terminal state; record the blocker and owner.
- A task is `done` only when acceptance criteria and required checks pass, or an explicitly approved exception is documented.
- Reopening a completed task requires a reason in the task history.

Copy `TASK_TEMPLATE.md` for new work. Use `created_at`, `updated_at`, `depends_on`, and `requirement_ids` as defined in `schemas/task.schema.yaml`. `TASK-001.md` is an example of a first project bootstrap task; adapt or replace it.

## FoundryMarches — задачи реализации

Импорт не означает реализацию. Этапы не сокращены; каждая задача содержит исходный состав и проверки этапа. TASK-002 и TASK-003 можно начать независимо; этап 2 требует обоих. Дальнейшая цепочка консервативная, не запрет параллелизма после явного пересмотра owned files и зависимостей.

| Этап | Задача | Состояние | Зависит от |
|---|---|---|---|
| 0. Спайк (1–2 дня) | [TASK-002](TASK-002.md) | in_progress | TASK-001 |
| 1. Основа | [TASK-003](TASK-003.md) | in_progress | TASK-001 |
| 2. Персонажи и Foundry | [TASK-004](TASK-004.md) | in_progress | TASK-002, TASK-003 |
| 3. Приключения | [TASK-005](TASK-005.md) | backlog | TASK-004 |
| 4. Награды и экономика | [TASK-006](TASK-006.md) | backlog | TASK-005 |
| 5. Wiki, Obsidian, Campaign Codex | [TASK-007](TASK-007.md) | backlog | TASK-006 |
| 6. Карты | [TASK-008](TASK-008.md) | backlog | TASK-007 |
| 7. Календарь и погода | [TASK-009](TASK-009.md) | backlog | TASK-008 |
| 8. Экспедиции | [TASK-010](TASK-010.md) | backlog | TASK-009 |
| 9. Фракции | [TASK-011](TASK-011.md) | backlog | TASK-010 |
| 10. Торговое предприятие | [TASK-012](TASK-012.md) | backlog | TASK-011 |
| 11. Расширенное | [TASK-013](TASK-013.md) | backlog | TASK-012 |

[TASK-001](TASK-001.md) — закрытый bootstrap: контекст и доступные проверки, не реализация основы приложения.
