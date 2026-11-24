package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync"
)

type blockingSyncExecutor struct {
	started chan struct{}
	release chan struct{}
}

func (s *blockingSyncExecutor) SyncAll(ctx context.Context) (foundrysync.SyncReport, error) {
	close(s.started)
	select {
	case <-s.release:
		return foundrysync.SyncReport{Discovered: 1, CharactersSynced: 1}, nil
	case <-ctx.Done():
		return foundrysync.SyncReport{}, ctx.Err()
	}
}

type immediateSyncExecutor struct{}

func (immediateSyncExecutor) SyncAll(context.Context) (foundrysync.SyncReport, error) {
	return foundrysync.SyncReport{Discovered: 2, SnapshotsOnly: 2}, nil
}

func TestCoordinatedSyncRunnerPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("FOUNDRY_SYNC_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("FOUNDRY_SYNC_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}

	const worldID = "00000000-0000-4000-8000-000000000094"
	_, err = db.ExecContext(ctx, `
		INSERT INTO foundry_worlds (id, name, relay_url, relay_client_id, key_secret_ref)
		VALUES ($1, 'sync runner integration test', 'https://relay.example.test', 'test-client', 'env:TEST')
		ON CONFLICT (id) DO NOTHING
	`, worldID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanupCancel()
		_, _ = db.ExecContext(cleanupCtx, "DELETE FROM foundry_sync_runs WHERE world_id = $1", worldID)
		_, _ = db.ExecContext(cleanupCtx, "DELETE FROM foundry_worlds WHERE id = $1", worldID)
	})

	blocking := &blockingSyncExecutor{started: make(chan struct{}), release: make(chan struct{})}
	first := &coordinatedSyncRunner{db: db, worldID: worldID, inner: blocking}
	firstDone := make(chan error, 1)
	go func() {
		_, err := first.SyncAll(ctx)
		firstDone <- err
	}()

	select {
	case <-blocking.started:
	case <-ctx.Done():
		t.Fatal("first sync did not start")
	}

	second := &coordinatedSyncRunner{db: db, worldID: worldID, inner: immediateSyncExecutor{}}
	if _, err := second.SyncAll(ctx); !errors.Is(err, foundrysync.ErrSyncInProgress) {
		t.Fatalf("concurrent sync error = %v, want ErrSyncInProgress", err)
	}

	close(blocking.release)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first sync failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("first sync did not finish")
	}

	// The session lock must be released so a later sync can acquire it.
	if _, err := second.SyncAll(ctx); err != nil {
		t.Fatalf("sync after lock release failed: %v", err)
	}

	var completed int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM foundry_sync_runs
		WHERE world_id = $1 AND status = 'succeeded'
	`, worldID).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if completed != 2 {
		t.Fatalf("persisted successful runs = %d, want 2", completed)
	}
}
