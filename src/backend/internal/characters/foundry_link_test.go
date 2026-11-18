package characters

import "testing"

type testSQLStateError string

func (e testSQLStateError) Error() string    { return string(e) }
func (e testSQLStateError) SQLState() string { return string(e) }

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(testSQLStateError("23505")) {
		t.Fatal("PostgreSQL unique violation must be recognized")
	}
	if isUniqueViolation(testSQLStateError("40001")) {
		t.Fatal("serialization failure must not be treated as a unique violation")
	}
}
