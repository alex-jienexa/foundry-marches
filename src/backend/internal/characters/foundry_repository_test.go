package characters

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
)

type recordingDBState struct {
	mu sync.Mutex
	queries []string
	actorMissing bool
	committed bool
	rolledBack bool
}

type recordingConnector struct { state *recordingDBState }
func (c recordingConnector) Connect(context.Context) (driver.Conn, error) { return &recordingConn{state: c.state}, nil }
func (c recordingConnector) Driver() driver.Driver { return recordingDriver{} }

type recordingDriver struct{}
func (recordingDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use Connector") }

type recordingConn struct { state *recordingDBState }
func (c *recordingConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("Prepare is not supported") }
func (c *recordingConn) Close() error { return nil }
func (c *recordingConn) Begin() (driver.Tx, error) { return &recordingTx{state: c.state}, nil }
func (c *recordingConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &recordingTx{state: c.state}, nil
}
func (c *recordingConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.queries = append(c.state.queries, query)
	if strings.Contains(query, "UPDATE foundry_actors") && c.state.actorMissing {
		return driver.RowsAffected(0), nil
	}
	return driver.RowsAffected(1), nil
}

type recordingTx struct { state *recordingDBState }
func (t *recordingTx) Commit() error {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	t.state.committed = true
	return nil
}
func (t *recordingTx) Rollback() error {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	t.state.rolledBack = true
	return nil
}

func testSyncRecord(t *testing.T) FoundrySyncRecord {
	t.Helper()
	decoded := pf2e.DecodedActor{
		Character: pf2e.CharacterDto{
			Identity: pf2e.CharacterIdentityDto{ID: "actor-id", UUID: "Actor.actor-id", Name: "Test Hero", Type: "character"},
			Items: []pf2e.CharacterItemDto{{ID: "item-1", Name: "Longsword"}},
		},
		RawSnapshot: []byte(`{"_id":"actor-id","name":"Test Hero"}`),
	}
	record, err := MapFoundryActor(FoundrySyncInput{
		WorldID: "world-id", CharacterID: "character-id", ActorUUID: "Actor.actor-id", Decoded: decoded,
	})
	if err != nil { t.Fatal(err) }
	return record
}

func TestFoundryRepositorySyncActorUsesOneTransaction(t *testing.T) {
	state := &recordingDBState{}
	db := sql.OpenDB(recordingConnector{state: state})
	defer db.Close()
	repo, err := NewFoundryRepository(db)
	if err != nil { t.Fatal(err) }
	if err := repo.SyncActor(context.Background(), testSyncRecord(t)); err != nil { t.Fatal(err) }

	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.committed { t.Fatal("transaction was not committed") }
	joined := strings.Join(state.queries, "\n")
	for _, expected := range []string{"SET LOCAL app.writer = 'foundry_sync'", "INSERT INTO characters", "UPDATE foundry_actors", "DELETE FROM inventory_items", "INSERT INTO inventory_items"} {
		if !strings.Contains(joined, expected) { t.Fatalf("expected SQL %q in transaction queries:\n%s", expected, joined) }
	}
	if strings.Index(joined, "INSERT INTO characters") > strings.Index(joined, "UPDATE foundry_actors") {
		t.Fatal("character row must be written before the Actor snapshot")
	}
}

func TestFoundryRepositoryRejectsUnlinkedActorAndRollsBack(t *testing.T) {
	state := &recordingDBState{actorMissing: true}
	db := sql.OpenDB(recordingConnector{state: state})
	defer db.Close()
	repo, err := NewFoundryRepository(db)
	if err != nil { t.Fatal(err) }
	err = repo.SyncActor(context.Background(), testSyncRecord(t))
	if !errors.Is(err, ErrActorNotLinked) { t.Fatalf("got %v, want ErrActorNotLinked", err) }
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.committed { t.Fatal("transaction with an unlinked Actor must not commit") }
	if !state.rolledBack { t.Fatal("transaction with an unlinked Actor must roll back") }
}
