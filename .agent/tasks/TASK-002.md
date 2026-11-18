---
id: TASK-002
title: 0. Спайк (1–2 дня)
status: in_progress
priority: high
assignee: ChatGPT
created_at: '2026-11-17'
updated_at: '2026-11-18'
estimate: null
parent: null
depends_on:
- TASK-001
requirement_ids:
- FM-REQ-001
files_owned:
- docs/architecture/decisions/spike-0-results.md
- .agent/decisions/
labels:
- stage-0
- planned
---

# 0. Спайк (1–2 дня)

## Objective

Реализовать полный этап 0 из [дорожной карты](../../docs/development/roadmap.md), не сокращая его до каркаса.

## Scope

relay у вас на хосте; чтение листа PF2e; монеты как предметы; запись XP с флагом операции; повышение уровня тестовому актору; хуки SSE; безголовая сессия; лимит self-hosted relay; ETag и `If-Match` на вашем WebDAV; выгрузка журналов Codex

## Sources and preconditions

- [FM-REQ-001](../requirements/FM-REQ-001.md), [полный раздел](../../docs/architecture/foundry.md), [SQL](../../docs/database/schema.sql).
- [Нерешённые вопросы](../../docs/development/open-questions.md), [пробелы исходников](../../docs/development/source-gaps.md).
- Внешние доступы/изменения в Foundry/WebDAV и деплой требуют отдельного разрешения. Для спайка нужны тестовые Actor/мир и заметки; текущий импорт таких разрешений не даёт.
- Пути files_owned для будущего кода предварительные; перед параллельной работой разбить на дочерние задачи с непересекающимся владением файлами.

## Acceptance criteria

- [ ] Состав этапа выполнен полностью, либо оставшаяся часть явно блокирует его закрытие.
- [ ] Проверки: пункты из 2.6, 2.9, 3.2, 3.6; для каждого — результат и вывод
- [ ] Готово, когда: есть решение «relay или собственный модуль» и образцы данных Codex
- [ ] API/бот/UI, миграции, тесты и документация согласованы в порядке исходного плана, где применимо.
- [ ] Открытые продуктовые решения не подменены предположениями; подтверждения записаны.

## Verification

Привести команды, версии и фактические результаты. Точные команды runtime будут определены при реализации; не копировать ещё несуществующий Compose/build как выполненный.

## Progress log

- 2026-11-17 — Начата работа в ветке `feat/task-002-integration-spike` от `1f70419`. Подтверждены версии Foundry VTT 14 build 367 и PF2e 8.7.0. Тестовый Actor `АНДРЕЙ` (`Actor.mxgc0DRWC8bA9ruI`) предоставлен владельцем для всех тестов записи; отдельного тестового мира нет. Сервер Arch Linux, Docker/Compose установлены; `catalog-container` вне scope и неприкосновенен. Установленный модуль ThreeHats REST API не настроен; владелец разрешил его перенастройку/переустановку. Внешние серверные команды и live-проверки пока не выполнялись.

- 2026-11-17 — Развёрнут self-hosted ThreeHats Relay 3.4.1 на Arch Linux; image зафиксирован digest `sha256:60df34aca78968f072a77e7d19358073aa514175e0ecec43406cb1a1ed64948e`, SQLite data persistent, host port привязан к loopback `127.0.0.1:3010`. Caddy проксирует `relay.jiene.ru` по HTTPS; Foundry доступен через `foundry.jiene.ru`. Секреты не внесены в репозиторий.
- 2026-11-17 — Foundry VTT 14 build 367 / PF2e 8.7.0; модуль ThreeHats REST API 3.4.1 переведён с публичного Relay на `wss://relay.jiene.ru/relay`. Владелец подтвердил вход в admin UI и успешный Pairing.
- 2026-11-17 — Read-only REST smoke test PASS: запрос `GET https://relay.jiene.ru/get?uuid=Actor.mxgc0DRWC8bA9ruI&clientId=fvtt_cca540151c686a43` с API key вернул `entity-result` для Actor `АНДРЕЙ`; PF2e 8.7.0, Foundry 14.367, level 1, XP 0, HP 0, items 0. Actor не изменялся; временный JSON удалён, shell variable с ключом очищена. Подробности в `docs/architecture/decisions/spike-0-results.md`.


- 2026-11-18 — Подтверждена полная синхронизация мира через приложение после синхронизации мира Foundry после обновления PF2e. Найдены и сохранены 11 Actor snapshots в `foundry_actors`; для всех `character_id IS NULL`, disposition `snapshot_only`. Это ожидаемо до явной привязки. Внутренний WorldID приложения — UUID; Relay identifier `westmarches` не подменяет его. Foundry Actor не изменялись.
- 2026-11-18 — Этап 0 остаётся `in_progress`: WebDAV ETag/If-Match, Campaign Codex exports, SSE/headless session и usage/quota не проверены.
