# Замена Foundry Relay без потери существующих ключей

> Статус: подготовленный runbook, не выполненная production-операция. До отдельного одобрения владельца действующий Relay, Foundry и production-приложение не переключать.

## Цель и границы

Заменить текущий self-hosted ThreeHats Foundry REST API Relay на обновлённую сборку с поддержкой headless-авторизации Foundry v14, сохранив существующие Relay API keys, client records, pairing и конфигурацию мира настолько, насколько допускает совместимость формата данных.

Проверенная тестовая конфигурация:
- Foundry 14.367, world westmarches, system pf2e.
- Тестовый Relay 3.4.1, отдельный контейнер foundry-relay-v14-headless-test, порт 3012.
- Headless username Puffy, Foundry user ID задаётся через FOUNDRY_USER_ID.
- Успешный POST /join, загрузка #ui-left, регистрация клиента и активная сессия подтверждены логами 2026-11-27.
- Тестовая SQLite: /app/data/relay-v14-test.db; это отдельная БД, не production.

Тестовая сборка и её SQLite не являются production replacement bundle и не должны подключаться к production volume.

## Какие данные и токены различать

1. **Relay API keys и client records.** Сохранность зависит от того, где и в каком виде текущая версия Relay хранит ключи, их хэши/секреты, client ID, scopes и связи с Foundry world. Сначала исследовать схему и миграции обеих версий. Не считать, что одного копирования API key из env достаточно.
2. **Foundry module pairing / Relay connection settings.** Могут находиться в настройках модуля Foundry и/или в Relay DB. Сохранять текущие module settings, world settings, pairing records и идентификаторы; не выполнять unpair и не переустанавливать модуль во время миграции.
3. **Ключ приложения Foundry Marches.** FOUNDRY_RELAY_API_KEY и FOUNDRY_RELAY_CLIENT_ID передаются backend из защищённого .env.prod. foundry_worlds.key_secret_ref хранит только ссылку env:FOUNDRY_RELAY_API_KEY, не сам секрет. При сохранении существующих Relay credentials эти значения не менять.
4. **Токены приложения.** FOUNDRY_SYNC_API_TOKEN — отдельный Bearer token для административного sync endpoint, не Relay API key. Для замены Relay он не требуется и не должен ротироваться.
5. **Headless user ID.** FOUNDRY_USER_ID — Foundry user ID (в тесте Puffy: gkYizCRHVPCjt0Xe), не Relay API key и не client ID.

Не публиковать значения секретов в логах, документации, GitHub или чате.

## Предварительные условия — ничего не переключая

1. Получить согласованное окно обслуживания и подтвердить владельцу, что на время cutover headless-соединение и Relay API могут быть недоступны.
2. Зафиксировать текущее состояние: версия и image digest старого Relay, Compose project/service/container, hostname/reverse proxy, host paths/volumes, SQLite path, config files, secrets source, Foundry module version/settings, client ID, scopes, список имён/ID ключей (без значений ключей), текущий world/client status.
3. Изучить код новой сборки и release notes: схема SQLite, migrations, способ хранения API keys и pairing, формат config, совместимость с существующей версией. Если нет гарантированной обратной совместимости — миграция данных отдельным проверяемым шагом, а не запуск новой версии на старой БД.
4. Убедиться, что новая сборка использует тот же Relay API contract, который ожидает Foundry Marches. Проверить read-only методы /healthz или фактический health endpoint, /clients, /search, /get и scopes на staging.
5. Уточнить, какие секреты нужны новой сборке для сохранения ключей (например, signing/encryption secret). Перенести их из secret store без вывода значений; не генерировать новые при переносе существующей базы, пока не выяснено их назначение.
6. Подготовить независимое место для staging: другой Compose project, container name, host port, SQLite/volume и reverse-proxy hostname. Не монтировать production DB в staging и не допускать подключения staging к production world module.
7. Подготовить и проверить backup/restore для Relay DB, конфигурации, secrets и текущего deployment definition. Резервная копия должна быть вне каталога, который может перезаписать deployment.

## Рекомендуемая последовательность cutover

### A. Подготовить резервную копию старого Relay

1. Сохранить Compose/config/deployment files и точные image references старой версии.
2. Сделать согласованный SQLite backup. Предпочтительно использовать SQLite online backup/API или остановить только Relay service и затем скопировать DB вместе с нужными sidecar/WAL файлами. Не копировать активный SQLite-файл обычным cp, если не обеспечена консистентность.
3. Сохранить все persistent data directories, конфиги, сертификаты/секреты и reverse-proxy config. Архив зашифровать, ограничить права и проверить checksum.
4. Проверить backup на отдельной копии: SQLite integrity check, чтение таблиц клиентов/ключей/scopes/pairing. Секреты не выводить.
5. Зафиксировать rollback: старый image digest, старый Compose config, старый DB backup и прежний маршрут reverse proxy.

### B. Проверить перенос ключей на изолированной копии

