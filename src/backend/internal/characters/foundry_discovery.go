package characters

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

type FoundryDiscoveryDisposition string

const (
	DiscoveryCharacterSynced FoundryDiscoveryDisposition = "character_synced"
	DiscoverySnapshotOnly    FoundryDiscoveryDisposition = "snapshot_only"
)

// SyncDiscoveredActor stores a snapshot for every discovered Actor. If the Actor
// is a character already linked to an application character, it also refreshes
// the normalized character and inventory projection in the same repository path.
func (r *FoundryRepository) SyncDiscoveredActor(ctx context.Context, worldID, actorUUID string, snapshot json.RawMessage) (FoundryDiscoveryDisposition, error) {
	if r == nil || r.db == nil {
		return "", errors.New("Foundry repository is not initialized")
	}
	if worldID == "" || actorUUID == "" || len(snapshot) == 0 || !json.Valid(snapshot) {
		return "", ErrSyncRecordInvalid
	}
	decoded, err := pf2e.DecodeActor(snapshot)
	if err != nil {
		return "", fmt.Errorf("decode discovered Actor %q: %w", actorUUID, err)
	}
	if decoded.Character.Identity.UUID != actorUUID {
		return "", fmt.Errorf("%w: requested %q, decoded %q", pf2e.ErrInvalidPayload, actorUUID, decoded.Character.Identity.UUID)
	}

	var linkedCharacterID sql.NullString
	err = r.db.QueryRowContext(ctx, "SELECT character_id::text FROM foundry_actors WHERE world_id = $1 AND actor_uuid = $2", worldID, actorUUID).Scan(&linkedCharacterID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("find application link for Actor %q: %w", actorUUID, err)
	}

	if decoded.Character.Identity.Type == "character" && linkedCharacterID.Valid && linkedCharacterID.String != "" {
		record, mapErr := MapFoundryActor(FoundrySyncInput{
			WorldID: worldID, CharacterID: linkedCharacterID.String,
			ActorUUID: actorUUID, Decoded: *decoded,
		})
		if mapErr != nil {
			return "", fmt.Errorf("map linked Actor %q: %w", actorUUID, mapErr)
		}
		if err := r.SyncActor(ctx, record); err != nil {
			return "", err
		}
		return DiscoveryCharacterSynced, nil
	}

	sum := sha256.Sum256(snapshot)
	_, err = r.db.ExecContext(ctx, "INSERT INTO foundry_actors (world_id, actor_uuid, snapshot, snapshot_hash, last_synced_at) VALUES ($1, $2, $3::jsonb, $4, now()) ON CONFLICT (world_id, actor_uuid) DO UPDATE SET snapshot = EXCLUDED.snapshot, snapshot_hash = EXCLUDED.snapshot_hash, last_synced_at = EXCLUDED.last_synced_at, updated_at = now()", worldID, actorUUID, string(snapshot), hex.EncodeToString(sum[:]))
	if err != nil {
		return "", fmt.Errorf("store discovered Actor %q snapshot: %w", actorUUID, err)
	}
	return DiscoverySnapshotOnly, nil
}
