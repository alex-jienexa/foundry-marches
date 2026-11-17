# Проверки PostgreSQL baseline

Исходные `schema_tests.sql` (166 проверок) и `schema_race_tests.sh` (9 проверок / 3 сценария двух сессий) сохранены побайтово. Race-скрипт использует реальные параллельные psql-сессии и задержки, а не mock.

## Запуск из корня репозитория

Нужны PostgreSQL 15 или 16 с `btree_gist`, клиент `psql`, Bash. Создайте **отдельную пустую** БД вне production:

```bash
export PGHOST=localhost PGPORT=5432 PGUSER=postgres PGDATABASE=foundry_marches_test
# Пароль — через .pgpass / секрет окружения, не в Git и не в командной строке.
createdb "$PGDATABASE"
psql -X -v ON_ERROR_STOP=1 -f docs/database/schema.sql
psql -X -v ON_ERROR_STOP=1 -f tests/database/schema_tests.sql
bash tests/database/schema_race_tests.sh
```

Не перезапускайте baseline или fixtures в той же БД: для нового полного прогона создайте новую пустую БД. Очистка/удаление — вручную после проверки адреса и имени. Проверки меняют данные; не запускать на production.

Успех: `passed 166 / 166`, `all race checks passed`, у всех команд код 0. Исходный race-скрипт использует `ON_ERROR_STOP=0` для ожидаемых отказов и не везде немедленно прерывается при ошибке fixture; CI обязан запускать его только после успешных DDL и функциональных тестов. Это не исчерпывающая верификация всех возможных гонок.

[Матрица CI](../../.github/workflows/ci.yml) повторяет прогон на PG15 и PG16. [Результаты выполненной локальной проверки](../../docs/verification.md).
