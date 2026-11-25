---
id: TASK-002
title: 0. Спайк интеграции Foundry Relay
status: in_progress
priority: high
assignee: ChatGPT
created_at: '2026-11-17'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-001
requirement_ids:
- FM-REQ-001
- FM-REQ-014
files_owned:
- docs/architecture/decisions/spike-0-results.md
- docs/architecture/foundry.md
- .agent/decisions/
labels:
- stage-0
- foundry
- read-only
---

# 0. Спайк интеграции Foundry Relay

## Objective

Подтвердить надёжность self-hosted Relay как транспорта для чтения Foundry и синхронизации Actor snapshots. Спайк не исследует и не реализует общую запись в Foundry.

## Context and binding constraints

- Foundry является источником истины. Обязательная граница — [ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md).
- Relay используется read-only. XP, level, coins, attributes, ownership и произвольные Actor writes исключены.
- Единственное будущее исключение — GM-authorized add/remove Item, отдельная задача TASK-019; оно не является частью общего write API.
- Подробности и исторические результаты: [Foundry architecture](../../docs/architecture/foundry.md), [spike results](../../docs/architecture/decisions/spike-0-results.md).

## Scope

- Self-hosted Relay deployment, pairing и безопасная конфигурация.
- Чтение PF2e Actor snapshot, инвентаря и Compendium references.
- World sync и проверка сохранения raw snapshot без потери неизвестных полей.
- Проверка SSE/hooks, headless session stability и Relay usage/quota, если поддерживается текущей версией.
- Проверка WebDAV ETag/If-Match и получение образцов Campaign Codex — это отдельный read-only контур, без записи в Foundry.
- Зафиксировать точные ограничения Relay и подтверждённые версии.

## Excluded

- Любая запись XP, level, coins, abilities, attributes, HP, ownership или произвольных Actor fields.
- Общий Relay write proxy.
- Изменения Foundry Actor в рамках тестов.
- Запись документов/журналов в Foundry.

## Acceptance criteria

- [x] Self-hosted Relay развёрнут, Foundry paired, GET Actor smoke test пройден без изменения Actor.
- [x] World sync сохраняет Actor snapshots; обнаруженные Actors остаются snapshot-only до явной привязки.
- [ ] Проверена полнота PF2e snapshot, включая реальные Actor Items и ссылки Compendium.
- [ ] Проверены SSE/hooks и headless session либо документированы ограничения/отсутствие поддержки.
- [ ] Проверены self-hosted usage/quota и WebDAV ETag/If-Match.
- [ ] Получены и описаны образцы Campaign Codex, если доступ предоставлен.
- [ ] Все результаты имеют команды, версии и фактический вывод; нет неразрешённых Foundry writes.

## Verification

Использовать только read-only Relay requests и отдельные тестовые WebDAV операции. После проверок сверить Actor snapshots и подтвердить отсутствие изменений Actor.

## Progress log

- 2026-11-17 — Создана из полного этапа v0.3.
- 2026-11-17/18 — Self-hosted Relay 3.4.1 развёрнут; pairing и GET Actor smoke test подтверждены; world sync сохранил 11 Actor snapshots.
- 2026-11-25 — Политика Foundry read-only закреплена ADR-0003. Исторические XP/level write spike checks исключены как противоречащие новому решению. Оставшиеся read-only проверки перечислены выше.
