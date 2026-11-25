package characters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// UnlinkActor detaches a Foundry Actor without deleting the application character.
// The existing owner is preserved unless a new owner is explicitly supplied.
// Any owner change is recorded by character_transfer_guard in the same transaction.
func (r *FoundryRepository) UnlinkActor(ctx context.Context, worldID, actorUUID, unlinkedByUserID, newOwnerUserID string) error {
	if r == nil || r.db == nil {
		return errors.New("Foundry repository is not initialized")
	}
	if worldID == "" || actorUUID == "" || unlinkedByUserID == "" {
		return errors.New("world ID, Actor UUID and unlinking user ID are required")
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin Foundry Actor unlink transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var characterID string
	var currentOwnerID sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT fa.character_id::text, c.owner_user_id::text
FROM foundry_actors fa
JOIN characters c ON c.id = fa.character_id
WHERE fa.world_id = $1 AND fa.actor_uuid = $2
FOR UPDATE OF fa, c
`, worldID, actorUUID).Scan(&characterID, &currentOwnerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrActorNotLinked
	}
	if err != nil {
		return fmt.Errorf("lock Foundry Actor link for unlink: %w", err)
	}

	if newOwnerUserID != "" && (!currentOwnerID.Valid || currentOwnerID.String != newOwnerUserID) {
		var configuredUserID, configuredReason string
		if err = tx.QueryRowContext(ctx,
			"SELECT set_config('app.user_id', $1, true), set_config('app.reason', 'Discord GM explicitly reassigned owner while unlinking Foundry Actor', true)",
			unlinkedByUserID,
		).Scan(&configuredUserID, &configuredReason); err != nil {
			return fmt.Errorf("set ownership audit context: %w", err)
		}
		if _, err = tx.ExecContext(ctx,
			"UPDATE characters SET owner_user_id = $2, updated_at = now() WHERE id = $1",
			characterID, newOwnerUserID,
		); err != nil {
			return fmt.Errorf("assign new character owner during unlink: %w", err)
		}
	}

	// character_id is the current link state. linked_at is NOT NULL and retains
	// the timestamp of the most recent successful link for historical context.
	result, err := tx.ExecContext(ctx, `
UPDATE foundry_actors
SET character_id = NULL, owner_confirmed_at = NULL, updated_at = now()
WHERE world_id = $1 AND actor_uuid = $2 AND character_id = $3
`, worldID, actorUUID, characterID)
	if err != nil {
		return fmt.Errorf("detach Foundry Actor: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check Foundry Actor unlink result: %w", err)
	}
	if affected != 1 {
		return ErrActorNotLinked
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, old_value, new_value)
VALUES ($1, 'user', 'foundry.actor.unlinked', 'foundry_actor', $2,
        jsonb_build_object('character_id', $3::text),
        jsonb_build_object('world_id', $4::text, 'character_id', NULL))
`, unlinkedByUserID, actorUUID, characterID, worldID)
	if err != nil {
		return fmt.Errorf("write Foundry Actor unlink audit record: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit Foundry Actor unlink: %w", err)
	}
	return nil
}
