package database

import (
	"errors"
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

func TestEmailSourcesUserScopedMutations(t *testing.T) {
	db := NewTestDB(t)

	owner := CreateTestUser(t, db)
	otherUser := CreateTestUserWithEmail(t, db, "other-user@example.com")

	source, err := db.CreateEmailSource(owner.ID, EmailSourceTypeSender, "owner@example.com", "Owner Source")
	require.NoError(t, err)

	t.Run("get by id is user scoped", func(t *testing.T) {
		got, err := db.GetEmailSourceByIDForUser(otherUser.ID, source.ID)
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("update by id is user scoped", func(t *testing.T) {
		err := db.UpdateEmailSourceForUser(otherUser.ID, source.ID, "Hacked", false)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrEmailSourceNotFound))

		unchanged, err := db.GetEmailSourceByID(source.ID)
		require.NoError(t, err)
		require.NotNil(t, unchanged)
		require.Equal(t, "Owner Source", unchanged.Name)
		require.True(t, unchanged.Enabled)
	})

	t.Run("delete by id is user scoped", func(t *testing.T) {
		err := db.DeleteEmailSourceForUser(otherUser.ID, source.ID)
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrEmailSourceNotFound))

		stillThere, err := db.GetEmailSourceByID(source.ID)
		require.NoError(t, err)
		require.NotNil(t, stillThere)
	})
}
