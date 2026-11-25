# Production smoke verification — 2026-11-25

Этот отчёт фиксирует результаты, сообщённые владельцем после запуска Production. Это smoke verification окружения, а не полный acceptance этапов 0–2 и не нагрузочный/аварийный тест.

## Deployment

- Host: Ubuntu 24.04 LTS, x86_64.
- Compose project: `foundry-marches-prod`.
- Deployment directory: `/opt/foundry-marches/`.
- Services: API, Discord bot, PostgreSQL 18 Alpine.
- API host binding: `127.0.0.1:8081 -> container:8080`.
- PostgreSQL host port не опубликован.
- Database, volume и Compose network отдельные от Dev.
- Container images: `ghcr.io/alex-jienexa/foundry-marches-api:sha-8a69f78` и `ghcr.io/alex-jienexa/foundry-marches-bot:sha-8a69f78`.

## Подтверждённые проверки

### Health endpoints

Владелец выполнил запросы к локальному API:

| Endpoint | Результат |
|---|---|
| `GET /livez` | HTTP 200, body `ok` |
| `GET /readyz` | HTTP 200, body `ok` |

### Integration status

Вывод Discord-команды `/integration-status`:

| Компонент | Результат |
|---|---|
| PostgreSQL | Доступен |
| Foundry Relay | Доступен, HTTP-сервис отвечает |
| Foundry | Доступен; найдено 18 Actor |

Фоновая Foundry sync включена с интервалом 5 минут. Discord bot подключён к Production guild; владелец подтвердил, что бот работает.

### GitHub publication

- Project checks workflow для commit `8a69f78` — success.
- Publish production images workflow для commit `8a69f78` — success.
- Оба контейнерных образа доступны в GHCR с immutable SHA tag.

## Что эти проверки не доказывают

- Регулярное резервное копирование и успешное восстановление из backup.
- Поведение при OOM, заполнении диска, потере Relay/Foundry или рестарте хоста.
- Корректность rollback после неудачного обновления.
- Полный acceptance roadmap, включая этапы 0–2.
- Нагрузочную устойчивость и долгосрочную стабильность.

До включения автоматического Production CD выполнить отдельную проверку целевого сервера, backup/restore и процедуры rollback. Не считать успешный первичный deployment достаточным разрешением на автоматический деплой при каждом push в `main`.
