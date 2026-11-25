---
id: FM-REQ-014
title: Foundry authority and restricted Relay write exception
type: constraint
status: approved
priority: must
created_at: '2026-11-25'
updated_at: '2026-11-25'
verification: inspection
owner: project-owner
related_tasks:
- TASK-002
- TASK-004
- TASK-015
- TASK-019
tags:
- foundry
- relay
- security
---

# Foundry authority and restricted Relay write exception

## Statement

Foundry VTT is the sole authoritative source for Actor sheet state and PF2e mechanics. Foundry integration through Relay is read-only except for explicit GM-authorized addition or removal of Items on an existing Actor.

## Rationale

The application must not maintain a competing version of the PF2e character sheet or overwrite changes made directly in Foundry. Item grants/removals are the one explicitly approved convenience operation for the Game Master.

## Acceptance criteria

- [ ] Actor snapshots are read from Foundry and preserve raw/unknown fields, nulls and empty objects.
- [ ] The application never writes XP, level, attributes, abilities, HP, AC, skills, class, ancestry, background, currency, ownership, permissions, actor fields/flags, or arbitrary Actor data.
- [ ] Application code has no generic Relay write/proxy operation.
- [ ] The only write-capable use case is adding/removing a Foundry Item on an existing Actor, initiated by a GM with the required permission.
- [ ] Item operations use stable identifiers, validate target Actor and Item, are auditable and idempotent, and verify the resulting Actor snapshot by reading it back.
- [ ] Relay read credentials are read-only; any credential required for the Item exception is separately scoped and never available to player-facing/general endpoints.
- [ ] Relay outages never cause local mirror mutation to be reported as a successful Foundry mutation.

## Verification

Static inspection, unit tests for authorization and operation routing, and integration tests for Item mutation idempotency/read-back. Tests must explicitly assert that forbidden Actor fields cannot be written.

## Notes

Decision: [ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md). This requirement is approved by the project owner on 2026-11-25; implementation and verification remain open.
