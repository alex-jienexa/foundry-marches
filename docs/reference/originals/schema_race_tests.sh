#!/usr/bin/env bash
# Concurrency checks for schema.sql v0.3. Run on a database that has schema.sql applied
# (a fresh one, or one used by schema_tests.sql): PGDATABASE=wm_test ./schema_race_tests.sh
# Each scenario starts two real sessions; the first holds its transaction open for a few seconds.
# Exit code is non-zero if any invariant is violated.
set -u
P="psql -X -q -At -v ON_ERROR_STOP=0"
FAILS=0
check() { if [ "$2" = "$3" ]; then echo "PASS $1"; else echo "FAIL $1 (expected '$3', got '$2')"; FAILS=$((FAILS+1)); fi; }
uid() { $P -c "SELECT gen_random_uuid()"; }

# ---- 1. two simultaneous approvals for a player with one free slot
U=$(uid)
$P -c "INSERT INTO users(id,display_name) VALUES ('$U','race-approve')" >/dev/null
for n in 1 2 3; do $P -c "INSERT INTO characters(owner_user_id,name,status) VALUES ('$U','race-$n','draft')" >/dev/null; done
$P -c "UPDATE characters SET status='pending_approval' WHERE name='race-1' AND owner_user_id='$U'" >/dev/null
( $P -c "BEGIN; UPDATE characters SET status='pending_approval' WHERE name='race-2' AND owner_user_id='$U'; SELECT pg_sleep(3); COMMIT;" >/dev/null 2>&1 ) &
sleep 1
OUT=$($P -c "UPDATE characters SET status='pending_approval' WHERE name='race-3' AND owner_user_id='$U'" 2>&1)
wait
check "approvals: exactly two characters hold slots" "$($P -c "SELECT count(*) FROM characters WHERE owner_user_id='$U' AND status='pending_approval'")" "2"
check "approvals: the loser got the slot-limit error" "$(echo "$OUT" | grep -c 'slot limit')" "1"

# ---- 2. two simultaneous purchases of the last two items
CUR=$($P -c "INSERT INTO currencies(key,name) VALUES ('race-gold-'||substr(md5(random()::text),1,6),'g') RETURNING id")
SYS=$($P -c "SELECT get_or_create_account('system','00000000-0000-0000-0000-000000000000','$CUR')")
mkchar() { local u c; u=$(uid); c=$(uid); $P -c "INSERT INTO users(id,display_name) VALUES ('$u','buyer')" >/dev/null
  $P -c "INSERT INTO characters(id,owner_user_id,name,status) VALUES ('$c','$u','buyer','draft')" >/dev/null
  $P -c "UPDATE characters SET status='pending_approval' WHERE id='$c'" >/dev/null
  $P -c "UPDATE characters SET status='active' WHERE id='$c'" >/dev/null
  local a; a=$($P -c "SELECT get_or_create_account('character','$c','$CUR')")
  local t; t=$($P -c "INSERT INTO ledger_transactions(idempotency_key,kind,source) VALUES (gen_random_uuid()::text,'mint','app') RETURNING id")
  $P -c "INSERT INTO ledger_entries(transaction_id,account_id,delta) VALUES ('$t','$a',1000),('$t','$SYS',-1000)" >/dev/null
  $P -c "UPDATE ledger_transactions SET state='posted' WHERE id='$t'" >/dev/null; echo "$c"; }
CA=$(mkchar); CB=$(mkchar)
ITEM=$($P -c "INSERT INTO items_catalog(source,name) VALUES ('custom','race item') RETURNING id")
SHOP=$($P -c "INSERT INTO shops(name) VALUES ('race shop') RETURNING id")
STOCK=$($P -c "INSERT INTO shop_stock(shop_id,catalog_id,price_amount,price_currency_id,stock_qty) VALUES ('$SHOP','$ITEM',10,'$CUR',2) RETURNING id")
( $P -c "BEGIN; SELECT purchase_item('$STOCK','$CA',2,'race-a-'||gen_random_uuid(),NULL); SELECT pg_sleep(3); COMMIT;" >/dev/null 2>&1 ) &
sleep 1
OUT=$($P -c "SELECT purchase_item('$STOCK','$CB',2,'race-b-'||gen_random_uuid(),NULL)" 2>&1)
wait
check "purchases: stock is exactly zero, never negative" "$($P -c "SELECT stock_qty FROM shop_stock WHERE id='$STOCK'")" "0"
check "purchases: exactly one purchase exists" "$($P -c "SELECT count(*) FROM purchases WHERE stock_id='$STOCK'")" "1"
check "purchases: the loser got 'not enough stock'" "$(echo "$OUT" | grep -c 'not enough stock')" "1"
check "purchases: the loser's money is untouched" "$($P -c "SELECT balance FROM accounts WHERE owner_id='$CB' AND currency_id='$CUR'")" "1000"
check "purchases: balances match the ledger" "$($P -c "SELECT count(*) FROM verify_account_balances()")" "0"

# ---- 3. two simultaneous confirmations for the last seat
GM=$(uid); $P -c "INSERT INTO users(id,display_name) VALUES ('$GM','race-gm')" >/dev/null
ADV=$($P -c "INSERT INTO adventures(title,creator_user_id,min_level,max_level,seats,starts_at,duration_min) VALUES ('race adventure','$GM',1,5,1,now()+interval '30 days',120) RETURNING id")
$P -c "UPDATE adventures SET status='approved' WHERE id='$ADV'" >/dev/null
$P -c "UPDATE adventures SET status='published' WHERE id='$ADV'" >/dev/null
$P -c "UPDATE adventures SET status='registration_open' WHERE id='$ADV'" >/dev/null
for C in $CA $CB; do
  UID_=$($P -c "SELECT owner_user_id FROM characters WHERE id='$C'")
  $P -c "INSERT INTO adventure_signups(adventure_id,character_id,user_id,state) VALUES ('$ADV','$C','$UID_','queued')" >/dev/null
done
( $P -c "BEGIN; UPDATE adventure_signups SET state='confirmed' WHERE adventure_id='$ADV' AND character_id='$CA'; SELECT pg_sleep(3); COMMIT;" >/dev/null 2>&1 ) &
sleep 1
OUT=$($P -c "UPDATE adventure_signups SET state='confirmed' WHERE adventure_id='$ADV' AND character_id='$CB'" 2>&1)
wait
check "seats: exactly one confirmed signup" "$($P -c "SELECT count(*) FROM adventure_signups WHERE adventure_id='$ADV' AND state='confirmed'")" "1"
check "seats: the loser got 'no free seats'" "$(echo "$OUT" | grep -c 'no free seats')" "1"

echo "----"; [ "$FAILS" -eq 0 ] && echo "all race checks passed" || echo "$FAILS race check(s) FAILED"
exit $FAILS
