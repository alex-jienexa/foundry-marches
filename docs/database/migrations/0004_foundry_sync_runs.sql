-- Durable execution history for scheduled and manual Foundry world refreshes.
-- Safe to re-run; does not modify existing campaign data.

CREATE TABLE IF NOT EXISTS foundry_sync_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    world_id uuid NOT NULL REFERENCES foundry_worlds(id),
    status text NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'succeeded', 'partial', 'failed', 'interrupted')),
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    discovered integer NOT NULL DEFAULT 0 CHECK (discovered >= 0),
    characters_synced integer NOT NULL DEFAULT 0 CHECK (characters_synced >= 0),
    snapshots_only integer NOT NULL DEFAULT 0 CHECK (snapshots_only >= 0),
    failed integer NOT NULL DEFAULT 0 CHECK (failed >= 0),
    error_message text,
    report_json jsonb,
    CHECK (
        (status = 'running' AND finished_at IS NULL)
        OR
        (status <> 'running' AND finished_at IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS foundry_sync_runs_world_started_idx
    ON foundry_sync_runs (world_id, started_at DESC);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'foundry_api') THEN
        EXECUTE 'GRANT SELECT, INSERT, UPDATE ON foundry_sync_runs TO foundry_api';
    END IF;
END
$$;
