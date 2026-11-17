---
id: FM-REQ-011
title: 10. Торговое предприятие
type: functional
status: proposed
priority: must
created_at: '2026-11-17'
updated_at: '2026-11-17'
verification: test
owner: project-owner
related_tasks:
- TASK-012
---

# 10. Торговое предприятие

## Statement

Реализовать весь состав этапа без сокращения: карточка, склад, персонал, события, действия, правила автоматики

## Rationale and source

Полный [план этапов](../../docs/development/roadmap.md); [профильный полный раздел](../../docs/architecture/overview.md). Ссылки на сценарии исходного ТЗ сохранены, но само ТЗ отсутствует — не достраивать его по предположениям. [Открытые вопросы](../../docs/development/open-questions.md) решает project-owner перед зависимой реализацией.

## Acceptance criteria

- [ ] Проверки: правило с `requires_gm_approval = false` отвергается БД
- [ ] Итог: сценарий H из ТЗ
- [ ] Для всех операций выполняются серверные permissions/CanSee, журналы/инварианты и безопасность из полных разделов.
- [ ] DB → API/curl → bot → UI, если применимо этапу; источники истины, контекст и тесты обновлены.

## Verification

Фактические команды и результаты фиксируются в задаче; текст плана не служит доказательством реализации.
