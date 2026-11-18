package discordapi

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lib/pq"

	"github.com/alex-jienexa/foundry-marches/internal/characters"
)

const (
	RegisterPath  = "/api/internal/discord/register"
	ActorsPath    = "/api/internal/discord/actors"
	CharactersPath = "/api/internal/discord/characters"
	PreviewPath   = "/api/internal/discord/link/preview"
	LinkPath      = "/api/internal/discord/link"
	GrantRolePath = "/api/internal/discord/roles/grant"
	MapRolePath   = "/api/internal/discord/roles/map"
)

type Handler struct {
	db          *sql.DB
	foundry     *characters.FoundryRepository
	apiToken    string
	guildID     string
	worldID     string
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
	return &Handler{db: db, foundry: foundry, apiToken: apiToken, guildID: guildID, worldID: worldID, adminUserIDs: admins}, nil
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
	case PreviewPath:
		h.previewLink(w, r)
	case LinkPath:
		h.link(w, r)
	case GrantRolePath:
		h.grantRole(w, r)
	case MapRolePath:
		h.mapRole(w, r)
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
	Level     string `json:"level,omitempty"`
}

type characterView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Owner  string `json:"owner_user_id,omitempty"`
}

type linkRequest struct {
	identityRequest
	ActorUUID   string `json:"actor_uuid"`
	CharacterID string `json:"character_id"`
}

type linkPreview struct {
	ActorUUID   string `json:"actor_uuid"`
	ActorName   string `json:"actor_name"`
	CharacterID string `json:"character_id"`
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
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, new_value)
VALUES ($1, 'user', 'discord.account.registered', 'discord_account', $2,
        jsonb_build_object('guild_id', $3, 'username', $4))
`, userID, req.DiscordUserID, req.GuildID, req.Username); err != nil {
		writeError(w, http.StatusInternalServerError, "could not write registration audit record")
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit registration")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": userID, "status": "registered"})
}

func (h *Handler) listActors(w http.ResponseWriter, r *http.Request) {
	var req identityRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	userID, ok := h.authorize(w, r.Context(), req.identityRequest, "foundry.manage")
	if !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `
SELECT actor_uuid, COALESCE(snapshot->>'name', actor_uuid), COALESCE(snapshot->'system'->>'details', '')
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
		if err := rows.Scan(&item.ActorUUID, &item.Name, &item.Level); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read Foundry Actor list")
			return
		}
		actors = append(actors, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read Foundry Actor list")
		return
	}
	_ = userID
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
SELECT id::text, name, status, COALESCE(owner_user_id::text, '')
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
		if err := rows.Scan(&item.ID, &item.Name, &item.Status, &item.Owner); err != nil {
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

func (h *Handler) previewLink(w http.ResponseWriter, r *http.Request) {
	var req linkRequest
	if !decodeRequest(w, r, &req) {
		return
	}
	if _, ok := h.authorize(w, r.Context(), req.identityRequest, "foundry.manage"); !ok {
		return
	}
	var result linkPreview
	err := h.db.QueryRowContext(r.Context(), `
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
	err := h.foundry.LinkActor(r.Context(), h.worldID, req.ActorUUID, req.CharacterID, userID)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "linked"})
	case errors.Is(err, characters.ErrActorNotFound), errors.Is(err, characters.ErrCharacterNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, characters.ErrActorNotCharacter), errors.Is(err, characters.ErrActorAlreadyLinked), errors.Is(err, characters.ErrCharacterAlreadyLinked):
		writeError(w, http.StatusConflict, err.Error())
	default:
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
	var targetID string
	err := h.db.QueryRowContext(r.Context(), "SELECT user_id::text FROM discord_accounts WHERE discord_id = $1 AND in_guild = true", req.TargetDiscordUserID).Scan(&targetID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "target Discord account is not registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not find target account")
		return
	}
	_, err = h.db.ExecContext(r.Context(), `
INSERT INTO user_roles (user_id, role_id, granted_by)
SELECT $1, id, $2 FROM roles WHERE key = $3
ON CONFLICT (user_id, role_id) DO NOTHING
`, targetID, actorID, req.RoleKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not grant role")
		return
	}
	_, _ = h.db.ExecContext(r.Context(), `
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, new_value)
VALUES ($1, 'user', 'rbac.role.granted', 'user', $2, jsonb_build_object('role', $3))
`, actorID, targetID, req.RoleKey)
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
	_, err := h.db.ExecContext(r.Context(), `
INSERT INTO discord_role_map (guild_id, discord_role_id, role_id)
SELECT $1, $2, id FROM roles WHERE key = $3
ON CONFLICT (guild_id, discord_role_id) DO UPDATE SET role_id = EXCLUDED.role_id
`, h.guildID, req.DiscordRoleID, req.RoleKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not map Discord role")
		return
	}
	_, _ = h.db.ExecContext(r.Context(), `
INSERT INTO audit_log (actor_user_id, actor_kind, action, entity_type, entity_id, new_value)
VALUES ($1, 'user', 'rbac.discord_role.mapped', 'discord_role', $2, jsonb_build_object('role', $3))
`, actorID, req.DiscordRoleID, req.RoleKey)
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
