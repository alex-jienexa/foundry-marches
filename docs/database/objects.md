# Остальные объекты SQL

Генерируемая полная выборка: расширение, домен, функции, DO (в том числе динамические триггеры), seed-данные. Табличный DDL и явные CREATE TRIGGER — в [tables.md](tables.md). Это документация, не отдельный файл миграции.

```sql
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE DOMAIN visibility AS text
  CHECK (VALUE IN ('draft', 'gm_only', 'roles', 'public'));

CREATE FUNCTION log_setting_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO campaign_settings_history (key, old_value, new_value, changed_by)
  VALUES (NEW.key, CASE WHEN TG_OP = 'UPDATE' THEN OLD.value END, NEW.value, NEW.updated_by);
  RETURN NEW;
END $$;

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

CREATE FUNCTION seal_calendar_versions() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  UPDATE calendar_versions SET sealed_at = now()
   WHERE calendar_id = NEW.calendar_id AND sealed_at IS NULL AND effective_from_day <= NEW.current_day;
  RETURN NEW;
END $$;

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

CREATE FUNCTION ledger_entry_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  PERFORM 1 FROM ledger_transactions WHERE id = NEW.transaction_id AND state = 'pending' FOR SHARE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'transaction % is not pending; entries cannot be added', NEW.transaction_id
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN NEW;
END $$;

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

CREATE FUNCTION job_attempt_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'DELETE' OR OLD.finished_at IS NOT NULL THEN
    RAISE EXCEPTION 'a finished integration attempt is immutable' USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN NEW;
END $$;

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

CREATE FUNCTION apply_supply_delta() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  UPDATE expedition_supply_items SET quantity = quantity + NEW.delta, updated_at = now()
   WHERE id = NEW.supply_item_id;
  RETURN NEW;
END $$;

CREATE FUNCTION notify_outbox() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN PERFORM pg_notify('wm_events', NEW.id::text); RETURN NEW; END $$;

CREATE FUNCTION assert_hexes_on_map(p_map uuid, p_path jsonb) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
  IF p_path IS NULL THEN RETURN; END IF;
  IF EXISTS (SELECT 1 FROM jsonb_array_elements(p_path) e
              LEFT JOIN hexes h ON h.map_id = p_map AND h.q = (e->>0)::integer AND h.r = (e->>1)::integer
             WHERE h.map_id IS NULL) THEN
    RAISE EXCEPTION 'path contains hexes that are not on map %', p_map USING ERRCODE = 'foreign_key_violation';
  END IF;
END $$;

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

CREATE FUNCTION app_user() RETURNS uuid LANGUAGE sql STABLE AS
$$ SELECT NULLIF(current_setting('app.user_id', true), '')::uuid $$;

CREATE FUNCTION app_reason() RETURNS text LANGUAGE sql STABLE AS
$$ SELECT NULLIF(current_setting('app.reason', true), '') $$;

CREATE FUNCTION app_event() RETURNS uuid LANGUAGE sql STABLE AS
$$ SELECT NULLIF(current_setting('app.event_id', true), '')::uuid $$;

CREATE FUNCTION current_game_day() RETURNS bigint LANGUAGE sql STABLE AS
$$ SELECT current_day FROM game_clock LIMIT 1 $$;

CREATE FUNCTION log_character_status() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND OLD.status = NEW.status THEN RETURN NULL; END IF;
  INSERT INTO character_status_history (character_id, from_status, to_status, changed_by, reason)
  VALUES (NEW.id, CASE WHEN TG_OP = 'UPDATE' THEN OLD.status END, NEW.status, app_user(), app_reason());
  RETURN NULL;
END $$;

CREATE FUNCTION log_signup_state() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'UPDATE' AND OLD.state = NEW.state THEN RETURN NULL; END IF;
  INSERT INTO signup_events (signup_id, from_state, to_state, actor_user_id, reason)
  VALUES (NEW.id, CASE WHEN TG_OP = 'UPDATE' THEN OLD.state END, NEW.state, app_user(), app_reason());
  RETURN NULL;
END $$;

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

CREATE FUNCTION get_or_create_account(p_owner_type text, p_owner_id uuid, p_currency uuid)
RETURNS uuid LANGUAGE plpgsql AS $$
DECLARE v uuid;
BEGIN
  INSERT INTO accounts (owner_type, owner_id, currency_id) VALUES (p_owner_type, p_owner_id, p_currency)
  ON CONFLICT (owner_type, owner_id, currency_id) DO UPDATE SET owner_id = EXCLUDED.owner_id
  RETURNING id INTO v;
  RETURN v;
END $$;

CREATE FUNCTION verify_account_balances()
RETURNS TABLE (account_id uuid, balance bigint, ledger_sum bigint) LANGUAGE sql STABLE AS $$
  SELECT a.id, a.balance, COALESCE(sum(e.delta), 0)::bigint
    FROM accounts a
    LEFT JOIN ledger_entries e ON e.account_id = a.id
    LEFT JOIN ledger_transactions t ON t.id = e.transaction_id AND t.state = 'posted'
   GROUP BY a.id, a.balance
  HAVING a.balance <> COALESCE(sum(e.delta) FILTER (WHERE t.id IS NOT NULL), 0)
$$;

CREATE FUNCTION reputation_value_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF COALESCE(current_setting('app.rep_post', true), '') <> 'on' THEN
    RAISE EXCEPTION 'reputation changes only through faction_reputation_log'
      USING ERRCODE = 'integrity_constraint_violation';
  END IF;
  RETURN NEW;
END $$;

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

CREATE FUNCTION expedition_release_members() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  UPDATE expedition_members SET left_at = now(), left_game_day = COALESCE(NEW.end_game_day, current_game_day())
   WHERE expedition_id = NEW.id AND left_at IS NULL;
  RETURN NULL;
END $$;

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

CREATE FUNCTION wiki_edit_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.current_rev < OLD.current_rev THEN RAISE EXCEPTION 'current_rev cannot go back'; END IF;
  IF (NEW.title, NEW.body_md, NEW.gm_notes, NEW.attributes) IS DISTINCT FROM
     (OLD.title, OLD.body_md, OLD.gm_notes, OLD.attributes) AND NEW.current_rev = OLD.current_rev THEN
    RAISE EXCEPTION 'an edit of a wiki page must create a new revision (current_rev + 1)';
  END IF;
  RETURN NEW;
END $$;

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

CREATE FUNCTION wiki_roles_consistency() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM wiki_pages p WHERE p.id = OLD.page_id AND p.visibility = 'roles')
     AND NOT EXISTS (SELECT 1 FROM wiki_page_roles WHERE page_id = OLD.page_id) THEN
    RAISE EXCEPTION 'wiki page % is restricted to roles but has none', OLD.page_id;
  END IF;
  RETURN NULL;
END $$;

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
```
