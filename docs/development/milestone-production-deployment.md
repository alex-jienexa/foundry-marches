# Веха: первый Production deployment и подготовка v0.2.0-beta

**Дата фиксации:** 2026-11-25  
**Статус:** Production deployment выполнен и проверен владельцем; веха зафиксирована как достигнутая.  
**Репозиторий:** `alex-jienexa/foundry-marches`  
**Ветка документации:** `feat/conventional-commits`

## Цель

Перевести FoundryMarches из работающей Dev-разработки в отдельное постоянно работающие Production-окружение, не затрагивая существующие сервисы владельца и не смешивая данные Dev и Production.

## Выполнено

### Production-инфраструктура

- Подготовлен отдельный каталог `/opt/foundry-marches/` и отдельный Compose project `foundry-marches-prod`.
- Развёрнуты отдельные API, Discord bot и PostgreSQL 18 Alpine.
- Production PostgreSQL создана как чистая независимая БД; перенос Dev-данных не выполнялся.
- API опубликован только на loopback `127.0.0.1:8081`; PostgreSQL не опубликован на host port.
- Контейнеры используют отдельную Docker network и отдельный PostgreSQL volume.
- Для сервисов заданы memory limits и hardening; секреты хранятся в локальном `.env.prod` с правами `600`.
- Используются immutable image tags по SHA: `sha-8a69f78`.
- Существующий проект `jienexyz`, его БД/volumes, Caddy, Redis, `catalog-container` и `rosochkinsad.ru` не изменялись.

### Интеграции и приложение

- Discord bot подключился к целевой гильдии.
- API успешно подключился к PostgreSQL, Foundry Relay и Foundry.
- Foundry world sync обнаруживает 18 Actor.
- Фоновая синхронизация включена с интервалом 5 минут.
- Проверка владельца через `/integration-status` подтвердила:
  - PostgreSQL — доступен;
  - Foundry Relay — доступен;
  - Foundry — доступен, найдено 18 Actor.
- `GET /livez` вернул HTTP 200 `ok`.
- `GET /readyz` вернул HTTP 200 `ok`.
- Владелец подтвердил, что бот работает; ожидается пользовательская обратная связь от игроков.

### Публикация контейнеров

- GitHub Actions workflow Project checks завершился успешно.
- Workflow Publish production images завершился успешно.
- В GHCR опубликованы API и bot images с SHA tag `sha-8a69f78`.
- Сформирован deployment bundle с Compose-файлами, bootstrap/migrations, backup script и production runbook.

### Инженерный процесс

В этой же feature-ветке добавлена обязательная политика Conventional Commits 1.0.0:
- `.agent/protocols/COMMIT_CONVENTIONS.md` — формат, типы, scopes, breaking changes и правила для агентов/контрибьюторов;
- `AGENTS.md` — обязательное чтение политики и требование для каждого нового commit;
- `.agent/agents/IMPLEMENTER.md` — правила формирования commit messages для implementer.

Политика подготовлена для будущей генерации release notes и SemVer. Генерация релизов и CD при этом ещё не включены.

## Границы и незавершённые проверки

- Production работает, но автоматический деплой на сервер не включён.
- До проектирования/включения CD необходимо отдельно проверить новый сервер и безопасный deployment/rollback workflow на нём.
- Нужны регулярные проверяемые PostgreSQL backups и тест восстановления; наличие backup script само по себе не доказывает успешный restore.
- Нужны наблюдение за RAM/OOM, диском и ростом логов на небольшом сервере.
- Не считать полную продуктовую roadmap (этапы 0–11) завершённой: это инфраструктурная/операционная веха, а не закрытие этапов 0–2.
- Обратная связь игроков ещё не собрана.

## Критерий достижения

Веха считается достигнутой: Production API и bot запущены отдельно от Dev и существующих проектов; БД изолирована; Relay/Foundry доступны; синхронизация работает; health checks и Discord integration status подтверждены владельцем.

## Следующий релиз

Первый релиз проекта планируется как **`v0.2.0`**, prerelease-метка — **`beta`** (предпочтительный Git tag: `v0.2.0-beta`). Не начинать историю релизов с `v1.0.0` или с более раннего номера без отдельного решения владельца.
