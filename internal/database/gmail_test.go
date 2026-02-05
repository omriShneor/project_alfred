package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceTopContactsAllowsSameEmailAcrossUsers(t *testing.T) {
	db := NewTestDB(t)

	user1 := CreateTestUser(t, db)
	user2 := CreateTestUserWithEmail(t, db, "user2@example.com")

	err := db.ReplaceTopContacts(user1.ID, []TopContact{
		{Email: "shared@example.com", Name: "Shared", EmailCount: 3},
	})
	require.NoError(t, err)

	err = db.ReplaceTopContacts(user2.ID, []TopContact{
		{Email: "shared@example.com", Name: "Shared", EmailCount: 2},
	})
	require.NoError(t, err)
}
