---
id: TASK-004
title: 2. Персонажи и Foundry
status: in_progress
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-18'
estimate: null
parent: null
depends_on:
- TASK-002
- TASK-003
requirement_ids:
- FM-REQ-003
files_owned:
- src/
- tests/
- docs/
- .github/workflows/ci.yml
labels:
- stage-2
- planned
---

# 2. Персонажи и Foundry

## Objective

Реализовать полный этап 2 из [дорожной карты](../../docs/development/roadmap.md), не сокращая его до каркаса.

## Scope

`FoundryGateway`, снимки актёров, привязка с подтверждением, одобрение, лимит, перевод в NPC, XP и уровень, очередь операций

## Sources and preconditions

- [FM-REQ-003](../requirements/FM-REQ-003.md), [полный раздел](../../docs/architecture/foundry.md), [SQL](../../docs/database/schema.sql).
- [Нерешённые вопросы](../../docs/development/open-questions.md), [пробелы исходников](../../docs/development/source-gaps.md).
- Внешние доступы/изменения в Foundry/WebDAV и деплой требуют отдельного разрешения. Для спайка нужны тестовые Actor/мир и заметки; текущий импорт таких разрешений не даёт.
- Пути files_owned для будущего кода предварительные; перед параллельной работой разбить на дочерние задачи с непересекающимся владением файлами.

## Acceptance criteria

- [ ] Состав этапа выполнен полностью, либо оставшаяся часть явно блокирует его закрытие.
- [ ] Проверки: привязка `POST /api/characters/link`; третий персонаж даёт 409; повтор операции XP не удваивает; остановка relay переводит задачу в «ожидает синхронизации», запуск — в «успех»
- [ ] Готово, когда: персонаж проходит путь от Actor до одобрения, XP и уровень сходятся
- [ ] API/бот/UI, миграции, тесты и документация согласованы в порядке исходного плана, где применимо.
- [ ] Открытые продуктовые решения не подменены предположениями; подтверждения записаны.

## Verification

Привести команды, версии и фактические результаты. Точные команды runtime будут определены при реализации; не копировать ещё несуществующий Compose/build как выполненный.

## Progress log

- 2026-11-17 — Создана из полного этапа v0.3.\n- 2026-11-18 — Начат persistence slice: добавлены DTO → persistence mapper и транзакционный PostgreSQL adapter для characters, foundry_actors и inventory_items. Полный TASK-004 остаётся в backlog до реализации link/approval/jobs/API и live PostgreSQL verification.

- 2026-11-18 — Добавлен read-only Relay HTTP adapter для `GET /get`: чтение и проверка Actor snapshot, разрешение Compendium Item для calculator, привязка клиента к одному world ID, лимит ответа и запрет redirect. Добавлены `httptest` проверки; ожидают локального запуска. В CI добавлены gofmt, `go test ./...` и `go vet ./...`. Live PostgreSQL verification и полный FoundryGateway остаются незавершёнными.

- 2026-11-18 — Добавлены opt-in PostgreSQL integration tests для `FoundryRepository`: реальный sync linked Actor, обновление snapshot, reconcile инвентаря с сохранением ID существующей Foundry-строки, удаление stale Foundry items, сохранение app-owned rows, повторная идемпотентная синхронизация, срабатывание ownership triggers и rollback при отсутствующей Actor link. Тесты требуют отдельную PostgreSQL 15+ БД с именем, оканчивающимся на `_test`, и заранее применённую `docs/database/schema.sql`; без `FOUNDRY_TEST_DATABASE_URL` пропускаются. Добавлен `lib/pq` и инструкция `docs/database/foundry-integration-tests.md`. Интеграционные тесты подключены к существующему CI `database` job: там поднимается временный PostgreSQL 15/16, применяется полная схема и запускается Go integration suite. Фактический CI run для новой ревизии ещё не наблюдался.


- 2026-11-18 — Live world sync подтверждён владельцем: 11 Actor обнаружены и сохранены в `foundry_actors`; все 11 пока `character_id IS NULL` и disposition `snapshot_only`. Синхронизация не создаёт персонажей и не назначает владельцев. Следующий bounded slice — явная привязка существующего application character к Actor с проверками world/type/unique link/authorization и тестами; затем approval и устойчивый sync lifecycle. TASK-004 остаётся `in_progress`, полный этап не закрыт.

- 2026-11-18 — После merge PR #1 в `main` создана отдельная ветка `feat/task-004-character-linking` от merge commit `618bf862dd070e3688709cc319cf55b042e96df0`. Аудит подтвердил, что `foundry_actors.character_id` уже имеет UNIQUE, а `(world_id, actor_uuid)` уникален; отдельная таблица/дублирующая миграция для связи не нужны. Добавлен транзакционный `FoundryRepository.LinkActor`: проверка существующего application character, наличия обнаруженного Actor в заданном UUID мира, `snapshot.type == character`, запрет конфликтных связей, идемпотентность одинакового запроса и append-only audit event. Не изменяет owner и `owner_confirmed_at`. Добавлены opt-in PostgreSQL integration tests на успешную привязку, повтор, конфликт Actor/character и отказ для NPC. Пользовательский HTTP endpoint пока намеренно не добавлен: API имеет только bearer token для sync и не имеет user identity/session/RBAC middleware; повторное использование sync token для привязки было бы обходом пользовательской авторизации. Следующий slice — определить/подключить безопасный user authorization boundary либо сначала завершить базовую авторизацию TASK-003. Проверки в среде разработки пока не запускались; CI run для этой feature branch отсутствует.
