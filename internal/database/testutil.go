package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// NewTestDB creates an in-memory SQLite database for testing.
// The database is automatically closed when the test completes.
func NewTestDB(t *testing.T) *DB {
	t.Helper()

	// Use in-memory database with shared cache for test isolation
	db, err := New(":memory:")
	require.NoError(t, err, "failed to create test database")

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// TimePtr returns a pointer to the given time.Time value
func TimePtr(t interface{ UTC() interface{} }) *interface{} {
	return nil // This is a placeholder - we'll use inline pointers
}

// StringPtr returns a pointer to the given string
func StringPtr(s string) *string {
	return &s
}

// Int64Ptr returns a pointer to the given int64
func Int64Ptr(i int64) *int64 {
	return &i
}
