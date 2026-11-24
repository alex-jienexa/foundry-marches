package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync"
)

type syncExecutor interface {
	SyncAll(context.Context) (foundrysync.SyncReport, error)
}

// coordinatedSyncRunner serializes manual and scheduled refreshes across API
// replicas with a PostgreSQL session advisory lock. The dedicated connection
// avoids holding an idle transaction open while Relay requests are in flight.
type coordinatedSyncRunner struct {
	db      *sql.DB
	worldID string
	inner   syncExecutor
}

const (
	foundrySyncLockID       int64 = 0x464D53594E43
	foundrySyncStaleAfter         = 15 * time.Minute
	foundrySyncUnlockTimeout      = 3 * time.Second
	foundrySyncPersistTimeout     = 5 * time.Second
)

func (r *coordinatedSyncRunner) SyncAll(ctx context.Context) (foundrysync.SyncReport, error) {
	if r == nil || r.db == nil || r.inner == nil || r.worldID == "" {
		return foundrysync.SyncReport{}, errors.New("Foundry sync runner is not initialized")
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return foundrysync.SyncReport{}, fmt.Errorf("reserve PostgreSQL connection for Foundry sync lock: %w", err)
	}
	defer conn.Close()

	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", foundrySyncLockID).Scan(&acquired); err != nil {
		return foundrysync.SyncReport{}, fmt.Errorf("acquire Foundry sync lock: %w", err)
	}
	if !acquired {
		return foundrysync.SyncReport{}, foundrysync.ErrSyncInProgress
	}
	defer releaseFoundrySyncLock(conn)

	runID, err := r.startRun(ctx, conn)
	if err != nil {
		return foundrysync.SyncReport{}, err
	}

	report, syncErr := r.inner.SyncAll(ctx)
	if persistErr := r.finishRun(runID, conn, report, syncErr); persistErr != nil {
		persistErr = fmt.Errorf("persist Foundry sync run %s: %w", runID, persistErr)
		if syncErr != nil {
			return report, errors.Join(syncErr, persistErr)
		}
		return report, persistErr
	}
	return report, syncErr
}

func (r *coordinatedSyncRunner) startRun(ctx context.Context, conn *sql.Conn) (string, error) {
	_, err := conn.ExecContext(ctx, `
		UPDATE foundry_sync_runs
		SET status = 'interrupted',
		    finished_at = now(),
		    error_message = COALESCE(error_message, 'API process stopped before the sync run was finalized')
		WHERE world_id = $1
		  AND status = 'running'
		  AND started_at < now() - $2::interval
	`, r.worldID, foundrySyncStaleAfter.String())
	if err != nil {
		return "", fmt.Errorf("mark stale Foundry sync runs as interrupted: %w", err)
	}

	var runID string
	err = conn.QueryRowContext(ctx, `
		INSERT INTO foundry_sync_runs (world_id, status)
		VALUES ($1, 'running')
		RETURNING id::text
	`, r.worldID).Scan(&runID)
	if err != nil {
		return "", fmt.Errorf("create Foundry sync run history record: %w", err)
	}
	return runID, nil
}

func (r *coordinatedSyncRunner) finishRun(
	runID string,
	conn *sql.Conn,
	report foundrysync.SyncReport,
	syncErr error,
) error {
	status := "succeeded"
	var errorMessage any
	if syncErr != nil {
		status = "failed"
		errorMessage = syncErr.Error()
	} else if report.Failed > 0 {
		status = "partial"
	}

	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode sync report: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), foundrySyncPersistTimeout)
	defer cancel()
	result, err := conn.ExecContext(ctx, `
		UPDATE foundry_sync_runs
		SET status = $2,
		    finished_at = now(),
		    discovered = $3,
		    characters_synced = $4,
		    snapshots_only = $5,
		    failed = $6,
		    error_message = $7,
		    report_json = $8::jsonb
		WHERE id = $1
	`, runID, status, report.Discovered, report.CharactersSynced, report.SnapshotsOnly, report.Failed, errorMessage, string(reportJSON))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("expected to update one sync run, updated %d", affected)
	}
	return nil
}

func releaseFoundrySyncLock(conn *sql.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), foundrySyncUnlockTimeout)
	defer cancel()

	var released bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", foundrySyncLockID).Scan(&released); err != nil {
		log.Printf("Failed to release Foundry sync advisory lock; discarding PostgreSQL connection: %v", err)
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		return
	}
	if !released {
		log.Printf("Foundry sync advisory lock was not held by its dedicated connection during release")
	}
}
