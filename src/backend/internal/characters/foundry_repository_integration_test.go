package characters

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQL integration tests are opt-in and require a dedicated database
// populated from docs/database/schema.sql. The suffix guard is intentional:
// these tests insert and delete fixture rows and must never target production.
func openFoundryIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("FOUNDRY_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("set FOUNDRY_TEST_DATABASE_URL to an isolated PostgreSQL *_test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		t.Fatal("FOUNDRY_TEST_DATABASE_URL must be a postgres:// URL")
	}
	databaseName := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasSuffix(databaseName, "_test") {
		t.Fatalf("refusing integration tests against database %q: database name must end in _test", databaseName)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	db.SetMaxOpenConns(4)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Fatalf("connect to PostgreSQL test database: %v", err)
	}
	var version int
	if err := db.QueryRowContext(ctx, "SELECT current_setting('server_version_num')::integer").Scan(&version); err != nil {
		db.Close()
		t.Fatalf("read PostgreSQL version: %v", err)
	}
	if version < 150000 {
		db.Close()
		t.Fatalf("PostgreSQL 15+ is required by schema v0.3; server_version_num=%d", version)
	}
	for _, table := range []string{"characters", "foundry_worlds", "foundry_actors", "inventory_items"} {
		var exists bool
		if err := db.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", "public."+table).Scan(&exists); err != nil {
			db.Close()
			t.Fatalf("check schema table %s: %v", table, err)
		}
		if !exists {
			db.Close()
			t.Fatalf("test database is missing public.%s; apply docs/database/schema.sql first", table)
		}
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func integrationUUID(t *testing.T) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatalf("generate fixture UUID: %v", err)
	}
	encoded := hex.EncodeToString(raw[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}

func integrationWorld(t *testing.T, db *sql.DB) string {
	t.Helper()
	worldID := integrationUUID(t)
	_, err := db.ExecContext(context.Background(), `
INSERT INTO foundry_worlds (id, name, relay_url, relay_client_id, key_secret_ref)
VALUES ($1, 'Foundry integration test', 'https://relay.invalid', 'integration-test', 'test-only')
`, worldID)
	if err != nil {
		t.Fatalf("insert test Foundry world: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM foundry_actors WHERE world_id = $1", worldID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM foundry_worlds WHERE id = $1", worldID)
	})
	return worldID
}

func integrationLinkedCharacter(t *testing.T, db *sql.DB, worldID string) (string, string) {
	t.Helper()
	characterID := integrationUUID(t)
	actorUUID := "Actor.integration-" + integrationUUID(t)
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO characters (id, name) VALUES ($1, 'Before Foundry sync')
`, characterID); err != nil {
		t.Fatalf("insert linked test character: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO foundry_actors (world_id, actor_uuid, character_id)
VALUES ($1, $2, $3)
`, worldID, actorUUID, characterID); err != nil {
		t.Fatalf("insert test Actor link: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM foundry_actors WHERE world_id = $1", worldID)
		_, _ = db.ExecContext(context.Background(), "DELETE FROM characters WHERE id = $1", characterID)
	})
	return characterID, actorUUID
}

func bindIntegrationActor(record *FoundrySyncRecord, worldID, characterID, actorUUID string) {
	record.WorldID = worldID
	record.CharacterID = characterID
	record.ActorUUID = actorUUID
	record.Character.ID = characterID
	actorID := strings.TrimPrefix(actorUUID, "Actor.")
	record.ActorSnapshot = []byte(fmt.Sprintf(`{"_id":%q,"name":%q}`, actorID, record.Character.Name))
	hash := sha256.Sum256(record.ActorSnapshot)
	record.SnapshotHash = hex.EncodeToString(hash[:])
}

