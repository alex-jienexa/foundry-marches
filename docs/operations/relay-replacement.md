# Замена Foundry Relay с выпуском новых ключей для Production и Development

> Статус: согласованная стратегия и runbook; production-операция ещё не выполнялась. План — новый Relay deployment и новая чистая база, выпуск двух независимых API credentials (Production и Development), затем переключение relay.jiene.ru. Старый Relay и его данные сохраняются для rollback до приёмки.

## Цель и границы

Заменить текущий self-hosted ThreeHats Foundry REST API Relay на обновлённую сборку с поддержкой headless-авторизации Foundry v14. На новом Relay создать два независимых API credentials: отдельный для Production API и отдельный для Development API. Старые API keys не переносить и не использовать на новом Relay; старый Relay, его БД и конфигурацию сохранить нетронутыми на период rollback. Foundry world/module pairing переключить на новый Relay только в согласованном cutover.

Проверенная тестовая конфигурация:
- Foundry 14.367, world westmarches, system pf2e.
- Тестовый Relay 3.4.1, отдельный контейнер foundry-relay-v14-headless-test, порт 3012.
- Headless username Puffy, Foundry user ID задаётся через FOUNDRY_USER_ID.
- Успешный POST /join, загрузка #ui-left, регистрация клиента и активная сессия подтверждены логами 2026-11-27.
- Тестовая SQLite: /app/data/relay-v14-test.db; это отдельная БД, не production.

Тестовая сборка и её SQLite не являются production replacement bundle и не должны подключаться к production volume.

## Какие данные и токены различать

1. **Новые Relay API credentials.** Выпустить в новом Relay два разных client ID/API key pair: один для Production, другой для Development. Каждый ключ — только с необходимыми read scopes; не копировать один ключ в оба окружения.
2. **Foundry module pairing / Relay connection settings.** Relay URL и pairing в Foundry module переключаются на новый Relay во время cutover. Сохранить старые настройки/URL для быстрого rollback; не удалять старые записи и не выполнять unpair до момента переключения.
3. **Production приложение.** На сервере обновить только FOUNDRY_RELAY_CLIENT_ID и FOUNDRY_RELAY_API_KEY в защищённом .env.prod. FOUNDRY_RELAY_BASE_URL остаётся https://relay.jiene.ru после переключения DNS/TLS/reverse-proxy upstream.
4. **Development приложение.** На рабочей станции обновить только FOUNDRY_RELAY_CLIENT_ID и FOUNDRY_RELAY_API_KEY в локальном .env.dev. Development запускается локально; его ключ нельзя помещать на production сервер или коммитить.
5. **Токены приложения.** FOUNDRY_SYNC_API_TOKEN — отдельный Bearer token административного sync endpoint, не Relay API key. Для этой миграции его не менять. Аналогично не менять Discord и PostgreSQL credentials.
6. **Headless user ID.** FOUNDRY_USER_ID — Foundry user ID Puffy (gkYizCRHVPCjt0Xe в проверенном тесте), не Relay API key и не Relay client ID. Настроить в окружении нового Relay.

Секреты вводятся непосредственно в защищённые env files/secret store; не публиковать их в Git, логах или чате. Новые API keys показывать только один раз при создании и сразу сохранять в соответствующие секретные хранилища.

## Предварительные условия — ничего не переключая

1. Подтвердить точные права двух ключей: минимум необходимые read scopes для Actor listing/search/get и нужных Compendium read endpoints. Не выдавать entity:write, chat:write, roll:execute, file:write или structure:write.
2. Подтвердить, что Development и Production API могут обращаться к одному Relay URL и одному Foundry world, но используют разные credentials и отдельные API/DB. Проверить, что Relay quotas позволяют два клиента.
3. Подготовить новый Relay параллельно на loopback-порту 3011 (или другом свободном порту), сохранив текущий Relay на 127.0.0.1:3010. Новый deployment получает отдельный Compose project, env file и data directory/SQLite. Не использовать старую или тестовую DB.
4. Старый Relay не выключать до успешной проверки нового и согласованного cutover.

## Предварительные условия — аудит и backup

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

