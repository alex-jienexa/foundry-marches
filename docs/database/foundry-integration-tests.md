# Foundry persistence: PostgreSQL integration tests

The `FoundryRepository` integration tests exercise the real schema triggers and constraints. They are opt-in so ordinary unit tests do not require a local PostgreSQL server.

## Requirements

- PostgreSQL 15 or newer (schema v0.3 uses PostgreSQL 15 features).
- The `btree_gist` extension available to the database role applying the schema.
- A **dedicated, disposable test database**. The test runner refuses a database whose name does not end in `_test`.
- The complete schema from `docs/database/schema.sql` already applied. The tests do not create, migrate, truncate, or drop schemas.

Example setup (run with a PostgreSQL role allowed to create the extension):

```bash
createdb foundry_marches_test
psql -v ON_ERROR_STOP=1 -d foundry_marches_test -f docs/database/schema.sql
```

Run from `src/backend/`:

```bash
FOUNDRY_TEST_DATABASE_URL='postgres://USER:PASSWORD@HOST:5432/foundry_marches_test?sslmode=disable' \\
  go test ./internal/characters -run PostgresIntegration -v
```

To run the whole backend suite with the integration tests enabled:

```bash
FOUNDRY_TEST_DATABASE_URL='postgres://USER:PASSWORD@HOST:5432/foundry_marches_test?sslmode=disable' go test ./...
```

The GitHub Actions `database` job applies the full schema to its disposable PostgreSQL 15/16 service and runs these integration tests automatically. Without `FOUNDRY_TEST_DATABASE_URL`, the integration tests are skipped. They use random UUIDs, only create their own fixture rows, and remove those rows with test cleanup. Never point this variable at production or a database containing campaign data.
