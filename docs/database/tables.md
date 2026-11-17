# Полный каталог таблиц

Генерируется из `schema.sql`. Здесь все колонки и исходный DDL; поздние ALTER, индексы и явные триггеры включены. Динамические триггеры и полные функции — в [objects.md](objects.md).

- [users](#users)
- [discord_accounts](#discord_accounts)
- [roles](#roles)
- [permissions](#permissions)
- [role_permissions](#role_permissions)
- [user_roles](#user_roles)
- [discord_role_map](#discord_role_map)
- [sessions](#sessions)
- [service_tokens](#service_tokens)
- [player_gm_notes](#player_gm_notes)
- [notification_prefs](#notification_prefs)
- [login_log](#login_log)
- [campaign_settings](#campaign_settings)
- [campaign_settings_history](#campaign_settings_history)
- [audit_log](#audit_log)
- [backups](#backups)
- [calendars](#calendars)
- [calendar_versions](#calendar_versions)
- [calendar_months](#calendar_months)
- [calendar_weekdays](#calendar_weekdays)
- [calendar_leap_rules](#calendar_leap_rules)
- [game_clock](#game_clock)
- [game_clock_history](#game_clock_history)
- [wiki_pages](#wiki_pages)
- [wiki_page_roles](#wiki_page_roles)
- [wiki_versions](#wiki_versions)
- [wiki_aliases](#wiki_aliases)
- [wiki_relations](#wiki_relations)
- [wiki_backlinks](#wiki_backlinks)
- [assets](#assets)
- [chronicle_entries](#chronicle_entries)
- [calendar_events](#calendar_events)
- [calendar_event_links](#calendar_event_links)
- [maps](#maps)
- [terrain_types](#terrain_types)
- [map_regions](#map_regions)
- [hexes](#hexes)
- [hex_features](#hex_features)
- [map_linear_features](#map_linear_features)
- [map_pins](#map_pins)
- [routes](#routes)
- [exploration_history](#exploration_history)
- [climates](#climates)
- [weather_regions](#weather_regions)
- [weather_types](#weather_types)
- [weather_records](#weather_records)
- [foundry_worlds](#foundry_worlds)
- [characters](#characters)
- [character_transfers](#character_transfers)
- [character_status_history](#character_status_history)
- [character_reviews](#character_reviews)
- [foundry_actors](#foundry_actors)
- [character_history](#character_history)
- [factions](#factions)
- [faction_reputation_tiers](#faction_reputation_tiers)
- [faction_reputation](#faction_reputation)
- [faction_reputation_log](#faction_reputation_log)
- [faction_relations](#faction_relations)
- [political_events](#political_events)
- [political_event_effects](#political_event_effects)
- [faction_relation_history](#faction_relation_history)
- [territory_control](#territory_control)
- [territory_history](#territory_history)
- [currencies](#currencies)
- [level_progression](#level_progression)
- [accounts](#accounts)
- [ledger_transactions](#ledger_transactions)
- [ledger_entries](#ledger_entries)
- [integration_jobs](#integration_jobs)
- [integration_job_attempts](#integration_job_attempts)
- [items_catalog](#items_catalog)
- [inventory_items](#inventory_items)
- [inventory_events](#inventory_events)
- [treasure_budget_rules](#treasure_budget_rules)
- [shops](#shops)
- [shop_stock](#shop_stock)
- [shop_requirements](#shop_requirements)
- [purchases](#purchases)
- [adventures](#adventures)
- [adventure_wiki_links](#adventure_wiki_links)
- [adventure_signups](#adventure_signups)
- [adventure_multi_char_grants](#adventure_multi_char_grants)
- [signup_events](#signup_events)
- [adventure_participants](#adventure_participants)
- [discord_channels](#discord_channels)
- [discord_messages](#discord_messages)
- [rewards](#rewards)
- [adventure_reports](#adventure_reports)
- [report_links](#report_links)
- [expeditions](#expeditions)
- [expedition_members](#expedition_members)
- [expedition_routes](#expedition_routes)
- [expedition_legs](#expedition_legs)
- [expedition_supply_items](#expedition_supply_items)
- [expedition_supply_ledger](#expedition_supply_ledger)
- [expedition_events](#expedition_events)
- [expedition_visited_hexes](#expedition_visited_hexes)
- [enterprises](#enterprises)
- [enterprise_stock](#enterprise_stock)
- [enterprise_staff](#enterprise_staff)
- [enterprise_suppliers](#enterprise_suppliers)
- [enterprise_upgrades](#enterprise_upgrades)
- [enterprise_event_rules](#enterprise_event_rules)
- [enterprise_events](#enterprise_events)
- [enterprise_actions](#enterprise_actions)
- [enterprise_log](#enterprise_log)
- [vaults](#vaults)
- [vault_files](#vault_files)
- [vault_conflicts](#vault_conflicts)
- [foundry_journals](#foundry_journals)
- [vault_sync_log](#vault_sync_log)
- [notifications](#notifications)
- [notification_deliveries](#notification_deliveries)
- [event_outbox](#event_outbox)
- [state_transitions](#state_transitions)

---

## users

[Источник SQL](schema.sql) · 10 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `display_name` | `text` | нет | — | — | — |
| `avatar_url` | `text` | да | — | — | — |
| `status` | `text` | нет | — | — | `'active'` |
| `timezone` | `text` | нет | — | — | `'UTC'` |
| `preferred_schedule` | `text` | да | — | — | — |
| `registered_at` | `timestamptz` | нет | — | — | `now()` |
| `last_seen_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## discord_accounts

[Источник SQL](schema.sql) · 10 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `user_id` | `uuid` | нет | — | да | — |
| `discord_id` | `text` | нет | — | да | — |
| `username` | `text` | нет | — | — | — |
| `avatar_hash` | `text` | да | — | — | — |
| `in_guild` | `boolean` | нет | — | — | `TRUE` |
| `oauth_refresh_token_enc` | `bytea` | да | — | — | — |
| `linked_at` | `timestamptz` | нет | — | — | `now()` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## roles

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `key` | `text` | нет | — | да | — |
| `name` | `text` | нет | — | — | — |
| `is_system` | `boolean` | нет | — | — | `FALSE` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  key text NOT NULL UNIQUE,                 -- owner, admin, gm, player, ...
  name text NOT NULL,
  is_system boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
```

## permissions

[Источник SQL](schema.sql) · 2 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `key` | `text` | нет | да | — | — |
| `description` | `text` | нет | — | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE permissions (
  key text PRIMARY KEY,                     -- characters.approve, rewards.distribute, ...
  description text NOT NULL
);
```

## role_permissions

[Источник SQL](schema.sql) · 2 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `role_id` | `uuid` | нет | да | — | — |
| `permission_key` | `text` | нет | да | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE role_permissions (
  role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_key text NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_key)
);
```

## user_roles

[Источник SQL](schema.sql) · 4 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `user_id` | `uuid` | нет | да | — | — |
| `role_id` | `uuid` | нет | да | — | — |
| `granted_by` | `uuid` | да | — | — | — |
| `granted_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE user_roles (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  granted_by uuid REFERENCES users(id),
  granted_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, role_id)
);
```

## discord_role_map

[Источник SQL](schema.sql) · 3 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `guild_id` | `text` | нет | да | — | — |
| `discord_role_id` | `text` | нет | да | — | — |
| `role_id` | `uuid` | нет | — | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE discord_role_map (
  guild_id text NOT NULL,
  discord_role_id text NOT NULL,
  role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (guild_id, discord_role_id)
);
```

## sessions

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `user_id` | `uuid` | нет | — | — | — |
| `token_hash` | `bytea` | нет | — | да | — |
| `expires_at` | `timestamptz` | нет | — | — | — |
| `ip` | `inet` | да | — | — | — |
| `user_agent` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash bytea NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  ip inet,
  user_agent text,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## service_tokens

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | да | — |
| `token_hash` | `bytea` | нет | — | да | — |
| `scopes` | `text[]` | нет | — | — | `'{}'` |
| `revoked_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE service_tokens (               -- the Discord bot and other services
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL UNIQUE,
  token_hash bytea NOT NULL UNIQUE,
  scopes text[] NOT NULL DEFAULT '{}',
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## player_gm_notes

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `user_id` | `uuid` | нет | — | — | — |
| `author_id` | `uuid` | нет | — | — | — |
| `body` | `text` | нет | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE player_gm_notes (              -- visible to GMs only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  author_id uuid NOT NULL REFERENCES users(id),
  body text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
```

## notification_prefs

[Источник SQL](schema.sql) · 3 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `user_id` | `uuid` | нет | да | — | — |
| `kind` | `text` | нет | да | — | — |
| `enabled` | `boolean` | нет | — | — | `TRUE` |

### Полный DDL и комментарии

```sql
CREATE TABLE notification_prefs (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  PRIMARY KEY (user_id, kind)
);
```

## login_log

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `user_id` | `uuid` | да | — | — | — |
| `ok` | `boolean` | нет | — | — | — |
| `ip` | `inet` | да | — | — | — |
| `detail` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE login_log (                    -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid REFERENCES users(id),
  ok boolean NOT NULL,
  ip inet,
  detail text,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## campaign_settings

[Источник SQL](schema.sql) · 4 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `key` | `text` | нет | да | — | — |
| `value` | `jsonb` | нет | — | — | — |
| `updated_by` | `uuid` | да | — | — | — |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE campaign_settings (            -- max_characters_per_player, max_party_level_spread, ...
  key text PRIMARY KEY,
  value jsonb NOT NULL,
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now()
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_log_setting_change AFTER INSERT OR UPDATE ON campaign_settings FOR EACH ROW EXECUTE PROCEDURE log_setting_change();
```

## campaign_settings_history

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `bigint` | нет | да | — | — |
| `key` | `text` | нет | — | — | — |
| `old_value` | `jsonb` | да | — | — | — |
| `new_value` | `jsonb` | да | — | — | — |
| `changed_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE campaign_settings_history (    -- append-only
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  key text NOT NULL,
  old_value jsonb,
  new_value jsonb,
  changed_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## audit_log

[Источник SQL](schema.sql) · 12 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `bigint` | нет | да | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `actor_user_id` | `uuid` | да | — | — | — |
| `actor_kind` | `text` | нет | — | — | — |
| `action` | `text` | нет | — | — | — |
| `entity_type` | `text` | нет | — | — | — |
| `entity_id` | `text` | да | — | — | — |
| `old_value` | `jsonb` | да | — | — | — |
| `new_value` | `jsonb` | да | — | — | — |
| `result` | `text` | нет | — | — | `'ok'` |
| `request_id` | `text` | да | — | — | — |
| `ip` | `inet` | да | — | — | — |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX audit_log_entity_idx ON audit_log (entity_type, entity_id);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_user_id, created_at DESC);
```

## backups

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `kind` | `text` | нет | — | — | — |
| `path` | `text` | нет | — | — | — |
| `size_bytes` | `bigint` | да | — | — | — |
| `sha256` | `text` | да | — | — | — |
| `verified_at` | `timestamptz` | да | — | — | — |
| `verify_result` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## calendars

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | — | — |
| `year_name` | `text` | да | — | — | — |
| `epoch_name` | `text` | да | — | — | — |
| `is_active` | `boolean` | нет | — | — | `FALSE` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE calendars (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  year_name text,
  epoch_name text,
  is_active boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE UNIQUE INDEX calendars_one_active ON calendars ((TRUE)) WHERE is_active;
```

## calendar_versions

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `calendar_id` | `uuid` | нет | — | — | — |
| `version` | `integer` | нет | — | — | — |
| `effective_from_day` | `bigint` | нет | — | — | — |
| `anchor_year` | `integer` | нет | — | — | — |
| `anchor_month_ord` | `smallint` | нет | — | — | `1` |
| `anchor_day` | `smallint` | нет | — | — | `1` |
| `anchor_weekday_ord` | `smallint` | нет | — | — | `1` |
| `sealed_at` | `timestamptz` | да | — | — | — |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_calendar_version_guard BEFORE INSERT OR DELETE OR UPDATE ON calendar_versions FOR EACH ROW EXECUTE PROCEDURE calendar_version_guard();
```

## calendar_months

[Источник SQL](schema.sql) · 5 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `version_id` | `uuid` | нет | да | — | — |
| `ord` | `smallint` | нет | да | — | — |
| `name` | `text` | нет | — | — | — |
| `days` | `smallint` | нет | — | — | — |
| `is_intercalary` | `boolean` | нет | — | — | `FALSE` |

### Полный DDL и комментарии

```sql
CREATE TABLE calendar_months (              -- intercalary periods are months with is_intercalary
  version_id uuid NOT NULL REFERENCES calendar_versions(id) ON DELETE CASCADE,
  ord smallint NOT NULL CHECK (ord > 0),
  name text NOT NULL,
  days smallint NOT NULL CHECK (days > 0),
  is_intercalary boolean NOT NULL DEFAULT false,  -- does not take part in the week cycle
  PRIMARY KEY (version_id, ord)
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_calendar_months_guard BEFORE INSERT OR DELETE OR UPDATE ON calendar_months FOR EACH ROW EXECUTE PROCEDURE calendar_child_guard();
```

## calendar_weekdays

[Источник SQL](schema.sql) · 3 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `version_id` | `uuid` | нет | да | — | — |
| `ord` | `smallint` | нет | да | — | — |
| `name` | `text` | нет | — | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE calendar_weekdays (
  version_id uuid NOT NULL REFERENCES calendar_versions(id) ON DELETE CASCADE,
  ord smallint NOT NULL CHECK (ord > 0),
  name text NOT NULL,
  PRIMARY KEY (version_id, ord)
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_calendar_weekdays_guard BEFORE INSERT OR DELETE OR UPDATE ON calendar_weekdays FOR EACH ROW EXECUTE PROCEDURE calendar_child_guard();
```

## calendar_leap_rules

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `version_id` | `uuid` | нет | — | — | — |
| `every_n_years` | `integer` | нет | — | — | — |
| `offset_years` | `integer` | нет | — | — | `0` |
| `month_ord` | `smallint` | нет | — | — | — |
| `extra_days` | `smallint` | нет | — | — | `1` |

### Полный DDL и комментарии

```sql
CREATE TABLE calendar_leap_rules (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  version_id uuid NOT NULL REFERENCES calendar_versions(id) ON DELETE CASCADE,
  every_n_years integer NOT NULL CHECK (every_n_years > 0),
  offset_years integer NOT NULL DEFAULT 0,
  month_ord smallint NOT NULL,
  extra_days smallint NOT NULL DEFAULT 1,
  FOREIGN KEY (version_id, month_ord) REFERENCES calendar_months(version_id, ord)
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_calendar_leap_guard BEFORE INSERT OR DELETE OR UPDATE ON calendar_leap_rules FOR EACH ROW EXECUTE PROCEDURE calendar_child_guard();
```

## game_clock

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `smallint` | нет | да | — | `1` |
| `calendar_id` | `uuid` | нет | — | — | — |
| `current_day` | `bigint` | нет | — | — | — |
| `mode` | `text` | нет | — | — | `'manual'` |
| `time_ratio` | `numeric(10, 3)` | нет | — | — | `0` |
| `last_auto_tick` | `timestamptz` | да | — | — | — |
| `updated_by` | `uuid` | да | — | — | — |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_seal_calendar AFTER INSERT OR UPDATE ON game_clock FOR EACH ROW EXECUTE PROCEDURE seal_calendar_versions();
```

## game_clock_history

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `bigint` | нет | да | — | — |
| `from_day` | `bigint` | нет | — | — | — |
| `to_day` | `bigint` | нет | — | — | — |
| `reason` | `text` | да | — | — | — |
| `changed_by` | `uuid` | да | — | — | — |
| `adventure_id` | `uuid` | да | — | — | — |
| `expedition_id` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
ALTER TABLE game_clock_history ADD FOREIGN KEY (adventure_id) REFERENCES adventures (id);
ALTER TABLE game_clock_history ADD FOREIGN KEY (expedition_id) REFERENCES expeditions (id);
```

## wiki_pages

[Источник SQL](schema.sql) · 14 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `type` | `text` | нет | — | — | — |
| `title` | `text` | нет | — | — | — |
| `slug` | `text` | нет | — | да | — |
| `parent_id` | `uuid` | да | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'draft'` |
| `attributes` | `jsonb` | нет | — | — | `'{}'` |
| `body_md` | `text` | нет | — | — | `''` |
| `gm_notes` | `text` | нет | — | — | `''` |
| `current_rev` | `integer` | нет | — | — | `1` |
| `archived_at` | `timestamptz` | да | — | — | — |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX wiki_pages_type_idx ON wiki_pages (type);
CREATE INDEX wiki_pages_parent_idx ON wiki_pages (parent_id);
CREATE TRIGGER trg_wiki_edit_guard BEFORE UPDATE ON wiki_pages FOR EACH ROW EXECUTE PROCEDURE wiki_edit_guard();
CREATE CONSTRAINT TRIGGER trg_wiki_page_consistency AFTER INSERT OR UPDATE ON wiki_pages DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE PROCEDURE wiki_page_consistency();
```

## wiki_page_roles

[Источник SQL](schema.sql) · 2 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `page_id` | `uuid` | нет | да | — | — |
| `role_id` | `uuid` | нет | да | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE wiki_page_roles (              -- visibility = 'roles'
  page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (page_id, role_id)
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE CONSTRAINT TRIGGER trg_wiki_roles_consistency AFTER DELETE ON wiki_page_roles DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE PROCEDURE wiki_roles_consistency();
```

## wiki_versions

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `page_id` | `uuid` | нет | — | — | — |
| `rev` | `integer` | нет | — | — | — |
| `title` | `text` | нет | — | — | — |
| `body_md` | `text` | нет | — | — | — |
| `gm_notes` | `text` | нет | — | — | `''` |
| `attributes` | `jsonb` | нет | — | — | `'{}'` |
| `author_id` | `uuid` | да | — | — | — |
| `source` | `text` | нет | — | — | — |
| `content_hash` | `text` | нет | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## wiki_aliases

[Источник SQL](schema.sql) · 2 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `page_id` | `uuid` | нет | — | — | — |
| `alias` | `text` | нет | — | да | — |

### Полный DDL и комментарии

```sql
CREATE TABLE wiki_aliases (                 -- old slugs and titles: renames do not break links
  page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  alias text NOT NULL UNIQUE
);
```

## wiki_relations

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `from_page_id` | `uuid` | нет | — | — | — |
| `to_page_id` | `uuid` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `attrs` | `jsonb` | нет | — | — | `'{}'` |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## wiki_backlinks

[Источник SQL](schema.sql) · 2 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `from_page_id` | `uuid` | нет | да | — | — |
| `to_page_id` | `uuid` | нет | да | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE wiki_backlinks (               -- derived from body_md on save
  from_page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  to_page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  PRIMARY KEY (from_page_id, to_page_id)
);
```

## assets

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `sha256` | `text` | нет | — | да | — |
| `mime` | `text` | нет | — | — | — |
| `path` | `text` | нет | — | — | — |
| `size_bytes` | `bigint` | нет | — | — | — |
| `uploaded_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE assets (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  sha256 text NOT NULL UNIQUE,
  mime text NOT NULL,
  path text NOT NULL,
  size_bytes bigint NOT NULL,
  uploaded_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## chronicle_entries

[Источник SQL](schema.sql) · 12 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `game_day` | `bigint` | нет | — | — | — |
| `title` | `text` | нет | — | — | — |
| `body_md` | `text` | нет | — | — | `''` |
| `visibility` | `visibility` | нет | — | — | `'draft'` |
| `state` | `text` | нет | — | — | `'draft'` |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `adventure_id` | `uuid` | да | — | — | — |
| `expedition_id` | `uuid` | да | — | — | — |
| `author_id` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX chronicle_day_idx ON chronicle_entries (game_day);
ALTER TABLE chronicle_entries ADD FOREIGN KEY (adventure_id) REFERENCES adventures (id);
ALTER TABLE chronicle_entries ADD FOREIGN KEY (expedition_id) REFERENCES expeditions (id);
```

## calendar_events

[Источник SQL](schema.sql) · 15 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `calendar_id` | `uuid` | нет | — | — | — |
| `title` | `text` | нет | — | — | — |
| `description` | `text` | да | — | — | — |
| `kind` | `text` | да | — | — | — |
| `start_day` | `bigint` | нет | — | — | — |
| `duration_days` | `integer` | нет | — | — | `1` |
| `recurrence` | `jsonb` | да | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `location_wiki_page_id` | `uuid` | да | — | — | — |
| `adventure_id` | `uuid` | да | — | — | — |
| `fired_at` | `timestamptz` | да | — | — | — |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX calendar_events_day_idx ON calendar_events (start_day);
ALTER TABLE calendar_events ADD FOREIGN KEY (adventure_id) REFERENCES adventures (id);
```

## calendar_event_links

[Источник SQL](schema.sql) · 3 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `event_id` | `uuid` | нет | да | — | — |
| `link_type` | `text` | нет | да | — | — |
| `ref_id` | `uuid` | нет | да | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE calendar_event_links (
  event_id uuid NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
  link_type text NOT NULL CHECK (link_type IN ('character', 'wiki', 'faction', 'adventure')),
  ref_id uuid NOT NULL,
  PRIMARY KEY (event_id, link_type, ref_id)
);
```

## maps

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | — | — |
| `width` | `integer` | нет | — | — | — |
| `height` | `integer` | нет | — | — | — |
| `hex_orientation` | `text` | нет | — | — | `'pointy'` |
| `hex_size_miles` | `numeric(8, 2)` | нет | — | — | — |
| `background_asset_id` | `uuid` | да | — | — | — |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## terrain_types

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `key` | `text` | нет | — | да | — |
| `name` | `text` | нет | — | — | — |
| `color` | `text` | нет | — | — | `'#888888'` |
| `icon` | `text` | да | — | — | — |
| `travel_cost_mult` | `numeric(4, 2)` | нет | — | — | `1` |
| `passable` | `boolean` | нет | — | — | `TRUE` |
| `is_custom` | `boolean` | нет | — | — | `FALSE` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## map_regions

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `color` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## hexes

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `map_id` | `uuid` | нет | да | — | — |
| `q` | `integer` | нет | да | — | — |
| `r` | `integer` | нет | да | — | — |
| `terrain_id` | `uuid` | да | — | — | — |
| `elevation_ft` | `integer` | да | — | — | — |
| `region_id` | `uuid` | да | — | — | — |
| `weather_region_id` | `uuid` | да | — | — | — |
| `fog_state` | `text` | нет | — | — | `'hidden'` |
| `gm_notes` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX hexes_fog_idx ON hexes (map_id, fog_state);
ALTER TABLE hexes ADD CONSTRAINT hexes_weather_region_fk FOREIGN KEY (weather_region_id, map_id) REFERENCES weather_regions (id, map_id) ON DELETE SET NULL (weather_region_id);
```

## hex_features

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `q` | `integer` | нет | — | — | — |
| `r` | `integer` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `icon` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## map_linear_features

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `name` | `text` | да | — | — | — |
| `path` | `jsonb` | нет | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE map_linear_features (          -- roads, rivers, bridges, borders
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('road', 'river', 'bridge', 'border')),
  name text,
  path jsonb NOT NULL,                      -- [[q,r], [q,r], ...]
  visibility visibility NOT NULL DEFAULT 'gm_only',
  created_at timestamptz NOT NULL DEFAULT now()
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_linear_path BEFORE INSERT OR UPDATE ON map_linear_features FOR EACH ROW EXECUTE PROCEDURE check_route_path();
```

## map_pins

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `q` | `integer` | нет | — | — | — |
| `r` | `integer` | нет | — | — | — |
| `label` | `text` | нет | — | — | — |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## routes

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `waypoints` | `jsonb` | нет | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE routes (                       -- named routes, independent of expeditions
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  map_id uuid NOT NULL REFERENCES maps(id) ON DELETE CASCADE,
  name text NOT NULL,
  waypoints jsonb NOT NULL,                 -- [[q,r], ...]
  visibility visibility NOT NULL DEFAULT 'gm_only',
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_routes_path BEFORE INSERT OR UPDATE ON routes FOR EACH ROW EXECUTE PROCEDURE check_route_path();
```

## exploration_history

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `q` | `integer` | нет | — | — | — |
| `r` | `integer` | нет | — | — | — |
| `previous_fog` | `text` | нет | — | — | — |
| `new_fog` | `text` | нет | — | — | — |
| `revealed_by` | `uuid` | да | — | — | — |
| `adventure_id` | `uuid` | да | — | — | — |
| `expedition_id` | `uuid` | да | — | — | — |
| `game_day` | `bigint` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX exploration_hex_idx ON exploration_history (map_id, q, r);
ALTER TABLE exploration_history ADD FOREIGN KEY (adventure_id) REFERENCES adventures (id);
ALTER TABLE exploration_history ADD FOREIGN KEY (expedition_id, map_id) REFERENCES expeditions (id, map_id);
```

## climates

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `key` | `text` | нет | — | да | — |
| `name` | `text` | нет | — | — | — |
| `params` | `jsonb` | нет | — | — | `'{}'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE climates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  key text NOT NULL UNIQUE,
  name text NOT NULL,
  params jsonb NOT NULL DEFAULT '{}',       -- seasonal curves, transition probabilities
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
```

## weather_regions

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `climate_id` | `uuid` | нет | — | — | — |
| `map_region_id` | `uuid` | да | — | — | — |
| `base_elevation_ft` | `integer` | да | — | — | — |
| `forecast_visible_to_players` | `boolean` | нет | — | — | `FALSE` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
ALTER TABLE weather_regions ADD UNIQUE (id, map_id);
ALTER TABLE weather_regions ADD FOREIGN KEY (map_region_id, map_id) REFERENCES map_regions (id, map_id) ON DELETE SET NULL (map_region_id);
```

## weather_types

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `key` | `text` | нет | — | да | — |
| `name` | `text` | нет | — | — | — |
| `severity` | `smallint` | нет | — | — | `0` |
| `travel_mult` | `numeric(4, 2)` | нет | — | — | `1` |
| `visibility_mult` | `numeric(4, 2)` | нет | — | — | `1` |
| `icon` | `text` | да | — | — | — |
| `is_custom` | `boolean` | нет | — | — | `FALSE` |

### Полный DDL и комментарии

```sql
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
```

## weather_records

[Источник SQL](schema.sql) · 19 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `weather_region_id` | `uuid` | нет | — | — | — |
| `game_day` | `bigint` | нет | — | — | — |
| `weather_type_id` | `uuid` | нет | — | — | — |
| `temp_c` | `numeric(5, 1)` | да | — | — | — |
| `cloud_pct` | `smallint` | да | — | — | — |
| `wind_kmh` | `smallint` | да | — | — | — |
| `wind_dir_deg` | `smallint` | да | — | — | — |
| `visibility_m` | `integer` | да | — | — | — |
| `humidity_pct` | `smallint` | да | — | — | — |
| `snow_depth_cm` | `smallint` | да | — | — | — |
| `special` | `jsonb` | нет | — | — | `'{}'` |
| `state` | `text` | нет | — | — | `'forecast'` |
| `source` | `text` | нет | — | — | — |
| `seed` | `bigint` | да | — | — | — |
| `set_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## foundry_worlds

[Источник SQL](schema.sql) · 12 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | — | — |
| `relay_url` | `text` | нет | — | — | — |
| `relay_client_id` | `text` | нет | — | — | — |
| `key_secret_ref` | `text` | нет | — | — | — |
| `foundry_version` | `text` | да | — | — | — |
| `pf2e_system_version` | `text` | да | — | — | — |
| `is_primary` | `boolean` | нет | — | — | `FALSE` |
| `status` | `text` | нет | — | — | `'unknown'` |
| `last_seen_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE UNIQUE INDEX foundry_one_primary ON foundry_worlds ((TRUE)) WHERE is_primary;
```

## characters

[Источник SQL](schema.sql) · 26 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `owner_user_id` | `uuid` | да | — | — | — |
| `former_owner_user_id` | `uuid` | да | — | — | — |
| `name` | `text` | нет | — | — | — |
| `portrait_url` | `text` | да | — | — | — |
| `status` | `text` | нет | — | — | `'draft'` |
| `ancestry` | `text` | да | — | — | — |
| `heritage` | `text` | да | — | — | — |
| `background` | `text` | да | — | — | — |
| `class` | `text` | да | — | — | — |
| `level` | `smallint` | нет | — | — | `1` |
| `xp` | `integer` | нет | — | — | `0` |
| `level_up_pending` | `boolean` | нет | — | — | `FALSE` |
| `hp_current` | `integer` | да | — | — | — |
| `hp_max` | `integer` | да | — | — | — |
| `ac` | `integer` | да | — | — | — |
| `speed_ft` | `smallint` | да | — | — | — |
| `size` | `text` | да | — | — | — |
| `attributes` | `jsonb` | нет | — | — | `'{}'` |
| `skills` | `jsonb` | нет | — | — | `'{}'` |
| `gm_notes` | `text` | да | — | — | — |
| `npc_wiki_page_id` | `uuid` | да | — | — | — |
| `approved_by` | `uuid` | да | — | — | — |
| `approved_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX characters_owner_idx ON characters (owner_user_id, status);
CREATE TRIGGER trg_character_slots BEFORE INSERT OR UPDATE OF status, owner_user_id ON characters FOR EACH ROW EXECUTE PROCEDURE enforce_character_slots();
CREATE TRIGGER trg_character_transfer BEFORE UPDATE OF owner_user_id ON characters FOR EACH ROW EXECUTE PROCEDURE character_transfer_guard();
CREATE TRIGGER trg_characters_foundry_owned BEFORE UPDATE ON characters FOR EACH ROW EXECUTE PROCEDURE characters_foundry_owned_guard();
CREATE TRIGGER trg_log_character_status AFTER INSERT OR UPDATE OF status ON characters FOR EACH ROW EXECUTE PROCEDURE log_character_status();
CREATE TRIGGER trg_character_transition BEFORE INSERT OR UPDATE OF status ON characters FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('character', 'status');
```

## character_transfers

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `character_id` | `uuid` | нет | — | — | — |
| `from_user_id` | `uuid` | да | — | — | — |
| `to_user_id` | `uuid` | да | — | — | — |
| `transferred_by` | `uuid` | да | — | — | — |
| `reason` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE character_transfers (          -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id),
  from_user_id uuid REFERENCES users(id),
  to_user_id uuid REFERENCES users(id),     -- NULL when the character becomes an NPC
  transferred_by uuid REFERENCES users(id),
  reason text,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## character_status_history

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `character_id` | `uuid` | нет | — | — | — |
| `from_status` | `text` | да | — | — | — |
| `to_status` | `text` | нет | — | — | — |
| `changed_by` | `uuid` | да | — | — | — |
| `reason` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE character_status_history (     -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id),
  from_status text,
  to_status text NOT NULL,
  changed_by uuid REFERENCES users(id),
  reason text,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## character_reviews

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `character_id` | `uuid` | нет | — | — | — |
| `reviewer_id` | `uuid` | нет | — | — | — |
| `decision` | `text` | нет | — | — | — |
| `comment` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE character_reviews (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  character_id uuid NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  reviewer_id uuid NOT NULL REFERENCES users(id),
  decision text NOT NULL CHECK (decision IN ('approved', 'rejected', 'changes_requested', 'comment')),
  comment text,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## foundry_actors

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `world_id` | `uuid` | нет | — | — | — |
| `actor_uuid` | `text` | нет | — | — | — |
| `character_id` | `uuid` | да | — | да | — |
| `owner_confirmed_at` | `timestamptz` | да | — | — | — |
| `linked_at` | `timestamptz` | нет | — | — | `now()` |
| `last_synced_at` | `timestamptz` | да | — | — | — |
| `snapshot` | `jsonb` | да | — | — | — |
| `snapshot_hash` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE foundry_actors (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  world_id uuid NOT NULL REFERENCES foundry_worlds(id),
  actor_uuid text NOT NULL,                 -- permanent Foundry UUID, never the name
  character_id uuid UNIQUE REFERENCES characters(id),
  owner_confirmed_at timestamptz,           -- the player confirmed ownership; the owner itself is characters.owner_user_id
  linked_at timestamptz NOT NULL DEFAULT now(),
  last_synced_at timestamptz,
  snapshot jsonb,
  snapshot_hash text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (world_id, actor_uuid)             -- one Actor cannot belong to two characters
);
```

## character_history

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `character_id` | `uuid` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `ref_type` | `text` | да | — | — | — |
| `ref_id` | `uuid` | да | — | — | — |
| `payload` | `jsonb` | нет | — | — | `'{}'` |
| `author_id` | `uuid` | да | — | — | — |
| `reason` | `text` | да | — | — | — |
| `source` | `text` | нет | — | — | `'app'` |
| `game_day` | `bigint` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX character_history_idx ON character_history (character_id, created_at DESC);
```

## factions

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | — | — |
| `kind` | `text` | нет | — | — | `'faction'` |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `parent_faction_id` | `uuid` | да | — | — | — |
| `summary` | `text` | да | — | — | — |
| `color` | `text` | да | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `power_rating` | `smallint` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_faction_parent_cycle BEFORE INSERT OR UPDATE OF parent_faction_id ON factions FOR EACH ROW EXECUTE PROCEDURE faction_parent_cycle_guard();
```

## faction_reputation_tiers

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `faction_id` | `uuid` | да | — | — | — |
| `ord` | `smallint` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `min_value` | `integer` | нет | — | — | — |
| `perks` | `jsonb` | нет | — | — | `'{}'` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_tier_monotonic BEFORE INSERT OR UPDATE ON faction_reputation_tiers FOR EACH ROW EXECUTE PROCEDURE tier_monotonic_guard();
```

## faction_reputation

[Источник SQL](schema.sql) · 4 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `character_id` | `uuid` | нет | да | — | — |
| `faction_id` | `uuid` | нет | да | — | — |
| `value` | `integer` | нет | — | — | `0` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE faction_reputation (           -- current value, maintained from the log by trigger
  character_id uuid NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
  faction_id uuid NOT NULL REFERENCES factions(id) ON DELETE CASCADE,
  value integer NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (character_id, faction_id)
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_reputation_value_guard BEFORE INSERT OR UPDATE ON faction_reputation FOR EACH ROW EXECUTE PROCEDURE reputation_value_guard();
```

## faction_reputation_log

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `character_id` | `uuid` | нет | — | — | — |
| `faction_id` | `uuid` | нет | — | — | — |
| `delta` | `integer` | нет | — | — | — |
| `reason` | `text` | да | — | — | — |
| `source_type` | `text` | да | — | — | — |
| `source_id` | `uuid` | да | — | — | — |
| `game_day` | `bigint` | да | — | — | — |
| `author_id` | `uuid` | да | — | — | — |
| `reversal_of` | `uuid` | да | — | да | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_apply_reputation AFTER INSERT ON faction_reputation_log FOR EACH ROW EXECUTE PROCEDURE apply_reputation_delta();
CREATE TRIGGER trg_reputation_reversal_guard BEFORE INSERT ON faction_reputation_log FOR EACH ROW EXECUTE PROCEDURE reputation_reversal_guard();
```

## faction_relations

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `faction_a` | `uuid` | нет | — | — | — |
| `faction_b` | `uuid` | нет | — | — | — |
| `standing` | `smallint` | нет | — | — | `0` |
| `kind` | `text` | нет | — | — | `'neutral'` |
| `senior_faction_id` | `uuid` | да | — | — | — |
| `since_game_day` | `bigint` | да | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `notes_gm` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_log_relation_change AFTER INSERT OR UPDATE OF standing, kind ON faction_relations FOR EACH ROW EXECUTE PROCEDURE log_relation_change();
```

## political_events

[Источник SQL](schema.sql) · 15 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `title` | `text` | нет | — | — | — |
| `description_md` | `text` | нет | — | — | `''` |
| `game_day` | `bigint` | нет | — | — | — |
| `state` | `text` | нет | — | — | `'draft'` |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `effects` | `jsonb` | нет | — | — | `'[]'` |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `adventure_id` | `uuid` | да | — | — | — |
| `expedition_id` | `uuid` | да | — | — | — |
| `applied_by` | `uuid` | да | — | — | — |
| `applied_at` | `timestamptz` | да | — | — | — |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
ALTER TABLE political_events ADD FOREIGN KEY (adventure_id) REFERENCES adventures (id);
ALTER TABLE political_events ADD FOREIGN KEY (expedition_id) REFERENCES expeditions (id);
CREATE TRIGGER trg_political_event_guard BEFORE UPDATE OF state ON political_events FOR EACH ROW EXECUTE PROCEDURE political_event_guard();
CREATE TRIGGER trg_political_event_transition BEFORE INSERT OR UPDATE OF state ON political_events FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('political_event', 'state');
```

## political_event_effects

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `event_id` | `uuid` | нет | — | — | — |
| `ord` | `smallint` | нет | — | — | — |
| `effect` | `jsonb` | нет | — | — | — |
| `state` | `text` | нет | — | — | `'pending'` |
| `apply_key` | `uuid` | нет | — | да | `gen_random_uuid()` |
| `applied_at` | `timestamptz` | да | — | — | — |
| `reverted_at` | `timestamptz` | да | — | — | — |
| `error` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_effect_transition BEFORE INSERT OR UPDATE OF state ON political_event_effects FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('effect', 'state');
```

## faction_relation_history

[Источник SQL](schema.sql) · 10 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `relation_id` | `uuid` | нет | — | — | — |
| `old_standing` | `smallint` | да | — | — | — |
| `new_standing` | `smallint` | нет | — | — | — |
| `old_kind` | `text` | да | — | — | — |
| `new_kind` | `text` | нет | — | — | — |
| `game_day` | `bigint` | да | — | — | — |
| `cause_event_id` | `uuid` | да | — | — | — |
| `author_id` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## territory_control

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `map_id` | `uuid` | нет | да | — | — |
| `q` | `integer` | нет | да | — | — |
| `r` | `integer` | нет | да | — | — |
| `faction_id` | `uuid` | нет | да | — | — |
| `control_type` | `text` | нет | — | — | — |
| `strength` | `smallint` | нет | — | — | `50` |
| `since_game_day` | `bigint` | да | — | — | — |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE UNIQUE INDEX territory_one_controller ON territory_control (map_id, q, r) WHERE control_type = 'controlled';
CREATE TRIGGER trg_log_territory_change AFTER INSERT OR DELETE OR UPDATE ON territory_control FOR EACH ROW EXECUTE PROCEDURE log_territory_change();
```

## territory_history

[Источник SQL](schema.sql) · 13 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `map_id` | `uuid` | нет | — | — | — |
| `q` | `integer` | нет | — | — | — |
| `r` | `integer` | нет | — | — | — |
| `faction_id` | `uuid` | нет | — | — | — |
| `old_control_type` | `text` | да | — | — | — |
| `new_control_type` | `text` | да | — | — | — |
| `old_strength` | `smallint` | да | — | — | — |
| `new_strength` | `smallint` | да | — | — | — |
| `game_day` | `bigint` | да | — | — | — |
| `cause_event_id` | `uuid` | да | — | — | — |
| `author_id` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## currencies

[Источник SQL](schema.sql) · 14 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `key` | `text` | нет | — | да | — |
| `name` | `text` | нет | — | — | — |
| `description` | `text` | да | — | — | — |
| `icon` | `text` | да | — | — | — |
| `scale` | `smallint` | нет | — | — | `0` |
| `transferable` | `boolean` | нет | — | — | `FALSE` |
| `player_editable` | `boolean` | нет | — | — | `FALSE` |
| `tradeable` | `boolean` | нет | — | — | `FALSE` |
| `allow_negative` | `boolean` | нет | — | — | `FALSE` |
| `foundry_bound` | `boolean` | нет | — | — | `FALSE` |
| `foundry_binding` | `jsonb` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## level_progression

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | — | — |
| `xp_per_level` | `integer` | нет | — | — | `1000` |
| `max_level` | `smallint` | нет | — | — | `20` |
| `is_default` | `boolean` | нет | — | — | `FALSE` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE level_progression (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  xp_per_level integer NOT NULL DEFAULT 1000 CHECK (xp_per_level > 0),
  max_level smallint NOT NULL DEFAULT 20,
  is_default boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE UNIQUE INDEX level_progression_one_default ON level_progression ((TRUE)) WHERE is_default;
```

## accounts

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `owner_type` | `text` | нет | — | — | — |
| `owner_id` | `uuid` | нет | — | — | — |
| `currency_id` | `uuid` | нет | — | — | — |
| `balance` | `bigint` | нет | — | — | `0` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_accounts_guard BEFORE INSERT OR UPDATE OF balance ON accounts FOR EACH ROW EXECUTE PROCEDURE accounts_guard();
```

## ledger_transactions

[Источник SQL](schema.sql) · 13 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `idempotency_key` | `text` | нет | — | да | — |
| `kind` | `text` | нет | — | — | — |
| `state` | `text` | нет | — | — | `'pending'` |
| `source` | `text` | нет | — | — | — |
| `author_id` | `uuid` | да | — | — | — |
| `reason` | `text` | да | — | — | — |
| `reversal_of` | `uuid` | да | — | да | — |
| `adventure_id` | `uuid` | да | — | — | — |
| `expedition_id` | `uuid` | да | — | — | — |
| `game_day` | `bigint` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `posted_at` | `timestamptz` | да | — | — | — |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_ledger_tx_guard BEFORE DELETE OR UPDATE ON ledger_transactions FOR EACH ROW EXECUTE PROCEDURE ledger_tx_guard();
ALTER TABLE ledger_transactions ADD FOREIGN KEY (adventure_id) REFERENCES adventures (id);
ALTER TABLE ledger_transactions ADD FOREIGN KEY (expedition_id) REFERENCES expeditions (id);
```

## ledger_entries

[Источник SQL](schema.sql) · 5 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `transaction_id` | `uuid` | нет | — | — | — |
| `account_id` | `uuid` | нет | — | — | — |
| `delta` | `bigint` | нет | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE ledger_entries (               -- append-only; the sum per currency within a transaction is 0
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id uuid NOT NULL REFERENCES ledger_transactions(id),
  account_id uuid NOT NULL REFERENCES accounts(id),
  delta bigint NOT NULL CHECK (delta <> 0),
  created_at timestamptz NOT NULL DEFAULT now()
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX ledger_entries_tx_idx ON ledger_entries (transaction_id);
CREATE INDEX ledger_entries_account_idx ON ledger_entries (account_id, created_at DESC);
CREATE TRIGGER trg_ledger_entry_guard BEFORE INSERT ON ledger_entries FOR EACH ROW EXECUTE PROCEDURE ledger_entry_guard();
```

## integration_jobs

[Источник SQL](schema.sql) · 17 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `target` | `text` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `operation_id` | `uuid` | нет | — | да | `gen_random_uuid()` |
| `payload` | `jsonb` | нет | — | — | — |
| `precondition` | `jsonb` | да | — | — | — |
| `state` | `text` | нет | — | — | `'pending'` |
| `attempts` | `integer` | нет | — | — | `0` |
| `max_attempts` | `integer` | нет | — | — | `8` |
| `next_attempt_at` | `timestamptz` | нет | — | — | `now()` |
| `last_error` | `text` | да | — | — | — |
| `result` | `jsonb` | да | — | — | — |
| `parent_type` | `text` | да | — | — | — |
| `parent_id` | `uuid` | да | — | — | — |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX integration_jobs_due_idx ON integration_jobs (state, next_attempt_at);
CREATE INDEX integration_jobs_parent_idx ON integration_jobs (parent_type, parent_id);
CREATE TRIGGER trg_job_transition BEFORE INSERT OR UPDATE OF state ON integration_jobs FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('job', 'state');
```

## integration_job_attempts

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `job_id` | `uuid` | нет | — | — | — |
| `started_at` | `timestamptz` | нет | — | — | `now()` |
| `finished_at` | `timestamptz` | да | — | — | — |
| `http_status` | `integer` | да | — | — | — |
| `error` | `text` | да | — | — | — |
| `response` | `jsonb` | да | — | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE integration_job_attempts (     -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id uuid NOT NULL REFERENCES integration_jobs(id),
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz,
  http_status integer,
  error text,
  response jsonb
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_job_attempt_guard BEFORE DELETE OR UPDATE ON integration_job_attempts FOR EACH ROW EXECUTE PROCEDURE job_attempt_guard();
```

## items_catalog

[Источник SQL](schema.sql) · 12 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `source` | `text` | нет | — | — | — |
| `foundry_uuid` | `text` | да | — | да | — |
| `slug` | `text` | да | — | — | — |
| `name` | `text` | нет | — | — | — |
| `item_level` | `smallint` | да | — | — | — |
| `price_cp` | `bigint` | да | — | — | — |
| `traits` | `jsonb` | нет | — | — | `'[]'` |
| `data` | `jsonb` | нет | — | — | `'{}'` |
| `synced_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## inventory_items

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `character_id` | `uuid` | нет | — | — | — |
| `catalog_id` | `uuid` | да | — | — | — |
| `foundry_item_id` | `text` | да | — | — | — |
| `name` | `text` | нет | — | — | — |
| `quantity` | `integer` | нет | — | — | `1` |
| `state` | `jsonb` | нет | — | — | `'{}'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_inventory_mirror_guard BEFORE INSERT OR DELETE OR UPDATE ON inventory_items FOR EACH ROW EXECUTE PROCEDURE inventory_mirror_guard();
```

## inventory_events

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `character_id` | `uuid` | нет | — | — | — |
| `catalog_id` | `uuid` | да | — | — | — |
| `delta_qty` | `integer` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `transaction_id` | `uuid` | да | — | — | — |
| `operation_id` | `uuid` | да | — | — | — |
| `author_id` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE UNIQUE INDEX inventory_events_operation_uq ON inventory_events (operation_id, character_id, catalog_id) WHERE operation_id IS NOT NULL;
```

## treasure_budget_rules

[Источник SQL](schema.sql) · 2 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `level` | `smallint` | нет | да | — | — |
| `budget_cp` | `bigint` | нет | — | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE treasure_budget_rules (        -- loaded from the PF2e tables, not hardcoded; informational
  level smallint PRIMARY KEY,
  budget_cp bigint NOT NULL
);
```

## shops

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | — | — |
| `description` | `text` | да | — | — | — |
| `owner_wiki_page_id` | `uuid` | да | — | — | — |
| `location_wiki_page_id` | `uuid` | да | — | — | — |
| `map_id` | `uuid` | да | — | — | — |
| `q` | `integer` | да | — | — | — |
| `r` | `integer` | да | — | — | — |
| `active` | `boolean` | нет | — | — | `TRUE` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## shop_stock

[Источник SQL](schema.sql) · 10 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `shop_id` | `uuid` | нет | — | — | — |
| `catalog_id` | `uuid` | нет | — | — | — |
| `price_amount` | `bigint` | нет | — | — | — |
| `price_currency_id` | `uuid` | нет | — | — | — |
| `stock_qty` | `integer` | да | — | — | — |
| `per_character_limit` | `integer` | да | — | — | — |
| `min_character_level` | `smallint` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## shop_requirements

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `shop_id` | `uuid` | да | — | — | — |
| `stock_id` | `uuid` | да | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `faction_id` | `uuid` | нет | — | — | — |
| `min_value` | `integer` | да | — | — | — |
| `min_tier_ord` | `smallint` | да | — | — | — |

### Полный DDL и комментарии

```sql
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
```

## purchases

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `stock_id` | `uuid` | нет | — | — | — |
| `character_id` | `uuid` | нет | — | — | — |
| `quantity` | `integer` | нет | — | — | — |
| `transaction_id` | `uuid` | да | — | — | — |
| `job_id` | `uuid` | да | — | — | — |
| `state` | `text` | нет | — | — | `'pending'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_purchase_transition BEFORE INSERT OR UPDATE OF state ON purchases FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('purchase', 'state');
```

## adventures

[Источник SQL](schema.sql) · 26 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `title` | `text` | нет | — | — | — |
| `description` | `text` | нет | — | — | `''` |
| `hook` | `text` | да | — | — | — |
| `creator_user_id` | `uuid` | нет | — | — | — |
| `gm_user_id` | `uuid` | да | — | — | — |
| `status` | `text` | нет | — | — | `'draft'` |
| `starts_at` | `timestamptz` | да | — | — | — |
| `duration_min` | `integer` | да | — | — | — |
| `meeting_place` | `text` | да | — | — | — |
| `region_id` | `uuid` | да | — | — | — |
| `map_id` | `uuid` | да | — | — | — |
| `target_q` | `integer` | да | — | — | — |
| `target_r` | `integer` | да | — | — | — |
| `target_wiki_page_id` | `uuid` | да | — | — | — |
| `min_level` | `smallint` | нет | — | — | — |
| `max_level` | `smallint` | нет | — | — | — |
| `seats` | `smallint` | нет | — | — | — |
| `kind` | `text` | да | — | — | — |
| `constraints` | `jsonb` | нет | — | — | `'{}'` |
| `foundry_scene_uuid` | `text` | да | — | — | — |
| `start_game_day` | `bigint` | да | — | — | — |
| `end_game_day` | `bigint` | да | — | — | — |
| `cancel_reason` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX adventures_status_idx ON adventures (status, starts_at);
CREATE TRIGGER trg_adventure_params_guard BEFORE UPDATE OF seats, min_level, max_level ON adventures FOR EACH ROW EXECUTE PROCEDURE adventure_params_guard();
CREATE TRIGGER trg_sync_signup_period AFTER UPDATE OF starts_at, duration_min ON adventures FOR EACH ROW EXECUTE PROCEDURE sync_signup_period();
CREATE TRIGGER trg_adventure_transition BEFORE INSERT OR UPDATE OF status ON adventures FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('adventure', 'status');
```

## adventure_wiki_links

[Источник SQL](schema.sql) · 2 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `adventure_id` | `uuid` | нет | да | — | — |
| `wiki_page_id` | `uuid` | нет | да | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE adventure_wiki_links (
  adventure_id uuid NOT NULL REFERENCES adventures(id) ON DELETE CASCADE,
  wiki_page_id uuid NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
  PRIMARY KEY (adventure_id, wiki_page_id)
);
```

## adventure_signups

[Источник SQL](schema.sql) · 15 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `adventure_id` | `uuid` | нет | — | — | — |
| `character_id` | `uuid` | нет | — | — | — |
| `user_id` | `uuid` | нет | — | — | — |
| `state` | `text` | нет | — | — | `'queued'` |
| `queued_at` | `timestamptz` | нет | — | — | `now()` |
| `manual_order` | `integer` | да | — | — | — |
| `offered_at` | `timestamptz` | да | — | — | — |
| `offer_expires_at` | `timestamptz` | да | — | — | — |
| `decided_by` | `uuid` | да | — | — | — |
| `decision_reason` | `text` | да | — | — | — |
| `level_at_signup` | `smallint` | нет | — | — | — |
| `during` | `tstzrange` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX signups_queue_idx ON adventure_signups (adventure_id, state, queued_at);
CREATE TRIGGER trg_signup_guard BEFORE INSERT OR UPDATE OF adventure_id, character_id, user_id, state, level_at_signup ON adventure_signups FOR EACH ROW EXECUTE PROCEDURE signup_guard();
CREATE TRIGGER trg_log_signup_state AFTER INSERT OR UPDATE OF state ON adventure_signups FOR EACH ROW EXECUTE PROCEDURE log_signup_state();
CREATE TRIGGER trg_signup_transition BEFORE INSERT OR UPDATE OF state ON adventure_signups FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('signup', 'state');
```

## adventure_multi_char_grants

[Источник SQL](schema.sql) · 6 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `adventure_id` | `uuid` | нет | да | — | — |
| `user_id` | `uuid` | нет | да | — | — |
| `max_characters` | `smallint` | нет | — | — | — |
| `granted_by` | `uuid` | нет | — | — | — |
| `reason` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE adventure_multi_char_grants (   -- the GM explicitly allows N characters of one player
  adventure_id uuid NOT NULL REFERENCES adventures(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id),
  max_characters smallint NOT NULL CHECK (max_characters >= 2),
  granted_by uuid NOT NULL REFERENCES users(id),
  reason text,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (adventure_id, user_id)
);
```

## signup_events

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `signup_id` | `uuid` | нет | — | — | — |
| `from_state` | `text` | да | — | — | — |
| `to_state` | `text` | нет | — | — | — |
| `actor_user_id` | `uuid` | да | — | — | — |
| `reason` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE signup_events (                -- append-only
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  signup_id uuid NOT NULL REFERENCES adventure_signups(id),
  from_state text,
  to_state text NOT NULL,
  actor_user_id uuid REFERENCES users(id),
  reason text,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## adventure_participants

[Источник SQL](schema.sql) · 4 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `adventure_id` | `uuid` | нет | да | — | — |
| `character_id` | `uuid` | нет | да | — | — |
| `level_at_start` | `smallint` | нет | — | — | — |
| `attended` | `boolean` | нет | — | — | `TRUE` |

### Полный DDL и комментарии

```sql
CREATE TABLE adventure_participants (       -- fixed when the adventure starts or completes
  adventure_id uuid NOT NULL REFERENCES adventures(id) ON DELETE CASCADE,
  character_id uuid NOT NULL REFERENCES characters(id),
  level_at_start smallint NOT NULL,
  attended boolean NOT NULL DEFAULT true,
  PRIMARY KEY (adventure_id, character_id)
);
```

## discord_channels

[Источник SQL](schema.sql) · 3 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `kind` | `text` | нет | да | — | — |
| `guild_id` | `text` | нет | — | — | — |
| `channel_id` | `text` | нет | — | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE discord_channels (
  kind text PRIMARY KEY,                    -- announcements, adventures, characters, rewards, trade, reports, world, calendar
  guild_id text NOT NULL,
  channel_id text NOT NULL
);
```

## discord_messages

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `kind` | `text` | нет | — | — | — |
| `ref_type` | `text` | нет | — | — | — |
| `ref_id` | `uuid` | нет | — | — | — |
| `channel_id` | `text` | нет | — | — | — |
| `message_id` | `text` | нет | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## rewards

[Источник SQL](schema.sql) · 17 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `adventure_id` | `uuid` | да | — | — | — |
| `expedition_id` | `uuid` | да | — | — | — |
| `character_id` | `uuid` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `currency_id` | `uuid` | да | — | — | — |
| `amount` | `bigint` | да | — | — | — |
| `catalog_id` | `uuid` | да | — | — | — |
| `quantity` | `integer` | да | — | — | — |
| `comment` | `text` | да | — | — | — |
| `transaction_id` | `uuid` | да | — | — | — |
| `job_id` | `uuid` | да | — | — | — |
| `state` | `text` | нет | — | — | `'pending'` |
| `game_day` | `bigint` | да | — | — | — |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX rewards_char_idx ON rewards (character_id, created_at DESC);
ALTER TABLE rewards ADD FOREIGN KEY (expedition_id) REFERENCES expeditions (id);
CREATE TRIGGER trg_reward_transition BEFORE INSERT OR UPDATE OF state ON rewards FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('reward', 'state');
```

## adventure_reports

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `adventure_id` | `uuid` | нет | — | — | — |
| `author_user_id` | `uuid` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `body_md` | `text` | нет | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## report_links

[Источник SQL](schema.sql) · 3 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `report_id` | `uuid` | нет | да | — | — |
| `link_type` | `text` | нет | да | — | — |
| `ref_id` | `uuid` | нет | да | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE report_links (
  report_id uuid NOT NULL REFERENCES adventure_reports(id) ON DELETE CASCADE,
  link_type text NOT NULL CHECK (link_type IN ('location', 'npc', 'faction', 'event', 'wiki')),
  ref_id uuid NOT NULL,
  PRIMARY KEY (report_id, link_type, ref_id)
);
```

## expeditions

[Источник SQL](schema.sql) · 18 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | — | — |
| `map_id` | `uuid` | нет | — | — | — |
| `adventure_id` | `uuid` | да | — | — | — |
| `leader_character_id` | `uuid` | да | — | — | — |
| `status` | `text` | нет | — | — | `'planning'` |
| `origin_q` | `integer` | нет | — | — | — |
| `origin_r` | `integer` | нет | — | — | — |
| `current_q` | `integer` | нет | — | — | — |
| `current_r` | `integer` | нет | — | — | — |
| `planned_days` | `integer` | да | — | — | — |
| `start_game_day` | `bigint` | да | — | — | — |
| `end_game_day` | `bigint` | да | — | — | — |
| `travel_rules` | `jsonb` | нет | — | — | `'{}'` |
| `gm_notes` | `text` | да | — | — | — |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX expeditions_status_idx ON expeditions (status);
CREATE TRIGGER trg_expedition_status_guard BEFORE UPDATE OF status ON expeditions FOR EACH ROW EXECUTE PROCEDURE expedition_status_guard();
CREATE TRIGGER trg_expedition_release AFTER UPDATE OF status ON expeditions FOR EACH ROW WHEN (new.status IN ('returned', 'aborted', 'failed')) EXECUTE PROCEDURE expedition_release_members();
CREATE TRIGGER trg_expedition_transition BEFORE INSERT OR UPDATE OF status ON expeditions FOR EACH ROW EXECUTE PROCEDURE enforce_state_transition('expedition', 'status');
```

## expedition_members

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `expedition_id` | `uuid` | нет | да | — | — |
| `character_id` | `uuid` | нет | да | — | — |
| `role` | `text` | да | — | — | — |
| `speed_ft_snapshot` | `smallint` | да | — | — | — |
| `joined_game_day` | `bigint` | да | — | — | — |
| `left_game_day` | `bigint` | да | — | — | — |
| `left_at` | `timestamptz` | да | — | — | — |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE UNIQUE INDEX expedition_one_active_per_char ON expedition_members (character_id) WHERE left_at IS NULL;
CREATE TRIGGER trg_expedition_member_guard BEFORE INSERT ON expedition_members FOR EACH ROW EXECUTE PROCEDURE expedition_member_guard();
```

## expedition_routes

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `expedition_id` | `uuid` | нет | — | — | — |
| `version` | `integer` | нет | — | — | — |
| `waypoints` | `jsonb` | нет | — | — | — |
| `planned_hexes` | `jsonb` | нет | — | — | `'[]'` |
| `estimated_days` | `numeric(6, 2)` | да | — | — | — |
| `is_current` | `boolean` | нет | — | — | `TRUE` |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE UNIQUE INDEX expedition_one_current_route ON expedition_routes (expedition_id) WHERE is_current;
CREATE TRIGGER trg_exp_route_path BEFORE INSERT OR UPDATE ON expedition_routes FOR EACH ROW EXECUTE PROCEDURE check_route_path();
```

## expedition_legs

[Источник SQL](schema.sql) · 20 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `expedition_id` | `uuid` | нет | — | — | — |
| `day_no` | `integer` | нет | — | — | — |
| `game_day` | `bigint` | да | — | — | — |
| `map_id` | `uuid` | нет | — | — | — |
| `from_q` | `integer` | нет | — | — | — |
| `from_r` | `integer` | нет | — | — | — |
| `to_q` | `integer` | нет | — | — | — |
| `to_r` | `integer` | нет | — | — | — |
| `path` | `jsonb` | нет | — | — | `'[]'` |
| `travel_hours` | `smallint` | да | — | — | — |
| `movement_budget` | `numeric(8, 2)` | да | — | — | — |
| `hexes_moved` | `smallint` | да | — | — | — |
| `weather_record_id` | `uuid` | да | — | — | — |
| `modifiers` | `jsonb` | нет | — | — | `'{}'` |
| `state` | `text` | нет | — | — | `'planned'` |
| `outcome` | `text` | да | — | — | — |
| `gm_notes` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
ALTER TABLE expedition_legs ADD UNIQUE (id, expedition_id);
CREATE TRIGGER trg_leg_path BEFORE INSERT OR UPDATE ON expedition_legs FOR EACH ROW EXECUTE PROCEDURE check_route_path();
```

## expedition_supply_items

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `expedition_id` | `uuid` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `unit` | `text` | нет | — | — | `'pcs'` |
| `owner_character_id` | `uuid` | да | — | — | — |
| `catalog_id` | `uuid` | да | — | — | — |
| `per_member_per_day` | `numeric(8, 3)` | нет | — | — | `0` |
| `quantity` | `numeric(12, 3)` | нет | — | — | `0` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
ALTER TABLE expedition_supply_items ADD UNIQUE (id, expedition_id);
```

## expedition_supply_ledger

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `expedition_id` | `uuid` | нет | — | — | — |
| `supply_item_id` | `uuid` | нет | — | — | — |
| `leg_id` | `uuid` | да | — | — | — |
| `delta` | `numeric(12, 3)` | нет | — | — | — |
| `reason` | `text` | нет | — | — | — |
| `game_day` | `bigint` | да | — | — | — |
| `author_id` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_apply_supply AFTER INSERT ON expedition_supply_ledger FOR EACH ROW EXECUTE PROCEDURE apply_supply_delta();
ALTER TABLE expedition_supply_ledger ADD FOREIGN KEY (supply_item_id, expedition_id) REFERENCES expedition_supply_items (id, expedition_id);
ALTER TABLE expedition_supply_ledger ADD FOREIGN KEY (leg_id, expedition_id) REFERENCES expedition_legs (id, expedition_id);
```

## expedition_events

[Источник SQL](schema.sql) · 12 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `expedition_id` | `uuid` | нет | — | — | — |
| `leg_id` | `uuid` | да | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `title` | `text` | нет | — | — | — |
| `description` | `text` | да | — | — | — |
| `gm_notes` | `text` | да | — | — | — |
| `visibility` | `visibility` | нет | — | — | `'gm_only'` |
| `resolved` | `boolean` | нет | — | — | `FALSE` |
| `game_day` | `bigint` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
ALTER TABLE expedition_events ADD FOREIGN KEY (leg_id, expedition_id) REFERENCES expedition_legs (id, expedition_id);
```

## expedition_visited_hexes

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `expedition_id` | `uuid` | нет | да | — | — |
| `map_id` | `uuid` | нет | — | — | — |
| `q` | `integer` | нет | да | — | — |
| `r` | `integer` | нет | да | — | — |
| `first_leg_id` | `uuid` | да | — | — | — |
| `first_game_day` | `bigint` | да | — | — | — |
| `times_visited` | `integer` | нет | — | — | `1` |
| `revealed` | `boolean` | нет | — | — | `FALSE` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
ALTER TABLE expedition_visited_hexes ADD FOREIGN KEY (first_leg_id, expedition_id) REFERENCES expedition_legs (id, expedition_id);
```

## enterprises

[Источник SQL](schema.sql) · 15 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `name` | `text` | нет | — | — | — |
| `kind` | `text` | да | — | — | — |
| `owner_character_id` | `uuid` | да | — | — | — |
| `owner_user_id` | `uuid` | да | — | — | — |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `map_id` | `uuid` | да | — | — | — |
| `q` | `integer` | да | — | — | — |
| `r` | `integer` | да | — | — | — |
| `level` | `smallint` | нет | — | — | `1` |
| `reputation` | `integer` | нет | — | — | `0` |
| `description` | `text` | да | — | — | — |
| `status` | `text` | нет | — | — | `'active'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_log_enterprise AFTER UPDATE OF reputation, level, status ON enterprises FOR EACH ROW EXECUTE PROCEDURE log_enterprise_change();
```

## enterprise_stock

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `enterprise_id` | `uuid` | нет | — | — | — |
| `catalog_id` | `uuid` | да | — | — | — |
| `name` | `text` | нет | — | — | — |
| `unit` | `text` | нет | — | — | `'pcs'` |
| `quantity` | `numeric(12, 3)` | нет | — | — | `0` |
| `unit_value_cp` | `bigint` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE TRIGGER trg_log_enterprise_stock AFTER INSERT OR UPDATE OF quantity ON enterprise_stock FOR EACH ROW EXECUTE PROCEDURE log_enterprise_change();
```

## enterprise_staff

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `enterprise_id` | `uuid` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `role` | `text` | да | — | — | — |
| `npc_wiki_page_id` | `uuid` | да | — | — | — |
| `wage_cp` | `bigint` | да | — | — | — |
| `hired_game_day` | `bigint` | да | — | — | — |
| `left_game_day` | `bigint` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## enterprise_suppliers

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `enterprise_id` | `uuid` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `reliability` | `smallint` | да | — | — | — |
| `wiki_page_id` | `uuid` | да | — | — | — |
| `notes` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE enterprise_suppliers (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  enterprise_id uuid NOT NULL REFERENCES enterprises(id) ON DELETE CASCADE,
  name text NOT NULL,
  reliability smallint CHECK (reliability BETWEEN 0 AND 100),
  wiki_page_id uuid REFERENCES wiki_pages(id),
  notes text,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

## enterprise_upgrades

[Источник SQL](schema.sql) · 9 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `enterprise_id` | `uuid` | нет | — | — | — |
| `name` | `text` | нет | — | — | — |
| `description` | `text` | да | — | — | — |
| `cost_amount` | `bigint` | да | — | — | — |
| `cost_currency_id` | `uuid` | да | — | — | — |
| `state` | `text` | нет | — | — | `'available'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## enterprise_event_rules

[Источник SQL](schema.sql) · 10 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `enterprise_id` | `uuid` | да | — | — | — |
| `name` | `text` | нет | — | — | — |
| `trigger_kind` | `text` | нет | — | — | — |
| `config` | `jsonb` | нет | — | — | `'{}'` |
| `template` | `jsonb` | нет | — | — | `'{}'` |
| `requires_gm_approval` | `boolean` | нет | — | — | `TRUE` |
| `active` | `boolean` | нет | — | — | `TRUE` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## enterprise_events

[Источник SQL](schema.sql) · 16 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `enterprise_id` | `uuid` | нет | — | — | — |
| `rule_id` | `uuid` | да | — | — | — |
| `title` | `text` | нет | — | — | — |
| `description` | `text` | да | — | — | — |
| `occurred_game_day` | `bigint` | да | — | — | — |
| `expires_game_day` | `bigint` | да | — | — | — |
| `conditions` | `jsonb` | нет | — | — | `'{}'` |
| `available_actions` | `jsonb` | нет | — | — | `'[]'` |
| `consequences` | `jsonb` | нет | — | — | `'{}'` |
| `cost` | `jsonb` | да | — | — | — |
| `reward` | `jsonb` | да | — | — | — |
| `state` | `text` | нет | — | — | `'draft'` |
| `created_by` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## enterprise_actions

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `event_id` | `uuid` | нет | — | — | — |
| `character_id` | `uuid` | нет | — | — | — |
| `action_kind` | `text` | нет | — | — | — |
| `chosen_at` | `timestamptz` | нет | — | — | `now()` |
| `gm_outcome` | `text` | да | — | — | — |
| `resolved_by` | `uuid` | да | — | — | — |
| `resolved_at` | `timestamptz` | да | — | — | — |

### Полный DDL и комментарии

```sql
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
```

## enterprise_log

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `enterprise_id` | `uuid` | нет | — | — | — |
| `event_id` | `uuid` | да | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `delta` | `jsonb` | нет | — | — | `'{}'` |
| `game_day` | `bigint` | да | — | — | — |
| `author_id` | `uuid` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## vaults

[Источник SQL](schema.sql) · 14 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `kind` | `text` | нет | — | — | — |
| `name` | `text` | нет | — | да | — |
| `webdav_url` | `text` | нет | — | — | — |
| `remote_root` | `text` | нет | — | — | `'/'` |
| `username` | `text` | нет | — | — | — |
| `password_enc` | `bytea` | нет | — | — | — |
| `categories` | `text[]` | нет | — | — | `'{}'` |
| `sync_interval_sec` | `integer` | нет | — | — | `300` |
| `last_scan_at` | `timestamptz` | да | — | — | — |
| `last_ok_at` | `timestamptz` | да | — | — | — |
| `last_error` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## vault_files

[Источник SQL](schema.sql) · 14 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `vault_id` | `uuid` | нет | — | — | — |
| `entity_type` | `text` | нет | — | — | — |
| `entity_id` | `uuid` | нет | — | — | — |
| `rel_path` | `text` | нет | — | — | — |
| `exported_rev` | `integer` | да | — | — | — |
| `exported_hash` | `text` | да | — | — | — |
| `remote_etag` | `text` | да | — | — | — |
| `remote_mtime` | `timestamptz` | да | — | — | — |
| `last_seen_hash` | `text` | да | — | — | — |
| `last_seen_at` | `timestamptz` | да | — | — | — |
| `state` | `text` | нет | — | — | `'synced'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## vault_conflicts

[Источник SQL](schema.sql) · 11 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `vault_file_id` | `uuid` | нет | — | — | — |
| `base_rev` | `integer` | да | — | — | — |
| `db_rev` | `integer` | да | — | — | — |
| `file_hash` | `text` | нет | — | — | — |
| `conflict_path` | `text` | нет | — | — | — |
| `state` | `text` | нет | — | — | `'open'` |
| `resolution` | `text` | да | — | — | — |
| `resolved_by` | `uuid` | да | — | — | — |
| `resolved_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## foundry_journals

[Источник SQL](schema.sql) · 14 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `world_id` | `uuid` | нет | — | — | — |
| `journal_uuid` | `text` | нет | — | — | — |
| `sheet_type` | `text` | да | — | — | — |
| `wiki_page_id` | `uuid` | да | — | да | — |
| `folder_path` | `text` | да | — | — | — |
| `snapshot` | `jsonb` | да | — | — | — |
| `snapshot_hash` | `text` | да | — | — | — |
| `exported_rev` | `integer` | да | — | — | — |
| `last_seen_hash` | `text` | да | — | — | — |
| `state` | `text` | нет | — | — | `'synced'` |
| `last_synced_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `updated_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## vault_sync_log

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `bigint` | нет | да | — | — |
| `vault_id` | `uuid` | нет | — | — | — |
| `direction` | `text` | нет | — | — | — |
| `entity_id` | `uuid` | да | — | — | — |
| `action` | `text` | нет | — | — | — |
| `result` | `text` | нет | — | — | — |
| `detail` | `text` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## notifications

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `user_id` | `uuid` | нет | — | — | — |
| `kind` | `text` | нет | — | — | — |
| `payload` | `jsonb` | нет | — | — | `'{}'` |
| `dedupe_key` | `text` | да | — | — | — |
| `read_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
CREATE TABLE notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind text NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}',
  dedupe_key text,                          -- the same event is never delivered twice to one user
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE UNIQUE INDEX notifications_dedupe_idx ON notifications (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL;
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);
```

## notification_deliveries

[Источник SQL](schema.sql) · 8 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `uuid` | нет | да | — | `gen_random_uuid()` |
| `notification_id` | `uuid` | нет | — | — | — |
| `channel` | `text` | нет | — | — | — |
| `state` | `text` | нет | — | — | `'pending'` |
| `attempts` | `integer` | нет | — | — | `0` |
| `last_error` | `text` | да | — | — | — |
| `delivered_at` | `timestamptz` | да | — | — | — |
| `created_at` | `timestamptz` | нет | — | — | `now()` |

### Полный DDL и комментарии

```sql
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
```

## event_outbox

[Источник SQL](schema.sql) · 7 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `id` | `bigint` | нет | да | — | — |
| `kind` | `text` | нет | — | — | — |
| `entity_type` | `text` | да | — | — | — |
| `entity_id` | `uuid` | да | — | — | — |
| `payload` | `jsonb` | нет | — | — | `'{}'` |
| `created_at` | `timestamptz` | нет | — | — | `now()` |
| `published_at` | `timestamptz` | да | — | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE event_outbox (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  kind text NOT NULL,
  entity_type text,
  entity_id uuid,
  payload jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz
);
```

### Поздние ограничения, индексы и явные триггеры

```sql
CREATE INDEX event_outbox_unpublished_idx ON event_outbox (id) WHERE published_at IS NULL;
CREATE TRIGGER trg_notify_outbox AFTER INSERT ON event_outbox FOR EACH ROW EXECUTE PROCEDURE notify_outbox();
```

## state_transitions

[Источник SQL](schema.sql) · 3 колонок.

| Колонка | Тип PostgreSQL | NULL | PK | UNIQUE inline | Default |
|---|---|---|---|---|---|
| `entity` | `text` | нет | — | — | — |
| `from_state` | `text` | да | — | — | — |
| `to_state` | `text` | нет | — | — | — |

### Полный DDL и комментарии

```sql
CREATE TABLE state_transitions (            -- the only allowed moves; from_state NULL = initial state
  entity text NOT NULL,
  from_state text,
  to_state text NOT NULL,
  UNIQUE NULLS NOT DISTINCT (entity, from_state, to_state)
);
```
