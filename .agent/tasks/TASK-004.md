---
id: TASK-004
title: 2. Персонажи и read-only интеграция Foundry
status: in_progress
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-002
- TASK-003
requirement_ids:
- FM-REQ-003
- FM-REQ-014
files_owned:
- src/backend/
- tests/
- docs/architecture/foundry.md
labels:
- stage-2
- characters
- foundry
- read-only
---

# 2. Персонажи и read-only интеграция Foundry

## Objective

Завершить жизненный цикл персонажа в приложении и надёжную синхронизацию Actor snapshots, не превращая приложение в альтернативный редактор листа PF2e.

## Binding architecture

[ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md) is mandatory. Foundry owns Actor sheet data and PF2e mechanics. The application may read/snapshot and calculate presentation from the snapshot, but must not write XP, level, coins, attributes, abilities, HP, AC, skills, class, ancestry, background, ownership or arbitrary Actor fields. GM Item add/remove is handled only by TASK-019.

## Scope

- FoundryGateway read interface and Relay read-only adapter.
- Lossless Actor snapshots, PF2e decoding/calculation and periodic/event-driven sync.
- Explicit Actor linking with authorization and confirmation.
- Character approval/rejection, owner confirmation and active-slot limits in application DB.
- NPC lifecycle as application-side status/association; do not mutate Foundry Actor permissions or data.
- Durable sync jobs, retry/recovery, idempotent snapshot reconciliation and operational visibility.
- Detect and record external Foundry changes in application history where relevant.

## Excluded

- XP/level progression writes or application-managed level-up.
- Currency/coin updates in Foundry.
- Actor owner/permission changes in Foundry.
- Generic Actor update, create, delete or arbitrary flags.
- Item add/remove operations (separate TASK-019).

## Acceptance criteria

- [x] Read-only Relay GET adapter and Actor snapshot persistence exist.
- [x] World sync and explicit Actor linking through Discord flow exist.
- [ ] Snapshot preservation tests cover unknown fields, nulls, empty objects and real Actor.items.
- [ ] Character approval/rejection and owner confirmation are implemented with audit and authorization.
- [ ] Character slot limits are enforced transactionally.
- [ ] NPC lifecycle is represented without writing to Foundry.
- [ ] Background sync jobs have retry, recovery, idempotency and observable states.
- [ ] Manual changes made in Foundry become visible in the application after sync and are never overwritten by the application.
- [ ] Forbidden Foundry writes (XP/level/coins/attributes/ownership/arbitrary Actor fields) are absent and covered by tests.
- [ ] Full stage verification is documented; the existence of deployed Production alone does not close this task.

## Verification

Unit and PostgreSQL integration tests for mapping, sync, linking, ownership, slot limits and job lifecycle. Relay integration tests must use read-only credentials. Inspect the full application for any generic write route or Relay write scope.

## Progress log

- 2026-11-18 — Actor snapshot persistence, Relay GET adapter, sync and explicit Discord linking implemented in slices.
- 2026-11-25 — Re-scoped to strict Foundry read-only integration. XP/level write requirements removed. Approval, ownership confirmation, slot limits, NPC lifecycle and robust job/retry remain open.
