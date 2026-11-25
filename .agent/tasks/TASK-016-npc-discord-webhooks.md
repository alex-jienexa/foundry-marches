---
id: TASK-016
title: Send Discord messages as Foundry NPCs
status: ready
priority: high
assignee: unassigned
created_at: '2026-11-25'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-003
requirement_ids:
- FM-REQ-002
files_owned:
- src/backend/
- src/
- tests/
- docs/
labels:
- discord
- npc
- webhooks
- security
---

# Send Discord messages as Foundry NPCs

## Objective

Позволить Мастеру отправлять ролевые сообщения в Discord от имени выбранного НПС через Discord webhooks, не выдавая игрокам управление webhook credentials.

## Scope

- GM-only command/flow to select NPC, channel and message.
- Dynamic webhook display name and avatar sourced from selected NPC data.
- Explicit allow-list of guild channels.
- Secure webhook creation/storage/rotation and deletion lifecycle.
- Validation and audit for every send and management operation.
- Rate limits, message length limits and safe error handling.

## Security constraints

- Never accept arbitrary webhook URLs from users.
- Do not expose webhook tokens in API responses, logs, audit payloads or Discord messages.
- Re-check GM authorization on every operation.
- Restrict allowed guild and channel IDs.
- Escape/validate mentions and prevent unintended mass-pings.
- Treat this as Discord presentation only; it does not write to Foundry.

## Acceptance criteria

- [ ] Only authorized GM can create/manage/send NPC webhook messages.
- [ ] NPC name/avatar are resolved from trusted application/Foundry snapshot data.
- [ ] Sending is limited to configured allow-list channels.
- [ ] Credentials are encrypted/secret-managed and never returned to clients.
- [ ] Every send is audited with actor, NPC, channel and result metadata (without secrets).
- [ ] Tests cover permission denial, invalid channel, credential rotation, rate limit and mention abuse.
- [ ] Test channel smoke check is documented.

## Verification

Unit/security tests and owner-approved test-channel end-to-end demonstration.

## Progress log

- 2026-11-25 — Created from project owner roadmap.
