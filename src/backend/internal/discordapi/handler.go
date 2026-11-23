package discordapi

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/lib/pq"

	"github.com/alex-jienexa/foundry-marches/internal/characters"
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

const (
	RegisterPath        = "/api/internal/discord/register"
	ActorsPath          = "/api/internal/discord/actors"
	CharactersPath      = "/api/internal/discord/characters"
	CharacterCreatePath = "/api/internal/discord/characters/create"
	PreviewPath         = "/api/internal/discord/link/preview"
	LinkPath            = "/api/internal/discord/link"
	GrantRolePath       = "/api/internal/discord/roles/grant"
	MapRolePath         = "/api/internal/discord/roles/map"
	DraftCharacterPath  = "/api/internal/discord/characters/draft"
	CharacterSearchPath = "/api/internal/discord/characters/search"
	CharacterDetailPath = "/api/internal/discord/characters/detail"
)

type Handler struct {
	db           *sql.DB
	foundry      *characters.FoundryRepository
	guildID      string
	worldID      string
	adminUserIDs map[string]struct{}
}

func NewHandler(db *sql.DB, foundry *characters.FoundryRepository, apiToken, guildID, worldID string, adminUserIDs []string) (*Handler, error) {
	if db == nil || foundry == nil {
		return nil, errors.New("database and Foundry repository are required")
	}
	if len(apiToken) < 32 || strings.TrimSpace(guildID) == "" || strings.TrimSpace(worldID) == "" {
		return nil, errors.New("Discord API token (32+ chars), guild ID and Foundry world ID are required")
	}
	admins := make(map[string]struct{}, len(adminUserIDs))
	for _, id := range adminUserIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			admins[id] = struct{}{}
		}
	}
	return &Handler{db: db, foundry: foundry, guildID: guildID, worldID: worldID, adminUserIDs: admins}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	switch r.URL.Path {
	case RegisterPath:
		h.register(w, r)
	case ActorsPath:
		h.listActors(w, r)
	case CharactersPath:
		h.listCharacters(w, r)
	case CharacterCreatePath:
		h.createCharacter(w, r)
	case PreviewPath:
		h.previewLink(w, r)
	case LinkPath:
		h.link(w, r)
	case GrantRolePath:
		h.grantRole(w, r)
	case MapRolePath:
		h.mapRole(w, r)
	case DraftCharacterPath:
		h.createCharacterDraft(w, r)
	case CharacterSearchPath:
		h.searchFoundryCharacters(w, r)
	case CharacterDetailPath:
		h.foundryCharacterDetails(w, r)
	default:
		http.NotFound(w, r)
	}
}

type identityRequest struct {
	GuildID        string   `json:"guild_id"`
	DiscordUserID  string   `json:"discord_user_id"`
	DiscordRoleIDs []string `json:"discord_role_ids"`
}

