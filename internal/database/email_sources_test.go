package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmailSourcesAllowSameIdentifierAcrossUsers(t *testing.T) {
	db := NewTestDB(t)

	user1 := CreateTestUser(t, db)
	user2 := CreateTestUserWithEmail(t, db, "user2@example.com")

	_, err := db.CreateEmailSource(user1.ID, EmailSourceTypeSender, "shared@example.com", "Shared")
	require.NoError(t, err)

	_, err = db.CreateEmailSource(user2.ID, EmailSourceTypeSender, "shared@example.com", "Shared")
	require.NoError(t, err)
}
