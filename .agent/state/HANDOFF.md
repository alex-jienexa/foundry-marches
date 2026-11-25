# Передача контекста FoundryMarches

## Состояние на 2026-11-25

Рабочая ветка: `feat/conventional-commits`. Последний commit, созданный при обновлении roadmap: `bd985641d2af3c9e0de67d8673426d1f20b971e9`. Перед следующими изменениями перепроверить HEAD и состояние ветки.

Ветка создана от `main` commit `8a69f78fb2a1bafe44fb8203a3a5e82656a0d5eb`. Не создавать PR и не выполнять merge: владелец делает это самостоятельно.

## Подтверждённый Production

- Production развёрнут в `/opt/foundry-marches/`, Compose project `foundry-marches-prod`.
- Отдельные API, Discord bot и PostgreSQL 18 Alpine; новая независимая Production БД, без переноса Dev данных.
- API доступен на loopback `127.0.0.1:8081`; PostgreSQL не опубликован на host port.
- Отдельные network и volume; API/Bot/PostgreSQL имеют memory limits и container hardening.
- Образы опубликованы в GHCR с immutable tag `sha-8a69f78`.
- Discord bot подключён к production guild.
- `/integration-status`: PostgreSQL, Foundry Relay и Foundry доступны; Foundry обнаруживает 18 Actor.
- Фоновая Foundry sync включена с интервалом 5 минут.
- `GET /livez` и `GET /readyz` вернули HTTP 200 `ok`.
- Владелец подтвердил работоспособность бота; пользовательская обратная связь ещё ожидается.
- Production deployment зафиксирован в `docs/development/milestone-production-deployment.md`.

### Изоляция и безопасность

Не трогать существующий Compose project `jienexyz`, его PostgreSQL volume/данные, Caddy, Redis, `catalog-container` и `rosochkinsad.ru`. Не удалять volumes и не выполнять destructive DB operations. Не просить/не публиковать значения `.env.prod`, токенов и ключей. Не менять firewall или Caddy без отдельного запроса.

На сервере Ubuntu 24.04, 1 vCPU, около 1.9 GiB RAM и 19 GiB root disk; свободное место после установки образов было около 4.7 GB. Наблюдать RAM/OOM и диск. Backup script поставлен в deployment bundle, но успешный backup/restore drill ещё не подтверждён.

## Инженерный процесс

Ветка содержит обязательную политику Conventional Commits:
- `.agent/protocols/COMMIT_CONVENTIONS.md`;
- обновлённый `AGENTS.md`;
- обновлённый `.agent/agents/IMPLEMENTER.md`.

Генерация release notes, GitHub Actions Release и CD не реализованы.

## Следующие приоритеты

См. [актуальную roadmap](../../docs/development/roadmap.md).

1. Получить обратную связь игроков, проверить backup schedule и тестовое восстановление, наблюдать ресурсы Production.
2. Реализовать read-only поиск предметов Foundry через Relay: Actor/Compendium Items, фильтры, пагинация и безопасная деградация при недоступности Relay.
3. Реализовать GM-only сообщения от имени НПС в Discord через webhooks; динамические имя/аватар, разрешённые каналы, аудит и безопасное управление credentials.
4. Реализовать GitHub Actions Release: генерация release notes из Conventional Commits, проверка версии и публикация первого prerelease **`v0.2.0-beta`** (релизная линия начинается с `v0.2.0`).
5. Проверить новый сервер и процедуру безопасного обновления/rollback. Только после подтверждения владельца рассматривать автоматический CD на Production.
6. Продолжать полные этапы 0–11 по roadmap. Не считать этапы 0–2 завершёнными только из-за наличия Production.

## Архитектурные ограничения

- Foundry — авторитетный источник PF2e механик и исходных данных; текущая интеграция read-only.
- Не изменять Foundry Actor и не реализовывать XP/level write без отдельного архитектурного решения и разрешения владельца.
- Сохранять raw Actor snapshots, неизвестные поля, null и пустые объекты.
- Различать реальные `Actor.items[]` и Compendium references в `system.items`.
- Не использовать локализованные имена Item как идентификаторы; использовать IDs, slugs, UUID и compendiumSource.
- Не исполнять произвольные PF2e Rule Elements.
- Не создавать персонажей автоматически и не выводить ownership косвенно.
- Не использовать Foundry sync token как пользовательскую авторизацию.
- Не включать Relay write capabilities в приложение без отдельного дизайна авторизации, аудита, идемпотентности и разрешения.
- Go backend находится в `src/backend/`; исходники приложения — только под `src/`. Документация и PAD — вне `src/`.

## Разработка и проверка

- Не утверждать, что тесты/сборка прошли, пока нет фактического результата владельца или CI.
- Пользователь предпочитает пошаговый цикл: дать команды → дождаться вывода → следующий шаг.
- Не создавать PR/merge.
- Для GitHub изменений использовать GitHub connector, предварительно проверить branch HEAD и релевантные файлы. Commit messages — Conventional Commits.
