---
id: TASK-014
title: Stabilize Production and prove backup recovery
status: ready
priority: critical
assignee: project-owner
created_at: '2026-11-25'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-003
requirement_ids:
- FM-REQ-002
files_owned:
- docs/deployment/
- docs/verification/
- scripts/
labels:
- production
- operations
---

# Stabilize Production and prove backup recovery

## Objective

Подтвердить, что текущий Production можно безопасно эксплуатировать и восстановить после сбоя, не затрагивая соседние сервисы.

## Context

Production запущен в отдельном Compose project `foundry-marches-prod`; API/Bot/PostgreSQL работают, Relay/Foundry доступны. Backup/restore drill ещё не подтверждён. Сервер ограничен примерно 1.9 GiB RAM и 19 GiB root disk.

## Scope

- Собрать обратную связь игроков и зарегистрировать дефекты.
- Проверить расписание и фактическое создание backup.
- Проверить целостность backup и выполнить restore только в отдельную тестовую БД/изолированную среду.
- Наблюдать RAM, OOM events, disk usage, Docker logs и health endpoints.
- Зафиксировать operational runbook и пороги реагирования.

## Safety constraints

Не трогать Compose project `jienexyz`, его volumes, Caddy, Redis, `catalog-container`, `rosochkinsad.ru`; не менять firewall; не удалять volumes; не выполнять destructive DB operations; не публиковать секреты.

## Acceptance criteria

- [ ] Игровая обратная связь собрана и заведён список проблем.
- [ ] Backup выполняется по расписанию и его артефакт проверяется.
- [ ] Restore успешно выполнен в отдельную пустую test DB; Production DB/volume не затрагивались.
- [ ] RAM/OOM/disk/log monitoring и предупреждения описаны и проверены.
- [ ] Runbook содержит backup, restore, диагностику и восстановление API/Bot.
- [ ] Нет вмешательства в перечисленные соседние сервисы.

## Verification

Предоставить команды и выводы владельцу; зафиксировать отчёт в `docs/verification/`. Не отмечать задачу done до фактического restore drill.

## Progress log

- 2026-11-25 — Создана по результатам Production deployment.
