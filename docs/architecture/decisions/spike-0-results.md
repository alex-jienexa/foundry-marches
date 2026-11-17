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
| Foundry relay connectivity | pending | Live-проверка не выполнялась |
| Actor snapshot | pending | Live-проверка не выполнялась |
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

1. Получить read-only inventory текущего Foundry module (точная версия/manifest/настройки без секретов) и Docker networks/mounts для Foundry container.
2. Изучить конкретный upstream relay release и compose file; проверить image provenance, volumes, exposed ports, env defaults, authentication and quota behavior.
3. Подготовить отдельный compose project и конфигурацию с loopback-only API binding; сначала показать владельцу команды и diff.
4. После отдельного подтверждения применить deployment и настроить Foundry module URL/key.
5. Выполнить smoke test `GET /clients` и read-only Actor snapshot.
6. Затем провести разрешённые write tests только на указанном тестовом Actor.

## 8. Открытые вопросы

- Точный способ хранения Foundry data/modules и Docker network configuration.
- Точный текущий manifest/version Foundry REST API module.
- Какой путь и сервер WebDAV; отдельный тестовый каталог и credentials.
- Обезличенные JSON exports Campaign Codex.
- Требуемая политика level-up: автоматическая или GM-confirmed (решение владельца ещё требуется).
