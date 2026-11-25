---
id: TASK-019
title: Implement GM-authorized Foundry Item add/remove
status: backlog
priority: high
assignee: unassigned
created_at: '2026-11-25'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-004
- TASK-015
requirement_ids:
- FM-REQ-014
- FM-REQ-003
files_owned:
- src/backend/
- tests/
- docs/architecture/foundry.md
labels:
- foundry
- relay-write-exception
- gm-only
- security
---

# Implement GM-authorized Foundry Item add/remove

## Objective

Реализовать единственное разрешённое исключение из read-only Foundry boundary: Мастер может добавить или удалить конкретный Item на существующем Actor.

## Allowed operations

- Add an existing Item to an existing Actor.
- Remove an existing Item from an existing Actor.
- Item categories include equipment, abilities and other supported Foundry Items.
- No Actor creation/deletion, no arbitrary Actor update, no XP/level/currency/attribute/ownership writes.

## Required controls

- GM permission checked server-side on every request.
- Explicit target Actor and Item identity; use stable UUID/ID/slug/compendiumSource, never localized name alone.
- Validate world, Actor type, Item compatibility, quantity and source before mutation.
- Use a dedicated, narrowly scoped write credential; read-only credentials remain the default.
- No generic Relay URL/endpoint passthrough.
- Durable operation ID, idempotency and audit event; do not blindly retry uncertain outcomes.
- Read Actor back from Foundry and verify the intended Item state before reporting success.
- If Relay is unavailable or read-back is inconclusive, leave operation pending/needs-GM-review; never claim success from local DB mutation.
- Update application snapshot only from the read-back/sync result.

## Acceptance criteria

- [ ] GM can add a supported Item by stable identifier to a selected Actor.
- [ ] GM can remove a selected Item with confirmation and clear semantics for quantity/stacking.
- [ ] Non-GM and unlinked/unauthorized Actor requests are denied.
- [ ] Repeated request with same operation ID does not duplicate effects.
- [ ] Audit records actor, initiator, action, Item identity, operation ID and outcome without credentials.
- [ ] Read-back verifies final Actor Items; ambiguous outcomes require GM review.
- [ ] Tests prove XP, level, currency, attributes, ownership and arbitrary Actor updates cannot be written through this path.
- [ ] Relay outage/timeout does not cause unsafe retry or false success.
- [ ] Production write scopes are not enabled until security review and owner approval.

## Verification

Unit, authorization, idempotency and PostgreSQL tests; mocked Relay contract tests; isolated Foundry test-world integration; owner-approved production rollout after acceptance.

## Progress log

- 2026-11-25 — Created to encode the sole permitted Foundry write exception.
