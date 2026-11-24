-- Self-service Foundry Actor linking with persistent GM undo.
-- Apply once to the existing foundry_marches database; safe to re-run.

CREATE TABLE IF NOT EXISTS character_link_actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    world_id uuid NOT NULL REFERENCES foundry_worlds(id),
    actor_uuid text NOT NULL,
    character_id uuid NOT NULL REFERENCES characters(id),
    owner_user_id uuid NOT NULL REFERENCES users(id),
    linked_by_user_id uuid NOT NULL REFERENCES users(id),
    state text NOT NULL DEFAULT 'linked' CHECK (state IN ('linked', 'undone')),
    created_at timestamptz NOT NULL DEFAULT now(),
    undone_at timestamptz,
    undone_by_user_id uuid REFERENCES users(id),
    CHECK (
        (state = 'linked' AND undone_at IS NULL AND undone_by_user_id IS NULL)
        OR
        (state = 'undone' AND undone_at IS NOT NULL AND undone_by_user_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS character_link_actions_actor_idx
    ON character_link_actions (world_id, actor_uuid, created_at DESC);
CREATE INDEX IF NOT EXISTS character_link_actions_owner_idx
    ON character_link_actions (owner_user_id, created_at DESC);

-- The existing slot trigger defaults to two when this setting is absent.
-- Seed the explicit campaign setting without overwriting a deliberate later change.
INSERT INTO campaign_settings (key, value)
VALUES ('max_characters_per_player', '2'::jsonb)
ON CONFLICT (key) DO NOTHING;

GRANT SELECT ON campaign_settings TO foundry_api;
GRANT SELECT, INSERT, UPDATE ON character_link_actions TO foundry_api;
GRANT UPDATE (owner_user_id, status, updated_at) ON characters TO foundry_api;
