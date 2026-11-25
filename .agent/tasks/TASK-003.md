---
id: TASK-003
title: 1. Основа приложения и Production operations
status: in_progress
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-001
requirement_ids:
- FM-REQ-002
files_owned:
- src/backend/
- src/frontend/
- tests/
- docs/
- .github/workflows/ci.yml
labels:
- stage-1
- foundation
- production
---

# 1. Основа приложения

## Objective

Завершить основу приложения: устойчивый Go API, PostgreSQL, Discord-first identity/authorization, аудит, миграции, контейнеризация, backup/restore и безопасная эксплуатация.

## Context

- Решение владельца: Discord bot/guild — основной пользовательский интерфейс; browser Discord OAuth и browser sessions не входят в текущую архитектуру.
- Foundry credentials и Relay token не являются пользовательской авторизацией.
- Foundry authority boundary: [ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md).
- Production уже развёрнут, но operational milestone не равен закрытию полного этапа.

## Scope

- Go backend/API, PostgreSQL schema/migrations and integration tests.
- Discord Gateway bot, registration, owner bootstrap, RBAC and revocation.
- Separate bot-to-API service authentication.
- Audit coverage for privileged actions and security-sensitive workflows.
- Docker Compose and reproducible clean installation.
- Production backup schedule, backup verification and restore drill.
- Resource/log monitoring and documented operational procedures.

## Excluded

- Discord OAuth/browser sessions unless separately approved.
- Any Foundry writes except the narrow GM Item add/remove exception in ADR-0003 / TASK-019.
- Automatic Production CD; gated by TASK-018 and explicit owner approval.

## Acceptance criteria

- [x] API, PostgreSQL, Discord bot and separate internal service authentication are deployed.
- [x] Production API health endpoints return HTTP 200; owner confirmed bot operation.
- [ ] RBAC administration/revocation and all privileged routes have complete authorization tests.
- [ ] Audit coverage is complete for privileged operations.
- [ ] Backup schedule is active and a restore into a separate empty test database is demonstrated.
- [ ] Clean installation and migrations are reproducible from documented commands.
- [ ] Production resource/log monitoring and incident/rollback runbook are verified.
- [ ] No Foundry write path exists outside the explicit Item exception.

## Verification

- CI: backend unit/integration tests, gofmt, go vet, database schema tests.
- Runtime: health/readiness, bot registration and permission tests.
- Operations: create backup, validate it, restore to isolated test DB and verify application data; do not touch Production data or volumes during the drill.

## Progress log

- 2026-11-18 — Go API, PostgreSQL, Discord Gateway bot, internal service token, registration and baseline RBAC implemented.
- 2026-11-25 — Production API/Bot/PostgreSQL deployed in isolated Compose project; /livez and /readyz HTTP 200; Relay/Foundry reachable. Backup/restore and full audit acceptance remain open.
