package foundrysync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alex-jienexa/foundry-marches/internal/characters"
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/relay"
)

var ErrSyncInProgress = errors.New("a Foundry world sync is already running")

type ActorGateway interface {
	ListActors(context.Context, string) ([]relay.ActorReference, error)
	GetActorSnapshot(context.Context, string, string) ([]byte, error)
}

type ActorSnapshotSink interface {
	SyncDiscoveredActor(context.Context, string, string, json.RawMessage) (characters.FoundryDiscoveryDisposition, error)
}

type ActorSyncResult struct {
	UUID        string                                  `json:"uuid"`
	Name        string                                  `json:"name"`
	ActorType   string                                  `json:"actorType,omitempty"`
	Disposition characters.FoundryDiscoveryDisposition `json:"disposition,omitempty"`
	Error       string                                  `json:"error,omitempty"`
}

type SyncReport struct {
	WorldID           string            `json:"worldId"`
	StartedAt         time.Time         `json:"startedAt"`
	FinishedAt        time.Time         `json:"finishedAt"`
	Discovered        int               `json:"discovered"`
	CharactersSynced  int               `json:"charactersSynced"`
	SnapshotsOnly     int               `json:"snapshotsOnly"`
	Failed            int               `json:"failed"`
	Results           []ActorSyncResult `json:"results"`
}

type SyncService struct {
	worldID string
	gateway ActorGateway
	sink    ActorSnapshotSink
	mu      sync.Mutex
}

func NewSyncService(worldID string, gateway ActorGateway, sink ActorSnapshotSink) (*SyncService, error) {
	if worldID == "" || gateway == nil || sink == nil {
		return nil, errors.New("world ID, Foundry gateway and Actor snapshot sink are required")
	}
	return &SyncService{worldID: worldID, gateway: gateway, sink: sink}, nil
}

// SyncAll discovers every world Actor and refreshes its snapshot. Individual
// Actor failures are recorded without aborting the rest of the world batch.
func (s *SyncService) SyncAll(ctx context.Context) (SyncReport, error) {
	if s == nil || s.gateway == nil || s.sink == nil {
		return SyncReport{}, errors.New("Foundry sync service is not initialized")
	}
	if !s.mu.TryLock() {
		return SyncReport{}, ErrSyncInProgress
	}
	defer s.mu.Unlock()

	report := SyncReport{WorldID: s.worldID, StartedAt: time.Now().UTC()}
	defer func() { report.FinishedAt = time.Now().UTC() }()

	actors, err := s.gateway.ListActors(ctx, s.worldID)
	if err != nil {
		return report, fmt.Errorf("discover Foundry world Actors: %w", err)
	}
	report.Discovered = len(actors)
	report.Results = make([]ActorSyncResult, 0, len(actors))

	for _, actor := range actors {
		result := ActorSyncResult{UUID: actor.UUID, Name: actor.Name}
		if err := ctx.Err(); err != nil {
			result.Error = err.Error()
			report.Failed++
			report.Results = append(report.Results, result)
			continue
		}
		snapshot, fetchErr := s.gateway.GetActorSnapshot(ctx, s.worldID, actor.UUID)
		if fetchErr != nil {
			result.Error = fetchErr.Error()
			report.Failed++
			report.Results = append(report.Results, result)
			continue
		}
		disposition, syncErr := s.sink.SyncDiscoveredActor(ctx, s.worldID, actor.UUID, snapshot)
		if syncErr != nil {
			result.Error = syncErr.Error()
			report.Failed++
			report.Results = append(report.Results, result)
			continue
		}
		result.Disposition = disposition
		switch disposition {
		case characters.DiscoveryCharacterSynced:
			report.CharactersSynced++
		case characters.DiscoverySnapshotOnly:
			report.SnapshotsOnly++
		default:
			result.Error = fmt.Sprintf("repository returned unknown sync disposition %q", disposition)
			report.Failed++
		}
		report.Results = append(report.Results, result)
	}
	report.FinishedAt = time.Now().UTC()
	return report, nil
}
