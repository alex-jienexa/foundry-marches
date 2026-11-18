-- Discord bot identity and RBAC permissions.
-- Apply once to an existing database after deploying this revision:
--   psql -X -v ON_ERROR_STOP=1 -d foundry_marches -f docs/database/migrations/0002_discord_bot_auth.sql
-- Safe to re-run. Does not modify or delete campaign data.

-- Base players can create their own characters.
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, 'characters.create'
FROM roles r
WHERE r.key = 'player'
ON CONFLICT DO NOTHING;

-- GMs can manage Foundry links and character approvals, plus game-facing content.
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key
FROM roles r
JOIN permissions p ON p.key IN (
  'characters.approve',
  'characters.edit_others',
  'characters.retire',
  'adventures.create',
  'adventures.publish',
  'adventures.approve_signups',
  'adventures.edit_others',
  'rewards.distribute',
  'wiki.edit',
  'wiki.view_secrets',
  'maps.edit',
  'maps.reveal_fog',
  'expeditions.manage',
  'factions.manage',
  'foundry.manage'
)
WHERE r.key = 'gm'
ON CONFLICT DO NOTHING;

-- Admin and owner are trusted campaign operators.
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key
FROM roles r
CROSS JOIN permissions p
WHERE r.key IN ('admin', 'owner')
ON CONFLICT DO NOTHING;

-- The API uses the least-privilege foundry_api database role.
GRANT SELECT ON users, discord_accounts, roles, permissions,
  role_permissions, user_roles, discord_role_map, characters, foundry_actors
  TO foundry_api;
GRANT INSERT, UPDATE ON users, discord_accounts TO foundry_api;
GRANT INSERT, UPDATE, DELETE ON user_roles, discord_role_map TO foundry_api;
GRANT INSERT ON audit_log TO foundry_api;
GRANT USAGE, SELECT ON SEQUENCE audit_log_id_seq TO foundry_api;
