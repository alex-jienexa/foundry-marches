-- Schema tests for schema.sql v0.3. Run on an EMPTY database right after schema.sql:
--   psql -X -v ON_ERROR_STOP=1 -d <db> -f schema_tests.sql
-- The run ends with an error (non-zero exit code) if any check failed, or if the number of
-- checks is not the expected one, so CI cannot go green over a failing test.
-- ok(name, sql)               : the statement must succeed
-- fails(name, sql, pattern)   : the statement must fail with a message containing the pattern
\set ON_ERROR_STOP on
\pset tuples_only on
\pset format unaligned
CREATE TEMP TABLE results(seq serial, name text, ok boolean, detail text);
CREATE FUNCTION pg_temp.ok(p_name text, p_sql text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN EXECUTE p_sql; INSERT INTO results(name, ok, detail) VALUES (p_name, true, 'ok');
EXCEPTION WHEN others THEN INSERT INTO results(name, ok, detail) VALUES (p_name, false, SQLERRM); END $$;
CREATE FUNCTION pg_temp.fails(p_name text, p_sql text, p_pattern text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN EXECUTE p_sql;
  INSERT INTO results(name, ok, detail) VALUES (p_name, false, 'expected an error, got none');
EXCEPTION WHEN others THEN
  IF SQLERRM ILIKE '%' || p_pattern || '%' THEN INSERT INTO results(name, ok, detail) VALUES (p_name, true, left(SQLERRM, 80));
  ELSE INSERT INTO results(name, ok, detail) VALUES (p_name, false, 'wrong error: ' || SQLERRM || ' (wanted: ' || p_pattern || ')'); END IF;
END $$;

-- ---------- fixtures
INSERT INTO users(id, display_name) VALUES
 ('11111111-1111-1111-1111-111111111111','alice'), ('22222222-2222-2222-2222-222222222222','bob'),
 ('33333333-3333-3333-3333-333333333333','gm'), ('44444444-4444-4444-4444-444444444444','carol');
INSERT INTO maps(id, name, width, height, hex_size_miles) VALUES
 ('aaaaaaaa-0000-0000-0000-000000000001','m1',10,10,6), ('aaaaaaaa-0000-0000-0000-000000000002','m2',10,10,6);
INSERT INTO hexes(map_id, q, r) SELECT m, q, 0 FROM unnest(ARRAY['aaaaaaaa-0000-0000-0000-000000000001'::uuid,'aaaaaaaa-0000-0000-0000-000000000002'::uuid]) m, generate_series(0,4) q;
INSERT INTO characters(id, owner_user_id, name, status, level) VALUES
 ('c0000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','a1','draft',1),
 ('c0000000-0000-0000-0000-000000000002','11111111-1111-1111-1111-111111111111','a2','draft',1),
 ('c0000000-0000-0000-0000-000000000003','11111111-1111-1111-1111-111111111111','a3','draft',1),
 ('c0000000-0000-0000-0000-000000000004','22222222-2222-2222-2222-222222222222','b1','draft',4),
 ('c0000000-0000-0000-0000-000000000005','44444444-4444-4444-4444-444444444444','d1','draft',1);

-- ================= 1. characters: state machine, slots, history
SELECT pg_temp.fails('character: draft -> active is illegal', $q$UPDATE characters SET status='active' WHERE name='a1'$q$, 'illegal character transition');
SELECT pg_temp.ok('character: draft -> pending_approval', $q$UPDATE characters SET status='pending_approval' WHERE name IN ('a1','a2')$q$);
SELECT pg_temp.fails('slots: third pending character is blocked', $q$UPDATE characters SET status='pending_approval' WHERE name='a3'$q$, 'slot limit');
SELECT pg_temp.ok('approve a1', $q$UPDATE characters SET status='active' WHERE name='a1'$q$);
SELECT pg_temp.ok('active -> unavailable keeps the slot', $q$UPDATE characters SET status='unavailable' WHERE name='a1'$q$);
SELECT pg_temp.ok('unavailable -> active', $q$UPDATE characters SET status='active' WHERE name='a1'$q$);
SELECT pg_temp.ok('approve a2', $q$UPDATE characters SET status='active' WHERE name='a2'$q$);
SELECT pg_temp.ok('bob and carol characters become active', $q$UPDATE characters SET status='pending_approval' WHERE name IN ('b1','d1'); UPDATE characters SET status='active' WHERE name IN ('b1','d1')$q$);
SELECT pg_temp.ok('status history is written automatically (5 rows for a1)', $q$SELECT 1/(SELECT (count(*) = 5)::int FROM character_status_history WHERE character_id='c0000000-0000-0000-0000-000000000001')$q$);
UPDATE campaign_settings SET value='1' WHERE key='max_characters_per_player';
SELECT pg_temp.ok('limit lowered: existing characters stay editable', $q$UPDATE characters SET hp_current=5 WHERE name='a1'$q$);
SELECT pg_temp.ok('limit lowered: active -> unavailable -> active is allowed', $q$UPDATE characters SET status='unavailable' WHERE name='a2'; UPDATE characters SET status='active' WHERE name='a2'$q$);
SELECT pg_temp.ok('a2 dies (frees a slot)', $q$UPDATE characters SET status='dead' WHERE name='a2'$q$);
SELECT pg_temp.fails('limit lowered: a new character cannot enter approval', $q$UPDATE characters SET status='pending_approval' WHERE name='a3'$q$, 'slot limit');
SELECT pg_temp.fails('limit lowered: revival is blocked when the slots are full', $q$UPDATE characters SET status='active' WHERE name='a2'$q$, 'slot limit');
UPDATE campaign_settings SET value='2' WHERE key='max_characters_per_player';
SELECT pg_temp.ok('limit restored: a3 can be approved', $q$UPDATE characters SET status='pending_approval' WHERE name='a3'; UPDATE characters SET status='active' WHERE name='a3'$q$);
SELECT pg_temp.ok('setting changes are logged', $q$SELECT 1/(SELECT (count(*) >= 2)::int FROM campaign_settings_history WHERE key='max_characters_per_player')$q$);

-- ================= 2. adventures and signups
INSERT INTO adventures(id,title,creator_user_id,min_level,max_level,seats,starts_at,duration_min) VALUES
 ('d0000000-0000-0000-0000-000000000001','adv1','33333333-3333-3333-3333-333333333333',1,5,2,'2026-12-01 18:00+00',240),
 ('d0000000-0000-0000-0000-000000000002','adv2','33333333-3333-3333-3333-333333333333',1,5,2,'2026-12-01 20:00+00',240);
SELECT pg_temp.fails('signup: adventure in draft', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','queued')$q$, 'not open for registration');
SELECT pg_temp.ok('adventure: draft -> approved -> published -> registration_open', $q$UPDATE adventures SET status='approved'; UPDATE adventures SET status='published'; UPDATE adventures SET status='registration_open'$q$);
SELECT pg_temp.fails('adventure: registration_open -> published is illegal', $q$UPDATE adventures SET status='published' WHERE title='adv1'$q$, 'illegal adventure transition');
SELECT pg_temp.ok('signup a1 on adv1; the period is filled automatically', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','confirmed'); SELECT 1/(SELECT (during IS NOT NULL)::int FROM adventure_signups LIMIT 1)$q$);
SELECT pg_temp.fails('level snapshot cannot be edited', $q$UPDATE adventure_signups SET level_at_signup=3$q$, 'snapshot');
SELECT pg_temp.ok('a supplied level_at_signup is ignored (real level 4 is stored)', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state,level_at_signup) VALUES ('d0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000004','22222222-2222-2222-2222-222222222222','queued',1); SELECT 1/(SELECT (level_at_signup = 4)::int FROM adventure_signups WHERE character_id='c0000000-0000-0000-0000-000000000004')$q$);
SELECT pg_temp.fails('spread: queued level-4 cannot take a seat next to level 1', $q$UPDATE adventure_signups SET state='confirmed' WHERE character_id='c0000000-0000-0000-0000-000000000004'$q$, 'level spread');
SELECT pg_temp.fails('overlap: same character on an overlapping adventure', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000002','c0000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','confirmed')$q$, 'exclusion constraint');
SELECT pg_temp.fails('ownership: bob signs up with the character of alice', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000003','22222222-2222-2222-2222-222222222222','queued')$q$, 'does not belong');
SELECT pg_temp.fails('multi-character: second character of alice without a grant', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000003','11111111-1111-1111-1111-111111111111','queued')$q$, 'at most 1');
INSERT INTO adventure_multi_char_grants VALUES ('d0000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111',2,'33333333-3333-3333-3333-333333333333','test');
SELECT pg_temp.ok('multi-character: allowed with a grant of 2', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000003','11111111-1111-1111-1111-111111111111','confirmed')$q$);
SELECT pg_temp.fails('seats: both seats taken, carol is refused', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000005','44444444-4444-4444-4444-444444444444','confirmed')$q$, 'no free seats');
SELECT pg_temp.ok('seats: carol can queue', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000005','44444444-4444-4444-4444-444444444444','queued')$q$);
SELECT pg_temp.fails('parameters: seats cannot drop below occupied seats', $q$UPDATE adventures SET seats=1 WHERE title='adv1'$q$, 'seats cannot be lower');
SELECT pg_temp.fails('parameters: level range cannot exclude existing signups', $q$UPDATE adventures SET min_level=2 WHERE title='adv1'$q$, 'new level range');
SELECT pg_temp.fails('signup state: confirmed -> queued is illegal', $q$UPDATE adventure_signups SET state='queued' WHERE character_id='c0000000-0000-0000-0000-000000000001'$q$, 'illegal signup transition');
SELECT pg_temp.ok('level snapshots can be refreshed by the controlled function', $q$UPDATE characters SET level=2 WHERE name='a1'; SELECT 1/(SELECT (recompute_signup_levels('d0000000-0000-0000-0000-000000000001') = 1)::int)$q$);
SELECT pg_temp.ok('adventure reschedule moves the signup periods', $q$UPDATE adventures SET starts_at='2026-12-02 18:00+00' WHERE title='adv1'; SELECT 1/(SELECT (count(*) = 4)::int FROM adventure_signups WHERE lower(during)='2026-12-02 18:00+00')$q$);
SELECT pg_temp.ok('signup state: confirmed -> withdrawn', $q$UPDATE adventure_signups SET state='withdrawn' WHERE character_id='c0000000-0000-0000-0000-000000000003'$q$);
SELECT pg_temp.fails('signup state: withdrawn -> confirmed is illegal', $q$UPDATE adventure_signups SET state='confirmed' WHERE character_id='c0000000-0000-0000-0000-000000000003'$q$, 'illegal signup transition');
SELECT pg_temp.ok('signup events are written automatically', $q$SELECT 1/(SELECT (count(*) >= 5)::int FROM signup_events)$q$);
UPDATE adventures SET status='registration_closed' WHERE title='adv2';
SELECT pg_temp.fails('signup: no new signups when registration is closed', $q$INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('d0000000-0000-0000-0000-000000000002','c0000000-0000-0000-0000-000000000005','44444444-4444-4444-4444-444444444444','queued')$q$, 'not open for registration');

-- ================= 3. transfer
SELECT pg_temp.fails('transfer is blocked while a signup is open', $q$UPDATE characters SET owner_user_id='22222222-2222-2222-2222-222222222222' WHERE name='a1'$q$, 'open signups');
UPDATE adventure_signups SET state='withdrawn' WHERE character_id='c0000000-0000-0000-0000-000000000001';
SELECT pg_temp.ok('transfer a1 to bob (bob has one free slot)', $q$SET LOCAL app.user_id='33333333-3333-3333-3333-333333333333'; UPDATE characters SET owner_user_id='22222222-2222-2222-2222-222222222222' WHERE name='a1'$q$);
SELECT pg_temp.ok('transfer is recorded and the former owner is kept', $q$SELECT 1/(SELECT count(*)::int FROM character_transfers t JOIN characters c ON c.id=t.character_id WHERE c.name='a1' AND c.former_owner_user_id='11111111-1111-1111-1111-111111111111')$q$);
SELECT pg_temp.fails('transfer to a player with full slots is blocked', $q$UPDATE characters SET owner_user_id='22222222-2222-2222-2222-222222222222' WHERE name='a3'$q$, 'slot limit');

-- ================= 4. Foundry-owned fields and inventory mirror
INSERT INTO foundry_worlds(id,name,relay_url,relay_client_id,key_secret_ref) VALUES ('f0000000-0000-0000-0000-000000000001','w','http://relay:3010','c','s');
INSERT INTO foundry_actors(world_id,actor_uuid,character_id) VALUES ('f0000000-0000-0000-0000-000000000001','Actor.abc','c0000000-0000-0000-0000-000000000004');
SELECT pg_temp.fails('Foundry-owned: the app cannot write xp', $q$UPDATE characters SET xp=999 WHERE name='b1'$q$, 'belong to the Foundry actor');
SELECT pg_temp.ok('app-owned: level_up_pending is writable', $q$UPDATE characters SET level_up_pending=true WHERE name='b1'$q$);
SELECT pg_temp.ok('the sync path can write xp', $q$SET LOCAL app.writer='foundry_sync'; UPDATE characters SET xp=999 WHERE name='b1'$q$);
SELECT pg_temp.fails('inventory mirror: the app cannot add items to a linked actor', $q$INSERT INTO inventory_items(character_id,name) VALUES ('c0000000-0000-0000-0000-000000000004','Sword')$q$, 'Foundry actor');
SELECT pg_temp.ok('inventory mirror: the sync path can', $q$SET LOCAL app.writer='foundry_sync'; INSERT INTO inventory_items(character_id,name) VALUES ('c0000000-0000-0000-0000-000000000004','Sword')$q$);
SELECT pg_temp.ok('inventory: a character without an actor is editable', $q$INSERT INTO inventory_items(character_id,name) VALUES ('c0000000-0000-0000-0000-000000000005','Rope')$q$);
INSERT INTO items_catalog(id,source,name) VALUES ('1c000000-0000-0000-0000-000000000001','custom','Potion');
INSERT INTO inventory_events(character_id,catalog_id,delta_qty,kind,operation_id) VALUES ('c0000000-0000-0000-0000-000000000004','1c000000-0000-0000-0000-000000000001',1,'grant','0b000000-0000-0000-0000-000000000001');
SELECT pg_temp.fails('inventory events: the same operation is not recorded twice', $q$INSERT INTO inventory_events(character_id,catalog_id,delta_qty,kind,operation_id) VALUES ('c0000000-0000-0000-0000-000000000004','1c000000-0000-0000-0000-000000000001',1,'grant','0b000000-0000-0000-0000-000000000001')$q$, 'duplicate key');

-- ================= 5. ledger
INSERT INTO currencies(id,key,name) VALUES ('cc000000-0000-0000-0000-000000000001','gold','Gold');
INSERT INTO accounts(id,owner_type,owner_id,currency_id) VALUES
 ('ac000000-0000-0000-0000-000000000001','system','00000000-0000-0000-0000-000000000000','cc000000-0000-0000-0000-000000000001'),
 ('ac000000-0000-0000-0000-000000000002','character','c0000000-0000-0000-0000-000000000004','cc000000-0000-0000-0000-000000000001');
INSERT INTO ledger_transactions(id,idempotency_key,kind,source) VALUES ('e0000000-0000-0000-0000-000000000001','k1','mint','app');
INSERT INTO ledger_entries(transaction_id,account_id,delta) VALUES ('e0000000-0000-0000-0000-000000000001','ac000000-0000-0000-0000-000000000002',1000);
SELECT pg_temp.fails('ledger: an unbalanced transaction cannot be posted', $q$UPDATE ledger_transactions SET state='posted' WHERE id='e0000000-0000-0000-0000-000000000001'$q$, 'not balanced');
INSERT INTO ledger_entries(transaction_id,account_id,delta) VALUES ('e0000000-0000-0000-0000-000000000001','ac000000-0000-0000-0000-000000000001',-1000);
SELECT pg_temp.ok('ledger: a balanced transaction is posted', $q$UPDATE ledger_transactions SET state='posted' WHERE id='e0000000-0000-0000-0000-000000000001'$q$);
SELECT pg_temp.ok('ledger: the balance follows the posting', $q$SELECT 1/(SELECT (balance = 1000)::int FROM accounts WHERE id='ac000000-0000-0000-0000-000000000002')$q$);
SELECT pg_temp.fails('ledger: no entries can be added to a posted transaction', $q$INSERT INTO ledger_entries(transaction_id,account_id,delta) VALUES ('e0000000-0000-0000-0000-000000000001','ac000000-0000-0000-0000-000000000002',10)$q$, 'not pending');
SELECT pg_temp.fails('ledger: other columns of a transaction cannot change', $q$UPDATE ledger_transactions SET reason='x' WHERE id='e0000000-0000-0000-0000-000000000001'$q$, 'only the state');
SELECT pg_temp.fails('ledger: duplicate idempotency key', $q$INSERT INTO ledger_transactions(idempotency_key,kind,source) VALUES ('k1','mint','app')$q$, 'duplicate key');
SELECT pg_temp.fails('ledger: the balance cannot be edited directly', $q$UPDATE accounts SET balance=100000$q$, 'only by posting');
SELECT pg_temp.fails('ledger: a new account must start at zero', $q$INSERT INTO accounts(owner_type,owner_id,currency_id,balance) VALUES ('character','c0000000-0000-0000-0000-000000000005','cc000000-0000-0000-0000-000000000001',50)$q$, 'zero balance');
SELECT pg_temp.fails('ledger: entries cannot be deleted', $q$DELETE FROM ledger_entries$q$, 'append-only');
INSERT INTO ledger_transactions(id,idempotency_key,kind,source) VALUES ('e0000000-0000-0000-0000-000000000002','k2','adjustment','app');
INSERT INTO ledger_entries(transaction_id,account_id,delta) VALUES ('e0000000-0000-0000-0000-000000000002','ac000000-0000-0000-0000-000000000002',-5000),('e0000000-0000-0000-0000-000000000002','ac000000-0000-0000-0000-000000000001',5000);
SELECT pg_temp.fails('ledger: a negative balance is refused', $q$UPDATE ledger_transactions SET state='posted' WHERE id='e0000000-0000-0000-0000-000000000002'$q$, 'negative balance');
INSERT INTO ledger_transactions(id,idempotency_key,kind,source,reversal_of) VALUES ('e0000000-0000-0000-0000-000000000003','k3','reversal','app','e0000000-0000-0000-0000-000000000001');
INSERT INTO ledger_entries(transaction_id,account_id,delta) VALUES ('e0000000-0000-0000-0000-000000000003','ac000000-0000-0000-0000-000000000002',-400),('e0000000-0000-0000-0000-000000000003','ac000000-0000-0000-0000-000000000001',400);
SELECT pg_temp.fails('ledger: a reversal must negate the original exactly', $q$UPDATE ledger_transactions SET state='posted' WHERE id='e0000000-0000-0000-0000-000000000003'$q$, 'exactly negate');
SELECT pg_temp.ok('ledger: reconciliation finds no mismatch', $q$SELECT 1/(1 - (SELECT count(*)::int FROM verify_account_balances()))$q$);
SELECT pg_temp.ok('accounts: get_or_create_account is idempotent', $q$SELECT 1/(SELECT (get_or_create_account('character','c0000000-0000-0000-0000-000000000003','cc000000-0000-0000-0000-000000000001') = get_or_create_account('character','c0000000-0000-0000-0000-000000000003','cc000000-0000-0000-0000-000000000001'))::int)$q$);

-- ================= 6. integration jobs
INSERT INTO integration_jobs(id,target,kind,payload) VALUES ('10b00000-0000-0000-0000-000000000001','foundry','apply_xp','{}');
INSERT INTO integration_job_attempts(id,job_id) VALUES ('a7700000-0000-0000-0000-000000000001','10b00000-0000-0000-0000-000000000001');
SELECT pg_temp.ok('attempt: filled in while running', $q$UPDATE integration_job_attempts SET http_status=200, finished_at=now()$q$);
SELECT pg_temp.fails('attempt: frozen after it finished', $q$UPDATE integration_job_attempts SET http_status=500$q$, 'immutable');
SELECT pg_temp.fails('job: pending -> succeeded is illegal', $q$UPDATE integration_jobs SET state='succeeded'$q$, 'illegal job transition');

-- ================= 7. maps, weather, locations
INSERT INTO map_regions(id,map_id,name) VALUES ('ee000000-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','reg1');
SELECT pg_temp.ok('hex region on the same map', $q$UPDATE hexes SET region_id='ee000000-0000-0000-0000-000000000001' WHERE map_id='aaaaaaaa-0000-0000-0000-000000000001' AND q=0$q$);
SELECT pg_temp.fails('hex region from another map', $q$UPDATE hexes SET region_id='ee000000-0000-0000-0000-000000000001' WHERE map_id='aaaaaaaa-0000-0000-0000-000000000002' AND q=0$q$, 'foreign key');
SELECT pg_temp.fails('adventure region without a map', $q$UPDATE adventures SET region_id='ee000000-0000-0000-0000-000000000001' WHERE title='adv2'$q$, 'check constraint');
SELECT pg_temp.fails('adventure region from another map', $q$UPDATE adventures SET map_id='aaaaaaaa-0000-0000-0000-000000000002', region_id='ee000000-0000-0000-0000-000000000001' WHERE title='adv2'$q$, 'foreign key');
SELECT pg_temp.fails('adventure target hex needs both coordinates', $q$UPDATE adventures SET map_id='aaaaaaaa-0000-0000-0000-000000000001', target_q=1 WHERE title='adv2'$q$, 'check constraint');
SELECT pg_temp.fails('shop location must be complete', $q$INSERT INTO shops(name,map_id) VALUES ('s','aaaaaaaa-0000-0000-0000-000000000001')$q$, 'check constraint');
SELECT pg_temp.fails('enterprise location must be complete', $q$INSERT INTO enterprises(name,map_id,q) VALUES ('e','aaaaaaaa-0000-0000-0000-000000000001',1)$q$, 'check constraint');
INSERT INTO climates(id,key,name) VALUES ('c1100000-0000-0000-0000-000000000001','k','k');
INSERT INTO weather_regions(id,map_id,name,climate_id) VALUES ('77700000-0000-0000-0000-000000000001','aaaaaaaa-0000-0000-0000-000000000001','r','c1100000-0000-0000-0000-000000000001');
INSERT INTO weather_types(id,key,name) VALUES ('77710000-0000-0000-0000-000000000001','clear','clear');
SELECT pg_temp.ok('weather: a forecast', $q$INSERT INTO weather_records(map_id,weather_region_id,game_day,weather_type_id,state,source) VALUES ('aaaaaaaa-0000-0000-0000-000000000001','77700000-0000-0000-0000-000000000001',10,'77710000-0000-0000-0000-000000000001','forecast','generated')$q$);
SELECT pg_temp.ok('weather: the actual record for the same day', $q$INSERT INTO weather_records(map_id,weather_region_id,game_day,weather_type_id,state,source) VALUES ('aaaaaaaa-0000-0000-0000-000000000001','77700000-0000-0000-0000-000000000001',10,'77710000-0000-0000-0000-000000000001','actual','manual')$q$);
SELECT pg_temp.fails('weather: a second forecast for the same day', $q$INSERT INTO weather_records(map_id,weather_region_id,game_day,weather_type_id,state,source) VALUES ('aaaaaaaa-0000-0000-0000-000000000001','77700000-0000-0000-0000-000000000001',10,'77710000-0000-0000-0000-000000000001','forecast','generated')$q$, 'duplicate key');
SELECT pg_temp.fails('weather: record with a region of another map', $q$INSERT INTO weather_records(map_id,weather_region_id,game_day,weather_type_id,state,source) VALUES ('aaaaaaaa-0000-0000-0000-000000000002','77700000-0000-0000-0000-000000000001',11,'77710000-0000-0000-0000-000000000001','forecast','generated')$q$, 'foreign key');

-- ================= 8. expeditions
INSERT INTO expeditions(id,name,map_id,origin_q,origin_r,current_q,current_r,leader_character_id) VALUES
 ('ec000000-0000-0000-0000-000000000001','e1','aaaaaaaa-0000-0000-0000-000000000001',0,0,0,0,'c0000000-0000-0000-0000-000000000005'),
 ('ec000000-0000-0000-0000-000000000002','e2','aaaaaaaa-0000-0000-0000-000000000001',0,0,0,0,NULL);
SELECT pg_temp.ok('route: waypoints on the map', $q$INSERT INTO expedition_routes(expedition_id,version,waypoints,planned_hexes) VALUES ('ec000000-0000-0000-0000-000000000001',1,'[[0,0],[2,0]]','[[0,0],[1,0],[2,0]]')$q$);
SELECT pg_temp.fails('route: a hex that is not on the map', $q$INSERT INTO expedition_routes(expedition_id,version,waypoints,is_current) VALUES ('ec000000-0000-0000-0000-000000000001',2,'[[0,0],[9,9]]',false)$q$, 'not on map');
SELECT pg_temp.fails('route: planned hexes that are not neighbours', $q$INSERT INTO expedition_routes(expedition_id,version,waypoints,planned_hexes,is_current) VALUES ('ec000000-0000-0000-0000-000000000001',3,'[[0,0]]','[[0,0],[2,0]]',false)$q$, 'not neighbours');
INSERT INTO expedition_legs(id,expedition_id,day_no,map_id,from_q,from_r,to_q,to_r,path) VALUES ('1e600000-0000-0000-0000-000000000001','ec000000-0000-0000-0000-000000000001',1,'aaaaaaaa-0000-0000-0000-000000000001',0,0,1,0,'[[0,0],[1,0]]');
SELECT pg_temp.fails('leg: the path must end at the destination', $q$INSERT INTO expedition_legs(expedition_id,day_no,map_id,from_q,from_r,to_q,to_r,path) VALUES ('ec000000-0000-0000-0000-000000000001',2,'aaaaaaaa-0000-0000-0000-000000000001',1,0,3,0,'[[1,0],[2,0]]')$q$, 'end at the destination');
SELECT pg_temp.fails('leg: a map different from the expedition map', $q$INSERT INTO expedition_legs(expedition_id,day_no,map_id,from_q,from_r,to_q,to_r) VALUES ('ec000000-0000-0000-0000-000000000001',3,'aaaaaaaa-0000-0000-0000-000000000002',0,0,1,0)$q$, 'foreign key');
INSERT INTO expedition_supply_items(id,expedition_id,kind,name,quantity) VALUES ('5b000000-0000-0000-0000-000000000001','ec000000-0000-0000-0000-000000000001','food','rations',10),('5b000000-0000-0000-0000-000000000002','ec000000-0000-0000-0000-000000000002','food','rations',5);
SELECT pg_temp.ok('supplies: consumption lowers the stock', $q$INSERT INTO expedition_supply_ledger(expedition_id,supply_item_id,leg_id,delta,reason) VALUES ('ec000000-0000-0000-0000-000000000001','5b000000-0000-0000-0000-000000000001','1e600000-0000-0000-0000-000000000001',-4,'consumed'); SELECT 1/(SELECT (quantity = 6)::int FROM expedition_supply_items WHERE id='5b000000-0000-0000-0000-000000000001')$q$);
SELECT pg_temp.fails('supplies: the stock cannot go below zero', $q$INSERT INTO expedition_supply_ledger(expedition_id,supply_item_id,delta,reason) VALUES ('ec000000-0000-0000-0000-000000000001','5b000000-0000-0000-0000-000000000001',-20,'consumed')$q$, 'check constraint');
SELECT pg_temp.fails('supplies: a leg of another expedition', $q$INSERT INTO expedition_supply_ledger(expedition_id,supply_item_id,leg_id,delta,reason) VALUES ('ec000000-0000-0000-0000-000000000002','5b000000-0000-0000-0000-000000000002','1e600000-0000-0000-0000-000000000001',-1,'consumed')$q$, 'foreign key');
SELECT pg_temp.fails('supplies: an item of another expedition', $q$INSERT INTO expedition_supply_ledger(expedition_id,supply_item_id,delta,reason) VALUES ('ec000000-0000-0000-0000-000000000001','5b000000-0000-0000-0000-000000000002',-1,'consumed')$q$, 'foreign key');
SELECT pg_temp.fails('events: a leg of another expedition', $q$INSERT INTO expedition_events(expedition_id,leg_id,kind,title) VALUES ('ec000000-0000-0000-0000-000000000002','1e600000-0000-0000-0000-000000000001','k','t')$q$, 'foreign key');
SELECT pg_temp.fails('visited hexes: first leg of another expedition', $q$INSERT INTO expedition_visited_hexes(expedition_id,map_id,q,r,first_leg_id) VALUES ('ec000000-0000-0000-0000-000000000002','aaaaaaaa-0000-0000-0000-000000000001',1,0,'1e600000-0000-0000-0000-000000000001')$q$, 'foreign key');
SELECT pg_temp.fails('expedition cannot start without members', $q$UPDATE expeditions SET status='active' WHERE name='e1'$q$, 'at least one member');
SELECT pg_temp.ok('members join an expedition', $q$INSERT INTO expedition_members(expedition_id,character_id) VALUES ('ec000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000004')$q$);
SELECT pg_temp.fails('a character cannot be in two active expeditions', $q$INSERT INTO expedition_members(expedition_id,character_id) VALUES ('ec000000-0000-0000-0000-000000000002','c0000000-0000-0000-0000-000000000004')$q$, 'duplicate key');
SELECT pg_temp.fails('the leader must be a member', $q$UPDATE expeditions SET status='active' WHERE name='e1'$q$, 'leader must be a member');
UPDATE expeditions SET leader_character_id='c0000000-0000-0000-0000-000000000004' WHERE name='e1';
SELECT pg_temp.ok('expedition starts', $q$UPDATE expeditions SET status='active' WHERE name='e1'$q$);
SELECT pg_temp.fails('expedition: planning -> returned is illegal', $q$UPDATE expeditions SET status='returned' WHERE name='e2'$q$, 'illegal expedition transition');
SELECT pg_temp.ok('finishing an expedition frees its members', $q$UPDATE expeditions SET status='returned' WHERE name='e1'; SELECT 1/(1 - (SELECT count(*)::int FROM expedition_members WHERE left_at IS NOT NULL AND left_at IS NULL)) ; SELECT 1/(SELECT (count(*) = 0)::int FROM expedition_members WHERE expedition_id='ec000000-0000-0000-0000-000000000001' AND left_at IS NULL)$q$);
SELECT pg_temp.fails('nobody can join a finished expedition', $q$INSERT INTO expedition_members(expedition_id,character_id) VALUES ('ec000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000005')$q$, 'cannot join');

-- ================= 9. calendar
INSERT INTO calendars(id,name,is_active) VALUES ('ca000000-0000-0000-0000-000000000001','Cal',true);
INSERT INTO calendar_versions(id,calendar_id,version,effective_from_day,anchor_year) VALUES ('cb000000-0000-0000-0000-000000000001','ca000000-0000-0000-0000-000000000001',1,0,4700);
INSERT INTO calendar_months(version_id,ord,name,days) VALUES ('cb000000-0000-0000-0000-000000000001',1,'Abadius',31),('cb000000-0000-0000-0000-000000000001',2,'Calistril',28);
SELECT pg_temp.ok('calendar is editable before the game clock starts', $q$UPDATE calendar_months SET days=30 WHERE ord=1$q$);
SELECT pg_temp.fails('the clock cannot start on a calendar without weekdays', $q$INSERT INTO game_clock(calendar_id,current_day) VALUES ('ca000000-0000-0000-0000-000000000001',40)$q$, 'weekdays must be numbered');
INSERT INTO calendar_weekdays(version_id,ord,name) VALUES ('cb000000-0000-0000-0000-000000000001',1,'Mon'),('cb000000-0000-0000-0000-000000000001',2,'Tue');
INSERT INTO game_clock(calendar_id,current_day) VALUES ('ca000000-0000-0000-0000-000000000001',40);
SELECT pg_temp.fails('sealed: a month cannot change', $q$UPDATE calendar_months SET days=29 WHERE ord=1$q$, 'sealed');
SELECT pg_temp.fails('sealed: a month cannot be added', $q$INSERT INTO calendar_months(version_id,ord,name,days) VALUES ('cb000000-0000-0000-0000-000000000001',3,'X',30)$q$, 'sealed');
SELECT pg_temp.fails('sealed: a weekday cannot be removed', $q$DELETE FROM calendar_weekdays WHERE ord=2$q$, 'sealed');
SELECT pg_temp.fails('sealed: the version cannot be deleted', $q$DELETE FROM calendar_versions$q$, 'sealed');
SELECT pg_temp.fails('a new version cannot start in the past', $q$INSERT INTO calendar_versions(calendar_id,version,effective_from_day,anchor_year) VALUES ('ca000000-0000-0000-0000-000000000001',2,30,4700)$q$, 'after the current game day');
INSERT INTO calendar_versions(id,calendar_id,version,effective_from_day,anchor_year,anchor_month_ord,anchor_day) VALUES ('cb000000-0000-0000-0000-000000000002','ca000000-0000-0000-0000-000000000001',2,100,4700,2,10);
INSERT INTO calendar_months(version_id,ord,name,days) VALUES ('cb000000-0000-0000-0000-000000000002',1,'Abadius',35),('cb000000-0000-0000-0000-000000000002',2,'Calistril',20);
INSERT INTO calendar_weekdays(version_id,ord,name) VALUES ('cb000000-0000-0000-0000-000000000002',1,'Mon');
SELECT pg_temp.fails('bypass: a month cannot be moved out of a sealed version', $q$UPDATE calendar_months SET version_id='cb000000-0000-0000-0000-000000000002', ord=3 WHERE version_id='cb000000-0000-0000-0000-000000000001' AND ord=2$q$, 'sealed');
SELECT pg_temp.fails('bypass: a month cannot be moved into a sealed version', $q$UPDATE calendar_months SET version_id='cb000000-0000-0000-0000-000000000001', ord=9 WHERE version_id='cb000000-0000-0000-0000-000000000002' AND ord=2$q$, 'sealed');
SELECT pg_temp.fails('an unsealed version cannot start before the previous one', $q$UPDATE calendar_versions SET effective_from_day=-50 WHERE id='cb000000-0000-0000-0000-000000000002'$q$, 'same order');
SELECT pg_temp.fails('an unsealed version cannot start on or before the current day', $q$UPDATE calendar_versions SET effective_from_day=30 WHERE id='cb000000-0000-0000-0000-000000000002'$q$, 'current game day');
SELECT pg_temp.ok('an unsealed version can still be moved later', $q$UPDATE calendar_versions SET effective_from_day=120 WHERE id='cb000000-0000-0000-0000-000000000002'$q$);
INSERT INTO calendars(id,name) VALUES ('ca000000-0000-0000-0000-000000000002','Other');
SELECT pg_temp.fails('a version cannot move to another calendar', $q$UPDATE calendar_versions SET calendar_id='ca000000-0000-0000-0000-000000000002' WHERE id='cb000000-0000-0000-0000-000000000002'$q$, 'another calendar');
INSERT INTO calendar_versions(id,calendar_id,version,effective_from_day,anchor_year,anchor_day) VALUES ('cb000000-0000-0000-0000-000000000003','ca000000-0000-0000-0000-000000000001',3,200,4700,99);
INSERT INTO calendar_months(version_id,ord,name,days) VALUES ('cb000000-0000-0000-0000-000000000003',1,'M',30);
INSERT INTO calendar_weekdays(version_id,ord,name) VALUES ('cb000000-0000-0000-0000-000000000003',1,'Mon');
SELECT pg_temp.fails('the clock cannot reach a version with an impossible anchor date', $q$UPDATE game_clock SET current_day=250$q$, 'anchor date does not exist');
SELECT pg_temp.ok('the clock reaching a valid version seals it', $q$UPDATE game_clock SET current_day=130; SELECT 1/(SELECT (sealed_at IS NOT NULL)::int FROM calendar_versions WHERE version=2)$q$);
INSERT INTO calendar_events(calendar_id,title,start_day) VALUES ('ca000000-0000-0000-0000-000000000001','Fair',150);

-- ================= 10. factions
INSERT INTO factions(id,name) VALUES ('fa000000-0000-0000-0000-000000000001','Guild'),('fa000000-0000-0000-0000-000000000002','Crown'),('fa000000-0000-0000-0000-000000000003','Order');
INSERT INTO faction_reputation_log(id,character_id,faction_id,delta) VALUES
 ('10600000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000004','fa000000-0000-0000-0000-000000000001',10),
 ('10600000-0000-0000-0000-000000000002','c0000000-0000-0000-0000-000000000004','fa000000-0000-0000-0000-000000000001',-3);
SELECT pg_temp.ok('reputation: the value equals the sum of the log', $q$SELECT 1/(SELECT (value = 7)::int FROM faction_reputation WHERE character_id='c0000000-0000-0000-0000-000000000004')$q$);
SELECT pg_temp.fails('reputation: the value cannot be edited directly', $q$UPDATE faction_reputation SET value=999$q$, 'only through');
SELECT pg_temp.fails('reputation: a reversal with another amount', $q$INSERT INTO faction_reputation_log(character_id,faction_id,delta,reversal_of) VALUES ('c0000000-0000-0000-0000-000000000004','fa000000-0000-0000-0000-000000000001',5,'10600000-0000-0000-0000-000000000002')$q$, 'must mirror');
SELECT pg_temp.fails('reputation: a reversal for another character', $q$INSERT INTO faction_reputation_log(character_id,faction_id,delta,reversal_of) VALUES ('c0000000-0000-0000-0000-000000000005','fa000000-0000-0000-0000-000000000001',3,'10600000-0000-0000-0000-000000000002')$q$, 'must mirror');
SELECT pg_temp.ok('reputation: an exact reversal', $q$INSERT INTO faction_reputation_log(id,character_id,faction_id,delta,reversal_of) VALUES ('10600000-0000-0000-0000-000000000003','c0000000-0000-0000-0000-000000000004','fa000000-0000-0000-0000-000000000001',3,'10600000-0000-0000-0000-000000000002'); SELECT 1/(SELECT (value = 10)::int FROM faction_reputation WHERE character_id='c0000000-0000-0000-0000-000000000004')$q$);
SELECT pg_temp.fails('reputation: a reversal cannot be reversed', $q$INSERT INTO faction_reputation_log(character_id,faction_id,delta,reversal_of) VALUES ('c0000000-0000-0000-0000-000000000004','fa000000-0000-0000-0000-000000000001',-3,'10600000-0000-0000-0000-000000000003')$q$, 'cannot itself be reversed');
INSERT INTO faction_reputation_tiers(faction_id,ord,name,min_value) VALUES (NULL,1,'Stranger',0),(NULL,2,'Friend',20);
SELECT pg_temp.fails('tiers: thresholds must grow with the tier number', $q$INSERT INTO faction_reputation_tiers(faction_id,ord,name,min_value) VALUES (NULL,3,'Bad',10)$q$, 'thresholds');
SELECT pg_temp.fails('tiers: two tiers with the same threshold', $q$INSERT INTO faction_reputation_tiers(faction_id,ord,name,min_value) VALUES (NULL,4,'Dup',20)$q$, 'thresholds');
SELECT pg_temp.ok('relation: created', $q$INSERT INTO faction_relations(id,faction_a,faction_b) VALUES ('2e000000-0000-0000-0000-000000000001','fa000000-0000-0000-0000-000000000001','fa000000-0000-0000-0000-000000000002')$q$);
SELECT pg_temp.ok('relation: history is written on change', $q$UPDATE faction_relations SET standing=-50, kind='rival'; SELECT 1/(SELECT (count(*) = 2)::int FROM faction_relation_history)$q$);
SELECT pg_temp.fails('relation: a vassal needs a senior side', $q$UPDATE faction_relations SET kind='vassal'$q$, 'check constraint');
SELECT pg_temp.fails('relation: the senior side must be one of the pair', $q$UPDATE faction_relations SET kind='vassal', senior_faction_id='fa000000-0000-0000-0000-000000000003'$q$, 'check constraint');
SELECT pg_temp.ok('relation: a vassal with a valid senior side', $q$UPDATE faction_relations SET kind='vassal', senior_faction_id='fa000000-0000-0000-0000-000000000001'$q$);
SELECT pg_temp.ok('territory: control, contest and release leave a history', $q$INSERT INTO territory_control(map_id,q,r,faction_id,control_type) VALUES ('aaaaaaaa-0000-0000-0000-000000000001',1,0,'fa000000-0000-0000-0000-000000000001','controlled'); UPDATE territory_control SET strength=80; DELETE FROM territory_control; SELECT 1/(SELECT (count(*) = 3)::int FROM territory_history)$q$);
INSERT INTO territory_control(map_id,q,r,faction_id,control_type) VALUES ('aaaaaaaa-0000-0000-0000-000000000001',2,0,'fa000000-0000-0000-0000-000000000001','controlled');
SELECT pg_temp.fails('territory: two controlling factions on one hex', $q$INSERT INTO territory_control(map_id,q,r,faction_id,control_type) VALUES ('aaaaaaaa-0000-0000-0000-000000000001',2,0,'fa000000-0000-0000-0000-000000000002','controlled')$q$, 'duplicate key');
SELECT pg_temp.ok('territory: a contesting faction is allowed', $q$INSERT INTO territory_control(map_id,q,r,faction_id,control_type) VALUES ('aaaaaaaa-0000-0000-0000-000000000001',2,0,'fa000000-0000-0000-0000-000000000002','contested')$q$);
SELECT pg_temp.ok('factions: a parent', $q$UPDATE factions SET parent_faction_id='fa000000-0000-0000-0000-000000000001' WHERE name='Crown'$q$);
SELECT pg_temp.fails('factions: a cycle in the hierarchy', $q$UPDATE factions SET parent_faction_id='fa000000-0000-0000-0000-000000000002' WHERE name='Guild'$q$, 'cycle');
SELECT pg_temp.fails('factions: a faction as its own parent', $q$UPDATE factions SET parent_faction_id=id WHERE name='Order'$q$, 'cycle');
INSERT INTO political_events(id,title,game_day) VALUES ('9e000000-0000-0000-0000-000000000001','War',130);
SELECT pg_temp.fails('political event: no effects, cannot be applied', $q$UPDATE political_events SET state='applied'$q$, 'every effect');
INSERT INTO political_event_effects(id,event_id,ord,effect) VALUES ('9e100000-0000-0000-0000-000000000001','9e000000-0000-0000-0000-000000000001',1,'{"op":"relation"}');
SELECT pg_temp.fails('political event: a pending effect blocks applying', $q$UPDATE political_events SET state='applied'$q$, 'every effect');
UPDATE political_event_effects SET state='applied', applied_at=now();
SELECT pg_temp.ok('political event: applied when every effect is applied', $q$UPDATE political_events SET state='applied'$q$);
SELECT pg_temp.fails('political event: applied -> draft is illegal', $q$UPDATE political_events SET state='draft'$q$, 'illegal political_event transition');
SELECT pg_temp.fails('political event: cannot be revoked while an effect is applied', $q$UPDATE political_events SET state='revoked'$q$, 'revert every');
UPDATE political_event_effects SET state='reverted', reverted_at=now();
SELECT pg_temp.ok('political event: revoked after the effects are reverted', $q$UPDATE political_events SET state='revoked'$q$);
SELECT pg_temp.fails('political event: a reverted effect cannot be applied again', $q$UPDATE political_event_effects SET state='applied'$q$, 'illegal effect transition');

-- ================= 11. shop purchase
INSERT INTO shops(id,name) VALUES ('5a000000-0000-0000-0000-000000000001','Smith');
INSERT INTO shop_stock(id,shop_id,catalog_id,price_amount,price_currency_id,stock_qty,per_character_limit) VALUES ('57000000-0000-0000-0000-000000000001','5a000000-0000-0000-0000-000000000001','1c000000-0000-0000-0000-000000000001',100,'cc000000-0000-0000-0000-000000000001',3,2);
SELECT pg_temp.ok('shop: a purchase takes the money, the stock and queues a job', $q$SELECT purchase_item('57000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000004',2,'buy-1',NULL); SELECT 1/(SELECT (stock_qty = 1)::int FROM shop_stock); SELECT 1/(SELECT (balance = 800)::int FROM accounts WHERE id='ac000000-0000-0000-0000-000000000002'); SELECT 1/(SELECT (count(*) = 1)::int FROM integration_jobs WHERE kind='grant_item')$q$);
SELECT pg_temp.ok('shop: a repeated request returns the same purchase', $q$SELECT 1/(SELECT (purchase_item('57000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000004',2,'buy-1',NULL) = (SELECT id FROM purchases LIMIT 1))::int); SELECT 1/(SELECT (count(*) = 1)::int FROM purchases)$q$);
SELECT pg_temp.fails('shop: the per-character limit', $q$SELECT purchase_item('57000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000004',1,'buy-2',NULL)$q$, 'limit per character');
INSERT INTO accounts(id,owner_type,owner_id,currency_id) VALUES ('ac000000-0000-0000-0000-000000000005','character','c0000000-0000-0000-0000-000000000005','cc000000-0000-0000-0000-000000000001');
INSERT INTO ledger_transactions(id,idempotency_key,kind,source) VALUES ('e0000000-0000-0000-0000-000000000005','k5','mint','app');
INSERT INTO ledger_entries(transaction_id,account_id,delta) VALUES ('e0000000-0000-0000-0000-000000000005','ac000000-0000-0000-0000-000000000005',50),('e0000000-0000-0000-0000-000000000005','ac000000-0000-0000-0000-000000000001',-50);
UPDATE ledger_transactions SET state='posted' WHERE id='e0000000-0000-0000-0000-000000000005';
SELECT pg_temp.fails('shop: not enough stock', $q$SELECT purchase_item('57000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000005',2,'buy-3',NULL)$q$, 'not enough stock');
SELECT pg_temp.fails('shop: not enough money (nothing is left behind)', $q$SELECT purchase_item('57000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000005',1,'buy-4',NULL)$q$, 'negative balance');
SELECT pg_temp.ok('shop: a failed attempt left no purchase, no stock change', $q$SELECT 1/(SELECT (count(*) = 1)::int FROM purchases); SELECT 1/(SELECT (stock_qty = 1)::int FROM shop_stock)$q$);
UPDATE shop_stock SET price_amount=50 WHERE id='57000000-0000-0000-0000-000000000001';
INSERT INTO shop_requirements(stock_id,kind,faction_id,min_value) VALUES ('57000000-0000-0000-0000-000000000001','faction_reputation','fa000000-0000-0000-0000-000000000003',10);
SELECT pg_temp.fails('shop: reputation requirement', $q$SELECT purchase_item('57000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000005',1,'buy-5',NULL)$q$, 'reputation');
INSERT INTO faction_reputation_log(character_id,faction_id,delta) VALUES ('c0000000-0000-0000-0000-000000000005','fa000000-0000-0000-0000-000000000003',12);
SELECT pg_temp.ok('shop: the requirement is met', $q$SELECT purchase_item('57000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000005',1,'buy-6',NULL); SELECT 1/(SELECT (stock_qty = 0)::int FROM shop_stock)$q$);
SELECT pg_temp.ok('shop: a failed grant returns the money and the stock exactly', $q$SELECT fail_purchase((SELECT id FROM purchases WHERE character_id='c0000000-0000-0000-0000-000000000005'),'Foundry refused'); SELECT 1/(SELECT (stock_qty = 1)::int FROM shop_stock); SELECT 1/(SELECT (balance = 50)::int FROM accounts WHERE id='ac000000-0000-0000-0000-000000000005'); SELECT 1/(1 - (SELECT count(*)::int FROM verify_account_balances()))$q$);
SELECT pg_temp.fails('shop: a purchase cannot be failed twice', $q$SELECT fail_purchase((SELECT id FROM purchases WHERE character_id='c0000000-0000-0000-0000-000000000005'),'again')$q$, 'only a pending purchase');
SELECT pg_temp.fails('shop requirement: shop and item at once', $q$INSERT INTO shop_requirements(shop_id,stock_id,kind,faction_id,min_value) VALUES ('5a000000-0000-0000-0000-000000000001','57000000-0000-0000-0000-000000000001','faction_reputation','fa000000-0000-0000-0000-000000000003',1)$q$, 'check constraint');
SELECT pg_temp.fails('shop requirement: tier kind with a value', $q$INSERT INTO shop_requirements(shop_id,kind,faction_id,min_value) VALUES ('5a000000-0000-0000-0000-000000000001','faction_tier','fa000000-0000-0000-0000-000000000003',1)$q$, 'check constraint');

-- ================= 12. wiki
SELECT pg_temp.ok('wiki: a page with its first version in one transaction', $q$SET CONSTRAINTS ALL DEFERRED; INSERT INTO wiki_pages(id,type,title,slug,body_md,current_rev) VALUES ('3b000000-0000-0000-0000-000000000001','article','T','t','body',1); INSERT INTO wiki_versions(page_id,rev,title,body_md,source,content_hash) VALUES ('3b000000-0000-0000-0000-000000000001',1,'T','body','web','h1'); SET CONSTRAINTS ALL IMMEDIATE$q$);
SELECT pg_temp.fails('wiki: a page without a matching version', $q$SET CONSTRAINTS ALL DEFERRED; INSERT INTO wiki_pages(type,title,slug,current_rev) VALUES ('article','U','u',7); SET CONSTRAINTS ALL IMMEDIATE$q$, 'no matching version');
SELECT pg_temp.fails('wiki: an edit without a new revision', $q$UPDATE wiki_pages SET body_md='changed' WHERE slug='t'$q$, 'new revision');
SELECT pg_temp.fails('wiki: the revision cannot go back', $q$UPDATE wiki_pages SET current_rev=0 WHERE slug='t'$q$, 'cannot go back');
SELECT pg_temp.ok('wiki: an edit with a new revision and version', $q$SET CONSTRAINTS ALL DEFERRED; UPDATE wiki_pages SET body_md='v2', current_rev=2 WHERE slug='t'; INSERT INTO wiki_versions(page_id,rev,title,body_md,source,content_hash) VALUES ('3b000000-0000-0000-0000-000000000001',2,'T','v2','web','h2'); SET CONSTRAINTS ALL IMMEDIATE$q$);
SELECT pg_temp.fails('wiki: versions are immutable', $q$UPDATE wiki_versions SET body_md='x'$q$, 'append-only');
SELECT pg_temp.fails('wiki: restricted to roles but no roles', $q$SET CONSTRAINTS ALL DEFERRED; UPDATE wiki_pages SET visibility='roles' WHERE slug='t'; SET CONSTRAINTS ALL IMMEDIATE$q$, 'restricted to roles');

-- ================= 13. enterprise
INSERT INTO enterprises(id,name) VALUES ('e7e00000-0000-0000-0000-000000000001','Trade house');
SELECT pg_temp.ok('enterprise: a reputation change is logged', $q$UPDATE enterprises SET reputation=50; SELECT 1/(SELECT (count(*) = 1)::int FROM enterprise_log WHERE kind='reputation')$q$);
SELECT pg_temp.ok('enterprise: a stock change is logged', $q$INSERT INTO enterprise_stock(id,enterprise_id,name,quantity) VALUES ('e5000000-0000-0000-0000-000000000001','e7e00000-0000-0000-0000-000000000001','Wine',10); UPDATE enterprise_stock SET quantity=7; SELECT 1/(SELECT (count(*) = 2)::int FROM enterprise_log WHERE kind='stock')$q$);
SELECT pg_temp.fails('enterprise: automation cannot skip the GM', $q$INSERT INTO enterprise_event_rules(name,trigger_kind,requires_gm_approval) VALUES ('r','schedule',false)$q$, 'check constraint');

-- ================= 14. notifications, vaults, outbox, audit
INSERT INTO notifications(id,user_id,kind,dedupe_key) VALUES ('90000000-0000-0000-0000-000000000001','11111111-1111-1111-1111-111111111111','adventure.published','adv1:published');
SELECT pg_temp.fails('notifications: the same event twice for one user', $q$INSERT INTO notifications(user_id,kind,dedupe_key) VALUES ('11111111-1111-1111-1111-111111111111','adventure.published','adv1:published')$q$, 'duplicate key');
INSERT INTO notification_deliveries(notification_id,channel) VALUES ('90000000-0000-0000-0000-000000000001','discord_dm');
SELECT pg_temp.fails('notifications: the same delivery channel twice', $q$INSERT INTO notification_deliveries(notification_id,channel) VALUES ('90000000-0000-0000-0000-000000000001','discord_dm')$q$, 'duplicate key');
SELECT pg_temp.fails('vault: plain http is refused', $q$INSERT INTO vaults(kind,name,webdav_url,username,password_enc) VALUES ('gm','v1','http://dav.example/','u','\x00')$q$, 'check constraint');
SELECT pg_temp.fails('vault: a path that leaves the root', $q$INSERT INTO vaults(kind,name,webdav_url,remote_root,username,password_enc) VALUES ('gm','v2','https://dav.example/','/a/../b','u','\x00')$q$, 'check constraint');
SELECT pg_temp.ok('vault: https and a normal root', $q$INSERT INTO vaults(kind,name,webdav_url,remote_root,username,password_enc) VALUES ('gm','v3','https://dav.example/','/Vault','u','\x00')$q$);
SELECT pg_temp.ok('outbox: events get increasing ids', $q$INSERT INTO event_outbox(kind) VALUES ('a'),('b'); SELECT 1/(SELECT (max(id) > min(id))::int FROM event_outbox)$q$);
INSERT INTO audit_log(actor_kind,action,entity_type) VALUES ('system','t','t');
SELECT pg_temp.fails('audit log: cannot be changed', $q$UPDATE audit_log SET result='x'$q$, 'append-only');
SELECT pg_temp.fails('audit log: cannot be deleted', $q$DELETE FROM audit_log$q$, 'append-only');

-- ================= report
SELECT (CASE WHEN ok THEN 'PASS ' ELSE 'FAIL ' END) || name || CASE WHEN ok THEN '' ELSE '  [' || detail || ']' END FROM results ORDER BY ok, seq;
SELECT '---- passed ' || count(*) FILTER (WHERE ok) || ' / ' || count(*) FROM results;
DO $$
DECLARE failed text; total integer;
BEGIN
  SELECT string_agg(name, '; '), count(*) INTO failed, total FROM (SELECT name FROM results WHERE NOT ok) f;
  IF failed IS NOT NULL THEN RAISE EXCEPTION 'schema tests failed: %', failed; END IF;
  SELECT count(*) INTO total FROM results;
  IF total <> 166 THEN RAISE EXCEPTION 'unexpected number of checks: % (expected 166)', total; END IF;
END $$;
