package relay

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListActorsSearchesWorldAndReturnsSortedActorReferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/relay/search" {
			t.Errorf("path = %q, want /relay/search", r.URL.Path)
		}
		for key, want := range map[string]string{
			"clientId": "relay-client", "filter": "Actor",
			"excludeCompendiums": "true", "minified": "true", "limit": "500",
		} {
			if got := r.URL.Query().Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		if got := r.Header.Get("x-api-key"); got != "test-secret" {
			t.Errorf("x-api-key = %q, want configured key", got)
		}
		fmt.Fprint(w, `{"type":"search-result","data":[
			{"uuid":"Actor.zeta","id":"zeta","name":"Zeta","documentType":"Actor"},
			{"uuid":"Actor.alpha","id":"alpha","name":"Alpha","documentType":"Actor"},
			{"uuid":"Actor.alpha","id":"alpha","name":"Alpha","documentType":"Actor"}
		]}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		WorldID: "world", BaseURL: server.URL + "/relay", ClientID: "relay-client", APIKey: "test-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.ListActors(context.Background(), "world")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].UUID != "Actor.alpha" || got[1].UUID != "Actor.zeta" {
		t.Fatalf("unexpected Actor references: %+v", got)
	}
}

func TestListActorsAcceptsResultObjectAndBuildsUUIDFromID(t *testing.T) {
	got, err := decodeActorSearchResponse([]byte(`{
		"data":{"results":[
			{"id":"npc-1","name":"Guard"},
			{"uuid":"Actor.hero","name":"Hero"}
		]}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].UUID != "Actor.hero" || got[1].UUID != "Actor.npc-1" {
		t.Fatalf("unexpected references: %+v", got)
	}
}

func TestListActorsRejectsIncompleteSearchAtRelayLimit(t *testing.T) {
	rows := make([]string, maxActorSearchResults)
	for i := range rows {
		rows[i] = fmt.Sprintf(`{"uuid":"Actor.actor-%03d","name":"Actor %03d"}`, i, i)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"type":"search-result","data":[%s]}`, strings.Join(rows, ","))
	}))
	defer server.Close()
	client, err := NewClient(Config{WorldID: "world", BaseURL: server.URL, ClientID: "client", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	actors, err := client.ListActors(context.Background(), "world")
	if !errors.Is(err, ErrActorListLimitReached) || len(actors) != maxActorSearchResults {
		t.Fatalf("got %d actors, err %v; want capped list and ErrActorListLimitReached", len(actors), err)
	}
}

func TestListActorsRejectsDifferentWorld(t *testing.T) {
	client, err := NewClient(Config{WorldID: "configured", BaseURL: "https://relay.example.test", ClientID: "client", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListActors(context.Background(), "other"); !errors.Is(err, ErrWorldMismatch) {
		t.Fatalf("got %v, want ErrWorldMismatch", err)
	}
}

func TestListActorsRetriesTransientServerFailure(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "temporarily unavailable", http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, `{"type":"search-result","data":[{"uuid":"Actor.hero","name":"Hero"}]}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{WorldID: "world", BaseURL: server.URL, ClientID: "client", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	actors, err := client.ListActors(context.Background(), "world")
	if err != nil {
		t.Fatal(err)
	}
	if len(actors) != 1 || actors[0].UUID != "Actor.hero" {
		t.Fatalf("unexpected Actor search result: %+v", actors)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}
