---
id: TASK-007
title: 5. Wiki, Obsidian, Campaign Codex
status: backlog
priority: high
assignee: unassigned
created_at: '2026-11-17'
updated_at: '2026-11-25'
estimate: null
parent: null
depends_on:
- TASK-006
requirement_ids:
- FM-REQ-006
files_owned:
- src/
- tests/
- docs/
- .github/workflows/ci.yml
labels:
- stage-5
- planned
---

# 5. Wiki, Obsidian, Campaign Codex

## Objective

Реализовать Wiki и синхронизацию знаний с Obsidian/WebDAV без потери версий и прав доступа, включая read-only импорт материалов Campaign Codex и Foundry journals.

## Scope

- Wiki с видимостью, версиями и контролем доступа.
- WebDAV-клиент для Obsidian.
- Синхронизация Wiki ↔ Obsidian с rev/hash и защитой от конфликтов.
- Проверка WebDAV ETag/If-Match, условных запросов и поведения при конфликте версий.
- Read-only импорт/чтение Foundry journals и Campaign Codex с сохранением источника, UUID и provenance.
- Получение и анализ обезличенных образцов Campaign Codex, определение поддерживаемого формата импорта.
- Экспорт собственных Wiki-материалов в Obsidian/WebDAV по отдельным правилам версионирования.

## Foundry boundary

Foundry journals/Campaign Codex are read-only inputs. Никакая запись, редактирование или удаление документов/журналов Foundry не входит в задачу. Запись в WebDAV/Obsidian и собственную Wiki разрешена по их отдельным правилам версионирования.

## Acceptance criteria

- [ ] gm_only-страница недоступна игроку (404).
- [ ] Устаревший rev создаёт конфликт, не затирая новую версию.
- [ ] Правка в Obsidian появляется в Wiki.
- [ ] WebDAV ETag/If-Match проверен интеграционными тестами; конфликт не перезаписывает более новую версию.
- [ ] Foundry journals/Codex импортируются read-only с сохранением источника и provenance.
- [ ] Получены и описаны обезличенные образцы Campaign Codex либо зафиксировано отсутствие доступных образцов и согласован fallback.
- [ ] Экспорт Wiki → Obsidian работает с ETag/If-Match и защитой от конфликтов.
- [ ] Ни один write endpoint не изменяет Foundry documents/journals.

## Verification

Access-control tests, revision conflict tests, WebDAV ETag/If-Match integration tests, import fixture tests for Campaign Codex/Foundry journals and static inspection of Foundry adapter.
