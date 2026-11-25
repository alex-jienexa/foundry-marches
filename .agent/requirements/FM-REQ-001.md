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

Подтвердить self-hosted Relay как безопасный и надёжный транспорт read-only доступа к Foundry Actor/PF2e данным, world sync и смежным read-only интеграциям. Foundry остаётся источником истины; Relay не используется для записи XP, уровня, валюты, характеристик, ownership или произвольных Actor полей.

## Rationale and source

Полный [план этапов](../../docs/development/roadmap.md), [Foundry architecture](../../docs/architecture/foundry.md), [ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md). Исходный документ содержит историческое предложение записи XP/level; оно отменено текущим решением владельца и не является acceptance criterion.

## Acceptance criteria

- [ ] Проверены Relay deployment/pairing, чтение PF2e Actor и real Actor Items.
- [ ] Проверено lossless сохранение snapshot и world sync.
- [ ] Проверены SSE/hooks, headless session и usage/quota либо задокументированы ограничения.
- [ ] Проверены WebDAV ETag/If-Match и доступные Campaign Codex samples.
- [ ] Используются read-only Relay credentials; тесты подтверждают отсутствие изменений Actor.
- [ ] Для всех проверок записаны команды, версии и фактические результаты.

## Verification

Только read-only Relay requests; сравнить Actor snapshots до/после. Не проводить тесты записи XP/level/coins.

## Notes

Единственное разрешённое исключение для будущего приложения — GM-authorized add/remove Item по ADR-0003; это отдельная задача TASK-019, не часть общего spike.
