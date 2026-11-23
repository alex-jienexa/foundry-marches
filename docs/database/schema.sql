-- =====================================================================
-- West Marches PF2e platform: PostgreSQL schema (v0.3)
-- Conventions:
--   * every table: uuid PK, created_at; mutable tables also updated_at (trigger)
--   * game dates are absolute day numbers (bigint) since the calendar epoch;
--     the calendar service converts day number <-> year/month/day
--   * append-only tables reject UPDATE/DELETE (trigger at the end of the file)
--   * statuses are text + CHECK (stable sets) or lookup tables (extensible sets)
--   * money/xp are bigint in the smallest unit of the currency
-- Requires PostgreSQL 15+ (UNIQUE NULLS NOT DISTINCT, ON DELETE SET NULL (column))
-- and the btree_gist extension (contrib).
--
-- Who owns which value (the rule behind the whole integration):
--   Foundry actor owns : name, portrait, ancestry, heritage, background, class, level, xp,
--                        hp, ac, speed, size, attributes, skills, inventory items, coins.
--                        The app mirrors them (characters.*, inventory_items) and may change them
--                        only through an integration job; the session variable app.writer =
--                        'foundry_sync' marks the sync code path (see characters guard).
--   App ledger owns   : the history and intent of every change (ledger_*, rewards, purchases),
--                        custom currencies (reputation, tokens, faction resources), shop prices,
--                        stock, queues, expeditions, factions, wiki.
--   Mismatch           : never overwritten silently; recorded as ledger kind 'external_adjustment'.
-- =====================================================================

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE DOMAIN visibility AS text
  CHECK (VALUE IN ('draft', 'gm_only', 'roles', 'public'));

-- ---------------------------------------------------------------------
-- 1. Identity, roles, permissions
-- ---------------------------------------------------------------------
CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  display_name text NOT NULL,
  avatar_url text,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'left_guild')),
  timezone text NOT NULL DEFAULT 'UTC',
  preferred_schedule text,
  registered_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE discord_accounts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL UNIQUE REFERENCES users(id),
  discord_id text NOT NULL UNIQUE,
  username text NOT NULL,
  avatar_hash text,
  in_guild boolean NOT NULL DEFAULT true,
  oauth_refresh_token_enc bytea,            -- encrypted at application level
  linked_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  key text NOT NULL UNIQUE,                 -- owner, admin, gm, player, ...
  name text NOT NULL,
  is_system boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
  key text PRIMARY KEY,                     -- characters.approve, rewards.distribute, ...
  description text NOT NULL
);

CREATE TABLE role_permissions (
  role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_key text NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_key)
);

CREATE TABLE user_roles (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  granted_by uuid REFERENCES users(id),
  granted_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, role_id)
);

CREATE TABLE discord_role_map (
  guild_id text NOT NULL,
  discord_role_id text NOT NULL,
  role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (guild_id, discord_role_id)
);

