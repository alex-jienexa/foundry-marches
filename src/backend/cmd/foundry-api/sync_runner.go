package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync"
)

// coordinatedSyncRunner uses a PostgreSQL session advisory lock so manual and
// scheduled syncs remain mutually exclusive across API replicas and restarts.
type coordinatedSyncRunner struct {
	db    *sql.DB
	inner interface {
		SyncAll(context.Context) (foundrysync.SyncReport, error)
	}
}

const foundrySyncLockID int64 = 0x464D53594E43

func (r *coordinatedSyncRunner) SyncAll(ctx context.Context) (foundrysync.SyncReport, error) {
	if r == nil || r.db == nil || r.inner == nil {
		return foundrysync.SyncReport{}, errors.New("Foundry sync runner is not initialized")
	}

	// A transaction-scoped advisory lock is automatically released on rollback,
	// including when the caller context is cancelled or the connection is lost.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return foundrysync.SyncReport{}, fmt.Errorf("begin transaction for Foundry sync lock: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var acquired bool
	if err := tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", foundrySyncLockID).Scan(&acquired); err != nil {
		return foundrysync.SyncReport{}, fmt.Errorf("acquire Foundry sync lock: %w", err)
	}
	if !acquired {
		return foundrysync.SyncReport{}, foundrysync.ErrSyncInProgress
	}

	return r.inner.SyncAll(ctx)
}
