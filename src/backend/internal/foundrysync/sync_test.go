package foundrysync

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alex-jienexa/foundry-marches/internal/characters"
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/relay"
)

type fakeActorGateway struct {
	actors    []relay.ActorReference
	snapshots map[string][]byte
	fail      map[string]error
}

func (f *fakeActorGateway) ListActors(context.Context, string) ([]relay.ActorReference, error) {
	return f.actors, nil
}
func (f *fakeActorGateway) GetActorSnapshot(_ context.Context, _, uuid string) ([]byte, error) {
	if err := f.fail[uuid]; err != nil {
		return nil, err
	}
	return f.snapshots[uuid], nil
}

type fakeActorSink struct {
	mu          sync.Mutex
	disposition map[string]characters.FoundryDiscoveryDisposition
	fail        map[string]error
	seen        []string
}

func (f *fakeActorSink) SyncDiscoveredActor(_ context.Context, _, uuid string, _ json.RawMessage) (characters.FoundryDiscoveryDisposition, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = append(f.seen, uuid)
	if err := f.fail[uuid]; err != nil {
		return "", err
	}
	return f.disposition[uuid], nil
}

func TestSyncAllProcessesEveryActorAndKeepsGoingAfterFailure(t *testing.T) {
	gateway := &fakeActorGateway{
		actors: []relay.ActorReference{
			{UUID: "Actor.hero", Name: "Hero"},
			{UUID: "Actor.npc", Name: "Guard"},
			{UUID: "Actor.broken", Name: "Broken"},
		},
		snapshots: map[string][]byte{
			"Actor.hero":   []byte(`{"_id":"hero","type":"character"}`),
			"Actor.npc":    []byte(`{"_id":"npc","type":"npc"}`),
			"Actor.broken": []byte(`{"_id":"broken","type":"character"}`),
		},
		fail: map[string]error{"Actor.broken": errors.New("relay timeout")},
	}
	sink := &fakeActorSink{disposition: map[string]characters.FoundryDiscoveryDisposition{
		"Actor.hero": characters.DiscoveryCharacterSynced,
		"Actor.npc":  characters.DiscoverySnapshotOnly,
	}}
	service, err := NewSyncService("world", gateway, sink)
	if err != nil {
		t.Fatal(err)
	}
	report, err := service.SyncAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Discovered != 3 || report.CharactersSynced != 1 || report.SnapshotsOnly != 1 || report.Failed != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.Results) != 3 || report.Results[1].ActorType != "npc" || report.Results[2].Error == "" {
		t.Fatalf("per-Actor result details missing: %+v", report.Results)
	}
	if len(sink.seen) != 2 {
		t.Fatalf("sink saw %d actors, want successful fetches only (2)", len(sink.seen))
	}
}

func TestSyncAllRejectsConcurrentRun(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	gateway := blockingGateway{started: started, release: release}
	sink := &fakeActorSink{disposition: map[string]characters.FoundryDiscoveryDisposition{
		"Actor.one": characters.DiscoverySnapshotOnly,
	}}
	service, err := NewSyncService("world", gateway, sink)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = service.SyncAll(context.Background())
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first sync did not start")
	}
	if _, err := service.SyncAll(context.Background()); !errors.Is(err, ErrSyncInProgress) {
		t.Fatalf("got %v, want ErrSyncInProgress", err)
	}
	close(release)
	<-done
}

type blockingGateway struct {
	started chan struct{}
	release chan struct{}
}

func (g blockingGateway) ListActors(context.Context, string) ([]relay.ActorReference, error) {
	close(g.started)
	<-g.release
	return []relay.ActorReference{{UUID: "Actor.one", Name: "One"}}, nil
}
func (blockingGateway) GetActorSnapshot(context.Context, string, string) ([]byte, error) {
	return []byte(`{"_id":"one","type":"npc"}`), nil
}

func TestNewSyncServiceRequiresDependencies(t *testing.T) {
	if _, err := NewSyncService("world", nil, &fakeActorSink{}); err == nil {
		t.Fatal("expected missing gateway to fail")
	}
	if _, err := NewSyncService("", &fakeActorGateway{}, &fakeActorSink{}); err == nil {
		t.Fatal("expected missing world ID to fail")
	}
}

func TestSyncAllPropagatesDiscoveryFailure(t *testing.T) {
	gateway := discoveryFailureGateway{}
	service, err := NewSyncService("world", gateway, &fakeActorSink{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SyncAll(context.Background()); err == nil || !strings.Contains(err.Error(), "discovery failed") {
		t.Fatalf("unexpected discovery error: %v", err)
	}
}

type discoveryFailureGateway struct{}

func (discoveryFailureGateway) ListActors(context.Context, string) ([]relay.ActorReference, error) {
	return nil, errors.New("discovery failed")
}
func (discoveryFailureGateway) GetActorSnapshot(context.Context, string, string) ([]byte, error) {
	return nil, nil
}
