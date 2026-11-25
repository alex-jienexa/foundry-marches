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

Подтвердить self-hosted Foundry Relay как надёжный и безопасный транспорт для read-only доступа к Foundry Actor/PF2e данным и синхронизации Actor snapshots. Задача фиксирует технические возможности и эксплуатационные ограничения Relay; она не реализует новый пользовательский функционал и не исследует интеграции с Wiki/Obsidian.

## Binding architecture and security constraints

- Foundry — единственный источник истины для Actor sheet data и PF2e mechanics.
- Relay и обычный ключ приложения используются только для чтения.
- Приложение не записывает XP, level, coins/currency, attributes, abilities, HP, AC, skills, class, ancestry, background, ownership/permissions, Actor flags или произвольные Actor fields.
- Общий Relay write proxy запрещён.
- Единственное отдельное будущее исключение — GM-authorized add/remove Item по ADR-0003 и TASK-019. Оно не входит в TASK-002.
- Не использовать write operations для проверки Relay в рамках этого spike.
- Ручные изменения в Foundry должны попадать в приложение через sync.

См. [ADR-0003](../decisions/ADR-0003-foundry-authority-and-read-only-relay.md), [FM-REQ-014](../requirements/FM-REQ-014.md) и [Foundry architecture](../../docs/architecture/foundry.md).

## Scope

- Self-hosted Relay deployment, pairing, HTTPS и REST connectivity.
- Read-only получение Actor и Compendium Item.
- Проверка world sync и сохранения raw Actor snapshot без потери неизвестных полей, null и пустых структур.
- Проверка наличия и структуры реальных Actor Items на фактическом экспорте.
- Проверка SSE/hooks и доставки событий (без Relay/Foundry writes).
- Проверка headless session lifecycle, если это нужно для текущей модели фоновой синхронизации; иначе зафиксировать, почему headless не требуется.
- Проверка фактических Relay request limits, per-key limits и usage accounting.
- Проверка, что активный ключ приложения имеет только необходимые read scopes.
- Документирование подтверждённых ограничений и операционной модели Relay.

## Explicitly out of scope

- WebDAV, ETag/If-Match, Obsidian и синхронизация Wiki.
- Campaign Codex и импорт/экспорт Foundry journals. Эти вопросы относятся к TASK-007.
- Любая запись XP, level, coins/currency, attributes, abilities, HP, AC, skills, ownership или произвольных Actor fields.
- Общий Relay write proxy, chat write, roll execution и произвольные entity writes.
- Реализация SSE-driven sync в приложении (если отдельно не поставлена задача).
- GM-authorized Item add/remove — отдельная TASK-019.

## Current evidence

Подтверждено по состоянию на аудит 2026-11-25:

- [x] Self-hosted Relay 3.4.1 развёрнут; HTTPS, pairing и REST access ранее подтверждены владельцем.
- [x] Actor GET реализован и проверен.
- [x] World sync реализован; текущая Production integration status показывает Foundry доступен и обнаруживает 18 Actors; фоновая синхронизация настроена на 5 минут.
- [x] Raw Actor snapshot сохраняется целиком; неизвестные поля и различия null / {} / [] сохраняются.
- [x] Фактический экспорт Actor от 2026-11-18 (docs/foundry/mxgc0DRWC8bA9ruI-20261118T073146Z-actor.json) содержит 16 реальных Actor Items разных типов, включая currency Items.
- [x] Actor Item decoder и сохранение stable identifiers реализованы и покрыты тестами.
- [x] Compendium references в Class/Ancestry и targeted ResolveCompendiumItem реализованы; resolver покрыт тестами.
- [x] Go Relay client предоставляет только GET/read методы; Relay write methods в клиенте отсутствуют.
- [x] Ограничение world actor listing в 500 результатов обнаруживается и возвращает ошибку вместо молчаливой неполной синхронизации.
- [x] Production smoke checks подтверждают PostgreSQL, Relay и Foundry availability; Foundry обнаружено 18 Actors.

