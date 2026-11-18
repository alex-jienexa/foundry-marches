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

func TestGetActorSnapshotUnwrapsAndValidatesRelayEnvelope(t *testing.T) {
	const actorUUID = "Actor.actor-123"
	const actorJSON = `{"_id":"actor-123","name":"Test Hero","type":"character","items":[]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get" {
			t.Errorf("path = %q, want /get", r.URL.Path)
		}
		if got := r.URL.Query().Get("uuid"); got != actorUUID {
			t.Errorf("uuid = %q, want %q", got, actorUUID)
		}
		if got := r.URL.Query().Get("clientId"); got != "relay-client" {
			t.Errorf("clientId = %q, want relay-client", got)
		}
		if got := r.Header.Get("x-api-key"); got != "test-secret" {
			t.Errorf("x-api-key = %q, want configured key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"type":"entity-result","data":%s}`, actorJSON)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		WorldID: "world", BaseURL: server.URL, ClientID: "relay-client", APIKey: "test-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.GetActorSnapshot(context.Background(), "application-world-id", actorUUID)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != actorJSON {
		t.Fatalf("snapshot = %s, want %s", got, actorJSON)
	}
}

func TestResolveCompendiumItemReturnsItemDocument(t *testing.T) {
	const itemUUID = "Compendium.pf2e.feats-srd.Item.item-123"
	const itemJSON = `{"_id":"item-123","name":"Power Attack","type":"feat","system":{}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("uuid"); got != itemUUID {
			t.Errorf("uuid = %q, want %q", got, itemUUID)
		}
		fmt.Fprintf(w, `{"type":"entity-result","data":%s}`, itemJSON)
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, ClientID: "client", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.ResolveCompendiumItem(context.Background(), itemUUID)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != itemJSON {
		t.Fatalf("Item = %s, want %s", got, itemJSON)
	}
}

func TestGetDocumentEncodesUUIDAndRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("uuid"); got != "Actor.with spaces/and?symbols" {
			t.Errorf("decoded uuid = %q", got)
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, ClientID: "client", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getDocument(context.Background(), "Actor.with spaces/and?symbols")
	if !errors.Is(err, ErrUnexpectedStatus) {
		t.Fatalf("got %v, want ErrUnexpectedStatus", err)
	}
}

func TestGetDocumentRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 33)))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL: server.URL, ClientID: "client", APIKey: "secret", MaxResponseBytes: 32,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getDocument(context.Background(), "Actor.test")
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("got %v, want ErrResponseTooLarge", err)
	}
}

func TestGetActorSnapshotRejectsDifferentActor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"type":"entity-result","data":{"_id":"other","name":"Other","type":"character","items":[]}}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, ClientID: "client", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetActorSnapshot(context.Background(), "world", "Actor.requested")
	if !errors.Is(err, ErrDocumentMismatch) {
		t.Fatalf("got %v, want ErrDocumentMismatch", err)
	}
}


func TestGetActorSnapshotRejectsDifferentConfiguredWorld(t *testing.T) {
	client, err := NewClient(Config{
		WorldID: "configured-world", BaseURL: "https://relay.example.test",
		ClientID: "client", APIKey: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetActorSnapshot(context.Background(), "another-world", "Actor.test")
	if !errors.Is(err, ErrWorldMismatch) {
		t.Fatalf("got %v, want ErrWorldMismatch", err)
	}
}

func TestNewClientRejectsUnsafeOrIncompleteConfiguration(t *testing.T) {
	tests := []Config{
		{WorldID: "world", BaseURL: "https://relay.example.test", ClientID: "client"},
		{WorldID: "world", BaseURL: "ftp://relay.example.test", ClientID: "client", APIKey: "secret"},
		{WorldID: "world", BaseURL: "https://user:pass@relay.example.test", ClientID: "client", APIKey: "secret"},
		{WorldID: "world", BaseURL: "https://relay.example.test?x=1", ClientID: "client", APIKey: "secret"},
	}
	for i, config := range tests {
		if _, err := NewClient(config); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("case %d: got %v, want ErrInvalidConfig", i, err)
		}
	}
}

func TestGetDocumentUsesConfiguredBasePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/relay/get" {
			t.Errorf("path = %q, want /relay/get", r.URL.Path)
		}
		fmt.Fprint(w, `{"type":"entity-result","data":{"_id":"x","name":"X","type":"character","items":[]}}`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		WorldID: "world", BaseURL: server.URL + "/relay/", ClientID: "client", APIKey: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetActorSnapshot(context.Background(), "world", "Actor.x"); err != nil {
		t.Fatal(err)
	}
}

func TestClientDoesNotFollowRedirectsWithAPIKey(t *testing.T) {
	var targetRequests int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetRequests++
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/get", http.StatusFound)
	}))
	defer redirect.Close()

	client, err := NewClient(Config{WorldID: "world", BaseURL: redirect.URL, ClientID: "client", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.getDocument(context.Background(), "Actor.test")
	if !errors.Is(err, ErrUnexpectedStatus) {
		t.Fatalf("got %v, want ErrUnexpectedStatus for redirect response", err)
	}
	if targetRequests != 0 {
		t.Fatalf("redirect target received %d requests; want none", targetRequests)
	}
}
