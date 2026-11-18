# Foundry world synchronization API

The backend exposes a first read-only world synchronization handler at:

- `POST /api/integrations/foundry/sync`

The handler is implemented in `src/backend/internal/foundrysync/httpapi`. The application HTTP composition root must mount it at the path above and place it behind the existing application authentication/authorization middleware. Only administrators with the `foundry.manage` permission should be allowed to invoke it. The handler deliberately does not implement a second, competing authentication system.

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
- The handler package is ready to mount, but this repository does not yet have an API server composition root. Mounting and production configuration belong to the next server-bootstrap task.
