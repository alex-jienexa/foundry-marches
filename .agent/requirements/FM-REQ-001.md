---
id: FM-REQ-001
title: 0. Спайк интеграции Foundry Relay
type: functional
status: proposed
priority: must
created_at: '2026-11-17'
updated_at: '2026-11-25'
verification: test
owner: project-owner
related_tasks:
- TASK-002
---

# 0. Спайк интеграции Foundry Relay

## Statement

Подтвердить self-hosted Relay как безопасный и надёжный транспорт read-only доступа к Foundry Actor/PF2e данным, world sync и синхронизации Actor snapshots. Foundry остаётся единственным источником истины для Actor sheet data и PF2e mechanics. Relay не используется для записи в Actor или другие Foundry documents.

## Rationale and source

[Foundry architecture](../../docs/architecture/foundry.md), [ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md), [TASK-002](../tasks/TASK-002.md).

Историческое предложение записи XP/level и произвольных Actor fields отменено текущим решением владельца и не является частью требования. Единственное отдельное будущее исключение — GM-authorized add/remove Item по TASK-019; оно не расширяет обычные read-only credentials и не относится к этому требованию.

WebDAV, ETag/If-Match, Obsidian, Campaign Codex и Foundry journals относятся к TASK-007, а не к этому требованию.

## Acceptance criteria

- [ ] Self-hosted Relay deployment, pairing и REST access проверены.
- [ ] Read-only получение PF2e Actor и реальных Actor Items проверено.
- [ ] Raw snapshot сохраняется без потери неизвестных полей, null и пустых структур; world sync проверен.
- [ ] SSE/hooks проверены end-to-end либо задокументированы фактические ограничения.
- [ ] Headless session lifecycle проверен, если он требуется эксплуатационной модели; иначе задокументировано, почему не требуется.
- [ ] Global/per-minute limits, per-key limits и usage accounting проверены.
- [ ] Активный ключ приложения имеет только необходимые read scopes.
- [ ] Для всех проверок записаны версии, команды и фактические результаты; тесты не изменяют Foundry Actor.

## Verification

Read-only Relay requests; SSE event test triggered by manual edit in Foundry UI; inspect effective non-secret Relay configuration and API key scopes. Сверить Actor snapshots до/после. Не проводить тесты записи XP/level/coins.

## Notes

Foundry — authority. Единственное разрешённое исключение для будущего приложения — GM-authorized add/remove Item по ADR-0003, отдельная задача TASK-019.