Эти пункты подтверждают наличие реализации и/или ранее полученные результаты. Они не означают, что все оставшиеся runtime acceptance criteria закрыты в текущем проходе.

## Remaining acceptance criteria

- [ ] Проверить фактические scopes активного Production API key. Обычный ключ должен содержать только реально необходимые read scopes; удалить из него все write-capable scopes после безопасной замены ключа и проверки зависимостей.
- [ ] Проверить SSE end-to-end: подписка, ручное изменение disposable Actor через Foundry UI, получение события и восстановление подписки. Не изменять Actor через Relay.
- [ ] Проверить headless lifecycle (start/status/read/stop) только если он нужен для заявленной unattended-модели. Если не нужен — записать обоснование и подтверждённую текущую модель работы фоновой синхронизации.
- [ ] Проверить фактическую конфигурацию global/per-minute limits, per-key limits и usage counters без нагрузочного тестирования Production.
- [ ] Убедиться, что обычный Production ключ не имеет entity:write, chat:write, roll:execute, file:write, structure:write и иных write scopes.
- [ ] Обновить spike report фактическими версиями, безопасными командами и результатами проверок.
- [ ] Обновить статус FM-REQ-001 после прохождения acceptance criteria.

## Verification and safety

- Использовать только read-only Relay requests; событие SSE инициировать ручным редактированием disposable Actor в Foundry UI.
- Не просить владельца присылать API keys, passwords, cookies, .env или другие секреты.
- Не печатать секреты в диагностических командах. Для env inspection использовать allowlist конкретных несекретных переменных.
- Не выполнять нагрузочные тесты на Production.
- При замене API key сначала создать least-privilege replacement, обновить конфигурацию приложения, проверить /integration-status и только затем отозвать старый ключ.
- Не выполнять Relay writes; отдельный Item mutation path рассматривается только в TASK-019.

## Progress log

- 2026-11-17/18 — Self-hosted Relay 3.4.1 развёрнут; pairing, HTTPS и Actor GET подтверждены; ранняя world sync проверка сохранила 11 Actor snapshots.
- 2026-11-18 — Сохранён экспорт реального PF2e Actor с 16 Actor Items, включая currency Items.
- 2026-11-25 — ADR-0003 закрепил Foundry authority и read-only Relay; исторические предложения XP/level/Actor writes исключены из критериев.
- 2026-11-25 — Аудит кода подтвердил read-only Relay client, Actor Item decoding, targeted Compendium resolution и world sync; текущая Production status показывает 18 Actors.
- 2026-11-25 — Scope TASK-002 уточнён: WebDAV/ETag/If-Match и Campaign Codex перенесены в TASK-007. Остались runtime-проверки scopes, SSE, headless (по необходимости) и usage/quota.
- 2026-11-25 — Проверена unattended-модель постоянного Foundry-клиента без изменений Relay: на Arch Linux создан изолированный Docker-контейнер `foundry-client` с Chromium, постоянным профилем и Playwright Core 1.48.2. Автоматизация открывает Foundry и ожидает автозаполнения имени GM-пользователя Chrome перед отправкой формы Join Game; пароль в скрипте не хранится. После входа Relay подтвердил WebSocket-клиент `Chrome on Linux` для мира `westmarches` (Foundry 14.367, PF2e) с прежним `clientId`. Проверены восстановление соединения после перезапуска контейнера в графическом режиме и первоначальный запуск/подключение в Chromium headless; Production Relay и его конфигурация не изменялись. Headless-клиент оставлен с тем же постоянным профилем; удаление Xvfb/VNC/Fluxbox и повторный тест перезапуска уже в headless остаются следующими эксплуатационными шагами. Во время загрузки страницы наблюдалась ошибка браузерного JS `Sync file not found`, не помешавшая подключению; её причина отдельно не установлена.
