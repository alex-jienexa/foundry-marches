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
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync"
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

const (
	RegisterPath        = "/api/internal/discord/register"
	ActorsPath          = "/api/internal/discord/actors"
	CharactersPath      = "/api/internal/discord/characters"
	CharacterCreatePath = "/api/internal/discord/characters/create"
	PreviewPath         = "/api/internal/discord/link/preview"
	LinkPath            = "/api/internal/discord/link"
	UnlinkPath          = "/api/internal/discord/unlink"
	GrantRolePath       = "/api/internal/discord/roles/grant"
	MapRolePath         = "/api/internal/discord/roles/map"
	DraftCharacterPath  = "/api/internal/discord/characters/draft"
	CharacterSearchPath = "/api/internal/discord/characters/search"
	CharacterDetailPath = "/api/internal/discord/characters/detail"
	MyCharactersPath    = "/api/internal/discord/characters/mine"
	ConnectActorsPath   = "/api/internal/discord/characters/connect/actors"
	ConnectActorPath    = "/api/internal/discord/characters/connect"
	UndoConnectPath     = "/api/internal/discord/characters/connect/undo"
	HelpStatusPath      = "/api/internal/discord/help-status"
	FoundrySyncPath     = "/api/internal/discord/foundry-sync"
)

type Handler struct {
	db           *sql.DB
	foundry      *characters.FoundryRepository
	guildID      string
	worldID      string
	adminUserIDs map[string]struct{}
	syncer       interface {
		SyncAll(context.Context) (foundrysync.SyncReport, error)
	}
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

func (h *Handler) SetSyncService(syncer interface {
	SyncAll(context.Context) (foundrysync.SyncReport, error)
}) {
	h.syncer = syncer
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
	case UnlinkPath:
		h.unlinkActor(w, r)
	case GrantRolePath:
		h.grantRole(w, r)
	case MapRolePath:
		h.mapRole(w, r)
	case DraftCharacterPath:
		h.createCharacterDraft(w, r)
	case CharacterSearchPath:
		h.searchFoundryCharacters(w, r)
	case MyCharactersPath:
		h.listMyCharacters(w, r)
	case ConnectActorsPath:
		h.listConnectableActors(w, r)
	case ConnectActorPath:
		h.connectActor(w, r)
	case UndoConnectPath:
		h.undoCharacterConnect(w, r)
	case HelpStatusPath:
		h.helpStatus(w, r)
	case CharacterDetailPath:
		h.foundryCharacterDetails(w, r)
	case FoundrySyncPath:
		h.syncFoundryWorld(w, r)
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
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	OwnerName string `json:"owner_name,omitempty"`
}

type linkRequest struct {
	identityRequest
	ActorUUID          string `json:"actor_uuid"`
	CharacterID        string `json:"character_id"`
	OwnerDiscordUserID string `json:"owner_discord_user_id,omitempty"`
}

type unlinkRequest struct {
	identityRequest
	ActorUUID          string `json:"actor_uuid"`
	OwnerDiscordUserID string `json:"owner_discord_user_id,omitempty"`
}

type createCharacterRequest struct {
	identityRequest
	Name               string `json:"name"`
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

type connectActorRequest struct {
	identityRequest
	ActorUUID string `json:"actor_uuid"`
}

type undoConnectRequest struct {
	identityRequest
	ActionID string `json:"action_id"`
}

type ownCharacterView struct {
	ID          string  `json:"id"`
	ActorUUID   string  `json:"actor_uuid"`
	Name        string  `json:"name"`
	PortraitURL *string `json:"portrait_url,omitempty"`
	Level       int     `json:"level"`
	XP          int     `json:"xp"`
	Ancestry    string  `json:"ancestry,omitempty"`
	Class       string  `json:"class,omitempty"`
	Status      string  `json:"status"`
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
SELECT c.id::text, c.name, c.status, COALESCE(u.display_name, '')
FROM characters c
LEFT JOIN users u ON u.id = c.owner_user_id
WHERE c.status <> 'archived'
ORDER BY c.name
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Status, &item.OwnerName); err != nil {
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

// listMyCharacters returns only the caller's linked, explicitly owner-confirmed
// Foundry Actors. It deliberately shares the same visibility boundary as player search.
func (h *Handler) helpStatus(w http.ResponseWriter, r *http.Request) {
	var req identityRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if err := h.validateGuild(req.GuildID, req.DiscordUserID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	var registered, linked bool
	err := h.db.QueryRowContext(r.Context(), `
SELECT
    EXISTS (
        SELECT 1
        FROM discord_accounts da
        JOIN users u ON u.id = da.user_id
        WHERE da.discord_id = $1
          AND da.in_guild = true
          AND u.status = 'active'
    ),
    EXISTS (
        SELECT 1
        FROM discord_accounts da
        JOIN users u ON u.id = da.user_id
        JOIN characters c ON c.owner_user_id = u.id
        JOIN foundry_actors fa ON fa.character_id = c.id
        WHERE da.discord_id = $1
          AND da.in_guild = true
          AND u.status = 'active'
          AND c.status IN ('draft', 'pending_approval', 'active', 'unavailable')
          AND fa.world_id = $2
          AND fa.snapshot->>'type' = 'character'
          AND fa.owner_confirmed_at IS NOT NULL
          AND fa.owner_confirmed_at > COALESCE(
              (SELECT MAX(ct.created_at) FROM character_transfers ct WHERE ct.character_id = c.id),
              fa.linked_at
          )
    )
`, req.DiscordUserID, h.worldID).Scan(&registered, &linked)
	if err != nil {
		log.Printf("Discord help status lookup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not determine account status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{
		"registered": registered,
		"linked":     linked,
	})
}

func (h *Handler) listMyCharacters(w http.ResponseWriter, r *http.Request) {
	var req identityRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	userID, _, ok := h.authorizeCharacterRead(w, r.Context(), req)
	if !ok {
		return
	}

	rows, err := h.db.QueryContext(r.Context(), `
SELECT fa.actor_uuid, c.id::text, c.name, c.portrait_url, c.level, c.xp,
       COALESCE(c.ancestry, ''), COALESCE(c.class, ''), c.status
FROM foundry_actors fa
JOIN characters c ON c.id = fa.character_id
WHERE fa.world_id = $1 AND c.owner_user_id = $2
  AND c.status IN ('draft', 'pending_approval', 'active', 'unavailable')
  AND fa.owner_confirmed_at IS NOT NULL
  AND fa.owner_confirmed_at > COALESCE(
    (SELECT MAX(ct.created_at) FROM character_transfers ct WHERE ct.character_id = c.id),
    fa.linked_at
  )
  AND fa.snapshot->>'type' = 'character'
ORDER BY c.name, fa.actor_uuid LIMIT 25
`, h.worldID, userID)
	if err != nil {
		log.Printf("Discord own character list failed: %v", err)
		writeError(w, 500, "could not list your linked characters")
		return
	}
	defer rows.Close()
	items := make([]ownCharacterView, 0)
	for rows.Next() {
		var item ownCharacterView
		var portrait sql.NullString
		if err := rows.Scan(&item.ActorUUID, &item.ID, &item.Name, &portrait, &item.Level, &item.XP, &item.Ancestry, &item.Class, &item.Status); err != nil {
			writeError(w, 500, "could not read your linked characters")
			return
		}
		if portrait.Valid {
			item.PortraitURL = &portrait.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "could not read your linked characters")
		return
	}
	if err := rows.Close(); err != nil {
		writeError(w, 500, "could not close your character list")
		return
	}

	var activeSlots, maxSlots int
	err = h.db.QueryRowContext(r.Context(), `
SELECT
  (SELECT count(*) FROM characters WHERE owner_user_id = $1 AND status IN ('pending_approval', 'active', 'unavailable')),
  COALESCE((SELECT (value #>> '{}')::integer FROM campaign_settings WHERE key = 'max_characters_per_player'), 2)
`, userID).Scan(&activeSlots, &maxSlots)
	if err != nil {
		log.Printf("Discord character slot lookup failed: %v", err)
		writeError(w, 500, "could not determine active character slots")
		return
	}
	remaining := maxSlots - activeSlots
	if remaining < 0 {
		remaining = 0
	}
	actors := make([]foundryCharacterSearchView, 0, len(items))
	for _, item := range items {
		actors = append(actors, foundryCharacterSearchView{ActorUUID: item.ActorUUID, Name: item.Name})
	}
	writeJSON(w, 200, map[string]any{"characters": items, "actors": actors, "active_slots": activeSlots, "max_active_slots": maxSlots, "remaining_slots": remaining})
}

func (h *Handler) listConnectableActors(w http.ResponseWriter, r *http.Request) {
	var req identityRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if _, _, ok := h.authorizeCharacterRead(w, r.Context(), req); !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `
SELECT actor_uuid, COALESCE(snapshot->>'name', actor_uuid)
FROM foundry_actors
WHERE world_id = $1 AND character_id IS NULL AND snapshot->>'type' = 'character'
ORDER BY COALESCE(snapshot->>'name', actor_uuid), actor_uuid LIMIT 100
`, h.worldID)
	if err != nil {
		writeError(w, 500, "could not list connectable Foundry Actors")
		return
	}
	defer rows.Close()
	items := make([]actorView, 0)
	for rows.Next() {
		var item actorView
		if err := rows.Scan(&item.ActorUUID, &item.Name); err != nil {
			writeError(w, 500, "could not read connectable Foundry Actors")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "could not read connectable Foundry Actors")
		return
	}
	writeJSON(w, 200, map[string]any{"actors": items})
}

func (h *Handler) connectActor(w http.ResponseWriter, r *http.Request) {
	var req connectActorRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	userID, _, ok := h.authorizeCharacterRead(w, r.Context(), req.identityRequest)
	if !ok {
		return
	}
	if strings.TrimSpace(req.ActorUUID) == "" {
		writeError(w, 400, "Actor UUID is required")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		writeError(w, 500, "could not start character connection")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var snapshot []byte
	err = tx.QueryRowContext(r.Context(), `
SELECT snapshot FROM foundry_actors
WHERE world_id = $1 AND actor_uuid = $2
  AND character_id IS NULL AND snapshot->>'type' = 'character'
FOR UPDATE
`, h.worldID, req.ActorUUID).Scan(&snapshot)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 409, "this Foundry Actor is already connected or unavailable")
		return
	}
	if err != nil {
		log.Printf("Discord self-connect Actor lookup failed: %v", err)
		writeError(w, 500, "could not read Foundry Actor")
		return
	}

	decoded, err := pf2e.DecodeActor(snapshot)
	if err != nil || decoded == nil || decoded.Character.Identity.Type != "character" || decoded.Character.Identity.UUID != req.ActorUUID {
		log.Printf("Discord self-connect Actor decode failed for %s: %v", req.ActorUUID, err)
		writeError(w, 422, "Foundry Actor snapshot is not a valid character")
			return
	}
	var characterID string
	if err = tx.QueryRowContext(r.Context(), "SELECT gen_random_uuid()::text").Scan(&characterID); err != nil {
		writeError(w, 500, "could not allocate character ID")
		return
	}
	record, err := characters.MapFoundryActor(characters.FoundrySyncInput{WorldID: h.worldID, CharacterID: characterID, ActorUUID: req.ActorUUID, Decoded: *decoded})
	if err != nil {
		log.Printf("Discord self-connect Actor mapping failed for %s: %v", req.ActorUUID, err)
		writeError(w, 422, "Foundry character data cannot be mapped")
		return
	}
	c := record.Character
	_, err = tx.ExecContext(r.Context(), `
INSERT INTO characters (
  id, owner_user_id, name, portrait_url, status, ancestry, heritage, background, class,
  level, xp, hp_current, hp_max, attributes, skills
) VALUES (
  $1, $2, $3, $4, 'pending_approval', $5, $6, $7, $8,
  COALESCE($9, 1), COALESCE($10, 0), $11, $12, $13::jsonb, $14::jsonb
)
`, characterID, userID, c.Name, nullableStringValue(c.PortraitURL), nullableStringValue(c.Ancestry),
		nullableStringValue(c.Heritage), nullableStringValue(c.Background), nullableStringValue(c.Class),
		nullableIntValue(c.Level), nullableIntValue(c.XP), nullableIntValue(c.HPCurrent), nullableIntValue(c.HPMax),
		string(c.Attributes), string(c.Skills))
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23514" && strings.Contains(pgErr.Message, "character slot limit") {
			writeError(w, 409, "the active character limit has been reached (2 of 2 slots occupied)")
			return
		}
		log.Printf("Discord self-connect character creation failed: %v", err)
		writeError(w, 500, "could not create application character")
			return
	}

	// The state machine requires a new character to enter pending_approval first.
	// Self-connect is immediately approved within this transaction; the intermediate
	// state is never visible outside the transaction.
	if _, err = tx.ExecContext(r.Context(), "UPDATE characters SET status = 'active', updated_at = now() WHERE id = $1", characterID); err != nil {
		log.Printf("Discord self-connect character activation failed: %v", err)
		writeError(w, 500, "could not activate application character")
			return
	}

	result, err := tx.ExecContext(r.Context(), `
UPDATE foundry_actors
SET character_id = $3, linked_at = now(),
    owner_confirmed_at = clock_timestamp(), updated_at = now()
WHERE world_id = $1 AND actor_uuid = $2 AND character_id IS NULL
`, h.worldID, req.ActorUUID, characterID)
	if err != nil {
		log.Printf("Discord self-connect Actor link failed: %v", err)
		writeError(w, 500, "could not link Foundry Actor")
		return
	}
	if affected, rowsErr := result.RowsAffected(); rowsErr != nil || affected != 1 {
		writeError(w, 409, "this Foundry Actor was connected by another request")
		return
	}

	if _, err = tx.ExecContext(r.Context(), `
INSERT INTO character_status_history (character_id, from_status, to_status, changed_by, reason)
VALUES
  ($1, NULL, 'pending_approval', $2, 'Player connected a Foundry character'),
  ($1, 'pending_approval', 'active', $2, 'Self-service connection is immediately approved')
`, characterID, userID); err != nil {
		log.Printf("Discord self-connect status history insert failed: %v", err)
		writeError(w, 500, "could not write character status history")
		return
	}

	var actionID string
	err = tx.QueryRowContext(r.Context(), `
INSERT INTO character_link_actions (world_id, actor_uuid, character_id, owner_user_id, linked_by_user_id)
VALUES ($1, $2, $3, $4, $4) RETURNING id::text
`, h.worldID, req.ActorUUID, characterID, userID).Scan(&actionID)
	if err != nil {
		log.Printf("Discord self-connect action insert failed: %v", err)
		writeError(w, 500, "could not create reversible link record")
		return
	}
	_, err = tx.ExecContext(r.Context(), `
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, old_value, new_value)
VALUES ($1, 'user', 'foundry.actor.self_linked', 'foundry_actor', $2,
        jsonb_build_object('character_id', NULL),
        jsonb_build_object('world_id', $3::text, 'actor_uuid', $2::text,
                           'character_id', $4::text, 'owner_user_id', ($1::uuid)::text, 'link_action_id', $5::text))
`, userID, req.ActorUUID, h.worldID, characterID, actionID)
	if err != nil {
		log.Printf("Discord self-connect audit insert failed: %v", err)
		writeError(w, 500, "could not write character link audit")
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, 500, "could not commit character connection")
		return
	}
	writeJSON(w, 201, map[string]any{
		"action_id": actionID, "character": map[string]any{"id": characterID, "name": c.Name, "status": "active"},
		"actor_uuid": req.ActorUUID, "owner_user_id": userID,
	})
}

