# Foundry world synchronization API

The backend exposes a read-only world synchronization handler at:

- `POST /api/integrations/foundry/sync`

The handler is implemented in `src/backend/internal/foundrysync/httpapi` and mounted by `src/backend/cmd/foundry-api`. The endpoint is protected by a dedicated Bearer token from `FOUNDRY_SYNC_API_TOKEN` (minimum 32 characters). This is a bootstrap credential; when the main application auth/permission middleware is implemented, replace it with the `foundry.manage` permission check rather than exposing a second permanent auth model.

## Docker Compose quick start

The repository root contains a Compose stack for PostgreSQL and the API. It creates a dedicated database role, applies the canonical schema and inserts the initial Foundry World record on the first database initialization.

1. Create a local environment file:

   ```sh
   cp .env.example .env
   ```

2. Generate secrets and put them in `.env`. Hex values avoid connection-string escaping problems:

   ```sh
   openssl rand -hex 32 # POSTGRES_PASSWORD
   openssl rand -hex 32 # POSTGRES_APP_PASSWORD
   openssl rand -hex 32 # FOUNDRY_SYNC_API_TOKEN
   ```

   Set `FOUNDRY_RELAY_CLIENT_ID` and `FOUNDRY_RELAY_API_KEY` to the values from the existing Relay. The default Relay URL is `https://relay.jiene.ru`.

3. Start the stack from the repository root:

   ```sh
   docker compose up -d --build
   docker compose ps
   docker compose logs -f api
   ```

4. Check the local API:

   ```sh
   curl -fsS http://127.0.0.1:8080/healthz
   ```

   The API is published only on host loopback. PostgreSQL is not published on a host port.

5. Trigger a world sync when ready:

   ```sh
   curl -fsS -X POST http://127.0.0.1:8080/api/integrations/foundry/sync \
     -H "Authorization: Bearer $FOUNDRY_SYNC_API_TOKEN"
   ```

   Load `.env` into the shell before using the final command, or substitute the token through a protected local environment. Never paste the token into source control or chat.

### Database lifecycle and safety

- The named `postgres_data` volume persists the database across container recreation and `docker compose down`.
- The schema and bootstrap scripts run only when PostgreSQL initializes an empty data directory. Editing `schema.sql` later does not automatically migrate an existing volume; use an explicit migration for subsequent schema changes.
- `docker compose down -v` permanently deletes the Compose database volume. Use it only when intentionally discarding all local database data.
- The Compose database is separate from any PostgreSQL instance already installed on the host.
- The default `FOUNDRY_WORLD_ID` is a stable local development UUID. Keep it unchanged while retaining the volume. If you change it after initialization, the existing database row is not rewritten automatically and the API will refuse to start.
- The Relay API key is passed to the API at runtime. The `foundry_worlds.key_secret_ref` column stores only the reference `env:FOUNDRY_RELAY_API_KEY`, never the key itself.
- The application DB role can read tables and write only `characters`, `foundry_actors` and `inventory_items`, which are required by the current sync repository.

### Manual run without Docker

Run from `src/backend/` after applying `docs/database/schema.sql` and creating a matching `foundry_worlds` row:

```sh
export DATABASE_URL='postgres://USER:PASSWORD@HOST:5432/DB?sslmode=disable'
export FOUNDRY_WORLD_ID='00000000-0000-4000-8000-000000000000'
export FOUNDRY_RELAY_BASE_URL='https://relay.example.test'
export FOUNDRY_RELAY_CLIENT_ID='your-foundry-client-id'
export FOUNDRY_RELAY_API_KEY='your-relay-api-key'
export FOUNDRY_SYNC_API_TOKEN='generate-a-random-secret-of-at-least-32-characters'
export HTTP_ADDR='127.0.0.1:8080'
go run ./cmd/foundry-api
```

The bootstrap defaults to loopback binding. Put it behind the existing HTTPS reverse proxy; do not expose it directly to the public network. Health check: `GET /healthz`. Sync: `POST /api/integrations/foundry/sync` with `Authorization: Bearer $FOUNDRY_SYNC_API_TOKEN`. Secrets belong in the host's secret manager or protected environment file and must never be committed.

## What the operation does

1. Calls Relay `GET /search` with `filter=Actor`, `excludeCompendiums=true`, `minified=true` and `limit=500`.
2. Fetches each discovered world Actor through `GET /get?uuid=Actor...`.
3. Decodes the Actor document and checks that its canonical UUID matches the search result.
4. Stores a snapshot for every discovered Actor in `foundry_actors`.
5. If the Actor has type `character` and is already linked to an application character, refreshes the character projection and inventory using the existing atomic repository transaction.
6. Stores NPCs, loot actors, vehicles and other unlinked Actor types as snapshots only. It does not create application characters, assign owners, change statuses or infer player ownership.
7. Records per-Actor failures and continues with the remaining Actors.

A world search that reaches Relay's documented 500-result limit is rejected rather than treated as a complete world inventory. This prevents a truncated search from being mistaken for a successful full sync.

## Response

A successful HTTP request returns a JSON report containing:

- `worldId`, `startedAt`, `finishedAt`
- `discovered`
- `charactersSynced`
- `snapshotsOnly`
- `failed`
- `results[]`, with Actor UUID, name, actual Foundry Actor type, disposition or per-Actor error

A discovery-level Relay error returns HTTP 502. A second concurrent sync for the same service instance returns HTTP 409. Per-Actor failures do not fail the HTTP operation; they appear in the report and increment `failed`.

## Current boundaries

- The operation is read-only with respect to Foundry. It only uses Relay search and get operations.
- The endpoint is synchronous and processes Actors sequentially to avoid overloading the GM/Relay session. A future job/worker API should replace the synchronous request for very large worlds.
- Relay search currently exposes at most 500 results in one request. Pagination is not assumed.
- Actors removed from Foundry are not yet pruned from `foundry_actors`; deletion reconciliation is a separate, explicit policy.
- NPC and other non-character Actor snapshots are persisted, but their system-specific projections are not yet decoded into dedicated application entities.
- The initial bootstrap is a dedicated integration service, not yet the full application API. It does not implement user sessions, role checks or per-user access control; the Bearer token grants access to the single administrative sync operation.
