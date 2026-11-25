package discordapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeCharacterRequestsPreservesDiscordOwnerMention(t *testing.T) {
	t.Run("create character", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, CharacterCreatePath, strings.NewReader(`{"guild_id":"guild-1","discord_user_id":"gm-1","discord_role_ids":["gm-role"],"name":"Андрей","owner_discord_user_id":"player-1"}`))
		rec := httptest.NewRecorder()
		var decoded createCharacterRequest
		if !decodeRequest(rec, req, &decoded) {
			t.Fatalf("decodeRequest rejected create payload: %s", rec.Body.String())
		}
		if decoded.Name != "Андрей" || decoded.OwnerDiscordUserID != "player-1" {
			t.Fatalf("decoded create request = %+v", decoded)
		}
	})
	t.Run("link character", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, LinkPath, strings.NewReader(`{"guild_id":"guild-1","discord_user_id":"gm-1","actor_uuid":"Actor.test","character_id":"character-1","owner_discord_user_id":"player-1"}`))
		rec := httptest.NewRecorder()
		var decoded linkRequest
		if !decodeRequest(rec, req, &decoded) {
			t.Fatalf("decodeRequest rejected link payload: %s", rec.Body.String())
		}
		if decoded.OwnerDiscordUserID != "player-1" {
			t.Fatalf("owner Discord ID = %q, want player-1", decoded.OwnerDiscordUserID)
		}
	})
}

func TestServiceTokenMiddleware(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := ServiceTokenMiddleware("a-very-long-test-service-token-value-123456", next)

	t.Run("rejects missing token", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodPost, RegisterPath, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized || called {
			t.Fatalf("status=%d called=%v, want 401 and next not called", rec.Code, called)
		}
	})

	t.Run("rejects wrong token", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodPost, RegisterPath, nil)
		req.Header.Set("Authorization", "Bearer wrong-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized || called {
			t.Fatalf("status=%d called=%v, want 401 and next not called", rec.Code, called)
		}
	})

	t.Run("accepts exact token", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodPost, RegisterPath, nil)
		req.Header.Set("Authorization", "Bearer a-very-long-test-service-token-value-123456")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent || !called {
			t.Fatalf("status=%d called=%v, want 204 and next called", rec.Code, called)
		}
	})
}

func TestDecodeRequestRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	type request struct {
		Name string `json:"name"`
	}

	t.Run("unknown field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x","admin":true}`))
		rec := httptest.NewRecorder()
		var decoded request
		if decodeRequest(rec, req, &decoded) {
			t.Fatal("decodeRequest accepted an unknown field")
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400", rec.Code)
		}
	})

	t.Run("trailing JSON value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"x"} {"name":"y"}`))
		rec := httptest.NewRecorder()
		var decoded request
		if decodeRequest(rec, req, &decoded) {
			t.Fatal("decodeRequest accepted multiple JSON values")
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want 400", rec.Code)
		}
	})
}

func TestDecodeRequestAcceptsEmbeddedIdentityFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"guild_id":"guild-1","discord_user_id":"user-1","discord_role_ids":["role-1"],"username":"player"}`))
	rec := httptest.NewRecorder()
	var decoded registerRequest
	if !decodeRequest(rec, req, &decoded) {
		t.Fatalf("decodeRequest rejected valid embedded identity fields: %s", rec.Body.String())
	}
	if decoded.GuildID != "guild-1" || decoded.DiscordUserID != "user-1" ||
		len(decoded.DiscordRoleIDs) != 1 || decoded.DiscordRoleIDs[0] != "role-1" ||
		decoded.Username != "player" {
		t.Fatalf("decoded request = %+v, identity fields were not preserved", decoded)
	}
}
