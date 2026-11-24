# Production operations

This document describes the operational model for the API, Discord bot, Foundry Relay and PostgreSQL. The application remains read-only with respect to Foundry Actor documents: background sync reads Actor snapshots and stores them in PostgreSQL; it never writes to Foundry.

## Environment isolation: development and production

Run two independent Compose projects from the same repository revision and Dockerfiles:

| Setting | Development | Production |
|---|---|---|
| Compose project | `foundry-marches-dev` | `foundry-marches-prod` |
| Environment file | `.env.dev` | `.env.prod` |
| Discord application/bot | Dedicated test application | Dedicated production application |
| Discord guild | Test server | Campaign server |
| PostgreSQL | Own container and volume | Own container and volume |
| API host port | 8080 by default | 8081 by default |
| Compose files | `compose.yaml` | `compose.yaml` + `compose.production.yaml` |

Do not point the development bot at the production API token, production guild, or production database. Use a separate Discord application and test guild. The two Compose project names create separate service names and PostgreSQL volumes. Never rename a project while expecting it to reuse the old database volume.

The production project starts with its own database volume; it does not copy data from the development project. If the existing local database contains campaign data that must become production data, do not assume that creating `.env.prod` migrates it. Take a verified backup and plan an explicit import into the new production volume, with a restore rehearsal first. This runbook intentionally does not automate a destructive database replacement or use `pg_restore --clean`.

The development template is `.env.example`; the production template is `.env.prod.example`. Copy them to ignored local files. The existing Compose project historically used the name `foundry-marches` when no project name was set. If your current local database must remain attached to the existing volume, keep that project name in its local environment; the template's `foundry-marches-dev` name intentionally creates a separate, empty development database.

```sh
cp .env.example .env.dev
cp .env.prod.example .env.prod
chmod 600 .env.dev .env.prod
```

Generate independent secrets for every environment. For example:

```sh
openssl rand -hex 32
```

Use separate values for `POSTGRES_PASSWORD`, `POSTGRES_APP_PASSWORD`, `FOUNDRY_SYNC_API_TOKEN`, and `DISCORD_BOT_API_TOKEN`. The two service tokens must differ. Never commit the resulting `.env.dev` or `.env.prod` files. On an already initialized PostgreSQL volume, changing `POSTGRES_APP_PASSWORD` in the environment file does not automatically change the existing `foundry_api` database role password; rotate the database role and application environment together using a controlled maintenance procedure.

## Background Foundry synchronization

The API owns automatic synchronization. The Discord bot no longer starts a competing sync during its own startup.

- The API runs one initial sync after startup, then repeats it every `FOUNDRY_SYNC_INTERVAL_SECONDS` (default 300 seconds; allowed range 30–86400).
- Set `FOUNDRY_SYNC_ENABLED=false` to disable only the scheduled sync. The authorized manual `/foundry-sync` command and sync HTTP endpoint remain available.
- A PostgreSQL session-level advisory lock serializes scheduled and manual syncs across API replicas. The runner holds a dedicated database connection for the full sync and explicitly releases the lock before returning the connection to the pool. If the process or connection dies, PostgreSQL releases the session lock automatically.
- Individual Actor fetch failures are recorded in the sync report; one failed Actor does not abort the entire world refresh.
- The Relay Actor-search and Actor-document requests used by sync retry transient network/read failures, HTTP 429 and HTTP 5xx responses up to three attempts with a bounded delay. Non-retryable client errors are returned immediately.
- API and bot handle SIGINT/SIGTERM and attempt graceful shutdown.

The in-memory mutex inside the sync service is still useful within one process; the PostgreSQL advisory lock provides cross-process coordination. If PostgreSQL is unavailable, sync cannot acquire its lock and will fail safely rather than running without coordination.

### Integration job queue boundary

