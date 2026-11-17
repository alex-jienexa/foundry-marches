# Spike 0 — результаты и план интеграционной проверки

Дата начала: 2026-11-17  
Статус: выполняется; интеграционные результаты не подтверждены.

## 1. Среда, подтверждённая владельцем

- Foundry VTT 14, build 367.
- Pathfinder 2e system 8.7.0.
- Тестовый Actor: `АНДРЕЙ`, UUID `Actor.mxgc0DRWC8bA9ruI`. Владелец прямо разрешил использовать его для тестов записи TASK-002.
- Отдельного тестового мира нет.
- Хост: Arch Linux; Docker и Docker Compose установлены.
- Известные контейнеры:
  - `foundryvtt-container`, image `foundryvtt:v14`, порт `30000`.
  - `catalog-container`, image `catalog-app:latest`, порт `127.0.0.1:8080`. Контейнер и его данные вне scope, не изменять.
- Установлен, но не настроен модуль ThreeHats `foundryvtt-rest-api`.
- WebDAV существует; тесты только в отдельной директории вне Obsidian campaign notes.

## 2. Архитектура выбранного внешнего компонента

Модуль Foundry `ThreeHats/foundryvtt-rest-api` — это Foundry-side companion module. Он подключается по WebSocket к отдельному relay-серверу. Внешнее приложение вызывает REST API relay; relay маршрутизирует запросы в подключённый Foundry world. Следовательно, установка только Foundry-модуля не создаёт self-hosted REST endpoint.

Upstream relay repository: https://github.com/ThreeHats/foundryvtt-rest-api-relay  
Foundry module repository: https://github.com/ThreeHats/foundryvtt-rest-api  
Upstream docs: https://foundryrestapi.com/docs/intro

Upstream README описывает Docker Compose deployment relay, HTTP endpoint на порту 3010, API-key authentication и SQLite по умолчанию; также перечисляет per-minute quota configuration. Документация изменяется upstream, поэтому перед установкой нужно pin-ить конкретный release/tag или image digest, а не использовать `latest` в постоянной конфигурации.

## 3. Предварительное архитектурное решение

Продолжить spike с upstream ThreeHats self-hosted relay как проверяемым кандидатом, а не начинать собственный relay с нуля. Проверить фактические возможности версии, особенно:
- совместимость с Foundry 14 / PF2e 8.7.0;
- чтение Actor и структуру PF2e system data;
- безопасные write operations и operation idempotency;
- SSE/webhook/change notifications;
- headless GM session;
- rate limiting и auth scope;
- item UUID grant;
- level-up semantics.

Не считать наличие endpoint в README доказательством прохождения интеграционного теста.

## 4. Сетевая схема

Предпочтительная схема — relay в отдельном Docker Compose project на том же Arch Linux host, с persistent data в отдельной директории. Если relay и Foundry находятся в одной Docker network, relay сможет обращаться к Foundry по DNS имени контейнера и внутреннему порту; если сеть общая не используется, потребуется аккуратно спроектированная общая сеть либо доступ к host gateway. Нельзя предполагать, что `localhost` внутри relay container указывает на Foundry container.

Relay API не следует публиковать в интернет без TLS/reverse proxy и firewall policy. Для первого smoke test предпочтительно привязать HTTP port к loopback и выполнять curl с хоста. Foundry module должен иметь сетевой маршрут к WebSocket endpoint; конкретный URL зависит от выбранной схемы.

## 5. Безопасность тестов

- Все Actor mutations только для UUID `Actor.mxgc0DRWC8bA9ruI`.
- Перед первым write test получить экспорт/снимок Actor и записать исходные XP, level, HP, AC, speed, inventory, currency.
- Каждую мутацию выполнять отдельно, читать Actor обратно, проверять ожидаемое состояние.
- XP idempotency: повторить тот же operation_id; ожидается отсутствие повторного начисления.
- Не запускать destructive tests над остальными Actor/world data.
- Не удалять существующий модуль до фиксации его версии, manifest, настроек и проверки доступности rollback.
- Не менять, перезапускать и не пересоздавать `catalog-container`; не использовать его volumes/networks для relay.
- WebDAV test directory — отдельный явно согласованный путь, например `/FoundryMarches-spike/`; не сканировать и не менять реальные заметки.
- Секреты не передавать в чат и не коммитить. Использовать root-owned env file с ограниченными правами или secret store.

## 6. Результаты

