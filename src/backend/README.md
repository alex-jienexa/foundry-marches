# Backend Go module

This directory is the root of the application's Go module. Backend commands, workers, and internal domain/integration packages live below this directory.

Run backend tests from here:

```sh
go test ./...
go vet ./...
```

Shared documentation and Foundry JSON fixtures remain in the repository-level `docs/` tree; tests resolve fixtures relative to their source file, not the process working directory.

## Run the Foundry integration API

For the complete local stack (PostgreSQL + API), use Docker Compose from the repository root. See [the integration guide](../../docs/integrations/foundry-world-sync-api.md).

To run only the API process manually, first configure the environment described in that guide, then run:

```sh
go run ./cmd/foundry-api
```