The database schema contains a generic `integration_jobs` queue for future Foundry operations. This deployment does **not** start a worker for that queue. The current integration contract is read-only, while some queued job kinds (for example granting an Item) would write to Foundry. Do not enable a generic worker until the project explicitly approves a write-capable Foundry integration, idempotency rules, failure compensation and operator review. The scheduled Actor snapshot refresh is the active background worker at this stage.

## Health and monitoring

- `GET /livez`: API process liveness; it does not depend on PostgreSQL or Foundry.
- `GET /readyz`: API readiness; verifies PostgreSQL connectivity.
- `GET /healthz`: compatibility alias for readiness, used by older deployment checks.
- `/integration-status` in Discord checks PostgreSQL, Relay HTTP reachability and an authenticated Foundry Actor listing.
- When `DISCORD_ADMIN_CHANNEL_ID` is configured, the bot periodically checks integrations and posts state changes and rate-limited command errors to the private admin channel.

Foundry/Relay outages do not make the API unready: existing synchronized character data remains available while the source is temporarily offline. Monitor Relay and Foundry using `/integration-status` and the private admin channel.

## Pre-deployment validation

Before merging or deploying, run the backend checks from the repository root:

```sh
cd src/backend
test -z "$(gofmt -l .)"
go test ./...
go vet ./...
go build ./cmd/foundry-api ./cmd/discord-bot
cd ../..
```

Validate the development Compose configuration and build both images before testing the bot in the development guild:

```sh
docker compose --env-file .env.dev -p foundry-marches-dev \
  -f compose.yaml config --quiet
docker compose --env-file .env.dev -p foundry-marches-dev \
  -f compose.yaml build
```

Do not treat a successful image build as proof of runtime connectivity. Verify registration, character lookup, manual sync, automatic sync logs and the private audit channel in the development guild.

## First deployment

### 1. Prepare the server

Install Docker Engine with the Compose v2 plugin, Git, and a host backup destination with sufficient free space. Keep the repository in a dedicated directory. Configure a firewall so PostgreSQL is not exposed publicly. The API is bound to loopback by default and should be exposed through a TLS reverse proxy if external HTTP access is required.

### 2. Configure production

Copy `.env.prod.example` to `.env.prod`, replace every placeholder, and set the real production Foundry world UUID, Relay credentials, production Discord application token/application ID/guild ID, private admin channel, campaign role and trusted admin IDs.

The production Discord bot must be invited to the production guild with only the permissions it needs: View Channels, Send Messages, Embed Links, Use Application Commands, and Manage Roles only if it assigns the campaign role. Place its role above the campaign role it must grant. Restrict the admin audit channel to trusted masters and the bot.

### 3. Validate Compose configuration

From the repository root:

```sh
docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml config --quiet
```

Do not paste rendered Compose configuration into public logs: it may contain interpolated secrets.

### 4. Existing database versus fresh installation

For a fresh PostgreSQL volume, Compose mounts the baseline schema, bootstrap script, and migrations 0002/0003/0004 into `/docker-entrypoint-initdb.d`. PostgreSQL executes these only during first initialization.

For an existing database, take a backup first, then apply the incremental migrations in order if they have not already been applied:

```sh
docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml exec -T postgres \
  psql -X -v ON_ERROR_STOP=1 -U foundry_admin -d foundry_marches \
  < docs/database/migrations/0002_discord_bot_auth.sql

docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml exec -T postgres \
  psql -X -v ON_ERROR_STOP=1 -U foundry_admin -d foundry_marches \
  < docs/database/migrations/0003_discord_character_self_connect.sql

docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml exec -T postgres \
  psql -X -v ON_ERROR_STOP=1 -U foundry_admin -d foundry_marches \
  < docs/database/migrations/0004_foundry_sync_runs.sql
```

Both migrations are designed to be safe to re-run. Do not reinitialize the database, remove its volume, or run `docker compose down -v`.

### 5. Start production services

```sh
docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml up -d --build

docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml ps
```

Inspect logs:

