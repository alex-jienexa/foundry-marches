# Синтаксические проверки диаграмм

Это tooling документации, **не frontend приложения**. Node.js 22, зависимости закреплены `package-lock.json`.

Из корня:

```bash
npm ci --prefix tools/docs --no-audit --no-fund
npm run check --prefix tools/docs
```

Проверяется разбор DBML (и соответствие числа таблиц/колонок/FK AST-инвентарю), а также синтаксис всех четырёх Mermaid. Это не визуальная проверка вёрстки и не round-trip в исполняемый PostgreSQL. Для DDL остаются отдельные реальные SQL-тесты.
