# FoundryMarches — контекст проекта

## Назначение и статус

Русскоязычная платформа West Marches PF2e для игроков и мастеров: интеграции Foundry VTT, Discord, Obsidian через существующий WebDAV и Campaign Codex.

Проект перешёл от импорта спецификации к реализации. В рабочей ветке существуют Go backend, PostgreSQL/Compose, базовый REST API, Relay adapter, PF2e decoder/calculator и слой сохранения Actor snapshots. Владелец подтвердил успешную синхронизацию мира Foundry: получены 11 Actor, сохранены как snapshots. Все 11 пока `snapshot_only`, поскольку нет явной связи с персонажами приложения. Полный продукт и этапы 0–11 не завершены.

Актуальные результаты: [verification](../../docs/verification.md). Дорожная карта: [roadmap](../../docs/development/roadmap.md).

## Полный scope

Пользователи и RBAC; персонажи, одобрение, NPC, лимиты и владение; Foundry-снимки, механики PF2e, монеты и предметы; приключения, очередь/предложения и Discord; double-entry журнал, награды, магазины; Wiki, хроника, отчёты, Obsidian/Codex; гекс-карта, туман, календарь/погода; экспедиции/припасы; фракции/репутация/территории; торговое предприятие; статистика/графы/расширенные отчёты. Безопасность, аудит, бэкапы и восстановление обязательны. Все этапы 0–11 сохранены.

Исходное ТЗ v1.0 с дополнениями не приложено: [пробелы исходников](../../docs/development/source-gaps.md). Не реконструировать отсутствующие требования по догадкам.

## Термины

| Термин | Смысл |
|---|---|
| Adventure | игровая сессия/приключение с регистрацией и очередью |
| Expedition | многодневный поход; не обязан соответствовать одному adventure |
| Actor | документ Foundry, владеет состоянием листа |
| Ledger | неизменяемая история и намерение изменения; свои валюты — в приложении |
| Integration job | каноническая операция внешней системы, не River job |
| CanSee | единая серверная политика видимости |
| Game day | абсолютный bigint-номер дня, не timestamp реального времени |
| Vault | существующее WebDAV-хранилище Markdown |

## Архитектурные ограничения

- Foundry — авторитетный источник механик PF2e. Текущий интеграционный контур read-only.
- Не создавать application characters автоматически и не назначать владельцев по косвенным признакам.
- Связь Actor ↔ application character создаётся только явной операцией с проверками уникальности и полномочий.
- Внутренний WorldID приложения — UUID. Relay identifier `westmarches` не заменяет его.
- Сохранять исходный Actor snapshot, неизвестные поля, null и пустые объекты.
- `Actor.items[]` — реальные документы Actor; `system.items` внутри Class/Ancestry/Background может быть таблицей ссылок, а не списком инвентаря.
- Не использовать локализованные названия Item как идентификаторы; предпочитать IDs, slugs, UUID/compendiumSource.
- Не исполнять произвольные PF2e Rule Elements; unsupported expressions/predicates обозначать как incomplete.
- Не записывать XP/уровни в Foundry без отдельного архитектурного решения и разрешения владельца.
- Не удалять Compose database volume и не выполнять destructive DB operations без отдельного разрешения.
- Секреты не хранить в Git и не передавать в чат.

## Стек и структура

Реализация размещается в `src/`: Go backend — `src/backend/`, frontend — `src/frontend/`, отдельный bot — `src/bot/`. Go module находится в `src/backend/go.mod`. Документация и PAD — вне `src/`.

Целевой стек исходного плана: Go, PostgreSQL, SolidJS/TypeScript, Discord bot, Caddy и Docker Compose. Не считать компоненты установленными/завершёнными только потому, что они присутствуют в целевой архитектуре.

## Проверки и начало работы

- Backend: из `src/backend/` — `go test ./...`, `go vet ./...`.
- Runtime: см. [Foundry world sync API guide](../../docs/integrations/foundry-world-sync-api.md).
- PostgreSQL baseline и артефакты: см. [verification](../../docs/verification.md).
- Состояние задач и следующий checkpoint: `../state/current.yaml` и `../state/HANDOFF.md`.
- Перед изменениями сверять актуальный HEAD рабочей ветки и читать назначенную задачу.