```sh
docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml logs --tail=100 api discord-bot postgres
```

The API must become healthy. The bot should report a successful Discord connection and command registration. The API performs the first world sync immediately after startup; check the API log for its report. Then run `/integration-status` and `/foundry-sync` in Discord as a GM to verify the production path.

### 6. Start development independently

Use a separate test Discord application and test guild. Configure `.env.dev` with `COMPOSE_PROJECT_NAME=foundry-marches-dev`, `APP_ENV=development`, a distinct set of tokens and the test guild ID:

```sh
docker compose --env-file .env.dev -p foundry-marches-dev \
  -f compose.yaml up -d --build

docker compose --env-file .env.dev -p foundry-marches-dev \
  -f compose.yaml ps
```

Development does not load the production Compose hardening overlay. It still has its own PostgreSQL volume and network. Keep development data disposable, but never delete production volumes as part of development cleanup.

## Backups

The repository includes `scripts/backup-postgres.sh`. It creates a PostgreSQL custom-format dump, verifies that the archive can be listed, writes a SHA-256 checksum, and applies a configurable local retention period.

Run manually:

```sh
ENV_FILE=.env.prod RETENTION_DAYS=30 bash scripts/backup-postgres.sh
```

The default backup directory is `backups/`, which is ignored by Git. Protect it with host filesystem permissions and copy backups off the server to independent storage. A backup stored only on the same server is not disaster recovery.

Schedule the script with the host's cron or systemd timer, for example daily at 03:15:

```cron
15 3 * * * cd /opt/foundry-marches && ENV_FILE=.env.prod RETENTION_DAYS=30 bash scripts/backup-postgres.sh >> /var/log/foundry-marches-backup.log 2>&1
```

Ensure the backup destination has enough free space and monitor failures. Retention only deletes matching old dump files inside the configured backup directory; it does not touch Docker volumes.

### Restore rehearsal

Do not test restore against the live production database. Create a separate temporary database on the production PostgreSQL instance or, preferably, an isolated recovery instance:

```sh
docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml exec postgres \
  createdb -U foundry_admin foundry_marches_restore_check

cat backups/foundry_marches_YYYYMMDDTHHMMSSZ.dump | \
  docker compose --env-file .env.prod -p foundry-marches-prod \
  -f compose.yaml -f compose.production.yaml exec -T postgres \
  pg_restore -U foundry_admin -d foundry_marches_restore_check --no-owner
```

Validate the restored schema and representative records before considering the backup process proven. Remove only the explicitly named temporary restore-check database after the rehearsal is complete; never substitute the production database name in cleanup commands.

## Updating production

1. Verify the change in the development Compose project and test guild.
2. Merge the reviewed change into `main` through the normal PR process.
3. On the server, fetch the approved `main` revision.
4. Create and verify a fresh PostgreSQL backup.
5. Review new migrations; apply them in order before starting code that depends on them.
6. Validate Compose configuration and run `up -d --build`.
7. Check container health, logs, `/integration-status`, character roster, and one manual sync.
8. Keep the previous image/revision and verified backup available until the new deployment is accepted.

Never use `docker compose down -v`, `docker volume rm`, or database drop/recreate commands as an update or troubleshooting step.

## Operational limits and future work

- Automatic sync is periodic and full-world, not incremental. Tune the interval to the world size and Relay capacity; do not set it below 30 seconds.
- Each scheduled or manual sync is recorded in `foundry_sync_runs` with status, timestamps, counts, error text and the structured report. The latest 20 runs are available through `GET /api/internal/discord/sync-history`, protected by `DISCORD_BOT_API_TOKEN`. Metrics are not yet exported to Prometheus.
- There is no generic integration queue worker by design while Foundry writes are prohibited.
- Backups are initiated by the host scheduler, not by an in-process goroutine.
- Production deployment still requires a real TLS reverse proxy, off-host backup storage, host monitoring and a rehearsed restore procedure. Compose alone does not provide those external services.