### B. Поднять новый Relay и выпустить credentials

1. Развернуть новую сборку под отдельным Compose project, контейнером и loopback-портом 3011; использовать отдельную чистую SQLite и отдельный data directory.
2. Настроить admin/encryption/JWT secrets и FOUNDRY_USER_ID через отдельный защищённый env file. Не использовать test DB relay-v14-test.db и не подключать новый Relay к старой production DB.
3. Создать два credentials в admin UI нового Relay: Foundry Marches Production и Foundry Marches Development. Ограничить scopes минимально необходимыми read-only правами. Если Relay поддерживает client/world restriction — ограничить каждый credential нужным client/world.
4. Сохранить Production client ID + API key только в серверном secret store/.env.prod; Development client ID + API key — только в локальном .env.dev разработчика. Не пересылать ключи через чат.
5. На новом Relay проверить оба ключа независимо: успешная аутентификация, ожидаемый client ID, нужные read endpoints, отказ write endpoints. Проверить quotas/usage для обоих ключей.
6. Проверить headless login Foundry v14 на отдельной тестовой сессии с FOUNDRY_USER_ID; подтвердить POST /join, загрузку игрового интерфейса и регистрацию одного headless client.
7. Запустить Foundry Marches Development локально с новым Development key и проверить /integration-status, read-only Actor sync и отсутствие 401/403. Затем на новом Relay проверить Production key без переключения production hostname, используя временный test hostname или защищённый локальный tunnel.

Новые credentials должны быть проверены до изменения .env.prod, .env.dev и Foundry module URL.

### C. Переключение relay.jiene.ru и Production/Development

1. Согласовать короткое окно обслуживания. Остановить/заморозить только процессы, которые создают новые Relay sessions или делают sync, чтобы избежать одновременных изменений в Relay DB. Не останавливать Foundry world без необходимости.
2. Сделать финальный согласованный backup непосредственно перед переключением.
3. В согласованное окно остановить/приостановить только задачи sync и операции, создающие Relay sessions. Foundry world не останавливать без необходимости.
4. Обновить Production .env.prod: поставить новый Production client ID и key. Обновить Development .env.dev на рабочей станции новым Development client ID и key. FOUNDRY_SYNC_API_TOKEN, DB credentials и Discord tokens не менять.
5. Сделать финальный backup старого Relay data/config и сохранить старый image/Compose/Caddy config. Старый контейнер пока остановить, но не удалять; его порт 3010 и data directory сохранить.
6. Изменить только Caddy upstream для relay.jiene.ru с 127.0.0.1:3010 на 127.0.0.1:3011 и выполнить config validation/reload. Публичный hostname и TLS оставить прежними.
7. Переключить Foundry module Relay URL/pairing на новый Relay endpoint wss://relay.jiene.ru/relay. Проверить, что Foundry world и один headless client появляются в новом Relay.
8. Запустить Production API с новым Production key и Development API с новым Development key. Проверить /integration-status и по одной read-only синхронизации в каждом окружении; убедиться, что каждый credential отображается отдельно в usage.
9. Сверить число Actors и sync reports с baseline, проверить логи на 401/403/429, pairing errors и restart loop.
10. Возобновить плановую синхронизацию. Старый Relay, DB, data directory и конфигурацию сохранять до окончания согласованного rollback window; не удалять их в ходе cutover.

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

- [ ] Новый Relay доступен через прежний HTTPS hostname relay.jiene.ru.
- [ ] Выпущены два разных API credentials: Production и Development.
- [ ] Оба ключа имеют только необходимые read-only scopes и проходят независимую проверку.
- [ ] Production .env.prod и Development .env.dev используют правильные, разные client IDs и API keys.
- [ ] Foundry module остаётся paired, мир виден в Relay.
- [ ] Headless login Foundry v14 проходит под заданным FOUNDRY_USER_ID.
- [ ] Регистрируется один ожидаемый headless client; повторный handshake не создаёт лишний независимый клиент.
- [ ] Foundry Marches /integration-status и read-only world sync успешны.
- [ ] Rollback проверен и старые данные сохранены до окончания rollback window.
