package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var unlocked bool
		if err := conn.QueryRowContext(unlockCtx, "SELECT pg_advisory_unlock($1)", foundrySyncLockID).Scan(&unlocked); err != nil {
			// Closing the dedicated connection also releases the session lock.
			return
		}
	}()

	return r.inner.SyncAll(ctx)
}
