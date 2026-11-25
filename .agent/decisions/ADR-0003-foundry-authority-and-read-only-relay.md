---
id: ADR-0003
title: Foundry is authoritative; Relay integration is read-only except GM item management
status: accepted
date: "2026-11-25"
deciders:
  - project-owner
related_requirements:
  - FM-REQ-014
related_tasks:
  - TASK-002
  - TASK-004
  - TASK-015
  - TASK-019
---

# ADR-0003: Foundry is authoritative; Relay integration is read-only except GM item management

## Context

Foundry VTT with PF2e owns the authoritative Actor sheet and all PF2e-derived character state. The application mirrors snapshots and provides campaign workflows, but must not become a competing writer for the Actor sheet. Earlier imported architecture text described broad Relay writes (XP, level, coins, ownership and arbitrary actor updates); that proposal conflicts with the project owner's current explicit direction and is superseded by this ADR.

## Decision

1. Foundry is the sole source of truth for Actor data and PF2e mechanics.
2. The application's Foundry integration is read-only by default. Relay is used to read Actors, Items, Compendium data, world structure required for supported reads, and change notifications where supported.
3. The application MUST NOT write XP, level, abilities, attributes, HP, AC, skills, class, ancestry, background, currency, ownership/permissions, actor identity, arbitrary Actor flags/system fields, or any other Actor state.
4. The only permitted write exception is GM-authorized addition or removal of an Item on a specific Actor. This covers equipment, abilities and other Foundry Items. It does not authorize arbitrary Actor updates or creation/deletion of Actors.
5. Item mutations must be explicit, narrow operations, addressed by stable Item UUID/ID/slug/source identifiers rather than localized display names. The implementation must validate the GM permission, target Actor, world, Item source and operation intent; record an audit event and idempotency key; avoid blind retries; and read the Actor back from Foundry before reporting success.
6. If Relay is unavailable, item mutations remain pending/failed safely and must not be simulated by mutating the application's mirror. The Foundry snapshot remains the observed truth.
7. Application-owned campaign data (adventure status, approvals, reputation, expedition state, shop stock, ledgers, etc.) remains writable in the application database according to its own authorization and audit rules. This does not make Foundry writable.
8. Relay credentials used for ordinary application reads must not include write scopes. If the item-management exception requires a write-capable credential, it must be separately scoped and protected; never expose it to players or generic API paths.

## Consequences

- Earlier proposed XP/level/currency/ownership write flows are removed from the implementation plan.
- External edits in Foundry are ingested through snapshots/sync and must not be overwritten by the application.
- Item management is a separate privileged capability and must not be generalized into a generic Relay proxy.
- Read-after-write verification, durable audit and idempotency are mandatory for the exception.
- Legacy architecture text below this decision is historical source material and must be interpreted through this ADR.

## Verification

- Code review and tests prove that no general Actor update/write operation is reachable from application flows.
- Tests prove that only GM-authorized Item add/remove operations can invoke Relay write methods.
- Integration tests verify idempotency, audit, read-back and safe behavior on Relay failure.

## Supersession

This ADR supersedes any conflicting proposed write behavior in the imported Foundry architecture plan. It does not delete the historical source text.
