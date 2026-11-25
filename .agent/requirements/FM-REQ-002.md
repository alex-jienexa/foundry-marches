---
id: FM-REQ-002
title: 1. Основа приложения
type: functional
status: proposed
priority: must
created_at: '2026-11-17'
updated_at: '2026-11-25'
verification: test
owner: project-owner
related_tasks:
- TASK-003
---

# 1. Основа приложения

## Statement

Реализовать Go backend, PostgreSQL schema/migrations, Discord-first identity and authorization, отдельную service authentication для bot-to-API, аудит, Docker Compose, резервное копирование и воспроизводимое развёртывание.

## Rationale and source

Полный [план этапов](../../docs/development/roadmap.md), [overview](../../docs/architecture/overview.md). По решению владельца основной пользовательский интерфейс — Discord bot/guild; Discord OAuth и browser sessions не входят в текущий scope. Foundry Relay token не является пользовательской авторизацией.

## Acceptance criteria

- [ ] API health/readiness и CI database/backend checks проходят.
- [ ] Discord identity, registration, RBAC, owner bootstrap and revocation работают с проверками доступа на сервере.
- [ ] Bot-to-API service token отделён от Foundry Relay credentials.
- [ ] Privileged operations имеют полный audit coverage.
- [ ] Backup/restore проверен восстановлением в отдельную пустую test DB.
- [ ] Чистая установка и обновление описаны и воспроизводимы.
- [ ] Foundry integration соблюдает FM-REQ-014.

## Verification

CI и runtime проверки; отдельный backup/restore drill без воздействия на Production data.

## Notes

Discord OAuth/browser session из исходного импортированного плана заменён Discord-first bot authentication по решению владельца. Этапное требование остаётся proposed до полного acceptance.
