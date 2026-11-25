# Контрольные точки 0–11

Это индекс к [полному плану](roadmap.md), не урезанный scope. Все 12 этапов сохранены. Даты/сроки не назначены; «1–2 дня» — оценка исходного спайка, не обязательство. Этапы 0–2 остаются частично выполненными; этапы 3–11 остаются в backlog.

## Обязательное решение по Foundry (2026-11-25)

По [ADR-0003](../../.agent/decisions/ADR-0003-foundry-authority-and-read-only-relay.md) Foundry является единственным источником истины для Actor и PF2e mechanics. Relay интеграция read-only. Единственное исключение — GM-authorized add/remove Item на существующем Actor с аудитом, идемпотентностью и read-back. Запись XP, уровня, валюты, характеристик, ownership и любых произвольных Actor fields запрещена. Эта граница распространяется на все этапы и задачи, включая исторический импортированный план.

## Сводка на 2026-11-25

- **Production deployment достигнут:** отдельные API, Discord bot и PostgreSQL; независимый Production volume; API только на loopback; GHCR images опубликованы по SHA; bot подключён; PostgreSQL, Relay и Foundry доступны; обнаружено 18 Actor; фоновая синхронизация включена; `/livez` и `/readyz` вернули HTTP 200. Подробности: [веха первого Production deployment](milestone-production-deployment.md).
- **Инженерный процесс:** Conventional Commits 1.0.0 закреплён; GitHub Actions Release/release notes и CD не реализованы.
- **Этап 0 — частично выполнен:** Relay и read-only world sync подтверждены. WebDAV ETag/If-Match, Campaign Codex, SSE/headless session и usage/quota остаются незавершёнными. XP/level write spike больше не требуется по ADR-0003.
- **Этап 1 — частично выполнен:** API, PostgreSQL, Compose, health checks, CI, Discord bot identity/RBAC и Production deployment существуют. Backup/restore drill, полный audit acceptance и мониторинг требуют проверки.
- **Этап 2 — частично выполнен:** Relay adapter, PF2e decoder/calculator, Actor snapshots, persistence, world sync и явная привязка Actor реализованы. Approval/ownership lifecycle, NPC lifecycle, job/retry model, item search и GM Item mutation остаются открытыми.
- **Этапы 3–11 — не начаты как полные продуктовые этапы.** Вертикальные задачи не закрывают этапы автоматически.

Smoke checks Production не являются полным тестированием отказоустойчивости, backup/restore или всех acceptance criteria.

## Текущие вертикальные задачи

- [TASK-014](../../.agent/tasks/TASK-014-production-stabilization.md) — стабилизация Production и backup/restore (ready).
- [TASK-015](../../.agent/tasks/TASK-015-foundry-item-search.md) — read-only Foundry Item search (ready).
- [TASK-016](../../.agent/tasks/TASK-016-npc-discord-webhooks.md) — GM-only NPC Discord webhooks (ready).
- [TASK-017](../../.agent/tasks/TASK-017-release-automation.md) — GitHub Actions Release (backlog).
- [TASK-018](../../.agent/tasks/TASK-018-production-deploy-gate.md) — проверка сервера и rollback gate (backlog).
- [TASK-019](../../.agent/tasks/TASK-019-gm-item-management.md) — GM Item add/remove exception (backlog).

## Полный продуктовый scope

Исходные этапы 0–11 сохранены в [roadmap](roadmap.md). Их проверки и продуктовый смысл остаются обязательными, кроме операций Foundry, прямо запрещённых ADR-0003. Приоритетная последовательность текущей работы: TASK-014 → TASK-015/TASK-016 → TASK-017 → TASK-018 → TASK-019; полные этапы продолжаются параллельно только при явном планировании зависимостей.

## Release policy

- Первый релиз: `v0.2.0-beta` (релизная линия `v0.2.0`).
- Автоматический Production CD запрещён до backup/restore drill, проверки сервера/rollback и явного подтверждения владельца.