1. Развернуть новую сборку под отдельным именем и на отдельном loopback-порту.
2. Восстановить backup в отдельную staging DB/volume; не использовать исходный production volume.
3. Если новая версия требует DB migrations — запускать их только на staging copy. Сохранить исходную копию без изменений.
4. Проверить, что прежние client IDs, ключевые записи, scopes и pairing records присутствуют после миграции. Сравнивать только безопасные метаданные/количество/ID, не значения секретов.
5. Проверить, что прежний FOUNDRY_RELAY_API_KEY (вводимый из secret store) аутентифицируется с прежним FOUNDRY_RELAY_CLIENT_ID. Убедиться, что read-only scopes остались read-only.
6. Проверить Foundry v14 headless login на отдельной тестовой сессии с FOUNDRY_USER_ID; подтвердить POST /join, загрузку игрового интерфейса и регистрацию одного headless client.
7. Выполнить read-only smoke test backend Foundry Marches: /integration-status и синхронизацию, убедиться, что Actor snapshots читаются. Не выполнять Relay write operations.

Если старые ключи не переживают миграцию, остановиться и определить поддерживаемый экспорт/импорт или контролируемую ротацию. Не удалять старые ключи и не отзывать их до завершения production cutover и проверки rollback.

### C. Production cutover

1. Согласовать короткое окно обслуживания. Остановить/заморозить только процессы, которые создают новые Relay sessions или делают sync, чтобы избежать одновременных изменений в Relay DB. Не останавливать Foundry world без необходимости.
2. Сделать финальный согласованный backup непосредственно перед переключением.
3. Остановить только старый Relay service. Не удалять контейнер/volume/DB; не выполнять down -v, prune или очистку каталогов.
4. Развернуть новую версию с production DB, прошедшей staging migration. Сохранить прежние persistent mount paths либо использовать явно восстановленную БД. Настроить FOUNDRY_USER_ID на целевой Foundry user ID и остальные секреты через secret store.
5. Сохранить внешний HTTPS hostname и маршрут reverse proxy (например, текущий relay.jiene.ru), чтобы Foundry module и Foundry Marches не пришлось перенастраивать. Переключать upstream только после health check новой версии.
6. Убедиться, что прежние client IDs и API keys работают, Foundry world снова виден, headless session установлена и backend /integration-status успешен.
7. Выполнить одну read-only синхронизацию и сравнить число обнаруженных Actors и результат с baseline. Проверить логи на 401/403/429, pairing errors и restart loop.
8. Возобновить плановую синхронизацию. Наблюдать Relay, Foundry и API до окончания согласованного периода.
9. Только после подтверждённой стабильности и отдельного решения владельца выводить старый image из эксплуатации. Старую БД/backup хранить в течение согласованного rollback window.

### D. Rollback

1. Остановить только новую версию Relay.
2. Вернуть прежний upstream/reverse-proxy route и прежний image/config.
3. Восстановить прежнюю БД только если новая версия изменила формат данных несовместимо. Не накатывать старую БД поверх новой без остановки Relay и проверки backup.
4. Проверить прежний Relay health, существующие client IDs/API key auth, Foundry world online и /integration-status.
5. Возобновить sync и проверить отсутствие пропусков. Новые ключи, созданные после cutover, при необходимости перенести отдельно; не отзывать их до сверки.
6. Зафиксировать причину rollback и сохранить обе диагностические копии.

## Запреты и стоп-условия

- Не переключать production, пока staging restore/migration не доказали сохранность ключей, client IDs, scopes и pairing.
- Не использовать тестовую БД relay-v14-test.db или тестовый volume как production DB.
- Не менять одновременно Relay, Foundry module и Foundry world settings без необходимости.
- Не выполнять unpair, массовое перевыпускание ключей, сброс БД, docker compose down -v, Docker prune или удаление persistent data.
- Не открывать тестовый порт 3012 во внешний ingress; staging доступен только через SSH tunnel.
- Не менять FOUNDRY_SYNC_API_TOKEN, Discord tokens или PostgreSQL volume в рамках Relay migration.
- Не выдавать Relay write scopes; текущая интеграция приложения остаётся read-only.
- Если невозможно подтвердить формат хранения/шифрования токенов, считать перенос ключей неподтверждённым и планировать отдельную контролируемую ротацию с owner approval.

## Критерии успешной замены

- [ ] Старые Relay client IDs и ключевые записи сохранены либо контролируемо импортированы.
- [ ] Старый API key приложения работает с прежним client ID; scopes соответствуют least privilege.
- [ ] Foundry module остаётся paired, мир виден в Relay.
- [ ] Headless login Foundry v14 проходит под заданным FOUNDRY_USER_ID.
- [ ] Регистрируется один ожидаемый headless client; повторный handshake не создаёт лишний независимый клиент.
- [ ] Foundry Marches /integration-status и read-only world sync успешны.
- [ ] Rollback проверен и старые данные сохранены до окончания rollback window.
