---
id: TASK-006
title: 4. Награды и экономика
status: backlog
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-005
requirement_ids:
- FM-REQ-005
files_owned:
- src/
- tests/
- docs/
- .github/workflows/ci.yml
labels:
- stage-4
- planned
---

# 4. Награды и экономика

## Objective

Реализовать экономику кампании и журнал проводок в приложении, сохраняя Foundry единственным источником истины для Actor sheet data.

## Scope

Валюты приложения, журнал, награды, магазины, бюджет сокровищ, каталог/выдача/удаление Foundry Items через строго ограниченный GM flow TASK-019.

## Foundry boundary

- XP и прочие PF2e поля читаются из Foundry; приложение не начисляет XP и не пишет валюту/монеты в Actor.
- Покупка/награда, требующая изменения инвентаря Foundry, завершается только после GM-authorized Item add operation и read-back. До подтверждения остаётся pending.
- Внешние изменения Foundry отражаются через sync; приложение не пытается «обратной проводкой» перезаписать Foundry.
- Валюты/репутация, принадлежащие приложению, ведутся отдельным журналом и не смешиваются с монетными Items Foundry.

## Acceptance criteria

- [ ] Валюты приложения имеют append-only ledger и корректные reversal entries внутри БД приложения.
- [ ] Покупка, требующая Foundry Item, остаётся pending до подтверждённого GM add/read-back.
- [ ] Недоступность Relay не приводит к ложному завершению операции или локальному изменению Foundry snapshot.
- [ ] Изменения XP/монет, сделанные вручную в Foundry, отображаются после sync и не перезаписываются приложением.
- [ ] Магазины, награды и бюджет сокровищ соответствуют полному исходному сценарию этапа.

## Verification

Параллельные ledger tests, retry/idempotency tests, Relay outage tests и интеграционный сценарий GM Item add/read-back. Полный stage acceptance остаётся обязательным.