func TestFoundryRepositorySyncActorPostgresIntegration(t *testing.T) {
	db := openFoundryIntegrationDB(t)
	ctx := context.Background()
	worldID := integrationWorld(t, db)
	characterID, actorUUID := integrationLinkedCharacter(t, db, worldID)

	// Existing Foundry row must retain its relational ID on refresh. A stale
	// Foundry row is removed, while an app/catalog-owned row (NULL source ID)
	// must survive reconciliation.
	preservedItemID := integrationUUID(t)
	seedTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin inventory fixture transaction: %v", err)
	}
	defer func() { _ = seedTx.Rollback() }()
	if _, err := seedTx.ExecContext(ctx, "SET LOCAL app.writer = 'foundry_sync'"); err != nil {
		t.Fatalf("set fixture writer: %v", err)
	}
	if _, err := seedTx.ExecContext(ctx, `
INSERT INTO inventory_items (id, character_id, foundry_item_id, name, quantity, state)
VALUES ($1, $2, 'item-1', 'Old name', 1, '{}'::jsonb),
       ($3, $2, 'stale-item', 'Stale Foundry item', 1, '{}'::jsonb),
       ($4, $2, NULL, 'App-owned catalog item', 7, '{"origin":"app"}'::jsonb)
`, preservedItemID, characterID, integrationUUID(t), integrationUUID(t)); err != nil {
		t.Fatalf("seed inventory rows: %v", err)
	}
	if err := seedTx.Commit(); err != nil {
		t.Fatalf("commit inventory fixture transaction: %v", err)
	}

	record := testSyncRecord(t)
	bindIntegrationActor(&record, worldID, characterID, actorUUID)
	quantity := 3
	record.Inventory[0].Quantity = &quantity
	record.Inventory[0].State = []byte(`{"id":"item-1","name":"Longsword","quantity":3}`)

	repo, err := NewFoundryRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SyncActor(ctx, record); err != nil {
		t.Fatalf("sync linked Actor: %v", err)
	}

	var characterName string
	if err := db.QueryRowContext(ctx, "SELECT name FROM characters WHERE id = $1", characterID).Scan(&characterName); err != nil {
		t.Fatalf("read synchronized character: %v", err)
	}
	if characterName != "Test Hero" {
		t.Fatalf("character name = %q, want Test Hero", characterName)
	}
	var snapshotHash string
	var syncedAt sql.NullTime
	if err := db.QueryRowContext(ctx, `
SELECT snapshot_hash, last_synced_at FROM foundry_actors
WHERE world_id = $1 AND actor_uuid = $2 AND character_id = $3
`, worldID, actorUUID, characterID).Scan(&snapshotHash, &syncedAt); err != nil {
		t.Fatalf("read synchronized Actor snapshot metadata: %v", err)
	}
	if snapshotHash != record.SnapshotHash || !syncedAt.Valid {
		t.Fatalf("snapshot metadata mismatch: hash=%q syncedAt=%v", snapshotHash, syncedAt)
	}

	var itemID string
	var gotQuantity int
	if err := db.QueryRowContext(ctx, `
SELECT id::text, quantity FROM inventory_items
WHERE character_id = $1 AND foundry_item_id = 'item-1'
`, characterID).Scan(&itemID, &gotQuantity); err != nil {
		t.Fatalf("read reconciled Foundry item: %v", err)
	}
	if itemID != preservedItemID || gotQuantity != 3 {
		t.Fatalf("Foundry item id/quantity = %s/%d, want %s/3", itemID, gotQuantity, preservedItemID)
	}
	var staleCount, appOwnedCount, totalCount int
	if err := db.QueryRowContext(ctx, `
SELECT
  count(*) FILTER (WHERE foundry_item_id = 'stale-item'),
  count(*) FILTER (WHERE foundry_item_id IS NULL AND name = 'App-owned catalog item'),
  count(*)
FROM inventory_items WHERE character_id = $1
`, characterID).Scan(&staleCount, &appOwnedCount, &totalCount); err != nil {
		t.Fatalf("count reconciled inventory: %v", err)
	}
	if staleCount != 0 || appOwnedCount != 1 || totalCount != 2 {
		t.Fatalf("inventory counts stale/app-owned/total = %d/%d/%d, want 0/1/2", staleCount, appOwnedCount, totalCount)
	}

	// A repeated snapshot is idempotent and keeps the existing item row ID.
	if err := repo.SyncActor(ctx, record); err != nil {
		t.Fatalf("repeat Actor sync: %v", err)
	}
	var repeatedItemID string
	if err := db.QueryRowContext(ctx, `
SELECT id::text FROM inventory_items WHERE character_id = $1 AND foundry_item_id = 'item-1'
`, characterID).Scan(&repeatedItemID); err != nil {
		t.Fatalf("read item after repeated sync: %v", err)
	}
	if repeatedItemID != preservedItemID {
		t.Fatalf("repeated sync changed inventory row ID: got %s want %s", repeatedItemID, preservedItemID)
	}

	// The database itself rejects writes to Foundry-owned fields outside the
	// transaction-local integration writer context.
	if _, err := db.ExecContext(ctx, "UPDATE characters SET name = 'Manual overwrite' WHERE id = $1", characterID); err == nil {
		t.Fatal("direct character write unexpectedly bypassed Foundry ownership trigger")
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO inventory_items (character_id, name) VALUES ($1, 'Manual inventory write')
`, characterID); err == nil {
		t.Fatal("direct inventory write unexpectedly bypassed Foundry ownership trigger")
	}
}

func TestFoundryRepositoryPostgresIntegrationRollsBackUnlinkedActor(t *testing.T) {
	db := openFoundryIntegrationDB(t)
	ctx := context.Background()
	worldID := integrationWorld(t, db)
	characterID := integrationUUID(t)

	record := testSyncRecord(t)
	bindIntegrationActor(&record, worldID, characterID, "Actor.unlinked-"+integrationUUID(t))

	repo, err := NewFoundryRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	err = repo.SyncActor(ctx, record)
	if err != ErrActorNotLinked {
		t.Fatalf("SyncActor error = %v, want ErrActorNotLinked", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM characters WHERE id = $1", characterID).Scan(&count); err != nil {
		t.Fatalf("check rolled-back character: %v", err)
	}
	if count != 0 {
		t.Fatalf("unlinked Actor sync left %d character rows behind; want rollback", count)
	}
}

func TestFoundryRepositorySyncDiscoveredNPCStoresSnapshotOnly(t *testing.T) {
	db := openFoundryIntegrationDB(t)
	ctx := context.Background()
	worldID := integrationWorld(t, db)
	repo, err := NewFoundryRepository(db)
	if err != nil {
		t.Fatal(err)
	}

	snapshot := []byte(`{"_id":"npc-1","name":"Roadside Guard","type":"npc","items":[]}`)
	disposition, err := repo.SyncDiscoveredActor(ctx, worldID, "Actor.npc-1", snapshot)
	if err != nil {
		t.Fatalf("sync discovered NPC: %v", err)
	}
	if disposition != DiscoverySnapshotOnly {
		t.Fatalf("disposition = %q, want %q", disposition, DiscoverySnapshotOnly)
	}

	var storedSnapshot string
	var characterID sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT snapshot::text, character_id::text FROM foundry_actors WHERE world_id = $1 AND actor_uuid = $2", worldID, "Actor.npc-1").Scan(&storedSnapshot, &characterID); err != nil {
		t.Fatalf("read discovered NPC snapshot: %v", err)
	}
	if !strings.Contains(storedSnapshot, "Roadside Guard") || characterID.Valid {
		t.Fatalf("NPC snapshot/link mismatch: snapshot=%s characterID=%v", storedSnapshot, characterID)
	}
}

func TestFoundryRepositorySyncDiscoveredLinkedCharacter(t *testing.T) {
	db := openFoundryIntegrationDB(t)
	ctx := context.Background()
	worldID := integrationWorld(t, db)
	characterID, actorUUID := integrationLinkedCharacter(t, db, worldID)
	repo, err := NewFoundryRepository(db)
	if err != nil {
		t.Fatal(err)
	}

	actorID := strings.TrimPrefix(actorUUID, "Actor.")
	snapshot := []byte(fmt.Sprintf(`{"_id":%q,"name":"Discovered Hero","type":"character","items":[]}`, actorID))
	disposition, err := repo.SyncDiscoveredActor(ctx, worldID, actorUUID, snapshot)
	if err != nil {
		t.Fatalf("sync discovered linked character: %v", err)
	}
	if disposition != DiscoveryCharacterSynced {
		t.Fatalf("disposition = %q, want %q", disposition, DiscoveryCharacterSynced)
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT name FROM characters WHERE id = $1", characterID).Scan(&name); err != nil {
		t.Fatalf("read synchronized character: %v", err)
	}
	if name != "Discovered Hero" {
		t.Fatalf("character name = %q, want Discovered Hero", name)
	}
}

func TestFoundryRepositoryLinkActorPostgresIntegration(t *testing.T) {
	db := openFoundryIntegrationDB(t)
	ctx := context.Background()
	worldID := integrationWorld(t, db)
	characterID := integrationUUID(t)
	actorUUID := "Actor.link-" + integrationUUID(t)
	userID := integrationUUID(t)

	if _, err := db.ExecContext(ctx, "INSERT INTO users (id, display_name) VALUES ($1, 'Link reviewer')", userID); err != nil {
		t.Fatalf("insert linking user: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO characters (id, name) VALUES ($1, 'Existing application character')", characterID); err != nil {
		t.Fatalf("insert existing application character: %v", err)
	}
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, "DELETE FROM characters WHERE id = $1", characterID) })
	if _, err := db.ExecContext(ctx, `
INSERT INTO foundry_actors (world_id, actor_uuid, snapshot)
VALUES ($1, $2, '{"_id":"linked","name":"Foundry Hero","type":"character"}'::jsonb)
`, worldID, actorUUID); err != nil {
		t.Fatalf("insert discovered Actor snapshot: %v", err)
	}

	repo, err := NewFoundryRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.LinkActor(ctx, worldID, actorUUID, characterID, userID); err != nil {
		t.Fatalf("link discovered Actor: %v", err)
	}

	var linkedCharacterID string
	var ownerID sql.NullString
	var ownerConfirmedAt sql.NullTime
	if err := db.QueryRowContext(ctx, `
SELECT fa.character_id::text, c.owner_user_id::text, fa.owner_confirmed_at
FROM foundry_actors fa JOIN characters c ON c.id = fa.character_id
WHERE fa.world_id = $1 AND fa.actor_uuid = $2
`, worldID, actorUUID).Scan(&linkedCharacterID, &ownerID, &ownerConfirmedAt); err != nil {
		t.Fatalf("read linked Actor: %v", err)
	}
	if linkedCharacterID != characterID || ownerID.Valid || ownerConfirmedAt.Valid {
		t.Fatalf("link changed unexpected ownership fields: character=%q owner=%v ownerConfirmed=%v", linkedCharacterID, ownerID, ownerConfirmedAt)
	}

	// Repeating the same explicit operation is safe and does not duplicate audit.
	if err := repo.LinkActor(ctx, worldID, actorUUID, characterID, userID); err != nil {
		t.Fatalf("repeat identical link: %v", err)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM audit_log
WHERE actor_user_id = $1 AND action = 'foundry.actor.linked' AND entity_id = $2
`, userID, actorUUID).Scan(&auditCount); err != nil {
		t.Fatalf("count link audit rows: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("link audit rows = %d, want exactly one", auditCount)
	}

	otherCharacterID := integrationUUID(t)
	if _, err := db.ExecContext(ctx, "INSERT INTO characters (id, name) VALUES ($1, 'Other character')", otherCharacterID); err != nil {
		t.Fatalf("insert other character: %v", err)
	}
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, "DELETE FROM characters WHERE id = $1", otherCharacterID) })
	if err := repo.LinkActor(ctx, worldID, actorUUID, otherCharacterID, userID); err != ErrActorAlreadyLinked {
		t.Fatalf("link already linked Actor error = %v, want ErrActorAlreadyLinked", err)
	}

	otherActorUUID := "Actor.link-" + integrationUUID(t)
	if _, err := db.ExecContext(ctx, `
INSERT INTO foundry_actors (world_id, actor_uuid, snapshot)
VALUES ($1, $2, '{"_id":"other","name":"Another Hero","type":"character"}'::jsonb)
`, worldID, otherActorUUID); err != nil {
		t.Fatalf("insert second discovered Actor: %v", err)
	}
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, "DELETE FROM foundry_actors WHERE world_id = $1", worldID) })
	if err := repo.LinkActor(ctx, worldID, otherActorUUID, characterID, userID); err != ErrCharacterAlreadyLinked {
		t.Fatalf("link second Actor to occupied character error = %v, want ErrCharacterAlreadyLinked", err)
	}
}

