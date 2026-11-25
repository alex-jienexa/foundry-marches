---
id: TASK-015
title: Implement read-only Foundry Item search
status: ready
priority: high
assignee: unassigned
created_at: '2026-11-25'
updated_at: '2026-11-25'
estimate: null
parent: TASK-004
depends_on:
- TASK-002
- TASK-004
requirement_ids:
- FM-REQ-003
- FM-REQ-014
files_owned:
- src/backend/
- src/frontend/
- tests/
- docs/
labels:
- foundry
- search
- read-only
---

# Implement read-only Foundry Item search

## Objective

Дать игрокам/Мастеру поиск предметов PF2e, используя Foundry как источник данных, не изменяя Actor и не создавая конкурирующий каталог.

## Scope

- Поиск по доступным Actor Items и Compendium Items.
- Фильтры, сортировка и пагинация.
- Отображение названия, traits, rarity, level, price и доступных системных данных при наличии.
- Различение реальных Actor Items и Compendium references.
- Кэширование с явной инвалидизацией при изменении версии PF2e/источника.
- Безопасная деградация при недоступности Relay.

## Constraints

- Только read-only Relay credentials и операции.
- Идентификаторы: Item UUID, `_id`, `system.slug`, `compendiumSource`; локализованные имена не являются ID.
- Не исполнять произвольные PF2e Rule Elements.
- Поиск не выдаёт предмет персонажу и не изменяет Foundry.

## Acceptance criteria

- [ ] Actor Items и Compendium Items различаются в API/DTO.
- [ ] Фильтры и пагинация имеют стабильный контракт и тесты.
- [ ] Ответы не теряют идентификаторы/источник Item.
- [ ] Relay outage возвращает контролируемую ошибку или stale cached result с явным признаком, без ложной свежести.
- [ ] Пользовательский поиск не вызывает Foundry write endpoint.
- [ ] Есть тесты на локализованные дубликаты, отсутствующие поля и неизвестные Item fields.

## Verification

Unit tests, Relay adapter tests with mocked responses, and end-to-end read-only search test. Проверить Relay credentials/scopes.

## Progress log

- 2026-11-25 — Создана как отдельный вертикальный поток поверх TASK-004.