type registerRequest struct {
	identityRequest
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

type actorView struct {
	ActorUUID string `json:"actor_uuid"`
	Name      string `json:"name"`
}

type characterView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type linkRequest struct {
	identityRequest
	ActorUUID string `json:"actor_uuid"`
	CharacterID string `json:"character_id"`
	OwnerDiscordUserID string `json:"owner_discord_user_id,omitempty"`
}

type createCharacterRequest struct {
	identityRequest
	Name string `json:"name"`
	OwnerDiscordUserID string `json:"owner_discord_user_id,omitempty"`
}

type createCharacterDraftRequest struct {
	identityRequest
	ActorUUID string `json:"actor_uuid"`
}

type characterSearchRequest struct {
	identityRequest
	Query string `json:"query"`
}

type characterDetailRequest struct {
	identityRequest
	ActorUUID string `json:"actor_uuid"`
}

type foundryCharacterSearchView struct {
	ActorUUID string `json:"actor_uuid"`
	Name      string `json:"name"`
}

type linkPreview struct {
	ActorUUID     string `json:"actor_uuid"`
	ActorName     string `json:"actor_name"`
	CharacterID   string `json:"character_id"`
	CharacterName string `json:"character_name"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if err := h.validateGuild(req.GuildID, req.DiscordUserID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if strings.TrimSpace(req.Username) == "" {
		writeError(w, http.StatusBadRequest, "Discord username is required")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start registration")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var userID string
	err = tx.QueryRowContext(r.Context(), "SELECT user_id::text FROM discord_accounts WHERE discord_id = $1 FOR UPDATE", req.DiscordUserID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(r.Context(), `
INSERT INTO users (display_name, avatar_url, status)
VALUES ($1, NULLIF($2, ''), 'active')
RETURNING id::text
`, req.Username, req.AvatarURL).Scan(&userID)
		if err == nil {
			_, err = tx.ExecContext(r.Context(), `
INSERT INTO discord_accounts (user_id, discord_id, username, in_guild)
VALUES ($1, $2, $3, true)
`, userID, req.DiscordUserID, req.Username)
		}
	} else if err == nil {
		_, err = tx.ExecContext(r.Context(), `
UPDATE users SET display_name = $2, avatar_url = NULLIF($3, ''), status = 'active', updated_at = now()
WHERE id = $1
`, userID, req.Username, req.AvatarURL)
		if err == nil {
			_, err = tx.ExecContext(r.Context(), `
UPDATE discord_accounts SET username = $2, in_guild = true, updated_at = now()
WHERE discord_id = $1
`, req.DiscordUserID, req.Username)
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not register Discord account")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `
INSERT INTO user_roles (user_id, role_id)
SELECT $1, id FROM roles WHERE key = 'player'
ON CONFLICT DO NOTHING
`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not assign default player role")
		return
	}
	if _, trusted := h.adminUserIDs[req.DiscordUserID]; trusted {
		if _, err = tx.ExecContext(r.Context(), `
INSERT INTO user_roles (user_id, role_id)
SELECT $1, id FROM roles WHERE key = 'owner'
ON CONFLICT DO NOTHING
`, userID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not assign configured owner role")
			return
		}
	}
	if _, err = tx.ExecContext(r.Context(), `
INSERT INTO audit_log (
    actor_user_id,
    actor_kind,
    action,
    entity_type,
    entity_id,
    new_value
)
VALUES (
    $1,
    'user',
    'discord.account.registered',
    'discord_account',
    $2,
    jsonb_build_object(
        'guild_id', $3::text,
        'username', $4::text
    )
)
`, userID, req.DiscordUserID, req.GuildID, req.Username); err != nil {
		log.Printf("Discord registration audit insert failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not write registration audit record")
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit registration")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": userID, "status": "registered"})
}

func (h *Handler) createCharacter(w http.ResponseWriter, r *http.Request) {
	var req createCharacterRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	userID, canManage, ok := h.authorizeCharacterRead(w, r.Context(), req.identityRequest)
	if !ok {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		writeError(w, http.StatusBadRequest, "character name must contain 1..80 characters")
		return
	}
	ownerID := userID
	if canManage {
		if strings.TrimSpace(req.OwnerDiscordUserID) == "" {
			writeError(w, http.StatusBadRequest, "a GM must explicitly select the character owner")
			return
		}
	} else if req.OwnerDiscordUserID != "" {
		writeError(w, http.StatusForbidden, "players can create characters only for their own account")
		return
	}
	if req.OwnerDiscordUserID != "" {
		err := h.db.QueryRowContext(r.Context(), `
SELECT da.user_id::text
FROM discord_accounts da
JOIN users u ON u.id = da.user_id
WHERE da.discord_id = $1 AND da.in_guild = true AND u.status = 'active'
`, req.OwnerDiscordUserID).Scan(&ownerID)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "the selected owner must register with /register first")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not resolve selected character owner")
			return
		}
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start character creation")
		return
	}
	defer func() { _ = tx.Rollback() }()
	var created characterView
	err = tx.QueryRowContext(r.Context(), `
INSERT INTO characters (owner_user_id, name, status)
VALUES ($1, $2, 'draft')
RETURNING id::text, name, status
`, ownerID, name).Scan(&created.ID, &created.Name, &created.Status)
	if err != nil {
		log.Printf("Discord character creation failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not create character draft")
		return
	}
	_, err = tx.ExecContext(r.Context(), `
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, new_value)
VALUES ($1, 'user', 'character.draft.created', 'character', $2,
        jsonb_build_object('source', 'discord', 'owner_user_id', $3::text, 'name', $4::text))
`, userID, created.ID, ownerID, name)
	if err != nil {
		log.Printf("Discord character creation audit failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not write character creation audit record")
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit character creation")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) listActors(w http.ResponseWriter, r *http.Request) {
	var req identityRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if _, ok := h.authorize(w, r.Context(), req, "foundry.manage"); !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `
SELECT actor_uuid, COALESCE(snapshot->>'name', actor_uuid)
FROM foundry_actors
WHERE world_id = $1 AND character_id IS NULL AND snapshot->>'type' = 'character'
ORDER BY COALESCE(snapshot->>'name', actor_uuid)
LIMIT 100
`, h.worldID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list unlinked Foundry Actors")
		return
	}
	defer rows.Close()
	actors := make([]actorView, 0)
	for rows.Next() {
		var item actorView
		if err := rows.Scan(&item.ActorUUID, &item.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read Foundry Actor list")
			return
		}
		actors = append(actors, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read Foundry Actor list")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"actors": actors})
}

func (h *Handler) listCharacters(w http.ResponseWriter, r *http.Request) {
	var req identityRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if _, ok := h.authorize(w, r.Context(), req, "foundry.manage"); !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `
SELECT id::text, name, status
FROM characters
WHERE status <> 'archived'
ORDER BY name
LIMIT 200
`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list application characters")
		return
	}
	defer rows.Close()
	items := make([]characterView, 0)
	for rows.Next() {
		var item characterView
		if err := rows.Scan(&item.ID, &item.Name, &item.Status); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read character list")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read character list")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"characters": items})
}

// searchFoundryCharacters searches persisted Actor snapshots. It never calls Relay,
// so repeated Discord lookups do not create additional Foundry requests.
func (h *Handler) searchFoundryCharacters(w http.ResponseWriter, r *http.Request) {
	var req characterSearchRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	userID, canManage, ok := h.authorizeCharacterRead(w, r.Context(), req.identityRequest)
	if !ok {
		return
	}
	query := strings.TrimSpace(req.Query)
	if len([]rune(query)) < 2 {
		writeError(w, http.StatusBadRequest, "search query must contain at least two characters")
		return
	}

	var rows *sql.Rows
	var err error
	if canManage {
		rows, err = h.db.QueryContext(r.Context(), `
SELECT actor_uuid, COALESCE(snapshot->>'name', actor_uuid)
FROM foundry_actors
WHERE world_id = $1
  AND snapshot->>'type' = 'character'
  AND snapshot->>'name' ILIKE $2
ORDER BY snapshot->>'name', actor_uuid
LIMIT 25
`, h.worldID, "%"+query+"%")
	} else {
		// Players can discover only their own drafts, pending approvals and active characters after ownership
		// has been explicitly confirmed on the Foundry Actor link.
		rows, err = h.db.QueryContext(r.Context(), `
SELECT fa.actor_uuid, c.name
FROM foundry_actors fa
JOIN characters c ON c.id = fa.character_id
WHERE fa.world_id = $1
  AND fa.owner_confirmed_at IS NOT NULL
  AND fa.owner_confirmed_at > COALESCE(
    (SELECT MAX(ct.created_at) FROM character_transfers ct WHERE ct.character_id = c.id),
    fa.linked_at
  )
  AND c.owner_user_id = $2
  AND c.status IN ('draft', 'pending_approval', 'active')
  AND c.name ILIKE $3
  AND fa.snapshot->>'type' = 'character'
ORDER BY c.name, fa.actor_uuid
LIMIT 25
`, h.worldID, userID, "%"+query+"%")
	}
	if err != nil {
		log.Printf("Discord Foundry character search failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not search Foundry characters")
		return
	}
	defer rows.Close()

	items := make([]foundryCharacterSearchView, 0)
	for rows.Next() {
		var item foundryCharacterSearchView
		if err := rows.Scan(&item.ActorUUID, &item.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read Foundry character search results")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read Foundry character search results")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"actors": items})
}

// foundryCharacterDetails decodes a saved Actor snapshot. GMs can inspect all
// character Actors; players can inspect only their own active, confirmed links.
func (h *Handler) foundryCharacterDetails(w http.ResponseWriter, r *http.Request) {
	var req characterDetailRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	userID, canManage, ok := h.authorizeCharacterRead(w, r.Context(), req.identityRequest)
	if !ok {
		return
	}
	if strings.TrimSpace(req.ActorUUID) == "" {
		writeError(w, http.StatusBadRequest, "Actor UUID is required")
		return
	}

	var snapshot []byte
	err := h.db.QueryRowContext(r.Context(), `
SELECT fa.snapshot
FROM foundry_actors fa
WHERE fa.world_id = $1
  AND fa.actor_uuid = $2
  AND fa.snapshot->>'type' = 'character'
  AND (
    $3::boolean = true
    OR EXISTS (
      SELECT 1
      FROM characters c
      WHERE c.id = fa.character_id
        AND c.owner_user_id = $4
        AND c.status IN ('draft', 'pending_approval', 'active')
        AND fa.owner_confirmed_at IS NOT NULL
        AND fa.owner_confirmed_at > COALESCE(
          (SELECT MAX(ct.created_at) FROM character_transfers ct WHERE ct.character_id = c.id),
          fa.linked_at
        )
    )
  )
`, h.worldID, req.ActorUUID, canManage, userID).Scan(&snapshot)
	if errors.Is(err, sql.ErrNoRows) {
		// Do not reveal whether another player's Actor UUID exists.
		writeError(w, http.StatusNotFound, "character was not found or is not available to this account")
		return
	}
	if err != nil {
		log.Printf("Discord Foundry character snapshot read failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not read Foundry character snapshot")
		return
	}

	decoded, err := pf2e.DecodeActor(snapshot)
	if err != nil {
		log.Printf("Discord Foundry character decode failed for %s: %v", req.ActorUUID, err)
		writeError(w, http.StatusUnprocessableEntity, "stored Foundry character snapshot cannot be decoded")
		return
	}
	writeJSON(w, http.StatusOK, decoded.Character)
}

// authorizeCharacterRead returns the registered application user and whether
// they have the global Foundry-management permission. Ordinary players are
// authorized only by the ownership predicates in the search/detail queries.
func (h *Handler) authorizeCharacterRead(w http.ResponseWriter, ctx context.Context, req identityRequest) (string, bool, bool) {
	if err := h.validateGuild(req.GuildID, req.DiscordUserID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return "", false, false
	}
	var userID string
	err := h.db.QueryRowContext(ctx, `
SELECT da.user_id::text
FROM discord_accounts da
JOIN users u ON u.id = da.user_id
WHERE da.discord_id = $1 AND da.in_guild = true AND u.status = 'active'
`, req.DiscordUserID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusForbidden, "use /register before accessing the application")
		return "", false, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not resolve Discord identity")
		return "", false, false
	}

	var canManage, canViewOwn bool
	err = h.db.QueryRowContext(ctx, `
SELECT
  EXISTS (
    SELECT 1
    FROM user_roles ur
    JOIN role_permissions rp ON rp.role_id = ur.role_id
    WHERE ur.user_id = $1 AND rp.permission_key = 'foundry.manage'
    UNION
    SELECT 1
    FROM discord_role_map drm
    JOIN role_permissions rp ON rp.role_id = drm.role_id
    WHERE drm.guild_id = $2
      AND drm.discord_role_id = ANY($3)
      AND rp.permission_key = 'foundry.manage'
  ),
  EXISTS (
    SELECT 1
    FROM user_roles ur
    JOIN role_permissions rp ON rp.role_id = ur.role_id
    WHERE ur.user_id = $1 AND rp.permission_key = 'characters.create'
    UNION
    SELECT 1
    FROM discord_role_map drm
    JOIN role_permissions rp ON rp.role_id = drm.role_id
    WHERE drm.guild_id = $2
      AND drm.discord_role_id = ANY($3)
      AND rp.permission_key = 'characters.create'
  )
`, userID, h.guildID, pq.Array(req.DiscordRoleIDs)).Scan(&canManage, &canViewOwn)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not evaluate character access")
		return "", false, false
	}
	if !canManage && !canViewOwn {
		writeError(w, http.StatusForbidden, "missing permission: characters.create")
		return "", false, false
	}
	return userID, canManage, true
}

func (h *Handler) createCharacterDraft(w http.ResponseWriter, r *http.Request) {
	var req createCharacterDraftRequest
	if !decodeRequest(w, r, &req) {
		return
	}

	userID, ok := h.authorize(w, r.Context(), req.identityRequest, "foundry.manage")
	if !ok {
		return
	}

	if strings.TrimSpace(req.ActorUUID) == "" {
		writeError(w, http.StatusBadRequest, "Actor UUID is required")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start character creation")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var actorName string
	var portraitURL sql.NullString

	err = tx.QueryRowContext(r.Context(), `
SELECT
    COALESCE(snapshot->>'name', actor_uuid),
    NULLIF(snapshot->>'img', '')
FROM foundry_actors
WHERE world_id = $1
  AND actor_uuid = $2
  AND character_id IS NULL
  AND snapshot->>'type' = 'character'
FOR UPDATE
`, h.worldID, req.ActorUUID).Scan(&actorName, &portraitURL)

	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusConflict, "Actor is unavailable, already linked, or has an incompatible type")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read Foundry Actor")
		return
	}

	// A retry must not create another draft for the same still-unlinked Actor.
	// The Actor row is locked above, so concurrent requests serialize here.
	var existing characterView
	err = tx.QueryRowContext(r.Context(), `
SELECT c.id::text, c.name, c.status
FROM audit_log al
JOIN characters c ON c.id::text = al.entity_id
WHERE al.action = 'character.draft.created'
  AND al.entity_type = 'character'
  AND al.new_value->>'actor_uuid' = $1
  AND al.new_value->>'world_id' = $2
ORDER BY al.created_at DESC
LIMIT 1
`, req.ActorUUID, h.worldID).Scan(&existing.ID, &existing.Name, &existing.Status)
	if err == nil {
		if err = tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not finish character draft lookup")
			return
		}
		writeJSON(w, http.StatusOK, existing)
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "could not check for an existing character draft")
		return
	}

	var characterID string

	err = tx.QueryRowContext(r.Context(), `
INSERT INTO characters (
    name,
    portrait_url,
    status
)
VALUES ($1, $2, 'draft')
RETURNING id::text
`, actorName, nullableSQLString(portraitURL)).Scan(&characterID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create character draft")
		return
	}

	_, err = tx.ExecContext(r.Context(), `
INSERT INTO audit_log (
    actor_user_id,
    actor_kind,
    action,
    entity_type,
    entity_id,
    new_value
)
VALUES (
    $1,
    'user',
    'character.draft.created',
    'character',
    $2,
    jsonb_build_object(
        'source', 'foundry',
        'actor_uuid', $3::text,
        'world_id', $4::text,
        'status', 'draft'
    )
)
`, userID, characterID, req.ActorUUID, h.worldID)

	if err != nil {
		log.Printf("Discord character draft audit insert failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not write character draft audit record")
		return
	}

	if err = tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit character draft")
		return
	}

	writeJSON(w, http.StatusCreated, characterView{
		ID:     characterID,
		Name:   actorName,
		Status: "draft",
	})
}

func nullableSQLString(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	return value.String
}

func (h *Handler) previewLink(w http.ResponseWriter, r *http.Request) {
	var req linkRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if _, ok := h.authorize(w, r.Context(), req.identityRequest, "foundry.manage"); !ok {
		return
	}
	if strings.TrimSpace(req.OwnerDiscordUserID) == "" {
		writeError(w, http.StatusBadRequest, "a Discord user mention is required to assign the character owner")
		return
	}
	var ownerUserID string
	err := h.db.QueryRowContext(r.Context(), `
SELECT da.user_id::text
FROM discord_accounts da
JOIN users u ON u.id = da.user_id
WHERE da.discord_id = $1 AND da.in_guild = true AND u.status = 'active'
`, req.OwnerDiscordUserID).Scan(&ownerUserID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "the selected owner must register with /register first")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not resolve selected character owner")
		return
	}
	var currentOwnerID sql.NullString
	err = h.db.QueryRowContext(r.Context(), "SELECT owner_user_id::text FROM characters WHERE id = $1", req.CharacterID).Scan(&currentOwnerID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusConflict, "application character is unavailable")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not inspect application character owner")
		return
	}
	if currentOwnerID.Valid && currentOwnerID.String != ownerUserID {
		writeError(w, http.StatusConflict, "application character already belongs to another user")
		return
	}
	var result linkPreview
	err = h.db.QueryRowContext(r.Context(), `
SELECT fa.actor_uuid, COALESCE(fa.snapshot->>'name', fa.actor_uuid), c.id::text, c.name
FROM foundry_actors fa
CROSS JOIN characters c
WHERE fa.world_id = $1 AND fa.actor_uuid = $2
  AND fa.character_id IS NULL AND fa.snapshot->>'type' = 'character'
  AND c.id = $3
  AND NOT EXISTS (SELECT 1 FROM foundry_actors occupied WHERE occupied.character_id = c.id)
`, h.worldID, req.ActorUUID, req.CharacterID).Scan(&result.ActorUUID, &result.ActorName, &result.CharacterID, &result.CharacterName)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusConflict, "Actor or character is unavailable, already linked, or has an incompatible type")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not validate requested link")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) link(w http.ResponseWriter, r *http.Request) {
	var req linkRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	userID, ok := h.authorize(w, r.Context(), req.identityRequest, "foundry.manage")
	if !ok {
		return
	}
	if strings.TrimSpace(req.OwnerDiscordUserID) == "" {
		writeError(w, http.StatusBadRequest, "a Discord user mention is required to assign the character owner")
		return
	}
	ownerUserID := ""
	{
		err := h.db.QueryRowContext(r.Context(), `
SELECT da.user_id::text
FROM discord_accounts da
JOIN users u ON u.id = da.user_id
WHERE da.discord_id = $1 AND da.in_guild = true AND u.status = 'active'
`, req.OwnerDiscordUserID).Scan(&ownerUserID)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "the selected owner must register with /register first")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not resolve selected character owner")
			return
		}
	}
	err := h.foundry.LinkActorWithOwner(r.Context(), h.worldID, req.ActorUUID, req.CharacterID, userID, ownerUserID)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "linked"})
	case errors.Is(err, characters.ErrActorNotFound), errors.Is(err, characters.ErrCharacterNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, characters.ErrActorNotCharacter), errors.Is(err, characters.ErrActorAlreadyLinked), errors.Is(err, characters.ErrCharacterAlreadyLinked), errors.Is(err, characters.ErrCharacterOwnerRequired), errors.Is(err, characters.ErrCharacterOwnerConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("Discord Foundry Actor link failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not link Foundry Actor")
	}
}

type roleRequest struct {
	identityRequest
	TargetDiscordUserID string `json:"target_discord_user_id"`
	RoleKey             string `json:"role_key"`
	DiscordRoleID       string `json:"discord_role_id"`
}

func (h *Handler) grantRole(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	actorID, ok := h.authorize(w, r.Context(), req.identityRequest, "settings.manage")
	if !ok {
		return
	}
	if req.RoleKey != "player" && req.RoleKey != "gm" {
		writeError(w, http.StatusBadRequest, "only player and gm roles can be granted through Discord")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start role grant")
		return
	}
	defer func() { _ = tx.Rollback() }()
	var targetID string
	err = tx.QueryRowContext(r.Context(), "SELECT user_id::text FROM discord_accounts WHERE discord_id = $1 AND in_guild = true FOR UPDATE", req.TargetDiscordUserID).Scan(&targetID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "target Discord account is not registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not find target account")
		return
	}
	_, err = tx.ExecContext(r.Context(), `
INSERT INTO user_roles (user_id, role_id, granted_by)
SELECT $1, id, $2 FROM roles WHERE key = $3
ON CONFLICT (user_id, role_id) DO NOTHING
`, targetID, actorID, req.RoleKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not grant role")
		return
	}
	_, err = tx.ExecContext(r.Context(), `
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, new_value)
VALUES ($1, 'user', 'rbac.role.granted', 'user', $2, jsonb_build_object('role', $3))
`, actorID, targetID, req.RoleKey)
	if err != nil {
		log.Printf("Discord registration audit insert failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not write role grant audit record")
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit role grant")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "granted", "role": req.RoleKey})
}

func (h *Handler) mapRole(w http.ResponseWriter, r *http.Request) {
	var req roleRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	actorID, ok := h.authorize(w, r.Context(), req.identityRequest, "settings.manage")
	if !ok {
		return
	}
	if req.RoleKey != "player" && req.RoleKey != "gm" {
		writeError(w, http.StatusBadRequest, "only player and gm roles can be mapped from Discord roles")
		return
	}
	if strings.TrimSpace(req.DiscordRoleID) == "" {
		writeError(w, http.StatusBadRequest, "Discord role ID is required")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start Discord role mapping")
		return
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(r.Context(), `
INSERT INTO discord_role_map (guild_id, discord_role_id, role_id)
SELECT $1, $2, id FROM roles WHERE key = $3
ON CONFLICT (guild_id, discord_role_id) DO UPDATE SET role_id = EXCLUDED.role_id
`, h.guildID, req.DiscordRoleID, req.RoleKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not map Discord role")
		return
	}
	_, err = tx.ExecContext(r.Context(), `
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, new_value)
VALUES ($1, 'user', 'rbac.discord_role.mapped', 'discord_role', $2, jsonb_build_object('role', $3))
`, actorID, req.DiscordRoleID, req.RoleKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not write Discord role mapping audit record")
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit Discord role mapping")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "mapped", "role": req.RoleKey})
}

func (h *Handler) authorize(w http.ResponseWriter, ctx context.Context, req identityRequest, permission string) (string, bool) {
	if err := h.validateGuild(req.GuildID, req.DiscordUserID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return "", false
	}
	var userID string
	err := h.db.QueryRowContext(ctx, `