func TestFoundryRepositoryLinkActorRejectsNonCharacterPostgresIntegration(t *testing.T) {
	db := openFoundryIntegrationDB(t)
	ctx := context.Background()
	worldID := integrationWorld(t, db)
	characterID := integrationUUID(t)
	actorUUID := "Actor.npc-" + integrationUUID(t)
	userID := integrationUUID(t)

	if _, err := db.ExecContext(ctx, "INSERT INTO users (id, display_name) VALUES ($1, 'Link reviewer')", userID); err != nil {
		t.Fatalf("insert linking user: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO characters (id, name) VALUES ($1, 'Existing character')", characterID); err != nil {
		t.Fatalf("insert application character: %v", err)
	}
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, "DELETE FROM characters WHERE id = $1", characterID) })
	if _, err := db.ExecContext(ctx, `
INSERT INTO foundry_actors (world_id, actor_uuid, snapshot)
VALUES ($1, $2, '{"_id":"npc","name":"Guard","type":"npc"}'::jsonb)
`, worldID, actorUUID); err != nil {
		t.Fatalf("insert NPC snapshot: %v", err)
	}
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, "DELETE FROM foundry_actors WHERE world_id = $1", worldID) })

	repo, err := NewFoundryRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.LinkActor(ctx, worldID, actorUUID, characterID, userID); err != ErrActorNotCharacter {
		t.Fatalf("link NPC error = %v, want ErrActorNotCharacter", err)
	}
	var linked bool
	if err := db.QueryRowContext(ctx, "SELECT character_id IS NOT NULL FROM foundry_actors WHERE world_id = $1 AND actor_uuid = $2", worldID, actorUUID).Scan(&linked); err != nil {
		t.Fatalf("check NPC link: %v", err)
	}
	if linked {
		t.Fatal("NPC must remain unlinked")
	}
}