| Проверка | Статус | Результат |
|---|---|---|
| Foundry relay connectivity | **PASS** | Self-hosted Relay 3.4.1 запущен; HTTPS и GM Pairing подтверждены владельцем 2026-11-17 |
| Actor snapshot | **PASS (read-only)** | `GET /get?uuid=Actor.mxgc0DRWC8bA9ruI&clientId=fvtt_cca540151c686a43` вернул `entity-result`; `data.name=АНДРЕЙ`, `data.type=character`, PF2e 8.7.0, Foundry 14.367, level 1, XP 0, HP 0, `items=[]` |
| PF2e coins as items | pending | Live-проверка не выполнялась |
| XP write + operation flag | pending | Live-проверка не выполнялась |
| XP idempotency | pending | Live-проверка не выполнялась |
| Level-up behavior | pending | Live-проверка не выполнялась |
| Grant item by UUID | pending | Live-проверка не выполнялась |
| SSE | pending | Live-проверка не выполнялась |
| Headless GM session | pending | Live-проверка не выполнялась |
| Self-hosted relay quota | pending | Live-проверка не выполнялась |
| WebDAV ETag / If-Match | pending | Нужен изолированный каталог и доступ |
| Campaign Codex export | pending | Нужны обезличенные образцы |

## 7. Следующие шаги

Deployment, Relay URL, Pairing и read-only Actor smoke test выполнены; актуальные параметры и доказательства приведены в §9.

1. Исследовать read-only inventory/coins. Текущий тестовый Actor вернул `items=[]`, поэтому для проверки coin items нужен отдельный тестовый setup или согласованное добавление тестовых данных.
2. Согласовать политику level-up: автоматическая или GM-confirmed.
3. Перед любыми write tests получить snapshot Actor и проверить восстановление; затем отдельно согласовать точную XP/item операцию.
4. Проверить XP operation flag и идемпотентность, grant item by UUID.
5. Проверить SSE hooks и реальную headless GM session без открытого браузера.
6. Проверить usage accounting и индивидуальные API-key limits; конфигурация `MONTHLY_REQUEST_LIMIT=0` сама по себе не является тестом квоты.
7. Подготовить отдельный WebDAV test directory и проверить ETag/If-Match.
8. Получить обезличенные JSON exports Campaign Codex и проверить импорт/экспортный формат.

## 8. Открытые вопросы

- Нет отдельного тестового мира; для write tests использовать только согласованный Actor `Actor.mxgc0DRWC8bA9ruI` с предварительным snapshot/restore plan.
- Требуется решение владельца по автоматическому или GM-confirmed level-up.
- Требуется отдельный WebDAV test path вне Obsidian campaign notes.
- Требуются обезличенные JSON exports Campaign Codex.
- Требуется отдельно проверить ограничения доступа к публично доступной Relay admin UI (IP allowlist сейчас выключен по решению владельца).


## 9. Live deployment и проверка подключения (2026-11-17)

### 9.1. Фактическая серверная конфигурация

Владелец выполнил конфигурацию на Arch Linux host; ниже записаны только несекретные параметры.

- Foundry: `foundryvtt-container`, image `foundryvtt:v14`; Foundry VTT 14 build 367, Node.js 26.7.0.
- Foundry data на host: `/opt/foundry/foundrydata`; в контейнере: `/home/node/foundrydata`.
- Foundry REST API module: ThreeHats, версия 3.4.1.
- Relay: `foundry-marches-relay`, image `threehats/foundryvtt-rest-api-relay:3.4.1@sha256:60df34aca78968f072a77e7d19358073aa514175e0ecec43406cb1a1ed64948e` (linux/amd64).
- Relay Compose project: `/opt/foundry-marches-relay`; SQLite, data dir `./data:/app/data`, host bind только `127.0.0.1:3010:3010`, Redis отключён, `ALLOW_HEADLESS=true`, `DISABLE_REGISTRATION=true`, `MONTHLY_REQUEST_LIMIT=0`, `ADMIN_SECURE_COOKIES=true`.
- Relay secrets (admin password, generated encryption key, JWT secret) хранятся вне репозитория: env file и Relay data volume. Их значения намеренно не записываются в этот отчёт.
- Caddy v2.11.4 используется как существующий reverse proxy. Не создавался второй proxy.
- `foundry.jiene.ru` проксируется на `127.0.0.1:30000`; `relay.jiene.ru` — на `127.0.0.1:3010`.
- Foundry options: `hostname=foundry.jiene.ru`, `proxyPort=443`, `proxySSL=true`; внутренний Foundry port остаётся 30000.
- Relay URL в настройках Foundry module изменён с публичного `wss://foundryrestapi.com/relay` на `wss://relay.jiene.ru/relay`.
- Владелец подтвердил успешную авторизацию в Relay admin UI и успешный Pairing Foundry с self-hosted Relay.
- HTTPS для обоих доменов проверен владельцем через curl/browser; Caddy получил Let's Encrypt сертификаты. Для Relay HTTP перенаправляется на HTTPS.
- Существующий `catalog-container` и сайт `rosochkinsad.ru` не затрагивались.

### 9.2. Проверка REST API и Actor

Владелец выполнил на сервере запрос (API key передавался через shell variable и не раскрывался в чате):

```bash
curl -fsS --get 'https://relay.jiene.ru/get' \
  --data-urlencode 'uuid=Actor.mxgc0DRWC8bA9ruI' \
  --data-urlencode 'clientId=fvtt_cca540151c686a43' \
  -H "x-api-key: $FOUNDRY_API_KEY" \
  -o /tmp/foundry-actor-read.json
```

