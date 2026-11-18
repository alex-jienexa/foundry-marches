# Foundry world synchronization API

The backend exposes a first read-only world synchronization handler at:

- `POST /api/integrations/foundry/sync`

The handler is implemented in `src/backend/internal/foundrysync/httpapi` and is mounted by the runnable bootstrap at `src/backend/cmd/foundry-api`. The bootstrap currently protects this administrative endpoint with a dedicated Bearer token from `FOUNDRY_SYNC_API_TOKEN` (minimum 32 characters). This is a bootstrap credential; when the main application auth/permission middleware is implemented, replace it with the `foundry.manage` permission check rather than exposing a second permanent auth model.

## What the operation does

1. Calls Relay `GET /search` with `filter=Actor`, `excludeCompendiums=true`, `minified=true` and `limit=500`.
2. Fetches each discovered world Actor through `GET /get?uuid=Actor...`.
3. Decodes the Actor document and checks that its canonical UUID matches the search result.
4. Stores a snapshot for every discovered Actor in `foundry_actors`.
5. If the Actor has type `character` and is already linked to an application character, refreshes the character projection and inventory using the existing atomic repository transaction.
6. Stores NPCs, loot actors, vehicles and other unlinked Actor types as snapshots only. It does not create application characters, assign owners, change statuses or infer player ownership.
7. Records per-Actor failures and continues with the remaining Actors.

A world search that reaches Relay's documented 500-result limit is rejected rather than treated as a complete world inventory. This prevents a truncated search from being mistaken for a successful full sync.

## Running the bootstrap API

Run from `src/backend/` after applying `docs/database/schema.sql` and creating the matching `foundry_worlds` row:

```sh
export DATABASE_URL='postgres://USER:PASSWORD@HOST:5432/DB?sslmode=disable'
export FOUNDRY_WORLD_ID='00000000-0000-0000-0000-000000000000'
export FOUNDRY_RELAY_BASE_URL='https://relay.example.test'
export FOUNDRY_RELAY_CLIENT_ID='your-foundry-client-id'
export FOUNDRY_RELAY_API_KEY='your-relay-api-key'
export FOUNDRY_SYNC_API_TOKEN='generate-a-random-secret-of-at-least-32-characters'
export HTTP_ADDR='127.0.0.1:8080'
go run ./cmd/foundry-api
```

The bootstrap defaults to loopback binding. Put it behind the existing HTTPS reverse proxy; do not expose it directly to the public network. Health check: `GET /healthz`. Sync: `POST /api/integrations/foundry/sync` with `Authorization: Bearer $FOUNDRY_SYNC_API_TOKEN`. Secrets belong in the host's secret manager or protected environment file and must never be committed.

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
