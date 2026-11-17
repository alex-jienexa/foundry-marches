# Компоненты и границы


Модули общаются через интерфейсы и доменные события и не лезут в таблицы друг друга.

| Модуль | Отвечает за | Использует |
| --- | --- | --- |
| identity, rbac | пользователи, Discord, роли, разрешения | — |
| characters | персонажи, статусы, одобрение, NPC | identity, foundrysync |
| foundrysync | адаптер Foundry, снимки, операции | characters, ledger |
| adventures | приключения, заявки, очередь | characters, calendar |
| ledger | счета, проводки, награды, XP | — |
| economy | магазины, инвентарь, каталог предметов | ledger, factions |
| enterprise | торговое предприятие и события | ledger, factions, calendar |
| wiki | статьи, связи, версии, хроника, отчёты | identity |
| vaultsync | WebDAV-клиент, импорт и экспорт Markdown, журналы Foundry, конфликты | wiki |
| maps | карты, гексы, туман, метки | wiki |
| expeditions | маршруты, припасы, дневные переходы | maps, calendar, weather, ledger, characters |
| factions | репутация, отношения, территории | maps, wiki, characters, ledger |
| calendar, weather | игровое время, события, погода | maps |
| notifications | Discord, настройки уведомлений | все (по событиям) |
| audit, settings | журнал, настройки кампании | все |


## Предлагаемое размещение будущего кода

`src/` пока содержит только описание. Команды API/worker могут разделять один Go-модуль; бот — отдельный процесс. Точная Go-структура (`cmd/`, `internal/`, пакеты), расположение frontend и версии зависимостей определяются задачей TASK-003, не объявлены уже реализованными. Циклы conceptual-зависимостей characters ↔ foundrysync разрешать портами/событиями, не циклическими Go-import.
