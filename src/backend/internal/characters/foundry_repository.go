package characters

import (
	"context"
	"database/sql"
	"encoding/json"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

var (
	ErrActorNotLinked = errors.New("Foundry Actor is not linked to this character")
	ErrSyncRecordInvalid = errors.New("invalid Foundry sync record")
)

type SQLDB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type FoundryRepository struct { db SQLDB }

func NewFoundryRepository(db SQLDB) (*FoundryRepository, error) {
	if db == nil { return nil, errors.New("database is required") }
	return &FoundryRepository{db: db}, nil
}

// SyncActor atomically mirrors one already-linked Actor. Owner, status, approval
// and app-owned ledger data are intentionally not modified.
func (r *FoundryRepository) SyncActor(ctx context.Context, record FoundrySyncRecord) error {
	if r == nil || r.db == nil { return errors.New("Foundry repository is not initialized") }
	if record.WorldID == "" || record.CharacterID == "" || record.ActorUUID == "" ||
		record.Character.ID != record.CharacterID || record.Character.Name == "" ||
		len(record.ActorSnapshot) == 0 || !json.Valid(record.ActorSnapshot) || record.SnapshotHash == "" {
		return ErrSyncRecordInvalid
	}
	sum := sha256.Sum256(record.ActorSnapshot)
	if record.SnapshotHash != hex.EncodeToString(sum[:]) { return ErrSyncRecordInvalid }
	if !json.Valid(record.Character.Attributes) || !json.Valid(record.Character.Skills) { return ErrSyncRecordInvalid }
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil { return fmt.Errorf("begin Foundry sync transaction: %w", err) }
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, "SET LOCAL app.writer = 'foundry_sync'"); err != nil {
		return fmt.Errorf("set Foundry sync writer: %w", err)
	}
	if err = upsertCharacter(ctx, tx, record.Character); err != nil { return err }
	if err = upsertFoundryActor(ctx, tx, record); err != nil { return err }
	if err = replaceInventory(ctx, tx, record.CharacterID, record.Inventory); err != nil { return err }
	if err = tx.Commit(); err != nil { return fmt.Errorf("commit Foundry sync transaction: %w", err) }
	return nil
}

type sqlExecer interface { ExecContext(context.Context, string, ...any) (sql.Result, error) }

func upsertCharacter(ctx context.Context, tx sqlExecer, c CharacterRecord) error {
	const query = `
INSERT INTO characters (
  id, name, portrait_url, ancestry, heritage, background, class, level, xp,
  hp_current, hp_max, attributes, skills
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, COALESCE($8, 1), COALESCE($9, 0),
  $10, $11, $12::jsonb, $13::jsonb
)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  portrait_url = COALESCE($3, characters.portrait_url),
  ancestry = COALESCE($4, characters.ancestry),
  heritage = COALESCE($5, characters.heritage),
  background = COALESCE($6, characters.background),
  class = COALESCE($7, characters.class),
  level = CASE WHEN $8::smallint IS NULL THEN characters.level ELSE EXCLUDED.level END,
  xp = CASE WHEN $9::integer IS NULL THEN characters.xp ELSE EXCLUDED.xp END,
  hp_current = COALESCE($10, characters.hp_current),
  hp_max = COALESCE($11, characters.hp_max),
  attributes = EXCLUDED.attributes,
  skills = EXCLUDED.skills
`
	_, err := tx.ExecContext(ctx, query, c.ID, c.Name, nullableString(c.PortraitURL),
		nullableString(c.Ancestry), nullableString(c.Heritage), nullableString(c.Background),
		nullableString(c.Class), nullableInt(c.Level), nullableInt(c.XP),
		nullableInt(c.HPCurrent), nullableInt(c.HPMax), []byte(c.Attributes), []byte(c.Skills))
	if err != nil { return fmt.Errorf("upsert character %s: %w", c.ID, err) }
	return nil
}

func upsertFoundryActor(ctx context.Context, tx sqlExecer, record FoundrySyncRecord) error {
	const query = `
UPDATE foundry_actors
SET snapshot = $4::jsonb,
    snapshot_hash = $5,
    last_synced_at = now(),
    updated_at = now()
WHERE world_id = $1
  AND actor_uuid = $2
  AND character_id = $3
`
	result, err := tx.ExecContext(ctx, query, record.WorldID, record.ActorUUID,
		record.CharacterID, []byte(record.ActorSnapshot), record.SnapshotHash)
	if err != nil { return fmt.Errorf("update linked Foundry Actor %s: %w", record.ActorUUID, err) }
	affected, err := result.RowsAffected()
	if err != nil { return fmt.Errorf("check Foundry Actor link: %w", err) }
	if affected == 0 { return ErrActorNotLinked }
	return nil
}

func replaceInventory(ctx context.Context, tx sqlExecer, characterID string, items []InventoryRecord) error {
	ids := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.FoundryItemID == "" || item.Name == "" || len(item.State) == 0 || !json.Valid(item.State) {
			return fmt.Errorf("%w: invalid inventory item", ErrSyncRecordInvalid)
		}
		if _, exists := seen[item.FoundryItemID]; exists {
			return fmt.Errorf("%w: duplicate Foundry item ID %s", ErrSyncRecordInvalid, item.FoundryItemID)
		}
		seen[item.FoundryItemID] = struct{}{}
		ids = append(ids, item.FoundryItemID)
	}
	encodedIDs, err := json.Marshal(ids)
	if err != nil { return fmt.Errorf("marshal inventory IDs: %w", err) }
	if _, err = tx.ExecContext(ctx, `
DELETE FROM inventory_items
WHERE character_id = $1
  AND foundry_item_id IS NOT NULL
  AND foundry_item_id NOT IN (SELECT jsonb_array_elements_text($2::jsonb))
`, characterID, encodedIDs); err != nil {
		return fmt.Errorf("remove stale Foundry inventory items: %w", err)
	}
	for _, item := range items {
		if item.Quantity == nil {
			// An absent source quantity is retained as null in state. On insert the
			// schema default applies; on update the previous indexed quantity is kept.
			_, err = tx.ExecContext(ctx, `
INSERT INTO inventory_items (character_id, foundry_item_id, name, state)
VALUES ($1, $2, $3, $4::jsonb)
ON CONFLICT (character_id, foundry_item_id) DO UPDATE SET
  name = EXCLUDED.name, state = EXCLUDED.state
`, characterID, item.FoundryItemID, item.Name, []byte(item.State))
		} else {
			_, err = tx.ExecContext(ctx, `
INSERT INTO inventory_items (character_id, foundry_item_id, name, quantity, state)
VALUES ($1, $2, $3, $4, $5::jsonb)
ON CONFLICT (character_id, foundry_item_id) DO UPDATE SET
  name = EXCLUDED.name, quantity = EXCLUDED.quantity, state = EXCLUDED.state
`, characterID, item.FoundryItemID, item.Name, *item.Quantity, []byte(item.State))
		}
		if err != nil { return fmt.Errorf("upsert Foundry item %s: %w", item.FoundryItemID, err) }
	}
	return nil
}

