package database

import (
	"fmt"
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

// TestUser represents a user created for testing
type TestUser struct {
	ID       int64
	GoogleID string
	Email    string
	Name     string
}

// CreateTestUser creates a test user in the database for testing purposes.
// Each call creates a unique user with an auto-generated email and Google ID.
var testUserCounter int64 = 0

func CreateTestUser(t *testing.T, db *DB) *TestUser {
	t.Helper()
	testUserCounter++

	googleID := fmt.Sprintf("test-google-id-%d", testUserCounter)
	email := fmt.Sprintf("testuser%d@example.com", testUserCounter)
	name := fmt.Sprintf("Test User %d", testUserCounter)

	result, err := db.Exec(`
		INSERT INTO users (google_id, email, name)
		VALUES (?, ?, ?)
	`, googleID, email, name)
	require.NoError(t, err, "failed to create test user")

	id, err := result.LastInsertId()
	require.NoError(t, err, "failed to get test user ID")

	return &TestUser{
		ID:       id,
		GoogleID: googleID,
		Email:    email,
		Name:     name,
	}
}

// CreateTestUserWithEmail creates a test user with a specific email
func CreateTestUserWithEmail(t *testing.T, db *DB, email string) *TestUser {
	t.Helper()
	testUserCounter++

	googleID := fmt.Sprintf("test-google-id-%d", testUserCounter)
	name := fmt.Sprintf("Test User %d", testUserCounter)

	result, err := db.Exec(`
		INSERT INTO users (google_id, email, name)
		VALUES (?, ?, ?)
	`, googleID, email, name)
	require.NoError(t, err, "failed to create test user with email")

	id, err := result.LastInsertId()
	require.NoError(t, err, "failed to get test user ID")

	return &TestUser{
		ID:       id,
		GoogleID: googleID,
		Email:    email,
		Name:     name,
	}
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
