---
id: FM-REQ-013
title: Сохранить полные артефакты и доступ агентов
type: constraint
status: approved
priority: must
created_at: '2026-11-17'
updated_at: '2026-11-17'
verification: test
owner: project-owner
related_tasks:
- TASK-001
---

# Statement

Перенести все четыре входных файла без потери исходного содержимого, предоставить полные рабочие разделы, SQL, тесты, навигацию PAD и локальный пакет для ручной загрузки. Не реализовывать/урезать продукт и не записывать изменения в удалённый репозиторий.

## Acceptance criteria

- [x] Хэши и размеры всех оригиналов сохранены, полное покрытие разделов проверяется.
- [x] В SQL не потеряны функции, триггеры, поздние FK и seeds.
- [x] Все этапы 0–11 и открытые вопросы доступны из контекста агентов.
- [x] DBML/ER явно производные, документация не выдаётся за реализацию.
- [x] PAD baseline и safety/protocols не повреждены.

## Verification

`python3 scripts/validate_artifacts.py`, `python3 scripts/generate_database_docs.py --check`; [точный отчёт](../../docs/verification.md). Выполнение реализации этапов не входит в это требование.