SELECT da.user_id::text
FROM discord_accounts da
JOIN users u ON u.id = da.user_id
WHERE da.discord_id = $1 AND da.in_guild = true AND u.status = 'active'
`, req.DiscordUserID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusForbidden, "use /register before accessing the application")
		return "", false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not resolve Discord identity")
		return "", false
	}
	var allowed bool
	err = h.db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM user_roles ur
  JOIN role_permissions rp ON rp.role_id = ur.role_id
  WHERE ur.user_id = $1 AND rp.permission_key = $2
  UNION
  SELECT 1
  FROM discord_role_map drm
  JOIN role_permissions rp ON rp.role_id = drm.role_id
  WHERE drm.guild_id = $3
    AND drm.discord_role_id = ANY($4)
    AND rp.permission_key = $2
)
`, userID, permission, h.guildID, pq.Array(req.DiscordRoleIDs)).Scan(&allowed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not evaluate application permissions")
		return "", false
	}
	if !allowed {
		writeError(w, http.StatusForbidden, fmt.Sprintf("missing permission: %s", permission))
		return "", false
	}
	return userID, true
}

func (h *Handler) validateGuild(guildID, discordUserID string) error {
	if guildID != h.guildID {
		return errors.New("this bot is restricted to the configured Discord server")
	}
	if strings.TrimSpace(discordUserID) == "" {
		return errors.New("Discord user ID is required")
	}
	return nil
}

func decodeRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request must contain one JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func ServiceTokenMiddleware(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		provided := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if !strings.HasPrefix(header, prefix) || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
