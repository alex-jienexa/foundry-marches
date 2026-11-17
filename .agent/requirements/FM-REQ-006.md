---
id: FM-REQ-006
title: 5. Wiki, Obsidian, Campaign Codex
type: functional
status: proposed
priority: must
created_at: '2026-11-17'
updated_at: '2026-11-17'
verification: test
owner: project-owner
related_tasks:
- TASK-007
---

# 5. Wiki, Obsidian, Campaign Codex

## Statement

Реализовать весь состав этапа без сокращения: Wiki с видимостью и версиями; WebDAV-клиент; журналы Foundry на чтение; затем запись

## Rationale and source

Полный [план этапов](../../docs/development/roadmap.md); [профильный полный раздел](../../docs/architecture/obsidian.md). Ссылки на сценарии исходного ТЗ сохранены, но само ТЗ отсутствует — не достраивать его по предположениям. [Открытые вопросы](../../docs/development/open-questions.md) решает project-owner перед зависимой реализацией.

## Acceptance criteria

- [ ] Проверки: `gm_only`-страница недоступна игроку (404); устаревший `rev` создаёт конфликт; правка в Obsidian появляется в Wiki
- [ ] Итог: сценарий I из ТЗ; Codex → Wiki → Obsidian без потерь
- [ ] Для всех операций выполняются серверные permissions/CanSee, журналы/инварианты и безопасность из полных разделов.
- [ ] DB → API/curl → bot → UI, если применимо этапу; источники истины, контекст и тесты обновлены.

## Verification

Фактические команды и результаты фиксируются в задаче; текст плана не служит доказательством реализации.
