package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type syncRunSummary struct {
	ID              string     \`json:"id"\`
	Status          string     \`json:"status"\`
	StartedAt       time.Time  \`json:"started_at"\`
	FinishedAt      *time.Time \`json:"finished_at,omitempty"\`
	Discovered      int        \`json:"discovered"\`
	CharactersSynced int       \`json:"characters_synced"\`
	SnapshotsOnly   int        \`json:"snapshots_only"\`
	Failed          int        \`json:"failed"\`
	ErrorMessage    string     \`json:"error_message,omitempty"\`
}

func syncHistoryHandler(db *sql.DB, worldID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, \`
			SELECT id::text, status, started_at, finished_at,
			       discovered, characters_synced, snapshots_only, failed,
			       COALESCE(error_message, '')
			FROM foundry_sync_runs
			WHERE world_id = $1
			ORDER BY started_at DESC
			LIMIT 20
		\`, worldID)
		if err != nil {
			http.Error(w, "sync history unavailable", http.StatusServiceUnavailable)
			return
		}
		defer rows.Close()

		runs := make([]syncRunSummary, 0, 20)
		for rows.Next() {
			var run syncRunSummary
			if err := rows.Scan(
				&run.ID, &run.Status, &run.StartedAt, &run.FinishedAt,
				&run.Discovered, &run.CharactersSynced, &run.SnapshotsOnly,
				&run.Failed, &run.ErrorMessage,
			); err != nil {
				http.Error(w, "sync history unavailable", http.StatusInternalServerError)
				return
			}
			runs = append(runs, run)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, "sync history unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(struct {
			Runs []syncRunSummary \`json:"runs"\`
		}{Runs: runs})
	}
}