Команда завершилась с кодом 0. JSON содержал:
- `type: "entity-result"`;
- `uuid: "Actor.mxgc0DRWC8bA9ruI"`;
- `data.name: "АНДРЕЙ"`, `data.type: "character"`;
- `data._id: "mxgc0DRWC8bA9ruI"`;
- `data.system.details.level.value: 1`;
- `data.system.details.xp.value: 0`;
- `data.system.attributes.hp.value: 0`;
- `data.items: []`;
- `data._stats.coreVersion: "14.367"`, `systemId: "pf2e"`, `systemVersion: "8.7.0"`.

В ответе сущность обёрнута в `data`; поля Actor находятся по путям `.data.name`, `.data.system...`, а не в корне JSON. `jq` на сервере не установлен, поэтому попытка форматированного вывода завершилась сообщением `jq: command not found`; это не повлияло на REST-запрос и получение JSON. Временный файл `/tmp/foundry-actor-read.json` удалён, переменная `FOUNDRY_API_KEY` очищена командой `unset`.

**Вывод:** сетевой путь, TLS, Pairing, аутентификация API key, маршрутизация Relay → GM Foundry client и чтение конкретного PF2e Actor подтверждены end-to-end. Проверка была read-only; Actor не изменялся.

### 9.3. Обновлённая матрица результатов

| Проверка | Статус | Комментарий |
|---|---|---|
| Self-hosted Relay deployment | **PASS** | Relay 3.4.1 запущен на host loopback; persistent SQLite data |
| Relay admin UI / login | **PASS** | Владелец подтвердил вход через HTTPS |
| Caddy / TLS для Relay | **PASS** | `https://relay.jiene.ru/` возвращает HTTP 200; HTTP → HTTPS redirect |
| Foundry module Pairing | **PASS** | Владелец подтвердил успешный Pairing с `wss://relay.jiene.ru/relay` |
| REST API authentication и routing | **PASS** | API key + clientId; `GET /get` вернул ответ от Foundry |
| PF2e Actor read | **PASS (read-only)** | Получен Actor `АНДРЕЙ`, UUID указан выше |
| PF2e coins as items | pending | Инвентарь тестового Actor пуст; модель валюты ещё не проверена |
| XP write + operation flag | pending | Запись не выполнялась |
| XP idempotency | pending | Повторная операция не проверялась |
| Level-up behavior | pending / owner decision | Требуется решение владельца о GM confirmation vs auto; запись не выполнялась |
| Grant item by UUID | pending | Не проверялось |
| SSE hooks | pending | Не проверялось |
| Headless GM session | pending | Настройка разрешена в Relay, фактическая работа без браузера GM не проверена |
| Self-hosted relay quota | pending | `MONTHLY_REQUEST_LIMIT=0` задан; фактическое поведение счётчиков/лимитов ещё не проверено |
| WebDAV ETag / If-Match | pending | Нужны отдельный тестовый каталог и тесты |
| Campaign Codex export | pending | Нужны обезличенные образцы |

### 9.4. Безопасность и оставшиеся ограничения

- Не сохранять API keys, admin password, `.env`, `.secrets.env`, JWT/encryption secrets, cookies или pairing credentials в Git.
- Relay admin UI доступен публично через HTTPS; регистрация отключена, secure cookies включены. IP allowlist изначально оставлен выключенным по решению владельца. Рекомендуется отдельно пересмотреть ограничение доступа к admin UI.
- `MONTHLY_REQUEST_LIMIT=0` означает отсутствие глобального месячного лимита в конфигурации Relay; это не заменяет проверку индивидуальных API key limits и фактического usage accounting.
- `ALLOW_HEADLESS=true` только разрешает headless workflow; это ещё не доказывает, что запрос успешно выполняется при закрытом GM browser.
- Тест чтения подтвердил структуру ответа Actor, но не наличие AC/speed: в фактическом ответе эти поля отсутствовали. Не считать критерий полного набора полей листа полностью закрытым.
- Значения HP=0 и пустой инвентарь зафиксированы как текущее состояние тестового Actor на момент чтения; не трактовать их как ошибку интеграции.

### 9.5. Следующие проверки

1. Read-only исследование инвентаря/денежных предметов PF2e на тестовом Actor; сейчас `items=[]`, поэтому для проверки representation coins потребуется отдельный согласованный тестовый setup или тестовые данные.
2. Проверить точную структуру GET response и поля PF2e, доступные для Actor этого типа.
3. Перед любыми write tests — зафиксировать исходный экспорт Actor, проверить backup/restore и подготовить отдельную операцию на том же тестовом Actor.
4. После отдельного подтверждения выполнить XP write/idempotency и item grant tests.
5. Отдельно проверить SSE, headless GM, relay usage/quota, WebDAV и Codex.
