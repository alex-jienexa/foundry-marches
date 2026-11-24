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

-- Set the campaign-wide active character limit to the requested two slots.
INSERT INTO campaign_settings (key, value)
VALUES ('max_characters_per_player', '2'::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();

GRANT SELECT ON campaign_settings TO foundry_api;
GRANT SELECT, INSERT, UPDATE ON character_link_actions TO foundry_api;
GRANT UPDATE (owner_user_id, status, updated_at) ON characters TO foundry_api;

COMMENT ON COLUMN foundry_actors.owner_confirmed_at IS
  'Timestamp when the current application owner claimed or was explicitly assigned this Foundry Actor. This does not verify Foundry account identity; player self-claims can be reversed by a GM.';