CREATE TABLE sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash bytea NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  ip inet,
  user_agent text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE service_tokens (               -- the Discord bot and other services
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL UNIQUE,
  token_hash bytea NOT NULL UNIQUE,
  scopes text[] NOT NULL DEFAULT '{}',
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE player_gm_notes (              -- visible to GMs only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  author_id uuid NOT NULL REFERENCES users(id),
  body text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE notification_prefs (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  PRIMARY KEY (user_id, kind)
);

CREATE TABLE login_log (                    -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid REFERENCES users(id),
  ok boolean NOT NULL,
  ip inet,
  detail text,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------
-- 2. Campaign settings, audit
-- ---------------------------------------------------------------------
CREATE TABLE campaign_settings (            -- max_characters_per_player, max_party_level_spread, ...
  key text PRIMARY KEY,
  value jsonb NOT NULL,
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE campaign_settings_history (    -- append-only
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  key text NOT NULL,
  old_value jsonb,
  new_value jsonb,
  changed_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE FUNCTION log_setting_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO campaign_settings_history (key, old_value, new_value, changed_by)
  VALUES (NEW.key, CASE WHEN TG_OP = 'UPDATE' THEN OLD.value END, NEW.value, NEW.updated_by);
  RETURN NEW;
END $$;
CREATE TRIGGER trg_log_setting_change AFTER INSERT OR UPDATE ON campaign_settings
  FOR EACH ROW EXECUTE FUNCTION log_setting_change();

CREATE TABLE audit_log (                    -- append-only
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  actor_user_id uuid REFERENCES users(id),
  actor_kind text NOT NULL CHECK (actor_kind IN ('user', 'bot', 'system', 'foundry', 'obsidian')),
  action text NOT NULL,
  entity_type text NOT NULL,
  entity_id text,
  old_value jsonb,
  new_value jsonb,
  result text NOT NULL DEFAULT 'ok',
  request_id text,
  ip inet
);
CREATE INDEX audit_log_entity_idx ON audit_log (entity_type, entity_id);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_user_id, created_at DESC);

CREATE TABLE backups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  kind text NOT NULL CHECK (kind IN ('database', 'files', 'maps', 'settings', 'full')),
  path text NOT NULL,
  size_bytes bigint,
  sha256 text,
  verified_at timestamptz,
  verify_result text,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------
-- 3. Calendar and game clock
-- ---------------------------------------------------------------------
CREATE TABLE calendars (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  year_name text,
  epoch_name text,
  is_active boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX calendars_one_active ON calendars ((true)) WHERE is_active;

-- Calendar rules are versioned. Game dates are stored as absolute day numbers, and a version
-- governs the days from effective_from_day up to the next version. The anchor says which
-- year/month/day/weekday the first day of the version is, so old dates keep their reading.
CREATE TABLE calendar_versions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  calendar_id uuid NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
  version integer NOT NULL CHECK (version > 0),
  effective_from_day bigint NOT NULL,
  anchor_year integer NOT NULL,
  anchor_month_ord smallint NOT NULL DEFAULT 1,
  anchor_day smallint NOT NULL DEFAULT 1,
  anchor_weekday_ord smallint NOT NULL DEFAULT 1,
  sealed_at timestamptz,                    -- set when the game clock reaches the version; then immutable
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (calendar_id, version),
  UNIQUE (calendar_id, effective_from_day)
);

CREATE TABLE calendar_months (              -- intercalary periods are months with is_intercalary
  version_id uuid NOT NULL REFERENCES calendar_versions(id) ON DELETE CASCADE,
  ord smallint NOT NULL CHECK (ord > 0),
  name text NOT NULL,
  days smallint NOT NULL CHECK (days > 0),
  is_intercalary boolean NOT NULL DEFAULT false,  -- does not take part in the week cycle
  PRIMARY KEY (version_id, ord)
);

CREATE TABLE calendar_weekdays (
  version_id uuid NOT NULL REFERENCES calendar_versions(id) ON DELETE CASCADE,
  ord smallint NOT NULL CHECK (ord > 0),
  name text NOT NULL,
  PRIMARY KEY (version_id, ord)
);

CREATE TABLE calendar_leap_rules (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  version_id uuid NOT NULL REFERENCES calendar_versions(id) ON DELETE CASCADE,
  every_n_years integer NOT NULL CHECK (every_n_years > 0),
  offset_years integer NOT NULL DEFAULT 0,
  month_ord smallint NOT NULL,
  extra_days smallint NOT NULL DEFAULT 1,
  FOREIGN KEY (version_id, month_ord) REFERENCES calendar_months(version_id, ord)
);

CREATE TABLE game_clock (                   -- single row
  id smallint PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  calendar_id uuid NOT NULL REFERENCES calendars(id),
  current_day bigint NOT NULL,
  mode text NOT NULL DEFAULT 'manual' CHECK (mode IN ('manual', 'auto')),
  time_ratio numeric(10, 3) NOT NULL DEFAULT 0 CHECK (time_ratio >= 0),  -- game days per real day
  last_auto_tick timestamptz,
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- A version can be sealed only when it is a complete calendar: months 1..n, weekdays 1..m,
-- an anchor that exists. Because the game clock seals versions, a broken calendar cannot start a game.
CREATE FUNCTION calendar_validate_version(p_version uuid) RETURNS void LANGUAGE plpgsql AS $$
DECLARE v calendar_versions%ROWTYPE; nm integer; mm integer; nw integer; mw integer; anchor_days integer;
BEGIN
  SELECT * INTO v FROM calendar_versions WHERE id = p_version;
  SELECT count(*), COALESCE(max(ord), 0) INTO nm, mm FROM calendar_months WHERE version_id = p_version;
  SELECT count(*), COALESCE(max(ord), 0) INTO nw, mw FROM calendar_weekdays WHERE version_id = p_version;
  IF nm = 0 OR nm <> mm THEN RAISE EXCEPTION 'calendar version %: months must be numbered 1..n without gaps', p_version; END IF;
  IF nw = 0 OR nw <> mw THEN RAISE EXCEPTION 'calendar version %: weekdays must be numbered 1..n without gaps', p_version; END IF;
  SELECT days INTO anchor_days FROM calendar_months WHERE version_id = p_version AND ord = v.anchor_month_ord;
  IF anchor_days IS NULL OR v.anchor_day < 1 OR v.anchor_day > anchor_days THEN
    RAISE EXCEPTION 'calendar version %: the anchor date does not exist', p_version;
  END IF;
  IF v.anchor_weekday_ord < 1 OR v.anchor_weekday_ord > nw THEN
    RAISE EXCEPTION 'calendar version %: the anchor weekday does not exist', p_version;
  END IF;
END $$;

CREATE FUNCTION calendar_version_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    IF EXISTS (SELECT 1 FROM calendar_versions
                WHERE calendar_id = NEW.calendar_id AND effective_from_day >= NEW.effective_from_day) THEN
      RAISE EXCEPTION 'a calendar version must start after the previous one';
    END IF;
    IF EXISTS (SELECT 1 FROM calendar_versions WHERE calendar_id = NEW.calendar_id)
       AND EXISTS (SELECT 1 FROM game_clock WHERE calendar_id = NEW.calendar_id
                    AND current_day >= NEW.effective_from_day) THEN
      RAISE EXCEPTION 'a new calendar version must start after the current game day';
    END IF;
    RETURN NEW;
  END IF;
  IF OLD.sealed_at IS NOT NULL THEN
    RAISE EXCEPTION 'calendar version % is sealed; create a new version instead', OLD.id
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  IF NEW.calendar_id <> OLD.calendar_id THEN
    RAISE EXCEPTION 'a calendar version cannot move to another calendar';
  END IF;
  IF NEW.effective_from_day <> OLD.effective_from_day OR NEW.version <> OLD.version THEN
    IF EXISTS (SELECT 1 FROM calendar_versions o
                WHERE o.calendar_id = NEW.calendar_id AND o.id <> NEW.id
                  AND ((o.version < NEW.version AND o.effective_from_day >= NEW.effective_from_day)
                    OR (o.version > NEW.version AND o.effective_from_day <= NEW.effective_from_day))) THEN
      RAISE EXCEPTION 'version numbers and start days of a calendar must stay in the same order';
    END IF;
    IF EXISTS (SELECT 1 FROM calendar_versions o WHERE o.calendar_id = NEW.calendar_id AND o.id <> NEW.id
                AND o.version < NEW.version)
       AND EXISTS (SELECT 1 FROM game_clock WHERE calendar_id = NEW.calendar_id
                    AND current_day >= NEW.effective_from_day) THEN
      RAISE EXCEPTION 'a calendar version cannot start on or before the current game day';
    END IF;
  END IF;
  IF NEW.sealed_at IS NOT NULL THEN PERFORM calendar_validate_version(NEW.id); END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_calendar_version_guard BEFORE INSERT OR UPDATE OR DELETE ON calendar_versions
  FOR EACH ROW EXECUTE FUNCTION calendar_version_guard();

-- Checks both the old and the new version: a row cannot be moved out of a sealed version
-- into an unsealed one (or the other way round).
CREATE FUNCTION calendar_child_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP <> 'INSERT' THEN
    IF EXISTS (SELECT 1 FROM calendar_versions WHERE id = OLD.version_id AND sealed_at IS NOT NULL) THEN
      RAISE EXCEPTION 'calendar version % is sealed; create a new version instead', OLD.version_id
        USING ERRCODE = 'integrity_constraint_violation';
    END IF;
  END IF;
  IF TG_OP <> 'DELETE' THEN
    IF EXISTS (SELECT 1 FROM calendar_versions WHERE id = NEW.version_id AND sealed_at IS NOT NULL) THEN
      RAISE EXCEPTION 'calendar version % is sealed; create a new version instead', NEW.version_id
        USING ERRCODE = 'integrity_constraint_violation';
    END IF;
  END IF;
  RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END $$;
CREATE TRIGGER trg_calendar_months_guard BEFORE INSERT OR UPDATE OR DELETE ON calendar_months
  FOR EACH ROW EXECUTE FUNCTION calendar_child_guard();
CREATE TRIGGER trg_calendar_weekdays_guard BEFORE INSERT OR UPDATE OR DELETE ON calendar_weekdays
  FOR EACH ROW EXECUTE FUNCTION calendar_child_guard();
CREATE TRIGGER trg_calendar_leap_guard BEFORE INSERT OR UPDATE OR DELETE ON calendar_leap_rules
  FOR EACH ROW EXECUTE FUNCTION calendar_child_guard();

CREATE FUNCTION seal_calendar_versions() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  UPDATE calendar_versions SET sealed_at = now()
   WHERE calendar_id = NEW.calendar_id AND sealed_at IS NULL AND effective_from_day <= NEW.current_day;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_seal_calendar AFTER INSERT OR UPDATE ON game_clock
  FOR EACH ROW EXECUTE FUNCTION seal_calendar_versions();

CREATE TABLE game_clock_history (           -- append-only
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  from_day bigint NOT NULL,
  to_day bigint NOT NULL,
  reason text,
  changed_by uuid REFERENCES users(id),
  adventure_id uuid,                        -- FK added at the end
  expedition_id uuid,                       -- FK added at the end
  created_at timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------
-- 4. Wiki, chronicle
-- ---------------------------------------------------------------------
CREATE TABLE wiki_pages (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  type text NOT NULL CHECK (type IN ('article', 'npc', 'player', 'settlement', 'region', 'faction',
    'organization', 'shop', 'enterprise', 'map', 'timeline', 'adventure', 'folder')),
  title text NOT NULL,
  slug text NOT NULL UNIQUE,
  parent_id uuid REFERENCES wiki_pages(id),
  visibility visibility NOT NULL DEFAULT 'draft',
  attributes jsonb NOT NULL DEFAULT '{}',
  body_md text NOT NULL DEFAULT '',
  gm_notes text NOT NULL DEFAULT '',        -- never returned to players
  current_rev integer NOT NULL DEFAULT 1,
  archived_at timestamptz,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX wiki_pages_type_idx ON wiki_pages (type);
CREATE INDEX wiki_pages_parent_idx ON wiki_pages (parent_id);

CREATE TABLE wiki_page_roles (              -- visibility = 'roles'
  page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (page_id, role_id)
);

CREATE TABLE wiki_versions (                -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  page_id uuid NOT NULL REFERENCES wiki_pages(id),
  rev integer NOT NULL,
  title text NOT NULL,
  body_md text NOT NULL,
  gm_notes text NOT NULL DEFAULT '',
  attributes jsonb NOT NULL DEFAULT '{}',
  author_id uuid REFERENCES users(id),
  source text NOT NULL CHECK (source IN ('web', 'obsidian', 'import', 'system')),
  content_hash text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (page_id, rev)
);

CREATE TABLE wiki_aliases (                 -- old slugs and titles: renames do not break links
  page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  alias text NOT NULL UNIQUE
);

CREATE TABLE wiki_relations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  from_page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  to_page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  kind text NOT NULL,                       -- lives_in, serves, controls, owns, located_in, ...
  attrs jsonb NOT NULL DEFAULT '{}',
  visibility visibility NOT NULL DEFAULT 'gm_only',
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (from_page_id, to_page_id, kind)
);

CREATE TABLE wiki_backlinks (               -- derived from body_md on save
  from_page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  to_page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  PRIMARY KEY (from_page_id, to_page_id)
);

CREATE TABLE assets (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  sha256 text NOT NULL UNIQUE,
  mime text NOT NULL,
  path text NOT NULL,
  size_bytes bigint NOT NULL,
  uploaded_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE chronicle_entries (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  game_day bigint NOT NULL,
  title text NOT NULL,
  body_md text NOT NULL DEFAULT '',
  visibility visibility NOT NULL DEFAULT 'draft',
  state text NOT NULL DEFAULT 'draft' CHECK (state IN ('draft', 'published')),
  wiki_page_id uuid REFERENCES wiki_pages(id),
  adventure_id uuid,                        -- FK added at the end
  expedition_id uuid,                       -- FK added at the end
  author_id uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX chronicle_day_idx ON chronicle_entries (game_day);

CREATE TABLE calendar_events (              -- holidays, fairs, deadlines, faction events; dates are absolute game days
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  calendar_id uuid NOT NULL REFERENCES calendars(id),
  title text NOT NULL,
  description text,
  kind text,
  start_day bigint NOT NULL,
  duration_days integer NOT NULL DEFAULT 1 CHECK (duration_days > 0),
  recurrence jsonb,                         -- e.g. {"every":"year","month":3,"day":15}; expanded by the calendar service
  visibility visibility NOT NULL DEFAULT 'gm_only',
  location_wiki_page_id uuid REFERENCES wiki_pages(id),
  adventure_id uuid,                        -- FK added at the end
  fired_at timestamptz,                     -- the worker fired the event when the clock reached it
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX calendar_events_day_idx ON calendar_events (start_day);

CREATE TABLE calendar_event_links (
  event_id uuid NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
  link_type text NOT NULL CHECK (link_type IN ('character', 'wiki', 'faction', 'adventure')),
  ref_id uuid NOT NULL,
  PRIMARY KEY (event_id, link_type, ref_id)
);

-- ---------------------------------------------------------------------
-- 5. Maps (hex grid, fog of war)
-- ---------------------------------------------------------------------
CREATE TABLE maps (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  width integer NOT NULL CHECK (width > 0),
  height integer NOT NULL CHECK (height > 0),
  hex_orientation text NOT NULL DEFAULT 'pointy' CHECK (hex_orientation IN ('pointy', 'flat')),
  hex_size_miles numeric(8, 2) NOT NULL CHECK (hex_size_miles > 0),   -- scale of one hex
  background_asset_id uuid REFERENCES assets(id),
  wiki_page_id uuid REFERENCES wiki_pages(id),
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE terrain_types (                -- extensible without code changes
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  key text NOT NULL UNIQUE,
  name text NOT NULL,
  color text NOT NULL DEFAULT '#888888',
  icon text,
  travel_cost_mult numeric(4, 2) NOT NULL DEFAULT 1 CHECK (travel_cost_mult > 0),
  passable boolean NOT NULL DEFAULT true,
  is_custom boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE map_regions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  name text NOT NULL,
  wiki_page_id uuid REFERENCES wiki_pages(id),
  color text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, map_id)                       -- target of composite foreign keys
);

CREATE TABLE hexes (
  map_id uuid NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  q integer NOT NULL,                       -- axial coordinates
  r integer NOT NULL,
  terrain_id uuid REFERENCES terrain_types(id),
  elevation_ft integer,
  region_id uuid,
  weather_region_id uuid,                   -- FK added after weather_regions
  fog_state text NOT NULL DEFAULT 'hidden' CHECK (fog_state IN ('hidden', 'explored', 'gm_only')),
  gm_notes text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (map_id, q, r),
  FOREIGN KEY (region_id, map_id) REFERENCES map_regions(id, map_id) ON DELETE SET NULL (region_id)
);
CREATE INDEX hexes_fog_idx ON hexes (map_id, fog_state);

CREATE TABLE hex_features (                 -- settlements, ruins, dungeons, ...
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL,
  q integer NOT NULL,
  r integer NOT NULL,
  kind text NOT NULL,
  name text NOT NULL,
  wiki_page_id uuid REFERENCES wiki_pages(id),
  visibility visibility NOT NULL DEFAULT 'gm_only',
  icon text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (map_id, q, r) REFERENCES hexes(map_id, q, r) ON DELETE CASCADE
);

CREATE TABLE map_linear_features (          -- roads, rivers, bridges, borders
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('road', 'river', 'bridge', 'border')),
  name text,
  path jsonb NOT NULL,                      -- [[q,r], [q,r], ...]
  visibility visibility NOT NULL DEFAULT 'gm_only',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE map_pins (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL,
  q integer NOT NULL,
  r integer NOT NULL,
  label text NOT NULL,
  wiki_page_id uuid REFERENCES wiki_pages(id),
  visibility visibility NOT NULL DEFAULT 'gm_only',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (map_id, q, r) REFERENCES hexes(map_id, q, r) ON DELETE CASCADE
);

CREATE TABLE routes (                       -- named routes, independent of expeditions
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  name text NOT NULL,
  waypoints jsonb NOT NULL,                 -- [[q,r], ...]
  visibility visibility NOT NULL DEFAULT 'gm_only',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE exploration_history (          -- append-only: who revealed what and when
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL,
  q integer NOT NULL,
  r integer NOT NULL,
  previous_fog text NOT NULL,
  new_fog text NOT NULL,
  revealed_by uuid REFERENCES users(id),
  adventure_id uuid,                        -- FK added at the end
  expedition_id uuid,                       -- FK added at the end
  game_day bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (map_id, q, r) REFERENCES hexes(map_id, q, r)
);
CREATE INDEX exploration_hex_idx ON exploration_history (map_id, q, r);

-- ---------------------------------------------------------------------
-- 6. Weather
-- ---------------------------------------------------------------------
CREATE TABLE climates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  key text NOT NULL UNIQUE,
  name text NOT NULL,
  params jsonb NOT NULL DEFAULT '{}',       -- seasonal curves, transition probabilities
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE weather_regions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  name text NOT NULL,
  climate_id uuid NOT NULL REFERENCES climates(id),
  map_region_id uuid,
  base_elevation_ft integer,
  forecast_visible_to_players boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE weather_regions ADD UNIQUE (id, map_id);
ALTER TABLE weather_regions ADD FOREIGN KEY (map_region_id, map_id)
  REFERENCES map_regions(id, map_id) ON DELETE SET NULL (map_region_id);
ALTER TABLE hexes ADD CONSTRAINT hexes_weather_region_fk
  FOREIGN KEY (weather_region_id, map_id) REFERENCES weather_regions(id, map_id)
  ON DELETE SET NULL (weather_region_id);

CREATE TABLE weather_types (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  key text NOT NULL UNIQUE,
  name text NOT NULL,
  severity smallint NOT NULL DEFAULT 0,
  travel_mult numeric(4, 2) NOT NULL DEFAULT 1 CHECK (travel_mult > 0),
  visibility_mult numeric(4, 2) NOT NULL DEFAULT 1 CHECK (visibility_mult > 0),
  icon text,
  is_custom boolean NOT NULL DEFAULT false
);

CREATE TABLE weather_records (              -- forecast and actual weather by region and day
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL,
  weather_region_id uuid NOT NULL,
  game_day bigint NOT NULL,
  weather_type_id uuid NOT NULL REFERENCES weather_types(id),
  temp_c numeric(5, 1),
  cloud_pct smallint CHECK (cloud_pct BETWEEN 0 AND 100),
  wind_kmh smallint CHECK (wind_kmh >= 0),
  wind_dir_deg smallint CHECK (wind_dir_deg BETWEEN 0 AND 359),
  visibility_m integer,
  humidity_pct smallint CHECK (humidity_pct BETWEEN 0 AND 100),
  snow_depth_cm smallint,
  special jsonb NOT NULL DEFAULT '{}',
  state text NOT NULL DEFAULT 'forecast' CHECK (state IN ('forecast', 'actual')),
  source text NOT NULL CHECK (source IN ('generated', 'manual')),
  seed bigint,
  set_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (weather_region_id, game_day, state),   -- one forecast and one actual record per day
  UNIQUE (id, map_id),
  FOREIGN KEY (weather_region_id, map_id) REFERENCES weather_regions(id, map_id) ON DELETE CASCADE
);

-- ---------------------------------------------------------------------
-- 7. Foundry, characters
-- ---------------------------------------------------------------------
CREATE TABLE foundry_worlds (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  relay_url text NOT NULL,
  relay_client_id text NOT NULL,
  key_secret_ref text NOT NULL,             -- reference to the secret, never the key itself
  foundry_version text,
  pf2e_system_version text,
  is_primary boolean NOT NULL DEFAULT false,
  status text NOT NULL DEFAULT 'unknown' CHECK (status IN ('unknown', 'online', 'offline')),
  last_seen_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX foundry_one_primary ON foundry_worlds ((true)) WHERE is_primary;
-- relay_url comes from the administrator only (never from player input); the application also
-- validates it against an allow-list of hosts before every call.

CREATE TABLE characters (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id uuid REFERENCES users(id),
  former_owner_user_id uuid REFERENCES users(id),
  name text NOT NULL,
  portrait_url text,
  status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending_approval', 'active',
    'unavailable', 'dead', 'retired', 'npc', 'archived')),
  ancestry text,
  heritage text,
  background text,
  class text,
  level smallint NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 30),   -- mirror of the Foundry value
  xp integer NOT NULL DEFAULT 0 CHECK (xp >= 0),                       -- mirror of the Foundry value
  level_up_pending boolean NOT NULL DEFAULT false,
  hp_current integer,
  hp_max integer,
  ac integer,
  speed_ft smallint,                        -- needed by expeditions
  size text,
  attributes jsonb NOT NULL DEFAULT '{}',
  skills jsonb NOT NULL DEFAULT '{}',
  gm_notes text,
  npc_wiki_page_id uuid REFERENCES wiki_pages(id),
  approved_by uuid REFERENCES users(id),
  approved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX characters_owner_idx ON characters (owner_user_id, status);

-- Limit of characters per player; the number comes from campaign_settings.
-- Moving between counted statuses does not take a new slot, so lowering the limit
-- never evicts existing characters; it only blocks new ones.
CREATE FUNCTION enforce_character_slots() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  lim integer; cnt integer;
  counted text[] := ARRAY['pending_approval', 'active', 'unavailable'];
BEGIN
  IF NEW.owner_user_id IS NULL OR NOT (NEW.status = ANY (counted)) THEN
    RETURN NEW;
  END IF;
  IF TG_OP = 'UPDATE' AND OLD.owner_user_id IS NOT DISTINCT FROM NEW.owner_user_id
     AND OLD.status = ANY (counted) THEN
    RETURN NEW;
  END IF;
  PERFORM pg_advisory_xact_lock(hashtextextended(NEW.owner_user_id::text, 0));
  SELECT COALESCE((SELECT (value #>> '{}')::integer FROM campaign_settings
                   WHERE key = 'max_characters_per_player'), 2) INTO lim;
  SELECT count(*) INTO cnt FROM characters
   WHERE owner_user_id = NEW.owner_user_id AND status = ANY (counted) AND id <> NEW.id;
  IF cnt >= lim THEN
    RAISE EXCEPTION 'character slot limit (%) reached for user %', lim, NEW.owner_user_id
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_character_slots BEFORE INSERT OR UPDATE OF status, owner_user_id ON characters
  FOR EACH ROW EXECUTE FUNCTION enforce_character_slots();

CREATE TABLE character_transfers (          -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id),
  from_user_id uuid REFERENCES users(id),
  to_user_id uuid REFERENCES users(id),     -- NULL when the character becomes an NPC
  transferred_by uuid REFERENCES users(id),
  reason text,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- Changing the owner: no open signups, not on an expedition, former owner remembered, history written.
-- The application sets app.user_id and app.reason for the transaction (SET LOCAL).
CREATE FUNCTION character_transfer_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.owner_user_id IS DISTINCT FROM OLD.owner_user_id THEN
    IF EXISTS (SELECT 1 FROM adventure_signups WHERE character_id = NEW.id
                AND state IN ('pending_confirmation', 'confirmed', 'offered', 'queued')) THEN
      RAISE EXCEPTION 'character % has open signups; withdraw them before the transfer', NEW.id;
    END IF;
    IF EXISTS (SELECT 1 FROM expedition_members WHERE character_id = NEW.id AND left_at IS NULL) THEN
      RAISE EXCEPTION 'character % is on an expedition', NEW.id;
    END IF;
    NEW.former_owner_user_id := OLD.owner_user_id;
    INSERT INTO character_transfers (character_id, from_user_id, to_user_id, transferred_by, reason)
    VALUES (NEW.id, OLD.owner_user_id, NEW.owner_user_id,
            NULLIF(current_setting('app.user_id', true), '')::uuid,
            NULLIF(current_setting('app.reason', true), ''));
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_character_transfer BEFORE UPDATE OF owner_user_id ON characters
  FOR EACH ROW EXECUTE FUNCTION character_transfer_guard();

-- Fields owned by the Foundry actor can be changed only by the sync code path (app.writer = 'foundry_sync')
-- as long as the character is linked to an actor.
CREATE FUNCTION characters_foundry_owned_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF COALESCE(current_setting('app.writer', true), '') <> 'foundry_sync'
     AND EXISTS (SELECT 1 FROM foundry_actors WHERE character_id = OLD.id)
     AND (NEW.name, NEW.portrait_url, NEW.ancestry, NEW.heritage, NEW.background, NEW.class, NEW.level,
          NEW.xp, NEW.hp_current, NEW.hp_max, NEW.ac, NEW.speed_ft, NEW.size, NEW.attributes, NEW.skills)
         IS DISTINCT FROM
         (OLD.name, OLD.portrait_url, OLD.ancestry, OLD.heritage, OLD.background, OLD.class, OLD.level,
          OLD.xp, OLD.hp_current, OLD.hp_max, OLD.ac, OLD.speed_ft, OLD.size, OLD.attributes, OLD.skills) THEN
    RAISE EXCEPTION 'these fields belong to the Foundry actor; change them through an integration job'
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_characters_foundry_owned BEFORE UPDATE ON characters
  FOR EACH ROW EXECUTE FUNCTION characters_foundry_owned_guard();

CREATE TABLE character_status_history (     -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id),
  from_status text,
  to_status text NOT NULL,
  changed_by uuid REFERENCES users(id),
  reason text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE character_reviews (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  reviewer_id uuid NOT NULL REFERENCES users(id),
  decision text NOT NULL CHECK (decision IN ('approved', 'rejected', 'changes_requested', 'comment')),
  comment text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE foundry_actors (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id uuid NOT NULL REFERENCES foundry_worlds(id),
  actor_uuid text NOT NULL,                 -- permanent Foundry UUID, never the name
  character_id uuid UNIQUE REFERENCES characters(id),
  owner_confirmed_at timestamptz,           -- an authorized GM explicitly assigned/confirmed this owner during the Foundry link; owner is characters.owner_user_id
  linked_at timestamptz NOT NULL DEFAULT now(),
  last_synced_at timestamptz,
  snapshot jsonb,
  snapshot_hash text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (world_id, actor_uuid)             -- one Actor cannot belong to two characters
);

CREATE TABLE character_history (            -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id),
  kind text NOT NULL,                       -- adventure, reward, purchase, sale, transfer, inventory, level, status, admin
  ref_type text,
  ref_id uuid,
  payload jsonb NOT NULL DEFAULT '{}',
  author_id uuid REFERENCES users(id),
  reason text,
  source text NOT NULL DEFAULT 'app',
  game_day bigint,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX character_history_idx ON character_history (character_id, created_at DESC);

-- ---------------------------------------------------------------------
-- 8. Factions
-- ---------------------------------------------------------------------
CREATE TABLE factions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  kind text NOT NULL DEFAULT 'faction' CHECK (kind IN ('faction', 'organization', 'government',
    'guild', 'cult', 'other')),
  wiki_page_id uuid REFERENCES wiki_pages(id),
  parent_faction_id uuid REFERENCES factions(id),
  summary text,
  color text,
  visibility visibility NOT NULL DEFAULT 'gm_only',
  power_rating smallint CHECK (power_rating BETWEEN 0 AND 100),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE faction_reputation_tiers (     -- faction_id IS NULL = default scale
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  faction_id uuid REFERENCES factions(id) ON DELETE CASCADE,
  ord smallint NOT NULL,
  name text NOT NULL,
  min_value integer NOT NULL,
  perks jsonb NOT NULL DEFAULT '{}',
  UNIQUE NULLS NOT DISTINCT (faction_id, ord),
  UNIQUE NULLS NOT DISTINCT (faction_id, min_value)
);

CREATE TABLE faction_reputation (           -- current value, maintained from the log by trigger
  character_id uuid NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  faction_id uuid NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
  value integer NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (character_id, faction_id)
);

CREATE TABLE faction_reputation_log (       -- append-only; correction = reversal entry
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id),
  faction_id uuid NOT NULL REFERENCES factions(id),
  delta integer NOT NULL CHECK (delta <> 0),
  reason text,
  source_type text,                         -- adventure, expedition, enterprise, political_event, manual
  source_id uuid,
  game_day bigint,
  author_id uuid REFERENCES users(id),
  reversal_of uuid UNIQUE REFERENCES faction_reputation_log(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE FUNCTION apply_reputation_delta() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  PERFORM set_config('app.rep_post', 'on', true);
  INSERT INTO faction_reputation (character_id, faction_id, value)
  VALUES (NEW.character_id, NEW.faction_id, NEW.delta)
  ON CONFLICT (character_id, faction_id)
  DO UPDATE SET value = faction_reputation.value + EXCLUDED.value, updated_at = now();
  PERFORM set_config('app.rep_post', 'off', true);
  RETURN NEW;
END $$;
CREATE TRIGGER trg_apply_reputation AFTER INSERT ON faction_reputation_log
  FOR EACH ROW EXECUTE FUNCTION apply_reputation_delta();

CREATE TABLE faction_relations (            -- symmetric; faction_a < faction_b
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  faction_a uuid NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
  faction_b uuid NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
  standing smallint NOT NULL DEFAULT 0 CHECK (standing BETWEEN -100 AND 100),
  kind text NOT NULL DEFAULT 'neutral' CHECK (kind IN ('ally', 'friendly', 'neutral', 'tense',
    'rival', 'war', 'vassal')),
  senior_faction_id uuid REFERENCES factions(id),  -- for kind = 'vassal'
  since_game_day bigint,
  visibility visibility NOT NULL DEFAULT 'gm_only',
  notes_gm text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (faction_a < faction_b),
  CHECK ((kind = 'vassal') = (senior_faction_id IS NOT NULL)),
  CHECK (senior_faction_id IS NULL OR senior_faction_id IN (faction_a, faction_b)),
  UNIQUE (faction_a, faction_b)
);

CREATE TABLE political_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  title text NOT NULL,
  description_md text NOT NULL DEFAULT '',
  game_day bigint NOT NULL,
  state text NOT NULL DEFAULT 'draft' CHECK (state IN ('draft', 'applied', 'revoked')),
  visibility visibility NOT NULL DEFAULT 'gm_only',
  effects jsonb NOT NULL DEFAULT '[]',      -- relation, territory, reputation changes; applied by the GM
  wiki_page_id uuid REFERENCES wiki_pages(id),
  adventure_id uuid,                        -- FK added at the end
  expedition_id uuid,                       -- FK added at the end
  applied_by uuid REFERENCES users(id),
  applied_at timestamptz,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- Applied effects of a political event. Each effect has its own state and key, so a repeated
-- request cannot apply it twice and a revocation refers to concrete applied effects.
CREATE TABLE political_event_effects (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id uuid NOT NULL REFERENCES political_events(id),
  ord smallint NOT NULL,
  effect jsonb NOT NULL,
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'applied', 'reverted', 'failed')),
  apply_key uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  applied_at timestamptz,
  reverted_at timestamptz,
  error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (event_id, ord)
);

CREATE TABLE faction_relation_history (     -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  relation_id uuid NOT NULL REFERENCES faction_relations(id),
  old_standing smallint,
  new_standing smallint NOT NULL,
  old_kind text,
  new_kind text NOT NULL,
  game_day bigint,
  cause_event_id uuid REFERENCES political_events(id),
  author_id uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE territory_control (
  map_id uuid NOT NULL,
  q integer NOT NULL,
  r integer NOT NULL,
  faction_id uuid NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
  control_type text NOT NULL CHECK (control_type IN ('controlled', 'contested', 'influence')),
  strength smallint NOT NULL DEFAULT 50 CHECK (strength BETWEEN 0 AND 100),
  since_game_day bigint,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (map_id, q, r, faction_id),
  FOREIGN KEY (map_id, q, r) REFERENCES hexes(map_id, q, r) ON DELETE CASCADE
);
-- at most one controlling faction per hex
CREATE UNIQUE INDEX territory_one_controller ON territory_control (map_id, q, r)
  WHERE control_type = 'controlled';

CREATE TABLE territory_history (            -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL,
  q integer NOT NULL,
  r integer NOT NULL,
  faction_id uuid NOT NULL REFERENCES factions(id),
  old_control_type text,
  new_control_type text,
  old_strength smallint,
  new_strength smallint,
  game_day bigint,
  cause_event_id uuid REFERENCES political_events(id),
  author_id uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (map_id, q, r) REFERENCES hexes(map_id, q, r)
);

-- ---------------------------------------------------------------------
-- 9. Ledger (currencies, XP, transactions), integration jobs
-- ---------------------------------------------------------------------
CREATE TABLE currencies (                   -- 'xp' and 'coin' (base unit: cp) are ordinary rows
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  key text NOT NULL UNIQUE,
  name text NOT NULL,
  description text,
  icon text,
  scale smallint NOT NULL DEFAULT 0,        -- decimal places for display
  transferable boolean NOT NULL DEFAULT false,
  player_editable boolean NOT NULL DEFAULT false,
  tradeable boolean NOT NULL DEFAULT false,
  allow_negative boolean NOT NULL DEFAULT false,
  foundry_bound boolean NOT NULL DEFAULT false,
  foundry_binding jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE level_progression (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  xp_per_level integer NOT NULL DEFAULT 1000 CHECK (xp_per_level > 0),
  max_level smallint NOT NULL DEFAULT 20,
  is_default boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX level_progression_one_default ON level_progression ((true)) WHERE is_default;

CREATE TABLE accounts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_type text NOT NULL CHECK (owner_type IN ('character', 'enterprise', 'shop', 'faction', 'system')),
  owner_id uuid NOT NULL,                   -- for 'system': the all-zero UUID
  currency_id uuid NOT NULL REFERENCES currencies(id),
  balance bigint NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (owner_type, owner_id, currency_id)
);

CREATE TABLE ledger_transactions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  idempotency_key text NOT NULL UNIQUE,
  kind text NOT NULL CHECK (kind IN ('reward', 'purchase', 'sale', 'transfer', 'reversal', 'mint', 'burn',
    'adjustment', 'external_adjustment', 'enterprise', 'faction')),
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'posted', 'void')),
  source text NOT NULL CHECK (source IN ('app', 'foundry', 'discord', 'system')),
  author_id uuid REFERENCES users(id),
  reason text,
  reversal_of uuid UNIQUE REFERENCES ledger_transactions(id),
  adventure_id uuid,                        -- FK added at the end
  expedition_id uuid,                       -- FK added at the end
  game_day bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  posted_at timestamptz
);

CREATE TABLE ledger_entries (               -- append-only; the sum per currency within a transaction is 0
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
  account_id uuid NOT NULL REFERENCES accounts(id),
  delta bigint NOT NULL CHECK (delta <> 0),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ledger_entries_tx_idx ON ledger_entries (transaction_id);
CREATE INDEX ledger_entries_account_idx ON ledger_entries (account_id, created_at DESC);

-- Entries may be added only to a pending transaction. FOR SHARE makes a concurrent posting wait
-- for the insert, so the posted sum always includes every entry.
CREATE FUNCTION ledger_entry_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  PERFORM 1 FROM ledger_transactions WHERE id = NEW.transaction_id AND state = 'pending' FOR SHARE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'transaction % is not pending; entries cannot be added', NEW.transaction_id
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_ledger_entry_guard BEFORE INSERT ON ledger_entries
  FOR EACH ROW EXECUTE FUNCTION ledger_entry_guard();

-- Posting a transaction (pending -> posted): balance check, balances updated.
CREATE FUNCTION ledger_tx_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    RAISE EXCEPTION 'ledger_transactions is append-only' USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  IF (to_jsonb(NEW) - 'state' - 'posted_at') IS DISTINCT FROM (to_jsonb(OLD) - 'state' - 'posted_at') THEN
    RAISE EXCEPTION 'only the state of a ledger transaction may change' USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  IF OLD.state <> 'pending' THEN
    RAISE EXCEPTION 'transaction % is already %', OLD.id, OLD.state USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  IF NEW.state = 'posted' THEN
    IF NOT EXISTS (SELECT 1 FROM ledger_entries WHERE transaction_id = NEW.id) THEN
      RAISE EXCEPTION 'transaction % has no entries', NEW.id;
    END IF;
    IF EXISTS (SELECT 1 FROM ledger_entries e JOIN accounts a ON a.id = e.account_id
               WHERE e.transaction_id = NEW.id
               GROUP BY a.currency_id HAVING sum(e.delta) <> 0) THEN
      RAISE EXCEPTION 'transaction % is not balanced', NEW.id;
    END IF;
    IF NEW.reversal_of IS NOT NULL THEN
      IF NOT EXISTS (SELECT 1 FROM ledger_transactions WHERE id = NEW.reversal_of AND state = 'posted') THEN
        RAISE EXCEPTION 'only a posted transaction can be reversed';
      END IF;
      IF EXISTS (SELECT 1
                   FROM (SELECT account_id, sum(delta) AS d FROM ledger_entries
                          WHERE transaction_id = NEW.id GROUP BY account_id) n
                   FULL JOIN (SELECT account_id, -sum(delta) AS d FROM ledger_entries
                               WHERE transaction_id = NEW.reversal_of GROUP BY account_id) o USING (account_id)
                  WHERE n.d IS DISTINCT FROM o.d) THEN
        RAISE EXCEPTION 'a reversal must exactly negate the original transaction';
      END IF;
    END IF;
    PERFORM set_config('app.ledger_post', 'on', true);
    UPDATE accounts a SET balance = a.balance + s.total, updated_at = now()
      FROM (SELECT account_id, sum(delta) AS total FROM ledger_entries
             WHERE transaction_id = NEW.id GROUP BY account_id) s
     WHERE a.id = s.account_id;
    PERFORM set_config('app.ledger_post', 'off', true);
    IF EXISTS (SELECT 1 FROM ledger_entries e
                 JOIN accounts a ON a.id = e.account_id
                 JOIN currencies c ON c.id = a.currency_id
                WHERE e.transaction_id = NEW.id AND a.balance < 0
                  AND NOT c.allow_negative AND a.owner_type <> 'system' AND NEW.kind <> 'reversal') THEN
      RAISE EXCEPTION 'negative balance in transaction %', NEW.id USING ERRCODE = 'check_violation';
    END IF;
    NEW.posted_at := now();
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_ledger_tx_guard BEFORE UPDATE OR DELETE ON ledger_transactions
  FOR EACH ROW EXECUTE FUNCTION ledger_tx_guard();

CREATE TABLE integration_jobs (             -- visible to GMs: retry, cancel, status
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  target text NOT NULL CHECK (target IN ('foundry', 'discord', 'obsidian')),
  kind text NOT NULL,                       -- apply_xp, apply_coins, grant_item, set_owner, ...
  operation_id uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),   -- also written to the Actor flags
  payload jsonb NOT NULL,
  precondition jsonb,                       -- expected state before the write
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'in_flight', 'awaiting_sync',
    'succeeded', 'failed', 'cancelled')),
  attempts integer NOT NULL DEFAULT 0,
  max_attempts integer NOT NULL DEFAULT 8,
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  last_error text,
  result jsonb,
  parent_type text,
  parent_id uuid,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX integration_jobs_due_idx ON integration_jobs (state, next_attempt_at);
CREATE INDEX integration_jobs_parent_idx ON integration_jobs (parent_type, parent_id);

CREATE TABLE integration_job_attempts (     -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id uuid NOT NULL REFERENCES integration_jobs(id),
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz,
  http_status integer,
  error text,
  response jsonb
);
-- An attempt row is filled in while the call is running (finished_at IS NULL) and is frozen afterwards.
CREATE FUNCTION job_attempt_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'DELETE' OR OLD.finished_at IS NOT NULL THEN
    RAISE EXCEPTION 'a finished integration attempt is immutable' USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_job_attempt_guard BEFORE UPDATE OR DELETE ON integration_job_attempts
  FOR EACH ROW EXECUTE FUNCTION job_attempt_guard();

-- ---------------------------------------------------------------------
-- 10. Items, inventory, shops
-- ---------------------------------------------------------------------
CREATE TABLE items_catalog (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  source text NOT NULL CHECK (source IN ('pf2e-compendium', 'foundry-world', 'custom')),
  foundry_uuid text UNIQUE,                 -- stable PF2e Item UUID
  slug text,
  name text NOT NULL,
  item_level smallint,
  price_cp bigint CHECK (price_cp >= 0),
  traits jsonb NOT NULL DEFAULT '[]',
  data jsonb NOT NULL DEFAULT '{}',
  synced_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE inventory_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  catalog_id uuid REFERENCES items_catalog(id),
  foundry_item_id text,
  name text NOT NULL,
  quantity integer NOT NULL DEFAULT 1 CHECK (quantity >= 0),
  state jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (character_id, foundry_item_id)
);

CREATE TABLE inventory_events (             -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id),
  catalog_id uuid REFERENCES items_catalog(id),
  delta_qty integer NOT NULL,
  kind text NOT NULL,
  transaction_id uuid REFERENCES ledger_transactions(id),
  operation_id uuid,
  author_id uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE treasure_budget_rules (        -- loaded from the PF2e tables, not hardcoded; informational
  level smallint PRIMARY KEY,
  budget_cp bigint NOT NULL
);

CREATE TABLE shops (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  description text,
  owner_wiki_page_id uuid REFERENCES wiki_pages(id),
  location_wiki_page_id uuid REFERENCES wiki_pages(id),
  map_id uuid,
  q integer,
  r integer,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((map_id IS NULL) = (q IS NULL) AND (q IS NULL) = (r IS NULL)),
  FOREIGN KEY (map_id, q, r) REFERENCES hexes(map_id, q, r)
);

CREATE TABLE shop_stock (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  shop_id uuid NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
  catalog_id uuid NOT NULL REFERENCES items_catalog(id),
  price_amount bigint NOT NULL CHECK (price_amount >= 0),
  price_currency_id uuid NOT NULL REFERENCES currencies(id),
  stock_qty integer CHECK (stock_qty >= 0),   -- NULL = unlimited
  per_character_limit integer CHECK (per_character_limit > 0),
  min_character_level smallint,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (shop_id, catalog_id)
);

CREATE TABLE shop_requirements (            -- faction and reputation requirements for a shop or an item
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  shop_id uuid REFERENCES shops(id) ON DELETE CASCADE,
  stock_id uuid REFERENCES shop_stock(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('faction_reputation', 'faction_tier')),
  faction_id uuid NOT NULL REFERENCES factions(id),
  min_value integer,
  min_tier_ord smallint,
  CHECK ((shop_id IS NULL) <> (stock_id IS NULL)),                      -- the whole shop or one item, not both
  CHECK ((kind = 'faction_reputation' AND min_value IS NOT NULL AND min_tier_ord IS NULL)
      OR (kind = 'faction_tier' AND min_tier_ord IS NOT NULL AND min_value IS NULL))
);

CREATE TABLE purchases (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  stock_id uuid NOT NULL REFERENCES shop_stock(id),
  character_id uuid NOT NULL REFERENCES characters(id),
  quantity integer NOT NULL CHECK (quantity > 0),
  transaction_id uuid REFERENCES ledger_transactions(id),
  job_id uuid REFERENCES integration_jobs(id),
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'confirmed', 'failed', 'cancelled')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------
-- 11. Adventures, signups, queue, rewards, reports
-- ---------------------------------------------------------------------
CREATE TABLE adventures (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  title text NOT NULL,
  description text NOT NULL DEFAULT '',
  hook text,
  creator_user_id uuid NOT NULL REFERENCES users(id),
  gm_user_id uuid REFERENCES users(id),
  status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending_approval', 'approved',
    'published', 'registration_open', 'registration_closed', 'running', 'completed', 'cancelled')),
  starts_at timestamptz,
  duration_min integer CHECK (duration_min > 0),
  meeting_place text,
  region_id uuid,
  map_id uuid,
  target_q integer,
  target_r integer,
  target_wiki_page_id uuid REFERENCES wiki_pages(id),
  min_level smallint NOT NULL,
  max_level smallint NOT NULL,
  seats smallint NOT NULL CHECK (seats > 0),
  kind text,
  constraints jsonb NOT NULL DEFAULT '{}',
  foundry_scene_uuid text,
  start_game_day bigint,
  end_game_day bigint,
  cancel_reason text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (min_level <= max_level),
  CHECK (region_id IS NULL OR map_id IS NOT NULL),
  CHECK ((target_q IS NULL) = (target_r IS NULL) AND (target_q IS NULL OR map_id IS NOT NULL)),
  FOREIGN KEY (map_id, target_q, target_r) REFERENCES hexes(map_id, q, r),
  FOREIGN KEY (region_id, map_id) REFERENCES map_regions(id, map_id)
);
CREATE INDEX adventures_status_idx ON adventures (status, starts_at);

CREATE TABLE adventure_wiki_links (
  adventure_id uuid NOT NULL REFERENCES adventures(id) ON DELETE CASCADE,
  wiki_page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  PRIMARY KEY (adventure_id, wiki_page_id)
);

CREATE TABLE adventure_signups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  adventure_id uuid NOT NULL REFERENCES adventures(id) ON DELETE CASCADE,
  character_id uuid NOT NULL REFERENCES characters(id),
  user_id uuid NOT NULL REFERENCES users(id),
  state text NOT NULL DEFAULT 'queued' CHECK (state IN ('pending_confirmation', 'confirmed', 'queued',
    'offered', 'declined', 'withdrawn', 'removed', 'expired')),
  queued_at timestamptz NOT NULL DEFAULT now(),
  manual_order integer,                     -- GM override of the queue order
  offered_at timestamptz,
  offer_expires_at timestamptz,
  decided_by uuid REFERENCES users(id),
  decision_reason text,
  level_at_signup smallint NOT NULL,        -- filled from the character when omitted
  during tstzrange,                         -- kept in sync with the adventure time
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (adventure_id, character_id),
  -- one character cannot hold two overlapping places
  EXCLUDE USING gist (character_id WITH =, during WITH &&)
    WHERE (state IN ('pending_confirmation', 'confirmed', 'offered') AND during IS NOT NULL)
);
CREATE INDEX signups_queue_idx ON adventure_signups (adventure_id, state, queued_at);

CREATE TABLE adventure_multi_char_grants (   -- the GM explicitly allows N characters of one player
  adventure_id uuid NOT NULL REFERENCES adventures(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id),
  max_characters smallint NOT NULL CHECK (max_characters >= 2),
  granted_by uuid NOT NULL REFERENCES users(id),
  reason text,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (adventure_id, user_id)
);

-- Everything a signup must satisfy, enforced in one place and serialised per adventure.
-- level_at_signup is a snapshot taken from the character; it cannot be supplied or edited by the caller.
CREATE FUNCTION signup_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  a adventures%ROWTYPE; c characters%ROWTYPE;
  active text[] := ARRAY['pending_confirmation', 'confirmed', 'offered', 'queued'];
  occupying text[] := ARRAY['pending_confirmation', 'confirmed', 'offered'];
  cnt integer; allowed integer; used integer; lo smallint; hi smallint; spread integer;
BEGIN
  SELECT * INTO a FROM adventures WHERE id = NEW.adventure_id FOR UPDATE;   -- serialises signups per adventure
  SELECT * INTO c FROM characters WHERE id = NEW.character_id;
  IF a.starts_at IS NOT NULL AND a.duration_min IS NOT NULL THEN
    NEW.during := tstzrange(a.starts_at, a.starts_at + make_interval(mins => a.duration_min));
  ELSE
    NEW.during := NULL;
  END IF;
  IF TG_OP = 'INSERT' THEN
    NEW.level_at_signup := c.level;
  ELSIF NEW.level_at_signup IS DISTINCT FROM OLD.level_at_signup
        AND COALESCE(current_setting('app.recompute_levels', true), '') <> 'on' THEN
    RAISE EXCEPTION 'level_at_signup is a snapshot; use recompute_signup_levels()'
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  IF NOT (NEW.state = ANY (active)) THEN RETURN NEW; END IF;

  -- adventure status: new signups only while registration is open; moves between active states
  -- (offer, confirmation) are allowed while registration is open or closed
  IF TG_OP = 'INSERT' AND a.status <> 'registration_open' THEN
    RAISE EXCEPTION 'adventure % is not open for registration (status %)', a.id, a.status;
  ELSIF TG_OP = 'UPDATE' AND OLD.state IS DISTINCT FROM NEW.state
        AND a.status NOT IN ('registration_open', 'registration_closed') THEN
    RAISE EXCEPTION 'adventure % does not accept signup changes (status %)', a.id, a.status;
  END IF;

  IF c.owner_user_id IS DISTINCT FROM NEW.user_id THEN
    RAISE EXCEPTION 'character % does not belong to user %', NEW.character_id, NEW.user_id
      USING ERRCODE = 'insufficient_privilege';
  END IF;
  IF c.status <> 'active' THEN RAISE EXCEPTION 'character % is not active', c.id; END IF;
  IF NEW.during IS NULL THEN RAISE EXCEPTION 'adventure % has no schedule yet', a.id; END IF;
  IF c.level NOT BETWEEN a.min_level AND a.max_level THEN
    RAISE EXCEPTION 'character level % is outside the adventure range %-%', c.level, a.min_level, a.max_level;
  END IF;

  SELECT count(*) INTO cnt FROM adventure_signups
   WHERE adventure_id = NEW.adventure_id AND user_id = NEW.user_id AND state = ANY (active) AND id <> NEW.id;
  SELECT COALESCE((SELECT max_characters FROM adventure_multi_char_grants
                    WHERE adventure_id = NEW.adventure_id AND user_id = NEW.user_id), 1) INTO allowed;
  IF cnt + 1 > allowed THEN
    RAISE EXCEPTION 'user % may enter at most % character(s) in this adventure', NEW.user_id, allowed;
  END IF;

  IF NEW.state = ANY (occupying) THEN
    SELECT count(*), min(level_at_signup), max(level_at_signup) INTO used, lo, hi FROM adventure_signups
     WHERE adventure_id = NEW.adventure_id AND state = ANY (occupying) AND id <> NEW.id;
    IF used >= a.seats THEN RAISE EXCEPTION 'no free seats in adventure %', a.id; END IF;
    SELECT COALESCE((SELECT (value #>> '{}')::integer FROM campaign_settings
                      WHERE key = 'max_party_level_spread'), 2) INTO spread;
    lo := LEAST(COALESCE(lo, NEW.level_at_signup), NEW.level_at_signup);
    hi := GREATEST(COALESCE(hi, NEW.level_at_signup), NEW.level_at_signup);
    IF hi - lo > spread THEN
      RAISE EXCEPTION 'party level spread % would exceed the limit %', hi - lo, spread;
    END IF;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_signup_guard BEFORE INSERT OR UPDATE OF adventure_id, character_id, user_id, state, level_at_signup
  ON adventure_signups FOR EACH ROW EXECUTE FUNCTION signup_guard();

-- The GM-controlled way to refresh the level snapshots (for example after a level-up).
-- It re-runs every signup check, so a spread violation aborts the whole refresh.
CREATE FUNCTION recompute_signup_levels(p_adventure uuid) RETURNS integer LANGUAGE plpgsql AS $$
DECLARE n integer;
BEGIN
  PERFORM set_config('app.recompute_levels', 'on', true);
  UPDATE adventure_signups s SET level_at_signup = c.level
    FROM characters c
   WHERE c.id = s.character_id AND s.adventure_id = p_adventure
     AND s.state IN ('pending_confirmation', 'confirmed', 'offered', 'queued')
     AND s.level_at_signup <> c.level;
  GET DIAGNOSTICS n = ROW_COUNT;
  PERFORM set_config('app.recompute_levels', 'off', true);
  RETURN n;
END $$;

-- Changing seats or the level range must not invalidate signups that already exist.
CREATE FUNCTION adventure_params_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.seats < (SELECT count(*) FROM adventure_signups WHERE adventure_id = NEW.id
                   AND state IN ('pending_confirmation', 'confirmed', 'offered')) THEN
    RAISE EXCEPTION 'seats cannot be lower than the number of occupied seats';
  END IF;
  IF EXISTS (SELECT 1 FROM adventure_signups s JOIN characters c ON c.id = s.character_id
              WHERE s.adventure_id = NEW.id
                AND s.state IN ('pending_confirmation', 'confirmed', 'offered', 'queued')
                AND (c.level < NEW.min_level OR c.level > NEW.max_level)) THEN
    RAISE EXCEPTION 'existing signups fall outside the new level range';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_adventure_params_guard BEFORE UPDATE OF seats, min_level, max_level ON adventures
  FOR EACH ROW EXECUTE FUNCTION adventure_params_guard();

CREATE FUNCTION sync_signup_period() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.starts_at IS DISTINCT FROM OLD.starts_at OR NEW.duration_min IS DISTINCT FROM OLD.duration_min THEN
    UPDATE adventure_signups
       SET during = CASE WHEN NEW.starts_at IS NULL THEN NULL
                         ELSE tstzrange(NEW.starts_at, NEW.starts_at + make_interval(mins => COALESCE(NEW.duration_min, 240))) END
     WHERE adventure_id = NEW.id;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_sync_signup_period AFTER UPDATE OF starts_at, duration_min ON adventures
  FOR EACH ROW EXECUTE FUNCTION sync_signup_period();

CREATE TABLE signup_events (                -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  signup_id uuid NOT NULL REFERENCES adventure_signups(id),
  from_state text,
  to_state text NOT NULL,
  actor_user_id uuid REFERENCES users(id),
  reason text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE adventure_participants (       -- fixed when the adventure starts or completes
  adventure_id uuid NOT NULL REFERENCES adventures(id) ON DELETE CASCADE,
  character_id uuid NOT NULL REFERENCES characters(id),
  level_at_start smallint NOT NULL,
  attended boolean NOT NULL DEFAULT true,
  PRIMARY KEY (adventure_id, character_id)
);

CREATE TABLE discord_channels (
  kind text PRIMARY KEY,                    -- announcements, adventures, characters, rewards, trade, reports, world, calendar
  guild_id text NOT NULL,
  channel_id text NOT NULL
);

CREATE TABLE discord_messages (             -- so that published messages can be edited later
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  kind text NOT NULL,
  ref_type text NOT NULL,
  ref_id uuid NOT NULL,
  channel_id text NOT NULL,
  message_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (ref_type, ref_id, kind)
);

CREATE TABLE rewards (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  adventure_id uuid REFERENCES adventures(id),
  expedition_id uuid,                       -- FK added at the end
  character_id uuid NOT NULL REFERENCES characters(id),
  kind text NOT NULL CHECK (kind IN ('xp', 'currency', 'item', 'special')),
  currency_id uuid REFERENCES currencies(id),
  amount bigint,
  catalog_id uuid REFERENCES items_catalog(id),
  quantity integer,
  comment text,
  transaction_id uuid REFERENCES ledger_transactions(id),
  job_id uuid REFERENCES integration_jobs(id),
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'awaiting_sync', 'applied', 'failed', 'reversed')),
  game_day bigint,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX rewards_char_idx ON rewards (character_id, created_at DESC);

CREATE TABLE adventure_reports (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  adventure_id uuid NOT NULL REFERENCES adventures(id) ON DELETE CASCADE,
  author_user_id uuid NOT NULL REFERENCES users(id),
  kind text NOT NULL CHECK (kind IN ('public', 'personal', 'gm', 'participant')),
  body_md text NOT NULL,
  visibility visibility NOT NULL DEFAULT 'gm_only',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE report_links (
  report_id uuid NOT NULL REFERENCES adventure_reports(id) ON DELETE CASCADE,
  link_type text NOT NULL CHECK (link_type IN ('location', 'npc', 'faction', 'event', 'wiki')),
  ref_id uuid NOT NULL,
  PRIMARY KEY (report_id, link_type, ref_id)
);

-- ---------------------------------------------------------------------
-- 12. Expeditions
-- ---------------------------------------------------------------------
CREATE TABLE expeditions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  map_id uuid NOT NULL REFERENCES maps(id),
  adventure_id uuid REFERENCES adventures(id),     -- an expedition may span several sessions
  leader_character_id uuid REFERENCES characters(id),
  status text NOT NULL DEFAULT 'planning' CHECK (status IN ('planning', 'active', 'returned', 'aborted', 'failed')),
  origin_q integer NOT NULL,
  origin_r integer NOT NULL,
  current_q integer NOT NULL,
  current_r integer NOT NULL,
  planned_days integer CHECK (planned_days > 0),
  start_game_day bigint,
  end_game_day bigint,
  travel_rules jsonb NOT NULL DEFAULT '{}',        -- overrides of the campaign travel rules
  gm_notes text,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (map_id, origin_q, origin_r) REFERENCES hexes(map_id, q, r),
  FOREIGN KEY (map_id, current_q, current_r) REFERENCES hexes(map_id, q, r),
  UNIQUE (id, map_id)
);
CREATE INDEX expeditions_status_idx ON expeditions (status);

CREATE TABLE expedition_members (
  expedition_id uuid NOT NULL REFERENCES expeditions(id) ON DELETE CASCADE,
  character_id uuid NOT NULL REFERENCES characters(id),
  role text,                                       -- navigator, scout, quartermaster, ...
  speed_ft_snapshot smallint,                      -- from the Foundry snapshot when joining
  joined_game_day bigint,
  left_game_day bigint,
  left_at timestamptz,
  PRIMARY KEY (expedition_id, character_id)
);
-- a character is in at most one active expedition
CREATE UNIQUE INDEX expedition_one_active_per_char ON expedition_members (character_id)
  WHERE left_at IS NULL;

CREATE TABLE expedition_routes (            -- versioned plan
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  expedition_id uuid NOT NULL REFERENCES expeditions(id) ON DELETE CASCADE,
  version integer NOT NULL,
  waypoints jsonb NOT NULL,                 -- [[q,r], ...]
  planned_hexes jsonb NOT NULL DEFAULT '[]',
  estimated_days numeric(6, 2),
  is_current boolean NOT NULL DEFAULT true,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (expedition_id, version)
);
CREATE UNIQUE INDEX expedition_one_current_route ON expedition_routes (expedition_id) WHERE is_current;

CREATE TABLE expedition_legs (              -- one row per travel day
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  expedition_id uuid NOT NULL REFERENCES expeditions(id) ON DELETE CASCADE,
  day_no integer NOT NULL CHECK (day_no > 0),
  game_day bigint,
  map_id uuid NOT NULL,
  from_q integer NOT NULL,
  from_r integer NOT NULL,
  to_q integer NOT NULL,
  to_r integer NOT NULL,
  path jsonb NOT NULL DEFAULT '[]',         -- hexes crossed during the day
  travel_hours smallint,
  movement_budget numeric(8, 2),            -- computed from the slowest member and terrain
  hexes_moved smallint,
  weather_record_id uuid,
  modifiers jsonb NOT NULL DEFAULT '{}',    -- applied terrain, weather and GM modifiers
  state text NOT NULL DEFAULT 'planned' CHECK (state IN ('planned', 'completed', 'skipped')),
  outcome text,
  gm_notes text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (expedition_id, day_no),
  FOREIGN KEY (map_id, from_q, from_r) REFERENCES hexes(map_id, q, r),
  FOREIGN KEY (map_id, to_q, to_r) REFERENCES hexes(map_id, q, r),
  FOREIGN KEY (expedition_id, map_id) REFERENCES expeditions(id, map_id),
  FOREIGN KEY (weather_record_id, map_id) REFERENCES weather_records(id, map_id)
);

CREATE TABLE expedition_supply_items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  expedition_id uuid NOT NULL REFERENCES expeditions(id) ON DELETE CASCADE,
  kind text NOT NULL,                       -- food, water, fodder, light, custom
  name text NOT NULL,
  unit text NOT NULL DEFAULT 'pcs',
  owner_character_id uuid REFERENCES characters(id),   -- NULL = shared stock
  catalog_id uuid REFERENCES items_catalog(id),
  per_member_per_day numeric(8, 3) NOT NULL DEFAULT 0 CHECK (per_member_per_day >= 0),
  quantity numeric(12, 3) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE expedition_supply_ledger (     -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  expedition_id uuid NOT NULL,
  supply_item_id uuid NOT NULL,
  leg_id uuid,
  delta numeric(12, 3) NOT NULL CHECK (delta <> 0),
  reason text NOT NULL CHECK (reason IN ('loaded', 'consumed', 'found', 'lost', 'spent', 'manual')),
  game_day bigint,
  author_id uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE FUNCTION apply_supply_delta() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  UPDATE expedition_supply_items SET quantity = quantity + NEW.delta, updated_at = now()
   WHERE id = NEW.supply_item_id;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_apply_supply AFTER INSERT ON expedition_supply_ledger
  FOR EACH ROW EXECUTE FUNCTION apply_supply_delta();

CREATE TABLE expedition_events (            -- encounters, hazards, discoveries along the way
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  expedition_id uuid NOT NULL REFERENCES expeditions(id) ON DELETE CASCADE,
  leg_id uuid,
  kind text NOT NULL,
  title text NOT NULL,
  description text,
  gm_notes text,
  visibility visibility NOT NULL DEFAULT 'gm_only',
  resolved boolean NOT NULL DEFAULT false,
  game_day bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE expedition_visited_hexes (
  expedition_id uuid NOT NULL REFERENCES expeditions(id) ON DELETE CASCADE,
  map_id uuid NOT NULL,
  q integer NOT NULL,
  r integer NOT NULL,
  first_leg_id uuid,
  first_game_day bigint,
  times_visited integer NOT NULL DEFAULT 1 CHECK (times_visited > 0),
  revealed boolean NOT NULL DEFAULT false,         -- fog opened by the GM for this visit
  PRIMARY KEY (expedition_id, q, r),
  FOREIGN KEY (map_id, q, r) REFERENCES hexes(map_id, q, r),
  FOREIGN KEY (expedition_id, map_id) REFERENCES expeditions(id, map_id)
);

-- ---------------------------------------------------------------------
-- 13. Trading enterprise
-- ---------------------------------------------------------------------
CREATE TABLE enterprises (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  kind text,
  owner_character_id uuid REFERENCES characters(id),
  owner_user_id uuid REFERENCES users(id),
  wiki_page_id uuid REFERENCES wiki_pages(id),
  map_id uuid,
  q integer,
  r integer,
  level smallint NOT NULL DEFAULT 1,
  reputation integer NOT NULL DEFAULT 0,
  description text,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'closed')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK ((map_id IS NULL) = (q IS NULL) AND (q IS NULL) = (r IS NULL)),
  FOREIGN KEY (map_id, q, r) REFERENCES hexes(map_id, q, r)
);

CREATE TABLE enterprise_stock (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  enterprise_id uuid NOT NULL REFERENCES enterprises(id) ON DELETE CASCADE,
  catalog_id uuid REFERENCES items_catalog(id),
  name text NOT NULL,
  unit text NOT NULL DEFAULT 'pcs',
  quantity numeric(12, 3) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  unit_value_cp bigint,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enterprise_staff (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  enterprise_id uuid NOT NULL REFERENCES enterprises(id) ON DELETE CASCADE,
  name text NOT NULL,
  role text,
  npc_wiki_page_id uuid REFERENCES wiki_pages(id),
  wage_cp bigint,
  hired_game_day bigint,
  left_game_day bigint,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enterprise_suppliers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  enterprise_id uuid NOT NULL REFERENCES enterprises(id) ON DELETE CASCADE,
  name text NOT NULL,
  reliability smallint CHECK (reliability BETWEEN 0 AND 100),
  wiki_page_id uuid REFERENCES wiki_pages(id),
  notes text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enterprise_upgrades (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  enterprise_id uuid NOT NULL REFERENCES enterprises(id) ON DELETE CASCADE,
  name text NOT NULL,
  description text,
  cost_amount bigint,
  cost_currency_id uuid REFERENCES currencies(id),
  state text NOT NULL DEFAULT 'available' CHECK (state IN ('available', 'in_progress', 'done')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enterprise_event_rules (       -- automation proposes events, the GM decides
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  enterprise_id uuid REFERENCES enterprises(id) ON DELETE CASCADE,   -- NULL = for all
  name text NOT NULL,
  trigger_kind text NOT NULL CHECK (trigger_kind IN ('schedule', 'calendar', 'reputation_threshold',
    'low_stock', 'order_completed')),
  config jsonb NOT NULL DEFAULT '{}',
  template jsonb NOT NULL DEFAULT '{}',
  requires_gm_approval boolean NOT NULL DEFAULT true CHECK (requires_gm_approval),
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enterprise_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  enterprise_id uuid NOT NULL REFERENCES enterprises(id) ON DELETE CASCADE,
  rule_id uuid REFERENCES enterprise_event_rules(id),
  title text NOT NULL,
  description text,
  occurred_game_day bigint,
  expires_game_day bigint,
  conditions jsonb NOT NULL DEFAULT '{}',
  available_actions jsonb NOT NULL DEFAULT '[]',
  consequences jsonb NOT NULL DEFAULT '{}',
  cost jsonb,
  reward jsonb,
  state text NOT NULL DEFAULT 'draft' CHECK (state IN ('draft', 'open', 'resolved', 'expired', 'cancelled')),
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enterprise_actions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id uuid NOT NULL REFERENCES enterprise_events(id) ON DELETE CASCADE,
  character_id uuid NOT NULL REFERENCES characters(id),
  action_kind text NOT NULL CHECK (action_kind IN ('fulfill_order', 'find_supplier', 'negotiate', 'organize_delivery',
    'advertise', 'hire', 'upgrade', 'change_assortment', 'solve_problem')),
  chosen_at timestamptz NOT NULL DEFAULT now(),
  gm_outcome text,
  resolved_by uuid REFERENCES users(id),
  resolved_at timestamptz
);

CREATE TABLE enterprise_log (               -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  enterprise_id uuid NOT NULL REFERENCES enterprises(id),
  event_id uuid REFERENCES enterprise_events(id),
  kind text NOT NULL,
  delta jsonb NOT NULL DEFAULT '{}',
  game_day bigint,
  author_id uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------
-- 14. Obsidian vaults over WebDAV
-- ---------------------------------------------------------------------
CREATE TABLE vaults (                       -- the app is a WebDAV CLIENT of the existing Obsidian server
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  kind text NOT NULL CHECK (kind IN ('gm', 'player')),   -- 'player' vault receives public content only
  name text NOT NULL UNIQUE,
  webdav_url text NOT NULL,
  remote_root text NOT NULL DEFAULT '/',
  username text NOT NULL,
  password_enc bytea NOT NULL,              -- encrypted at application level, key kept in a separate secret
  CONSTRAINT vault_url_https CHECK (webdav_url ~* '^https://'),
  CONSTRAINT vault_root_no_traversal CHECK (remote_root !~ '(^|/)\.\.(/|$)'),
  categories text[] NOT NULL DEFAULT '{}',
  sync_interval_sec integer NOT NULL DEFAULT 300 CHECK (sync_interval_sec >= 30),
  last_scan_at timestamptz,
  last_ok_at timestamptz,
  last_error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE vault_files (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_id uuid NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
  entity_type text NOT NULL,                -- wiki_page, adventure_report, chronicle_entry, ...
  entity_id uuid NOT NULL,
  rel_path text NOT NULL,
  exported_rev integer,
  exported_hash text,
  remote_etag text,                         -- sent back as If-Match so a PUT cannot overwrite a newer edit
  remote_mtime timestamptz,
  last_seen_hash text,
  last_seen_at timestamptz,
  state text NOT NULL DEFAULT 'synced' CHECK (state IN ('synced', 'pending_import', 'conflict',
    'pending_delete', 'orphan')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (vault_id, rel_path),
  UNIQUE (vault_id, entity_type, entity_id)
);

CREATE TABLE vault_conflicts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  vault_file_id uuid NOT NULL REFERENCES vault_files(id) ON DELETE CASCADE,
  base_rev integer,
  db_rev integer,
  file_hash text NOT NULL,
  conflict_path text NOT NULL,              -- _conflicts/<id>.<time>.md on the WebDAV server
  state text NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'resolved')),
  resolution text CHECK (resolution IN ('keep_app', 'keep_file', 'merged')),
  resolved_by uuid REFERENCES users(id),
  resolved_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE foundry_journals (              -- Foundry journals, including Campaign Codex sheets
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id uuid NOT NULL REFERENCES foundry_worlds(id),
  journal_uuid text NOT NULL,
  sheet_type text,                          -- the Campaign Codex sheet type, once its data format is confirmed
  wiki_page_id uuid UNIQUE REFERENCES wiki_pages(id),
  folder_path text,
  snapshot jsonb,
  snapshot_hash text,
  exported_rev integer,                     -- wiki revision last written to the journal
  last_seen_hash text,
  state text NOT NULL DEFAULT 'synced' CHECK (state IN ('synced', 'pending_import', 'conflict', 'orphan')),
  last_synced_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (world_id, journal_uuid)
);

CREATE TABLE vault_sync_log (               -- append-only
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  vault_id uuid NOT NULL REFERENCES vaults(id),
  direction text NOT NULL CHECK (direction IN ('export', 'import')),
  entity_id uuid,
  action text NOT NULL,
  result text NOT NULL,
  detail text,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------
-- 15. Notifications
-- ---------------------------------------------------------------------
CREATE TABLE notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind text NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}',
  dedupe_key text,                          -- the same event is never delivered twice to one user
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX notifications_dedupe_idx ON notifications (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL;
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);

CREATE TABLE notification_deliveries (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  notification_id uuid NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
  channel text NOT NULL CHECK (channel IN ('discord_dm', 'discord_channel', 'web')),
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'sent', 'failed')),
  attempts integer NOT NULL DEFAULT 0,
  last_error text,
  delivered_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (notification_id, channel)
);

-- Durable event log for SSE, Discord and the worker. NOTIFY is only a wake-up signal;
-- a client that reconnects replays events with id > its Last-Event-ID.
CREATE TABLE event_outbox (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  kind text NOT NULL,
  entity_type text,
  entity_id uuid,
  payload jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);
CREATE INDEX event_outbox_unpublished_idx ON event_outbox (id) WHERE published_at IS NULL;
CREATE FUNCTION notify_outbox() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN PERFORM pg_notify('wm_events', NEW.id::text); RETURN NEW; END $$;
CREATE TRIGGER trg_notify_outbox AFTER INSERT ON event_outbox FOR EACH ROW EXECUTE FUNCTION notify_outbox();

-- ---------------------------------------------------------------------
-- 16. Foreign keys that close the cycles
-- ---------------------------------------------------------------------
ALTER TABLE game_clock_history ADD FOREIGN KEY (adventure_id) REFERENCES adventures(id);
ALTER TABLE game_clock_history ADD FOREIGN KEY (expedition_id) REFERENCES expeditions(id);
ALTER TABLE chronicle_entries ADD FOREIGN KEY (adventure_id) REFERENCES adventures(id);
ALTER TABLE calendar_events ADD FOREIGN KEY (adventure_id) REFERENCES adventures(id);
ALTER TABLE chronicle_entries ADD FOREIGN KEY (expedition_id) REFERENCES expeditions(id);
ALTER TABLE exploration_history ADD FOREIGN KEY (adventure_id) REFERENCES adventures(id);
ALTER TABLE exploration_history ADD FOREIGN KEY (expedition_id, map_id) REFERENCES expeditions(id, map_id);
ALTER TABLE political_events ADD FOREIGN KEY (adventure_id) REFERENCES adventures(id);
ALTER TABLE political_events ADD FOREIGN KEY (expedition_id) REFERENCES expeditions(id);
ALTER TABLE ledger_transactions ADD FOREIGN KEY (adventure_id) REFERENCES adventures(id);
ALTER TABLE ledger_transactions ADD FOREIGN KEY (expedition_id) REFERENCES expeditions(id);
ALTER TABLE rewards ADD FOREIGN KEY (expedition_id) REFERENCES expeditions(id);

-- Records of one expedition may refer only to legs, supplies and members of the same expedition.
ALTER TABLE expedition_legs ADD UNIQUE (id, expedition_id);
ALTER TABLE expedition_supply_items ADD UNIQUE (id, expedition_id);
ALTER TABLE expedition_supply_ledger ADD FOREIGN KEY (supply_item_id, expedition_id)
  REFERENCES expedition_supply_items(id, expedition_id);
ALTER TABLE expedition_supply_ledger ADD FOREIGN KEY (leg_id, expedition_id)
  REFERENCES expedition_legs(id, expedition_id);
ALTER TABLE expedition_events ADD FOREIGN KEY (leg_id, expedition_id)
  REFERENCES expedition_legs(id, expedition_id);
ALTER TABLE expedition_visited_hexes ADD FOREIGN KEY (first_leg_id, expedition_id)
  REFERENCES expedition_legs(id, expedition_id);

-- Paths stored as jsonb ([[q,r],...]) must stay on the map of their owner.
CREATE FUNCTION assert_hexes_on_map(p_map uuid, p_path jsonb) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
  IF p_path IS NULL THEN RETURN; END IF;
  IF EXISTS (SELECT 1 FROM jsonb_array_elements(p_path) e
              LEFT JOIN hexes h ON h.map_id = p_map AND h.q = (e->>0)::integer AND h.r = (e->>1)::integer
             WHERE h.map_id IS NULL) THEN
    RAISE EXCEPTION 'path contains hexes that are not on map %', p_map USING ERRCODE = 'foreign_key_violation';
  END IF;
END $$;

-- A path is contiguous when every step goes to one of the six axial neighbours.
CREATE FUNCTION assert_path_contiguous(p_path jsonb, p_fq integer, p_fr integer, p_tq integer, p_tr integer)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE n integer; i integer; dq integer; dr integer;
BEGIN
  IF p_path IS NULL OR jsonb_array_length(p_path) = 0 THEN RETURN; END IF;
  n := jsonb_array_length(p_path);
  IF p_fq IS NOT NULL AND ((p_path -> 0 ->> 0)::integer <> p_fq OR (p_path -> 0 ->> 1)::integer <> p_fr) THEN
    RAISE EXCEPTION 'the path must start at the origin hex';
  END IF;
  IF p_tq IS NOT NULL AND ((p_path -> (n - 1) ->> 0)::integer <> p_tq OR (p_path -> (n - 1) ->> 1)::integer <> p_tr) THEN
    RAISE EXCEPTION 'the path must end at the destination hex';
  END IF;
  FOR i IN 1 .. n - 1 LOOP
    dq := (p_path -> i ->> 0)::integer - (p_path -> (i - 1) ->> 0)::integer;
    dr := (p_path -> i ->> 1)::integer - (p_path -> (i - 1) ->> 1)::integer;
    IF NOT ((dq, dr) IN ((1, 0), (1, -1), (0, -1), (-1, 0), (-1, 1), (0, 1))) THEN
      RAISE EXCEPTION 'the path jumps between hexes that are not neighbours (step %)', i;
    END IF;
  END LOOP;
END $$;

CREATE FUNCTION check_route_path() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE m uuid;
BEGIN
  IF TG_TABLE_NAME = 'routes' THEN PERFORM assert_hexes_on_map(NEW.map_id, NEW.waypoints);
  ELSIF TG_TABLE_NAME = 'map_linear_features' THEN PERFORM assert_hexes_on_map(NEW.map_id, NEW.path);
  ELSIF TG_TABLE_NAME = 'expedition_legs' THEN
    PERFORM assert_hexes_on_map(NEW.map_id, NEW.path);
    PERFORM assert_path_contiguous(NEW.path, NEW.from_q, NEW.from_r, NEW.to_q, NEW.to_r);
  ELSIF TG_TABLE_NAME = 'expedition_routes' THEN
    SELECT map_id INTO m FROM expeditions WHERE id = NEW.expedition_id;
    PERFORM assert_hexes_on_map(m, NEW.waypoints);
    PERFORM assert_hexes_on_map(m, NEW.planned_hexes);
    PERFORM assert_path_contiguous(NEW.planned_hexes, NULL, NULL, NULL, NULL);
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_routes_path BEFORE INSERT OR UPDATE ON routes FOR EACH ROW EXECUTE FUNCTION check_route_path();
CREATE TRIGGER trg_linear_path BEFORE INSERT OR UPDATE ON map_linear_features FOR EACH ROW EXECUTE FUNCTION check_route_path();
CREATE TRIGGER trg_leg_path BEFORE INSERT OR UPDATE ON expedition_legs FOR EACH ROW EXECUTE FUNCTION check_route_path();
CREATE TRIGGER trg_exp_route_path BEFORE INSERT OR UPDATE ON expedition_routes FOR EACH ROW EXECUTE FUNCTION check_route_path();

-- ---------------------------------------------------------------------
-- 16b. Helpers, automatic history, guards, shop purchase
-- The application sets these per transaction with SET LOCAL / set_config(..., true):
--   app.user_id, app.reason, app.event_id (political event), app.writer ('foundry_sync')
-- ---------------------------------------------------------------------
CREATE FUNCTION app_user() RETURNS uuid LANGUAGE sql STABLE AS
$$ SELECT NULLIF(current_setting('app.user_id', true), '')::uuid $$;
CREATE FUNCTION app_reason() RETURNS text LANGUAGE sql STABLE AS
$$ SELECT NULLIF(current_setting('app.reason', true), '') $$;
CREATE FUNCTION app_event() RETURNS uuid LANGUAGE sql STABLE AS
$$ SELECT NULLIF(current_setting('app.event_id', true), '')::uuid $$;
CREATE FUNCTION current_game_day() RETURNS bigint LANGUAGE sql STABLE AS
$$ SELECT current_day FROM game_clock LIMIT 1 $$;

-- Status changes, signup changes, relation and territory changes write their own history.
CREATE FUNCTION log_character_status() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND OLD.status = NEW.status THEN RETURN NULL; END IF;
  INSERT INTO character_status_history (character_id, from_status, to_status, changed_by, reason)
  VALUES (NEW.id, CASE WHEN TG_OP = 'UPDATE' THEN OLD.status END, NEW.status, app_user(), app_reason());
  RETURN NULL;
END $$;
CREATE TRIGGER trg_log_character_status AFTER INSERT OR UPDATE OF status ON characters
  FOR EACH ROW EXECUTE FUNCTION log_character_status();

CREATE FUNCTION log_signup_state() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND OLD.state = NEW.state THEN RETURN NULL; END IF;
  INSERT INTO signup_events (signup_id, from_state, to_state, actor_user_id, reason)
  VALUES (NEW.id, CASE WHEN TG_OP = 'UPDATE' THEN OLD.state END, NEW.state, app_user(), app_reason());
  RETURN NULL;
END $$;
CREATE TRIGGER trg_log_signup_state AFTER INSERT OR UPDATE OF state ON adventure_signups
  FOR EACH ROW EXECUTE FUNCTION log_signup_state();

CREATE FUNCTION log_relation_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND OLD.standing = NEW.standing AND OLD.kind = NEW.kind THEN RETURN NULL; END IF;
  INSERT INTO faction_relation_history (relation_id, old_standing, new_standing, old_kind, new_kind,
                                        game_day, cause_event_id, author_id)
  VALUES (NEW.id, CASE WHEN TG_OP = 'UPDATE' THEN OLD.standing END, NEW.standing,
          CASE WHEN TG_OP = 'UPDATE' THEN OLD.kind END, NEW.kind,
          current_game_day(), app_event(), app_user());
  RETURN NULL;
END $$;
CREATE TRIGGER trg_log_relation_change AFTER INSERT OR UPDATE OF standing, kind ON faction_relations
  FOR EACH ROW EXECUTE FUNCTION log_relation_change();

CREATE FUNCTION log_territory_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'DELETE' THEN
    INSERT INTO territory_history (map_id, q, r, faction_id, old_control_type, new_control_type,
                                  old_strength, new_strength, game_day, cause_event_id, author_id)
    VALUES (OLD.map_id, OLD.q, OLD.r, OLD.faction_id, OLD.control_type, NULL, OLD.strength, NULL,
            current_game_day(), app_event(), app_user());
    RETURN OLD;
  END IF;
  IF TG_OP = 'UPDATE' AND OLD.control_type = NEW.control_type AND OLD.strength = NEW.strength THEN
    RETURN NULL;
  END IF;
  INSERT INTO territory_history (map_id, q, r, faction_id, old_control_type, new_control_type,
                                old_strength, new_strength, game_day, cause_event_id, author_id)
  VALUES (NEW.map_id, NEW.q, NEW.r, NEW.faction_id,
          CASE WHEN TG_OP = 'UPDATE' THEN OLD.control_type END, NEW.control_type,
          CASE WHEN TG_OP = 'UPDATE' THEN OLD.strength END, NEW.strength,
          current_game_day(), app_event(), app_user());
  RETURN NULL;
END $$;
CREATE TRIGGER trg_log_territory_change AFTER INSERT OR UPDATE OR DELETE ON territory_control
  FOR EACH ROW EXECUTE FUNCTION log_territory_change();

-- Enterprise values are projections of enterprise_log: every change leaves a record.
CREATE FUNCTION log_enterprise_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE ev uuid := NULLIF(current_setting('app.enterprise_event_id', true), '')::uuid;
BEGIN
  IF TG_TABLE_NAME = 'enterprises' THEN
    IF NEW.reputation IS DISTINCT FROM OLD.reputation THEN
      INSERT INTO enterprise_log (enterprise_id, event_id, kind, delta, game_day, author_id)
      VALUES (NEW.id, ev, 'reputation', jsonb_build_object('old', OLD.reputation, 'new', NEW.reputation),
              current_game_day(), app_user());
    END IF;
    IF NEW.level IS DISTINCT FROM OLD.level THEN
      INSERT INTO enterprise_log (enterprise_id, event_id, kind, delta, game_day, author_id)
      VALUES (NEW.id, ev, 'level', jsonb_build_object('old', OLD.level, 'new', NEW.level),
              current_game_day(), app_user());
    END IF;
    IF NEW.status IS DISTINCT FROM OLD.status THEN
      INSERT INTO enterprise_log (enterprise_id, event_id, kind, delta, game_day, author_id)
      VALUES (NEW.id, ev, 'status', jsonb_build_object('old', OLD.status, 'new', NEW.status),
              current_game_day(), app_user());
    END IF;
  ELSE
    IF TG_OP = 'INSERT' OR NEW.quantity IS DISTINCT FROM OLD.quantity THEN
      INSERT INTO enterprise_log (enterprise_id, event_id, kind, delta, game_day, author_id)
      VALUES (NEW.enterprise_id, ev, 'stock',
              jsonb_build_object('item', NEW.name, 'stock_id', NEW.id,
                                 'old', CASE WHEN TG_OP = 'UPDATE' THEN OLD.quantity END, 'new', NEW.quantity),
              current_game_day(), app_user());
    END IF;
  END IF;
  RETURN NULL;
END $$;
CREATE TRIGGER trg_log_enterprise AFTER UPDATE OF reputation, level, status ON enterprises
  FOR EACH ROW EXECUTE FUNCTION log_enterprise_change();
CREATE TRIGGER trg_log_enterprise_stock AFTER INSERT OR UPDATE OF quantity ON enterprise_stock
  FOR EACH ROW EXECUTE FUNCTION log_enterprise_change();

-- Balances change only when a ledger transaction is posted. In production the application role
-- should additionally have no UPDATE privilege on accounts.balance.
CREATE FUNCTION accounts_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    IF NEW.balance <> 0 THEN RAISE EXCEPTION 'a new account starts with a zero balance'; END IF;
  ELSIF NEW.balance IS DISTINCT FROM OLD.balance
        AND COALESCE(current_setting('app.ledger_post', true), '') <> 'on' THEN
    RAISE EXCEPTION 'balances change only by posting a ledger transaction'
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_accounts_guard BEFORE INSERT OR UPDATE OF balance ON accounts
  FOR EACH ROW EXECUTE FUNCTION accounts_guard();

CREATE FUNCTION get_or_create_account(p_owner_type text, p_owner_id uuid, p_currency uuid)
RETURNS uuid LANGUAGE plpgsql AS $$
DECLARE v uuid;
BEGIN
  INSERT INTO accounts (owner_type, owner_id, currency_id) VALUES (p_owner_type, p_owner_id, p_currency)
  ON CONFLICT (owner_type, owner_id, currency_id) DO UPDATE SET owner_id = EXCLUDED.owner_id
  RETURNING id INTO v;
  RETURN v;
END $$;

-- Reconciliation: every account whose balance differs from the sum of its posted entries.
CREATE FUNCTION verify_account_balances()
RETURNS TABLE (account_id uuid, balance bigint, ledger_sum bigint) LANGUAGE sql STABLE AS $$
  SELECT a.id, a.balance, COALESCE(sum(e.delta), 0)::bigint
    FROM accounts a
    LEFT JOIN ledger_entries e ON e.account_id = a.id
    LEFT JOIN ledger_transactions t ON t.id = e.transaction_id AND t.state = 'posted'
   GROUP BY a.id, a.balance
  HAVING a.balance <> COALESCE(sum(e.delta) FILTER (WHERE t.id IS NOT NULL), 0)
$$;

-- Reputation: the current value changes only through the log; a reversal must mirror its original.
CREATE FUNCTION reputation_value_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF COALESCE(current_setting('app.rep_post', true), '') <> 'on' THEN
    RAISE EXCEPTION 'reputation changes only through faction_reputation_log'
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_reputation_value_guard BEFORE INSERT OR UPDATE ON faction_reputation
  FOR EACH ROW EXECUTE FUNCTION reputation_value_guard();

CREATE FUNCTION reputation_reversal_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE o faction_reputation_log%ROWTYPE;
BEGIN
  IF NEW.reversal_of IS NULL THEN RETURN NEW; END IF;
  SELECT * INTO o FROM faction_reputation_log WHERE id = NEW.reversal_of;
  IF o.character_id <> NEW.character_id OR o.faction_id <> NEW.faction_id OR o.delta <> -NEW.delta THEN
    RAISE EXCEPTION 'a reputation reversal must mirror the original record (same character, faction, opposite delta)';
  END IF;
  IF o.reversal_of IS NOT NULL THEN RAISE EXCEPTION 'a reversal cannot itself be reversed'; END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_reputation_reversal_guard BEFORE INSERT ON faction_reputation_log
  FOR EACH ROW EXECUTE FUNCTION reputation_reversal_guard();

CREATE FUNCTION tier_monotonic_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM faction_reputation_tiers t
              WHERE t.faction_id IS NOT DISTINCT FROM NEW.faction_id AND t.id <> NEW.id
                AND ((t.ord < NEW.ord AND t.min_value >= NEW.min_value)
                  OR (t.ord > NEW.ord AND t.min_value <= NEW.min_value))) THEN
    RAISE EXCEPTION 'reputation tiers must have thresholds that grow with the tier number';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_tier_monotonic BEFORE INSERT OR UPDATE ON faction_reputation_tiers
  FOR EACH ROW EXECUTE FUNCTION tier_monotonic_guard();

CREATE FUNCTION faction_parent_cycle_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.parent_faction_id IS NOT NULL AND (
       NEW.parent_faction_id = NEW.id OR EXISTS (
         WITH RECURSIVE up AS (
           SELECT id, parent_faction_id FROM factions WHERE id = NEW.parent_faction_id
           UNION
           SELECT f.id, f.parent_faction_id FROM factions f JOIN up ON f.id = up.parent_faction_id)
         SELECT 1 FROM up WHERE id = NEW.id)) THEN
    RAISE EXCEPTION 'faction hierarchy cannot contain a cycle';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_faction_parent_cycle BEFORE INSERT OR UPDATE OF parent_faction_id ON factions
  FOR EACH ROW EXECUTE FUNCTION faction_parent_cycle_guard();

-- A political event is applied only when every effect has been applied, and revoked only when none is left.
CREATE FUNCTION political_event_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.state = 'applied' THEN
    IF NOT EXISTS (SELECT 1 FROM political_event_effects WHERE event_id = NEW.id)
       OR EXISTS (SELECT 1 FROM political_event_effects WHERE event_id = NEW.id AND state <> 'applied') THEN
      RAISE EXCEPTION 'every effect of the event must be applied first';
    END IF;
    NEW.applied_at := COALESCE(NEW.applied_at, now());
  ELSIF NEW.state = 'revoked' AND OLD.state = 'applied' THEN
    IF EXISTS (SELECT 1 FROM political_event_effects WHERE event_id = NEW.id AND state = 'applied') THEN
      RAISE EXCEPTION 'revert every applied effect before revoking the event';
    END IF;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_political_event_guard BEFORE UPDATE OF state ON political_events
  FOR EACH ROW EXECUTE FUNCTION political_event_guard();

-- Expeditions: a start needs members (and the leader among them); finishing frees the members.
CREATE FUNCTION expedition_status_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.status = 'active' THEN
    IF NOT EXISTS (SELECT 1 FROM expedition_members WHERE expedition_id = NEW.id AND left_at IS NULL) THEN
      RAISE EXCEPTION 'an expedition needs at least one member to start';
    END IF;
    IF NEW.leader_character_id IS NOT NULL AND NOT EXISTS (
         SELECT 1 FROM expedition_members WHERE expedition_id = NEW.id
            AND character_id = NEW.leader_character_id AND left_at IS NULL) THEN
      RAISE EXCEPTION 'the leader must be a member of the expedition';
    END IF;
    NEW.start_game_day := COALESCE(NEW.start_game_day, current_game_day());
  ELSIF NEW.status IN ('returned', 'aborted', 'failed') THEN
    NEW.end_game_day := COALESCE(NEW.end_game_day, current_game_day());
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_expedition_status_guard BEFORE UPDATE OF status ON expeditions
  FOR EACH ROW EXECUTE FUNCTION expedition_status_guard();

CREATE FUNCTION expedition_release_members() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  UPDATE expedition_members SET left_at = now(), left_game_day = COALESCE(NEW.end_game_day, current_game_day())
   WHERE expedition_id = NEW.id AND left_at IS NULL;
  RETURN NULL;
END $$;
CREATE TRIGGER trg_expedition_release AFTER UPDATE OF status ON expeditions
  FOR EACH ROW WHEN (NEW.status IN ('returned', 'aborted', 'failed'))
  EXECUTE FUNCTION expedition_release_members();

CREATE FUNCTION expedition_member_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE st text;
BEGIN
  SELECT status INTO st FROM expeditions WHERE id = NEW.expedition_id;
  IF st NOT IN ('planning', 'active') AND NEW.left_at IS NULL THEN
    RAISE EXCEPTION 'members cannot join an expedition that is %', st;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM characters WHERE id = NEW.character_id AND status = 'active') THEN
    RAISE EXCEPTION 'only active characters can join an expedition';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_expedition_member_guard BEFORE INSERT ON expedition_members
  FOR EACH ROW EXECUTE FUNCTION expedition_member_guard();

-- Inventory rows mirror the Foundry actor: only the sync path writes them for linked characters.
CREATE FUNCTION inventory_mirror_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE cid uuid := CASE WHEN TG_OP = 'DELETE' THEN OLD.character_id ELSE NEW.character_id END;
BEGIN
  IF COALESCE(current_setting('app.writer', true), '') <> 'foundry_sync'
     AND EXISTS (SELECT 1 FROM foundry_actors WHERE character_id = cid) THEN
    RAISE EXCEPTION 'inventory belongs to the Foundry actor; change it through an integration job'
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END $$;
CREATE TRIGGER trg_inventory_mirror_guard BEFORE INSERT OR UPDATE OR DELETE ON inventory_items
  FOR EACH ROW EXECUTE FUNCTION inventory_mirror_guard();
-- the same integration operation never records the same item movement twice
CREATE UNIQUE INDEX inventory_events_operation_uq ON inventory_events (operation_id, character_id, catalog_id)
  WHERE operation_id IS NOT NULL;

-- Wiki: an edit bumps current_rev; the current revision exists and equals the page;
-- a page restricted to roles has at least one role. Checked at commit, so the page and its
-- version can be written in one transaction in any order.
CREATE FUNCTION wiki_edit_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.current_rev < OLD.current_rev THEN RAISE EXCEPTION 'current_rev cannot go back'; END IF;
  IF (NEW.title, NEW.body_md, NEW.gm_notes, NEW.attributes) IS DISTINCT FROM
     (OLD.title, OLD.body_md, OLD.gm_notes, OLD.attributes) AND NEW.current_rev = OLD.current_rev THEN
    RAISE EXCEPTION 'an edit of a wiki page must create a new revision (current_rev + 1)';
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER trg_wiki_edit_guard BEFORE UPDATE ON wiki_pages
  FOR EACH ROW EXECUTE FUNCTION wiki_edit_guard();

CREATE FUNCTION wiki_page_consistency() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM wiki_versions v
                  WHERE v.page_id = NEW.id AND v.rev = NEW.current_rev AND v.title = NEW.title
                    AND v.body_md = NEW.body_md AND v.gm_notes = NEW.gm_notes AND v.attributes = NEW.attributes) THEN
    RAISE EXCEPTION 'wiki page % has no matching version for revision %', NEW.id, NEW.current_rev;
  END IF;
  IF NEW.visibility = 'roles' AND NOT EXISTS (SELECT 1 FROM wiki_page_roles WHERE page_id = NEW.id) THEN
    RAISE EXCEPTION 'wiki page % is restricted to roles but has none', NEW.id;
  END IF;
  RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER trg_wiki_page_consistency AFTER INSERT OR UPDATE ON wiki_pages
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION wiki_page_consistency();

CREATE FUNCTION wiki_roles_consistency() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM wiki_pages p WHERE p.id = OLD.page_id AND p.visibility = 'roles')
     AND NOT EXISTS (SELECT 1 FROM wiki_page_roles WHERE page_id = OLD.page_id) THEN
    RAISE EXCEPTION 'wiki page % is restricted to roles but has none', OLD.page_id;
  END IF;
  RETURN NULL;
END $$;
CREATE CONSTRAINT TRIGGER trg_wiki_roles_consistency AFTER DELETE ON wiki_page_roles
  DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION wiki_roles_consistency();

-- Shop purchase: one transaction that locks the item, checks every rule, takes the money, reserves
-- the stock and queues the Foundry job. Money is taken immediately; if Foundry finally fails,
-- fail_purchase() returns the money and the stock with an exact reversal.
CREATE FUNCTION character_tier_ord(p_character uuid, p_faction uuid) RETURNS smallint
LANGUAGE plpgsql STABLE AS $$
DECLARE val integer; scale_owner uuid; res smallint;
BEGIN
  SELECT COALESCE(value, 0) INTO val FROM faction_reputation WHERE character_id = p_character AND faction_id = p_faction;
  val := COALESCE(val, 0);
  scale_owner := CASE WHEN EXISTS (SELECT 1 FROM faction_reputation_tiers WHERE faction_id = p_faction)
                      THEN p_faction END;
  SELECT max(ord) INTO res FROM faction_reputation_tiers
   WHERE faction_id IS NOT DISTINCT FROM scale_owner AND min_value <= val;
  RETURN res;
END $$;

CREATE FUNCTION purchase_item(p_stock uuid, p_character uuid, p_qty integer, p_idem text, p_author uuid)
RETURNS uuid LANGUAGE plpgsql AS $$
DECLARE
  st shop_stock%ROWTYPE; sh shops%ROWTYPE; ch characters%ROWTYPE; r record;
  v_tx uuid; v_purchase uuid; v_job uuid; total bigint; bought integer; val integer;
  v_char_acct uuid; v_shop_acct uuid;
BEGIN
  IF p_qty IS NULL OR p_qty <= 0 THEN RAISE EXCEPTION 'quantity must be positive'; END IF;
  SELECT p.id INTO v_purchase FROM purchases p JOIN ledger_transactions t ON t.id = p.transaction_id
   WHERE t.idempotency_key = p_idem;
  IF FOUND THEN RETURN v_purchase; END IF;                       -- repeated request: same result

  SELECT * INTO st FROM shop_stock WHERE id = p_stock FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION 'no such item in a shop'; END IF;
  SELECT * INTO sh FROM shops WHERE id = st.shop_id;
  IF NOT sh.active THEN RAISE EXCEPTION 'the shop is closed'; END IF;
  SELECT * INTO ch FROM characters WHERE id = p_character;
  IF NOT FOUND OR ch.status <> 'active' THEN RAISE EXCEPTION 'only an active character can buy'; END IF;
  IF st.min_character_level IS NOT NULL AND ch.level < st.min_character_level THEN
    RAISE EXCEPTION 'character level is too low for this item';
  END IF;
  IF st.stock_qty IS NOT NULL AND st.stock_qty < p_qty THEN RAISE EXCEPTION 'not enough stock'; END IF;
  IF st.per_character_limit IS NOT NULL THEN
    SELECT COALESCE(sum(quantity), 0) INTO bought FROM purchases
     WHERE stock_id = p_stock AND character_id = p_character AND state IN ('pending', 'confirmed');
    IF bought + p_qty > st.per_character_limit THEN RAISE EXCEPTION 'purchase limit per character exceeded'; END IF;
  END IF;
  FOR r IN SELECT * FROM shop_requirements WHERE shop_id = st.shop_id OR stock_id = st.id LOOP
    IF r.kind = 'faction_reputation' THEN
      SELECT COALESCE((SELECT value FROM faction_reputation
                        WHERE character_id = p_character AND faction_id = r.faction_id), 0) INTO val;
      IF val < r.min_value THEN RAISE EXCEPTION 'reputation with the faction is too low'; END IF;
    ELSIF COALESCE(character_tier_ord(p_character, r.faction_id), -1) < r.min_tier_ord THEN
      RAISE EXCEPTION 'reputation tier with the faction is too low';
    END IF;
  END LOOP;

  total := st.price_amount * p_qty;
  IF total <= 0 THEN RAISE EXCEPTION 'free items are granted as rewards, not bought'; END IF;
  v_char_acct := get_or_create_account('character', p_character, st.price_currency_id);
  v_shop_acct := get_or_create_account('shop', st.shop_id, st.price_currency_id);
  INSERT INTO ledger_transactions (idempotency_key, kind, source, author_id, reason)
  VALUES (p_idem, 'purchase', 'app', p_author, 'purchase') RETURNING id INTO v_tx;
  INSERT INTO ledger_entries (transaction_id, account_id, delta)
  VALUES (v_tx, v_char_acct, -total), (v_tx, v_shop_acct, total);
  UPDATE ledger_transactions SET state = 'posted' WHERE id = v_tx;   -- fails when the balance is not enough
  IF st.stock_qty IS NOT NULL THEN UPDATE shop_stock SET stock_qty = stock_qty - p_qty WHERE id = p_stock; END IF;
  INSERT INTO purchases (stock_id, character_id, quantity, transaction_id)
  VALUES (p_stock, p_character, p_qty, v_tx) RETURNING id INTO v_purchase;
  INSERT INTO integration_jobs (target, kind, payload, parent_type, parent_id, created_by)
  VALUES ('foundry', 'grant_item',
          jsonb_build_object('character_id', p_character, 'catalog_id', st.catalog_id,
                             'quantity', p_qty, 'purchase_id', v_purchase),
          'purchase', v_purchase, p_author) RETURNING id INTO v_job;
  UPDATE purchases SET job_id = v_job WHERE id = v_purchase;
  RETURN v_purchase;
END $$;

CREATE FUNCTION fail_purchase(p_purchase uuid, p_reason text) RETURNS void LANGUAGE plpgsql AS $$
DECLARE pu purchases%ROWTYPE; v_rev uuid;
BEGIN
  SELECT * INTO pu FROM purchases WHERE id = p_purchase FOR UPDATE;
  IF NOT FOUND OR pu.state <> 'pending' THEN RAISE EXCEPTION 'only a pending purchase can be failed'; END IF;
  INSERT INTO ledger_transactions (idempotency_key, kind, source, reason, reversal_of)
  VALUES ('reversal:' || pu.transaction_id, 'reversal', 'app', p_reason, pu.transaction_id) RETURNING id INTO v_rev;
  INSERT INTO ledger_entries (transaction_id, account_id, delta)
  SELECT v_rev, account_id, -sum(delta) FROM ledger_entries WHERE transaction_id = pu.transaction_id GROUP BY account_id;
  UPDATE ledger_transactions SET state = 'posted' WHERE id = v_rev;
  UPDATE shop_stock SET stock_qty = stock_qty + pu.quantity WHERE id = pu.stock_id AND stock_qty IS NOT NULL;
  UPDATE purchases SET state = 'failed' WHERE id = p_purchase;
END $$;

-- ---------------------------------------------------------------------
-- 17. Generic triggers: updated_at and append-only tables
-- ---------------------------------------------------------------------
CREATE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at := now(); RETURN NEW; END $$;

CREATE FUNCTION forbid_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'table % is append-only (corrections are made with a reversing record)', TG_TABLE_NAME
    USING ERRCODE = 'integrity_constraint_violation';
END $$;

DO $$
DECLARE t text;
BEGIN
  FOR t IN SELECT table_name FROM information_schema.columns
            WHERE table_schema = 'public' AND column_name = 'updated_at'
              AND table_name NOT IN (SELECT table_name FROM information_schema.views WHERE table_schema = 'public')
  LOOP
    EXECUTE format('CREATE TRIGGER trg_%s_updated BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION set_updated_at()', t, t);
  END LOOP;

  FOREACH t IN ARRAY ARRAY[
    'audit_log', 'campaign_settings_history', 'game_clock_history', 'login_log',
    'character_status_history', 'character_history', 'ledger_entries', 'inventory_events',
    'signup_events', 'exploration_history', 'expedition_supply_ledger', 'faction_reputation_log',
    'faction_relation_history', 'territory_history', 'enterprise_log', 'wiki_versions',
    'vault_sync_log', 'character_transfers'
  ] LOOP
    EXECUTE format('CREATE TRIGGER trg_%s_append_only BEFORE UPDATE OR DELETE ON %I FOR EACH ROW EXECUTE FUNCTION forbid_mutation()', t, t);
  END LOOP;
END $$;

-- ---------------------------------------------------------------------
-- 18. State machines, seed data
-- ---------------------------------------------------------------------
CREATE TABLE state_transitions (            -- the only allowed moves; from_state NULL = initial state
  entity text NOT NULL,
  from_state text,
  to_state text NOT NULL,
  UNIQUE NULLS NOT DISTINCT (entity, from_state, to_state)
);

CREATE FUNCTION enforce_state_transition() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE old_s text; new_s text;
BEGIN
  new_s := to_jsonb(NEW) ->> TG_ARGV[1];
  IF TG_OP = 'INSERT' THEN
    old_s := NULL;
  ELSE
    old_s := to_jsonb(OLD) ->> TG_ARGV[1];
    IF old_s IS NOT DISTINCT FROM new_s THEN RETURN NEW; END IF;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM state_transitions
                  WHERE entity = TG_ARGV[0] AND from_state IS NOT DISTINCT FROM old_s AND to_state = new_s) THEN
    RAISE EXCEPTION 'illegal % transition: % -> %', TG_ARGV[0], COALESCE(old_s, '(new)'), new_s
      USING ERRCODE = 'check_violation';
  END IF;
  RETURN NEW;
END $$;

CREATE TRIGGER trg_signup_transition BEFORE INSERT OR UPDATE OF state ON adventure_signups
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('signup', 'state');
CREATE TRIGGER trg_adventure_transition BEFORE INSERT OR UPDATE OF status ON adventures
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('adventure', 'status');
CREATE TRIGGER trg_character_transition BEFORE INSERT OR UPDATE OF status ON characters
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('character', 'status');
CREATE TRIGGER trg_job_transition BEFORE INSERT OR UPDATE OF state ON integration_jobs
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('job', 'state');
CREATE TRIGGER trg_expedition_transition BEFORE INSERT OR UPDATE OF status ON expeditions
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('expedition', 'status');
CREATE TRIGGER trg_purchase_transition BEFORE INSERT OR UPDATE OF state ON purchases
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('purchase', 'state');
CREATE TRIGGER trg_reward_transition BEFORE INSERT OR UPDATE OF state ON rewards
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('reward', 'state');
CREATE TRIGGER trg_political_event_transition BEFORE INSERT OR UPDATE OF state ON political_events
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('political_event', 'state');
CREATE TRIGGER trg_effect_transition BEFORE INSERT OR UPDATE OF state ON political_event_effects
  FOR EACH ROW EXECUTE FUNCTION enforce_state_transition('effect', 'state');

INSERT INTO state_transitions (entity, from_state, to_state) VALUES
  -- signups
  ('signup', NULL, 'queued'), ('signup', NULL, 'confirmed'), ('signup', NULL, 'pending_confirmation'),
  ('signup', 'queued', 'offered'), ('signup', 'queued', 'confirmed'), ('signup', 'queued', 'withdrawn'),
  ('signup', 'queued', 'removed'), ('signup', 'queued', 'declined'),
  ('signup', 'offered', 'confirmed'), ('signup', 'offered', 'declined'), ('signup', 'offered', 'expired'),
  ('signup', 'offered', 'removed'), ('signup', 'offered', 'withdrawn'),
  ('signup', 'pending_confirmation', 'confirmed'), ('signup', 'pending_confirmation', 'declined'),
  ('signup', 'pending_confirmation', 'removed'), ('signup', 'pending_confirmation', 'withdrawn'),
  ('signup', 'confirmed', 'withdrawn'), ('signup', 'confirmed', 'removed'),
  ('signup', 'withdrawn', 'queued'), ('signup', 'expired', 'queued'),
  -- adventures
  ('adventure', NULL, 'draft'), ('adventure', NULL, 'pending_approval'),
  ('adventure', 'draft', 'pending_approval'), ('adventure', 'draft', 'approved'), ('adventure', 'draft', 'cancelled'),
  ('adventure', 'pending_approval', 'approved'), ('adventure', 'pending_approval', 'draft'),
  ('adventure', 'pending_approval', 'cancelled'),
  ('adventure', 'approved', 'published'), ('adventure', 'approved', 'cancelled'),
  ('adventure', 'published', 'registration_open'), ('adventure', 'published', 'cancelled'),
  ('adventure', 'registration_open', 'registration_closed'), ('adventure', 'registration_open', 'running'),
  ('adventure', 'registration_open', 'cancelled'),
  ('adventure', 'registration_closed', 'registration_open'), ('adventure', 'registration_closed', 'running'),
  ('adventure', 'registration_closed', 'cancelled'),
  ('adventure', 'running', 'completed'), ('adventure', 'running', 'cancelled'),
  ('adventure', 'cancelled', 'draft'),
  -- characters
  ('character', NULL, 'draft'), ('character', NULL, 'pending_approval'), ('character', NULL, 'npc'),
  ('character', 'draft', 'pending_approval'), ('character', 'draft', 'archived'),
  ('character', 'pending_approval', 'active'), ('character', 'pending_approval', 'draft'),
  ('character', 'pending_approval', 'archived'),
  ('character', 'active', 'unavailable'), ('character', 'active', 'dead'), ('character', 'active', 'retired'),
  ('character', 'active', 'npc'),
  ('character', 'unavailable', 'active'), ('character', 'unavailable', 'dead'),
  ('character', 'unavailable', 'retired'), ('character', 'unavailable', 'npc'),
  ('character', 'dead', 'archived'), ('character', 'dead', 'active'),
  ('character', 'retired', 'archived'), ('character', 'retired', 'active'),
  ('character', 'npc', 'active'), ('character', 'npc', 'archived'),
  -- integration jobs
  ('job', NULL, 'pending'),
  ('job', 'pending', 'in_flight'), ('job', 'pending', 'cancelled'),
  ('job', 'in_flight', 'succeeded'), ('job', 'in_flight', 'failed'), ('job', 'in_flight', 'pending'),
  ('job', 'in_flight', 'awaiting_sync'),
  ('job', 'awaiting_sync', 'pending'), ('job', 'awaiting_sync', 'succeeded'), ('job', 'awaiting_sync', 'cancelled'),
  ('job', 'failed', 'pending'), ('job', 'failed', 'cancelled'),
  -- expeditions
  ('expedition', NULL, 'planning'),
  ('expedition', 'planning', 'active'), ('expedition', 'planning', 'aborted'),
  ('expedition', 'active', 'returned'), ('expedition', 'active', 'aborted'), ('expedition', 'active', 'failed'),
  -- purchases and rewards
  ('purchase', NULL, 'pending'),
  ('purchase', 'pending', 'confirmed'), ('purchase', 'pending', 'failed'), ('purchase', 'pending', 'cancelled'),
  ('purchase', 'failed', 'pending'),
  ('reward', NULL, 'pending'),
  ('reward', 'pending', 'awaiting_sync'), ('reward', 'pending', 'applied'), ('reward', 'pending', 'failed'),
  ('reward', 'awaiting_sync', 'applied'), ('reward', 'awaiting_sync', 'failed'), ('reward', 'awaiting_sync', 'pending'),
  ('reward', 'failed', 'pending'), ('reward', 'applied', 'reversed'),
  -- political events and their effects
  ('political_event', NULL, 'draft'),
  ('political_event', 'draft', 'applied'), ('political_event', 'draft', 'revoked'),
  ('political_event', 'applied', 'revoked'),
  ('effect', NULL, 'pending'),
  ('effect', 'pending', 'applied'), ('effect', 'pending', 'failed'),
  ('effect', 'failed', 'pending'), ('effect', 'applied', 'reverted');

INSERT INTO permissions (key, description) VALUES
  ('characters.create', 'Create own characters'),
  ('characters.approve', 'Approve or reject characters'),
  ('characters.edit_others', 'Edit other players characters'),
  ('characters.retire', 'Retire a character or turn it into an NPC'),
  ('adventures.create', 'Create adventures'),
  ('adventures.publish', 'Publish adventures'),
  ('adventures.approve_signups', 'Manage signups and the queue'),
  ('adventures.edit_others', 'Edit other people adventures'),
  ('rewards.distribute', 'Distribute rewards'),
  ('wiki.edit', 'Edit the wiki'),
  ('wiki.view_secrets', 'See GM notes and hidden pages'),
  ('maps.edit', 'Edit maps'),
  ('maps.reveal_fog', 'Reveal hexes'),
  ('expeditions.manage', 'Manage expeditions (routes, supplies, travel days)'),
  ('factions.manage', 'Manage factions, relations, territory, reputation'),
  ('economy.manage', 'Manage economy'),
  ('calendar.manage', 'Manage calendar and game time'),
  ('weather.manage', 'Manage weather'),
  ('foundry.manage', 'Manage the Foundry integration'),
  ('settings.manage', 'Manage campaign settings');

INSERT INTO roles (key, name, is_system) VALUES
  ('owner', 'Owner', true), ('admin', 'Administrator', true), ('gm', 'Game master', true), ('player', 'Player', true);

INSERT INTO campaign_settings (key, value) VALUES
  ('max_characters_per_player', '2'),
  ('max_party_level_spread', '2'),
  ('signup_offer_ttl_minutes', '720'),
  ('queue_priority', '["level_fit","waiting_longest","fewest_recent_games","earliest_signup"]');
