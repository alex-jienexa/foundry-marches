# Requirements

Requirements describe stable product behavior or quality constraints. Keep them implementation-independent where practical.

## Naming and lifecycle

- File name: `PAD-REQ-NNN-short-name.md`.
- IDs are stable and never reused.
- Lifecycle: `proposed`, `approved`, `implemented`, `verified`, `deprecated`.
- Each requirement has observable acceptance criteria and a verification method.
- Link tasks that implement or verify the requirement; do not mark a requirement verified solely because code exists.

Copy `REQUIREMENT_TEMPLATE.md` for each new requirement. Requirement IDs use the `PAD-REQ-NNN` form; priorities are `must`, `should`, `could`, or `wont`. The template is intentionally not a requirement record.

## FoundryMarches — трассировка полного плана

Этапные требования FM-REQ-001–013 сохраняют статус `proposed`: это перенос исходного плана, а не утверждение владельцем всех открытых решений. Их формулировки обновлены с учётом действующей Foundry boundary. [FM-REQ-014](FM-REQ-014.md) — отдельное утверждённое архитектурное ограничение.

| Этап / ограничение | Требование | Задача |
|---|---|---|
| 0. Спайк Relay | [FM-REQ-001](FM-REQ-001.md) | [TASK-002](../tasks/TASK-002.md) |
| 1. Основа | [FM-REQ-002](FM-REQ-002.md) | [TASK-003](../tasks/TASK-003.md) |
| 2. Персонажи и Foundry | [FM-REQ-003](FM-REQ-003.md) | [TASK-004](../tasks/TASK-004.md) |
| 3. Приключения | FM-REQ-004 | TASK-005 |
| 4. Награды и экономика | FM-REQ-005 | TASK-006 |
| 5. Wiki, Obsidian, Campaign Codex | FM-REQ-006 | TASK-007 |
| 6. Карты | FM-REQ-007 | TASK-008 |
| 7. Календарь и погода | FM-REQ-008 | TASK-009 |
| 8. Экспедиции | FM-REQ-009 | TASK-010 |
| 9. Фракции | FM-REQ-010 | TASK-011 |
| 10. Торговое предприятие | FM-REQ-011 | TASK-012 |
| 11. Расширенное | FM-REQ-012 | TASK-013 |
| Foundry authority / restricted Relay writes | [FM-REQ-014](FM-REQ-014.md) | [TASK-004](../tasks/TASK-004.md), [TASK-019](../tasks/TASK-019-gm-item-management.md) |

[FM-REQ-013](FM-REQ-013.md) — сохранность артефактов; [PAD-REQ-001](PAD-REQ-001.md) — исходное требование PAD.
