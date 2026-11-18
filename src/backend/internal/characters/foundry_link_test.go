package characters

import (
	"context"
	"testing"
)

type linkTestDB struct{}

func (linkTestDB) BeginTx(context.Context, interfaceTxOptions) (*sql.Tx, error) {
	panic("BeginTx must not be called for invalid input")
}
func (linkTestDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	panic("ExecContext must not be called for invalid input")
}
func (linkTestDB) QueryRowContext(context.Context, string, ...any) *sql.Row {
	panic("QueryRowContext must not be called for invalid input")
}

type interfaceTxOptions = sql.TxOptions

type testSQLStateError string

func (e testSQLStateError) Error() string  { return string(e) }
func (e testSQLStateError) SQLState() string { return string(e) }

func TestLinkActorRequiresExplicitIdentifiers(t *testing.T) {
	repo := &FoundryRepository{}
	if err := repo.LinkActor(context.Background(), "world", "Actor.id", "character", "user"); err == nil {
		t.Fatal("nil repository must be rejected")
	}

	db := &sql.DB{}
	_ = db
}

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(testSQLStateError("23505")) {
		t.Fatal("PostgreSQL unique violation must be recognized")
	}
	if isUniqueViolation(testSQLStateError("40001")) {
		t.Fatal("serialization failure must not be treated as a unique violation")
	}
}