func (h *Handler) undoCharacterConnect(w http.ResponseWriter, r *http.Request) {
	var req undoConnectRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	masterID, ok := h.authorize(w, r.Context(), req.identityRequest, "foundry.manage")
	if !ok {
		return
	}
	if strings.TrimSpace(req.ActionID) == "" {
		writeError(w, 400, "link action ID is required")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		writeError(w, 500, "could not start link undo")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var worldID, actorUUID, characterID, ownerID, state string
	err = tx.QueryRowContext(r.Context(), `
SELECT world_id::text, actor_uuid, character_id::text, owner_user_id::text, state
FROM character_link_actions WHERE id = $1 FOR UPDATE
`, req.ActionID).Scan(&worldID, &actorUUID, &characterID, &ownerID, &state)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, 404, "link action was not found")
		return
	}
	if err != nil {
		log.Printf("Discord link undo action lookup failed: %v", err)
		writeError(w, 500, "could not read link action")
		return
	}
	if state == "undone" {
		if err = tx.Commit(); err != nil {
			writeError(w, 500, "could not finish link undo lookup")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "already_undone", "actor_uuid": actorUUID})
			return
	}
	if worldID != h.worldID {
		writeError(w, 409, "link action belongs to another Foundry world")
		return
	}

	// Keep the same lock order as the GM link path: Character first, then Actor.
	var currentOwnerID sql.NullString
	var characterName, currentStatus string
	err = tx.QueryRowContext(r.Context(), "SELECT owner_user_id::text, name, status FROM characters WHERE id = $1 FOR UPDATE", characterID).Scan(&currentOwnerID, &characterName, &currentStatus)
	if err != nil || !currentOwnerID.Valid || currentOwnerID.String != ownerID {
		writeError(w, 409, "the character owner has changed since this action; automatic undo is unsafe")
			return
	}
	if currentStatus != "active" {
		writeError(w, 409, "the character status has changed since this action; automatic undo is unsafe")
			return
	}
	var currentCharacterID sql.NullString
	err = tx.QueryRowContext(r.Context(), `
SELECT character_id::text FROM foundry_actors WHERE world_id = $1 AND actor_uuid = $2 FOR UPDATE
`, worldID, actorUUID).Scan(&currentCharacterID)
	if err != nil || !currentCharacterID.Valid || currentCharacterID.String != characterID {
		writeError(w, 409, "the Actor link has changed since this action; automatic undo is unsafe")
			return
	}
	var configuredUserID, configuredReason string
	if err = tx.QueryRowContext(r.Context(), `
SELECT set_config('app.user_id', $1, true),
       set_config('app.reason', 'GM reversed a self-service Foundry character connection', true)
`, masterID).Scan(&configuredUserID, &configuredReason); err != nil {
		writeError(w, 500, "could not set undo audit context")
		return
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE characters SET owner_user_id = NULL, status = 'draft', updated_at = now() WHERE id = $1", characterID); err != nil {
		log.Printf("Discord character link undo owner release failed: %v", err)
		writeError(w, 409, "the character cannot be released safely (it may have an active signup or expedition)")
			return
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE foundry_actors SET character_id = NULL, owner_confirmed_at = NULL, updated_at = now() WHERE world_id = $1 AND actor_uuid = $2 AND character_id = $3", worldID, actorUUID, characterID); err != nil {
		writeError(w, 500, "could not detach Foundry Actor")
			return
	}
	if _, err = tx.ExecContext(r.Context(), `
INSERT INTO character_status_history (character_id, from_status, to_status, changed_by, reason)
VALUES ($1, $2, 'draft', $3, 'GM reversed self-service Foundry connection')
`, characterID, currentStatus, masterID); err != nil {
		writeError(w, 500, "could not write undo status history")
		return
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE character_link_actions SET state = 'undone', undone_at = now(), undone_by_user_id = $2 WHERE id = $1 AND state = 'linked'", req.ActionID, masterID); err != nil {
		writeError(w, 500, "could not update link action state")
			return
	}
	if _, err = tx.ExecContext(r.Context(), `
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, old_value, new_value)
VALUES ($1, 'user', 'foundry.actor.self_link_undone', 'foundry_actor', $2,
        jsonb_build_object('character_id', $3::text, 'owner_user_id', $4::text),
        jsonb_build_object('world_id', $5::text, 'character_id', NULL, 'link_action_id', $6::text))
`, masterID, actorUUID, characterID, ownerID, worldID, req.ActionID); err != nil {
	writeError(w, 500, "could not write link undo audit")
	return
}
	if err = tx.Commit(); err != nil {
		writeError(w, 500, "could not commit link undo")
		return
	}
	writeJSON(w, 200, map[string]any{"status": "undone", "actor_uuid": actorUUID, "character_name": characterName})
}

func nullableStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
func nullableIntValue(value *int) any {
	if value == nil {
		return nil
	}
	return *value
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
  AND c.status IN ('draft', 'pending_approval', 'active', 'unavailable')
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
// character Actors; players can inspect only their own linked drafts, pending approvals, active and unavailable characters.
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
        AND c.status IN ('draft', 'pending_approval', 'active', 'unavailable')
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
	err = h.foundry.LinkActorWithOwner(r.Context(), h.worldID, req.ActorUUID, req.CharacterID, userID, ownerUserID)
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

func (h *Handler) syncFoundryWorld(w http.ResponseWriter, r *http.Request) {
	var req identityRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if _, ok := h.authorize(w, r.Context(), req, "foundry.manage"); !ok {
		return
	}
	if h.syncer == nil {
		writeError(w, http.StatusServiceUnavailable, "Foundry sync service is unavailable")
		return
	}
	report, err := h.syncer.SyncAll(r.Context())
	if err != nil {
		if errors.Is(err, foundrysync.ErrSyncInProgress) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		log.Printf("Discord-triggered Foundry sync failed: %v", err)
		writeError(w, http.StatusBadGateway, "Foundry sync failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handler) unlinkActor(w http.ResponseWriter, r *http.Request) {
	var req unlinkRequest
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

	newOwnerID := ""
	if strings.TrimSpace(req.OwnerDiscordUserID) != "" {
		err := h.db.QueryRowContext(r.Context(), `
SELECT da.user_id::text
FROM discord_accounts da
JOIN users u ON u.id = da.user_id
WHERE da.discord_id = $1 AND da.in_guild = true AND u.status = 'active'
`, req.OwnerDiscordUserID).Scan(&newOwnerID)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "the selected owner must register with /register first")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not resolve selected character owner")
			return
		}
	}
	err := h.foundry.UnlinkActor(r.Context(), h.worldID, req.ActorUUID, userID, newOwnerID)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
	case errors.Is(err, characters.ErrActorNotLinked):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("Discord Foundry Actor unlink failed: %v", err)
		writeError(w, http.StatusInternalServerError, "could not unlink Foundry Actor")
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
