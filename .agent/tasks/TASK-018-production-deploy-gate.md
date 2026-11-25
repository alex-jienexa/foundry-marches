---
id: TASK-018
title: Validate Production host and safe update/rollback procedure
status: backlog
priority: high
assignee: project-owner
created_at: '2026-11-25'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-014
requirement_ids:
- FM-REQ-002
files_owned:
- docs/deployment/
- docs/verification/
- .github/workflows/
labels:
- production
- deployment
- rollback
- gate
---

# Validate Production host and safe update/rollback procedure

## Objective

Проверить пригодность нового сервера и доказать безопасный процесс обновления и отката до включения автоматического Production CD.

## Scope

- Validate host capacity, disk/RAM headroom, Docker/Compose, network isolation and secret handling.
- Define immutable image reference by SHA/digest.
- Deploy a candidate version manually with pre/post health checks.
- Preserve previous known-good image/config and document rollback.
- Verify that failure during rollout does not affect existing unrelated services.
- Prepare CD workflow design but keep it disabled until owner approval.

## Hard gate

Automatic Production CD MUST remain disabled until TASK-014 backup/restore and this task's server/update/rollback acceptance are complete and the project owner explicitly approves enabling it.

## Acceptance criteria

- [ ] Server capacity and isolation reviewed with actual measurements.
- [ ] Backup/restore from TASK-014 passed.
- [ ] Manual immutable-image update procedure demonstrated.
- [ ] Health/readiness checks and rollback to previous image demonstrated.
- [ ] Existing `jienexyz` services, Caddy, Redis and unrelated containers remain untouched.
- [ ] Owner explicitly approves the CD gate before any workflow can deploy automatically.

## Verification

A recorded manual deployment/rollback drill and owner sign-off. Until then status remains backlog/blocked and no CD workflow is activated.

## Progress log

- 2026-11-25 — Created as mandatory prerequisite for automatic Production deployment.
