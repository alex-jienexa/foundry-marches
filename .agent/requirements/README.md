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

Статус этапных требований `proposed`: это перенос плана, не утверждение владельцем всех открытых решений. Все этапы 0–11 сохранены; основное ТЗ в архиве отсутствует.

| Этап | Требование | Задача |
|---|---|---|
| 0. Спайк (1–2 дня) | [FM-REQ-001](FM-REQ-001.md) | [TASK-002](../tasks/TASK-002.md) |
| 1. Основа | [FM-REQ-002](FM-REQ-002.md) | [TASK-003](../tasks/TASK-003.md) |
| 2. Персонажи и Foundry | [FM-REQ-003](FM-REQ-003.md) | [TASK-004](../tasks/TASK-004.md) |
| 3. Приключения | [FM-REQ-004](FM-REQ-004.md) | [TASK-005](../tasks/TASK-005.md) |
| 4. Награды и экономика | [FM-REQ-005](FM-REQ-005.md) | [TASK-006](../tasks/TASK-006.md) |
| 5. Wiki, Obsidian, Campaign Codex | [FM-REQ-006](FM-REQ-006.md) | [TASK-007](../tasks/TASK-007.md) |
| 6. Карты | [FM-REQ-007](FM-REQ-007.md) | [TASK-008](../tasks/TASK-008.md) |
| 7. Календарь и погода | [FM-REQ-008](FM-REQ-008.md) | [TASK-009](../tasks/TASK-009.md) |
| 8. Экспедиции | [FM-REQ-009](FM-REQ-009.md) | [TASK-010](../tasks/TASK-010.md) |
| 9. Фракции | [FM-REQ-010](FM-REQ-010.md) | [TASK-011](../tasks/TASK-011.md) |
| 10. Торговое предприятие | [FM-REQ-011](FM-REQ-011.md) | [TASK-012](../tasks/TASK-012.md) |
| 11. Расширенное | [FM-REQ-012](FM-REQ-012.md) | [TASK-013](../tasks/TASK-013.md) |

[FM-REQ-013](FM-REQ-013.md) — сохранность артефактов; [PAD-REQ-001](PAD-REQ-001.md) — исходное требование PAD.
