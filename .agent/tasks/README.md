# Tasks

Tasks are bounded execution units. A task must be small enough to review and have a clear completion contract.

## Naming and lifecycle

- File name: `TASK-NNN-short-name.md`; IDs use the `TASK-NNN` form.
- IDs are stable and never reused.
- Lifecycle: `backlog`, `ready`, `in_progress`, `blocked`, `review`, `done`, `cancelled`.
- `blocked` can be entered from any non-terminal state; record the blocker and owner.
- A task is `done` only when acceptance criteria and required checks pass, or an explicitly approved exception is documented.
- Reopening a completed task requires a reason in the task history.

Copy `TASK_TEMPLATE.md` for new work. Use `created_at`, `updated_at`, `depends_on`, and `requirement_ids` as defined in `schemas/task.schema.yaml`.

## Mandatory Foundry boundary

[ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md) is binding for every task:
- Foundry is the only source of truth for Actor sheet data and PF2e mechanics.
- Relay access is read-only by default. Do not write XP, level, currency, attributes, abilities, ownership, permissions or arbitrary Actor data.
- The only write exception is GM-authorized add/remove Item on an existing Actor, implemented only through the narrow scope in [TASK-019](TASK-019-gm-item-management.md).
- Never implement a generic Relay write proxy. Every Item mutation requires authorization, audit, idempotency and read-back verification.
- Application-owned campaign state remains writable in the application DB; this does not grant Foundry write authority.

## FoundryMarches — задачи реализации

Исходные этапы 0–11 сохранены. Небольшие вертикальные задачи могут выполняться раньше полного этапа, но не закрывают его автоматически.

| Этап / поток | Задача | Состояние | Зависит от |
|---|---|---|---|
| 0. Спайк Relay | [TASK-002](TASK-002.md) | in_progress | TASK-001 |
| 1. Основа | [TASK-003](TASK-003.md) | in_progress | TASK-001 |
| 2. Персонажи / Foundry | [TASK-004](TASK-004.md) | in_progress | TASK-002, TASK-003 |
| 3. Приключения | [TASK-005](TASK-005.md) | backlog | TASK-004 |
| 4. Награды и экономика | [TASK-006](TASK-006.md) | backlog | TASK-005 |
| 5. Wiki / Obsidian / Codex | [TASK-007](TASK-007.md) | backlog | TASK-006 |
| 6. Карты | [TASK-008](TASK-008.md) | backlog | TASK-007 |
| 7. Календарь и погода | [TASK-009](TASK-009.md) | backlog | TASK-008 |
| 8. Экспедиции | [TASK-010](TASK-010.md) | backlog | TASK-009 |
| 9. Фракции | [TASK-011](TASK-011.md) | backlog | TASK-010 |
| 10. Торговое предприятие | [TASK-012](TASK-012.md) | backlog | TASK-011 |
| 11. Расширенное | [TASK-013](TASK-013.md) | backlog | TASK-012 |
| Production stabilization | [TASK-014](TASK-014-production-stabilization.md) | ready | TASK-003 |
| Foundry Item search (read-only) | [TASK-015](TASK-015-foundry-item-search.md) | ready | TASK-002, TASK-004 |
| NPC Discord webhooks | [TASK-016](TASK-016-npc-discord-webhooks.md) | ready | TASK-003 |
| GitHub Actions Release | [TASK-017](TASK-017-release-automation.md) | backlog | TASK-003 |
| Server validation / safe deployment | [TASK-018](TASK-018-production-deploy-gate.md) | backlog | TASK-014 |
| GM Item add/remove exception | [TASK-019](TASK-019-gm-item-management.md) | backlog | TASK-004, TASK-015 |

[TASK-001](TASK-001.md) — закрытый bootstrap: контекст и доступные проверки, не реализация основы приложения.

## Status interpretation

- Stages 0–2 remain `in_progress`; Production deployment is an operational milestone, not full stage acceptance.
- Stages 3–11 remain `backlog`.
- New tasks 014–019 are focused workstreams and do not replace the full stage acceptance criteria.
