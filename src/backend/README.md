# Backend Go module

This directory is the root of the application's Go module. Backend commands, workers, and internal domain/integration packages live below this directory.

Run backend tests from here:

```sh
go test ./...
go vet ./...
```

Shared documentation and Foundry JSON fixtures remain in the repository-level `docs/` tree; tests resolve fixtures relative to their source file, not the process working directory.
