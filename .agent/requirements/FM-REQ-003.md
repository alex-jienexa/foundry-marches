---
id: FM-REQ-003
title: 2. Персонажи и read-only интеграция Foundry
type: functional
status: proposed
priority: must
created_at: '2026-11-17'
updated_at: '2026-11-25'
verification: test
owner: project-owner
related_tasks:
- TASK-004
- TASK-019
---

# 2. Персонажи и read-only интеграция Foundry

## Statement

Реализовать чтение и lossless snapshots Actor, синхронизацию, явную привязку Actor к персонажу, approval/ownership lifecycle приложения, лимиты персонажей и NPC lifecycle. Foundry остаётся единственным источником истины для листа PF2e; приложение не записывает в Actor XP, уровень, валюту, характеристики, ownership или произвольные поля.

Единственное разрешённое исключение — отдельная GM-authorized операция добавления/удаления Item на существующем Actor согласно FM-REQ-014.

## Rationale and source

[Roadmap](../../docs/development/roadmap.md), [Foundry architecture](../../docs/architecture/foundry.md), [ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md).

## Acceptance criteria

- [ ] Actor snapshots и реальные Actor.items синхронизируются без потери неизвестных полей, null и пустых объектов.
- [ ] Привязка Actor и approval/ownership lifecycle защищены серверной авторизацией и аудитом.
- [ ] Лимиты персонажей enforced transactionally.
- [ ] NPC lifecycle не требует изменения Foundry Actor.
- [ ] Background sync имеет retry/recovery и не перезаписывает Foundry.
- [ ] Изменения, сделанные вручную в Foundry, появляются в приложении после sync.
- [ ] Запись запрещённых Actor полей отсутствует и проверяется тестами.
- [ ] Add/remove Item разрешён только Мастеру по FM-REQ-014.

## Verification

Unit/integration tests, PostgreSQL tests, read-only Relay integration tests и отдельные security tests для GM Item operations.

## Notes

XP/level progression write и синхронизация уровня обратно в Foundry исключены из требования. Само этапное требование остаётся proposed до полного acceptance.
