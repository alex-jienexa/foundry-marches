package characters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrActorNotFound          = errors.New("Foundry Actor was not discovered in this world")
	ErrCharacterNotFound      = errors.New("application character was not found")
	ErrActorNotCharacter      = errors.New("only Foundry character Actors can be linked")
	ErrActorAlreadyLinked     = errors.New("Foundry Actor is already linked to another character")
	ErrCharacterAlreadyLinked = errors.New("application character is already linked to another Foundry Actor")
	ErrCharacterOwnerRequired = errors.New("an owner must be assigned before linking the Foundry Actor")
	ErrCharacterOwnerConflict = errors.New("application character already belongs to another user")
)

// LinkActor explicitly links a previously discovered Foundry character Actor to an
// existing application character. It never creates a character, assigns an owner,
// or confirms player ownership. Repeating the same link is idempotent.
func (r *FoundryRepository) LinkActor(ctx context.Context, worldID, actorUUID, characterID, linkedByUserID string) error {
	return r.LinkActorWithOwner(ctx, worldID, actorUUID, characterID, linkedByUserID, "")
}

// LinkActorWithOwner links a Foundry Actor and explicitly assigns an owner when
// supplied. An existing owner is never silently replaced. Once linked, an
// assigned owner is marked confirmed for this link.
func (r *FoundryRepository) LinkActorWithOwner(ctx context.Context, worldID, actorUUID, characterID, linkedByUserID, ownerUserID string) error {
	if r == nil || r.db == nil {
		return errors.New("Foundry repository is not initialized")
	}
	if worldID == "" || actorUUID == "" || characterID == "" || linkedByUserID == "" {
		return errors.New("world ID, Actor UUID, character ID and linking user ID are required")
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin Foundry Actor link transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Lock the target first. The unique constraint on foundry_actors.character_id
	// remains the final concurrency guard across different Actor/world rows.
	var foundCharacterID string
	var currentOwnerID sql.NullString
	if err = tx.QueryRowContext(ctx,
		"SELECT id::text, owner_user_id::text FROM characters WHERE id = $1 FOR UPDATE", characterID,
	).Scan(&foundCharacterID, &currentOwnerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCharacterNotFound
		}
		return fmt.Errorf("lock application character: %w", err)
	}

	var actorType string
	var existingCharacterID sql.NullString
	if err = tx.QueryRowContext(ctx,
		"SELECT COALESCE(snapshot->>'type', ''), character_id::text FROM foundry_actors WHERE world_id = $1 AND actor_uuid = $2 FOR UPDATE",
		worldID, actorUUID,
	).Scan(&actorType, &existingCharacterID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrActorNotFound
		}
		return fmt.Errorf("lock discovered Foundry Actor: %w", err)
	}
	if actorType != "character" {
		return ErrActorNotCharacter
	}
	if ownerUserID != "" && currentOwnerID.Valid && currentOwnerID.String != ownerUserID {
		return ErrCharacterOwnerConflict
	}
	if !currentOwnerID.Valid && ownerUserID == "" {
		return ErrCharacterOwnerRequired
	}
	if !currentOwnerID.Valid && ownerUserID != "" {
		if _, err = tx.ExecContext(ctx, "SELECT set_config('app.user_id', $1, true), set_config('app.reason', 'Discord GM assigned character owner during Foundry link', true)", linkedByUserID); err != nil {
			return fmt.Errorf("set ownership audit context: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE characters SET owner_user_id = $2, updated_at = now() WHERE id = $1 AND owner_user_id IS NULL", characterID, ownerUserID); err != nil {
			return fmt.Errorf("assign character owner: %w", err)
		}
	}
	if existingCharacterID.Valid {
		if existingCharacterID.String == characterID {
			return tx.Commit()
		}
		return ErrActorAlreadyLinked
	}

	result, err := tx.ExecContext(ctx,
		"UPDATE foundry_actors SET character_id = $3, linked_at = now(), owner_confirmed_at = CASE WHEN $4::boolean THEN clock_timestamp() ELSE owner_confirmed_at END, updated_at = now() WHERE world_id = $1 AND actor_uuid = $2 AND character_id IS NULL",
		worldID, actorUUID, characterID, currentOwnerID.Valid || ownerUserID != "",
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrCharacterAlreadyLinked
		}
		return fmt.Errorf("link Foundry Actor to application character: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check Foundry Actor link result: %w", err)
	}
	if affected != 1 {
		return ErrActorAlreadyLinked
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO audit_log (
    actor_user_id,
    actor_kind,
    action,
    entity_type,
    entity_id,
    old_value,
    new_value
)
VALUES (
    $1,
    'user',
    'foundry.actor.linked',
    'foundry_actor',
    $2::text,
    jsonb_build_object('character_id', NULL),
    jsonb_build_object(
        'world_id', $3::text,
        'actor_uuid', $2::text,
        'character_id', $4::text
    )
)
`, linkedByUserID, actorUUID, worldID, characterID)
	if err != nil {
		return fmt.Errorf("write Foundry Actor link audit record: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit Foundry Actor link: %w", err)
	}
	return nil
}

type sqlStateError interface {
	SQLState() string
}

func isUniqueViolation(err error) bool {
	var stateErr sqlStateError
	return errors.As(err, &stateErr) && stateErr.SQLState() == "23505"
}
