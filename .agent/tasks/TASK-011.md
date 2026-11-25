---
id: TASK-011
title: 9. Фракции
status: backlog
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-010
requirement_ids:
- FM-REQ-010
files_owned:
- src/
- tests/
- docs/
- .github/workflows/ci.yml
labels:
- stage-9
- planned
---

# 9. Фракции

## Objective

Реализовать репутацию, отношения, территории и политические события как application-owned campaign state.

## Scope

Репутация, ступени, отношения, территории, политические события, журнал изменений и reversal entries в БД приложения.

## Foundry boundary

Все фракционные значения принадлежат приложению, если отдельным решением не определено иное. «Обратная запись» означает корректирующую проводку в журнале приложения, а не запись в Foundry. Никаких Foundry Actor writes.

## Acceptance criteria

- [ ] Значение репутации равно сумме append-only журнала.
- [ ] Корректировка создаёт reversal/adjustment entry, не изменяя историю.
- [ ] Применение события пишет историю.
- [ ] Игрок не видит скрытые фракционные данные.
- [ ] Репутация влияет на доступность товаров, а территории отображаются на карте.
- [ ] Все операции проходят server-side authorization и audit.

## Verification

Ledger consistency, reversal, visibility and event-history tests.
