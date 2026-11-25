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

Реализовать Wiki и синхронизацию знаний с Obsidian/WebDAV без потери версий и прав доступа.

## Scope

Wiki с видимостью и версиями; WebDAV-клиент; импорт/чтение журналов Foundry и Campaign Codex; экспорт между Wiki и Obsidian с rev/hash/ETag/If-Match.

## Foundry boundary

Foundry journals/Campaign Codex are read-only inputs. Никакая запись, редактирование или удаление документов/журналов Foundry не входит в задачу. Запись в WebDAV/Obsidian и собственную Wiki разрешена по их отдельным правилам версионирования.

## Acceptance criteria

- [ ] `gm_only`-страница недоступна игроку (404).
- [ ] Устаревший `rev` создаёт конфликт, не затирая новую версию.
- [ ] Правка в Obsidian появляется в Wiki.
- [ ] Foundry journals/Codex импортируются read-only с сохранением источника и provenance.
- [ ] Экспорт Wiki → Obsidian работает с ETag/If-Match и защитой от конфликтов.
- [ ] Ни один write endpoint не изменяет Foundry documents/journals.

## Verification

Access-control tests, revision conflict tests, WebDAV ETag/If-Match integration tests and static inspection of Foundry adapter.
