\set ON_ERROR_STOP on

-- Values are supplied by the PostgreSQL container environment. The Relay key itself
-- is never stored in this table; key_secret_ref is only a reference to the runtime secret.
\getenv world_id FOUNDRY_WORLD_ID
\getenv world_name FOUNDRY_WORLD_NAME
\getenv relay_url FOUNDRY_RELAY_BASE_URL
\getenv relay_client_id FOUNDRY_RELAY_CLIENT_ID
\getenv relay_secret_ref FOUNDRY_RELAY_SECRET_REF
\getenv app_password POSTGRES_APP_PASSWORD

SELECT format('CREATE ROLE foundry_api LOGIN PASSWORD %L', :'app_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'foundry_api')
\gexec

ALTER ROLE foundry_api LOGIN PASSWORD :'app_password';

GRANT CONNECT ON DATABASE foundry_marches TO foundry_api;
GRANT USAGE ON SCHEMA public TO foundry_api;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO foundry_api;
GRANT INSERT, UPDATE, DELETE
  ON TABLE characters, foundry_actors, inventory_items
  TO foundry_api;

INSERT INTO foundry_worlds (
  id, name, relay_url, relay_client_id, key_secret_ref, is_primary, status
)
VALUES (
  :'world_id'::uuid,
  :'world_name',
  :'relay_url',
  :'relay_client_id',
  :'relay_secret_ref',
  true,
  'unknown'
);
