# Передача контекста FoundryMarches

## Текущее состояние

TASK-002 начата в ветке `feat/task-002-integration-spike`, созданной от публичного `main` commit `1f70419cb3f7f5ba673cd7803c244335bf50d352`. Запись через GitHub теперь работает. Ветка не слита в main. Локальный git status недоступен через GitHub connector.

## Подтверждено владельцем

- Foundry VTT 14, build 367; PF2e 8.7.0.
- Тестовый Actor `АНДРЕЙ`, UUID `Actor.mxgc0DRWC8bA9ruI`; владелец разрешил выполнять тестовую работу над ним.
- Хост Arch Linux; Docker и Docker Compose установлены.
- `foundryvtt-container` публикует порт 30000.
- `catalog-container` публикует 127.0.0.1:8080 и категорически вне scope; не трогать.
- Установлен, но не настроен модуль ThreeHats `foundryvtt-rest-api`; владелец разрешил перенастроить/переустановить.
- WebDAV доступен; тестировать только в отдельной директории вне Obsidian notes.
- Владелец просит подробные инструкции.

## Выполнено в этой ветке

- TASK-002 переведена в `in_progress`.
- Обновлён `state/current.yaml`.
- Создан `docs/architecture/decisions/spike-0-results.md` с первичными выводами и планом.
- Подтверждено, что module repo и relay repo — два отдельных компонента.
- Никаких live-запросов, изменений Foundry Actor, WebDAV или Codex не выполнялось.
- Результаты интеграций пока не подтверждены.

## Блокеры и правила безопасности

1. Не изменять основной игровой Actor до согласования конкретного тестового Actor, допустимых операций, резервного снимка и плана восстановления.
2. Для relay нужны способ установки/версии и безопасный способ передать конфигурацию; не публиковать API keys в репозитории/чате.
3. WebDAV тестировать только в отдельном изолированном каталоге, например `/FoundryMarches-spike/`, с отдельными тестовыми файлами; не читать/перезаписывать реальные заметки кампании без отдельного разрешения.
4. Нужны обезличенные JSON-экспорты журналов Campaign Codex.
5. Отсутствующее исходное ТЗ v1.0/два дополнения остаются gap; не реконструировать сценарии по догадкам.

## Следующие действия

1. Получить read-only Docker networks/mounts и текущий manifest/version Foundry module.
2. Определить точный relay product/repository/version и deployment method; проверить его API/schema/config.
3. Подготовить WebDAV test directory и получить read/write ETag/If-Match tests.
4. Получить Codex JSON exports и исследовать формат без записи обратно.
5. Провести тесты Foundry read, coin inventory, XP idempotency, level-up, item UUID, SSE, headless session, relay quota.
6. Зафиксировать таблицу результатов с версиями, командами, фактическими ответами, ограничениями и итогом relay vs Foundry Bridge.
7. TASK-003 остаётся независимой и может выполняться параллельно при разделении владения файлами.

## Текущий commit

Изменения PAD/TASK-002 в ветке `feat/task-002-integration-spike`; см. историю ветки. Live-интеграции ещё не проверялись.

## Live-обновление — 2026-11-17

### Подтверждено владельцем
- Relay 3.4.1 развёрнут на Arch Linux host в `/opt/foundry-marches-relay`; SQLite, persistent `./data:/app/data`, host bind `127.0.0.1:3010:3010`.
- Образ закреплён как `threehats/foundryvtt-rest-api-relay:3.4.1@sha256:60df34aca78968f072a77e7d19358073aa514175e0ecec43406cb1a1ed64948e` (linux/amd64).
- Caddy проксирует `relay.jiene.ru` → `127.0.0.1:3010`; HTTPS и HTTP→HTTPS redirect проверены. Foundry работает через `foundry.jiene.ru`.
- Foundry options: hostname `foundry.jiene.ru`, proxyPort 443, proxySSL true.
- Foundry module URL изменён на `wss://relay.jiene.ru/relay`; admin login и Pairing успешны.
- REST read smoke test: `GET /get` с UUID `Actor.mxgc0DRWC8bA9ruI` и clientId `fvtt_cca540151c686a43` вернул entity-result с Actor name `АНДРЕЙ`, type character, Foundry 14.367, PF2e 8.7.0, level 1, XP 0, HP 0, items [].
- Проверка была read-only; временный JSON удалён, API key в репозиторий/чат не попадал.
- Подробный журнал, точная конфигурация и результаты — `docs/architecture/decisions/spike-0-results.md`.

### Не считать завершённым
Coins/inventory model, XP write + operation flag/idempotency, level-up, item UUID grant, SSE, headless session, фактический quota/usage accounting, WebDAV ETag/If-Match и Campaign Codex export остаются pending. Не выполнять write tests до snapshot/restore подготовки и подтверждения конкретной операции владельцем.
