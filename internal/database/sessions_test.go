package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhatsAppSessionStorage(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	t.Run("get non-existent session returns nil", func(t *testing.T) {
		session, err := db.GetWhatsAppSession(user.ID)
		require.NoError(t, err)
		assert.Nil(t, session)
	})

	t.Run("save and retrieve session", func(t *testing.T) {
		err := db.SaveWhatsAppSession(user.ID, "+1234567890", "12345@s.whatsapp.net", true)
		require.NoError(t, err)

		session, err := db.GetWhatsAppSession(user.ID)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.Equal(t, user.ID, session.UserID)
		assert.Equal(t, "+1234567890", session.PhoneNumber)
		assert.Equal(t, "12345@s.whatsapp.net", session.DeviceJID)
		assert.True(t, session.Connected)
		assert.NotNil(t, session.ConnectedAt)
	})

	t.Run("update connection status", func(t *testing.T) {
		err := db.UpdateWhatsAppConnected(user.ID, false)
		require.NoError(t, err)

		session, err := db.GetWhatsAppSession(user.ID)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.False(t, session.Connected)
	})

	t.Run("update device JID", func(t *testing.T) {
		err := db.UpdateWhatsAppDeviceJID(user.ID, "67890@s.whatsapp.net")
		require.NoError(t, err)

		session, err := db.GetWhatsAppSession(user.ID)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.Equal(t, "67890@s.whatsapp.net", session.DeviceJID)
	})

	t.Run("list users with sessions", func(t *testing.T) {
		// First reconnect
		err := db.UpdateWhatsAppConnected(user.ID, true)
		require.NoError(t, err)

		users, err := db.ListUsersWithWhatsAppSession()
		require.NoError(t, err)
		assert.Contains(t, users, user.ID)
	})

	t.Run("delete session", func(t *testing.T) {
		err := db.DeleteWhatsAppSession(user.ID)
		require.NoError(t, err)

		session, err := db.GetWhatsAppSession(user.ID)
		require.NoError(t, err)
		assert.Nil(t, session)
	})
}

func TestTelegramSessionStorage(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	t.Run("get non-existent session returns nil", func(t *testing.T) {
		session, err := db.GetTelegramSession(user.ID)
		require.NoError(t, err)
		assert.Nil(t, session)
	})

	t.Run("save and retrieve session", func(t *testing.T) {
		err := db.SaveTelegramSession(user.ID, "+1234567890", true)
		require.NoError(t, err)

		session, err := db.GetTelegramSession(user.ID)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.Equal(t, user.ID, session.UserID)
		assert.Equal(t, "+1234567890", session.PhoneNumber)
		assert.True(t, session.Connected)
		assert.NotNil(t, session.ConnectedAt)
	})

	t.Run("update connection status", func(t *testing.T) {
		err := db.UpdateTelegramConnected(user.ID, false)
		require.NoError(t, err)

		session, err := db.GetTelegramSession(user.ID)
		require.NoError(t, err)
		require.NotNil(t, session)

		assert.False(t, session.Connected)
	})

	t.Run("list users with sessions", func(t *testing.T) {
		// First reconnect
		err := db.UpdateTelegramConnected(user.ID, true)
		require.NoError(t, err)

		users, err := db.ListUsersWithTelegramSession()
		require.NoError(t, err)
		assert.Contains(t, users, user.ID)
	})

	t.Run("delete session", func(t *testing.T) {
		err := db.DeleteTelegramSession(user.ID)
		require.NoError(t, err)

		session, err := db.GetTelegramSession(user.ID)
		require.NoError(t, err)
		assert.Nil(t, session)
	})
}

func TestSessionIsolation(t *testing.T) {
	db := NewTestDB(t)

	// Create two users
	user1 := CreateTestUser(t, db)
	user2 := CreateTestUserWithEmail(t, db, "user2@example.com")

	// Create sessions for both users
	err := db.SaveWhatsAppSession(user1.ID, "+1111111111", "user1@wa.net", true)
	require.NoError(t, err)

	err = db.SaveWhatsAppSession(user2.ID, "+2222222222", "user2@wa.net", true)
	require.NoError(t, err)

	// Verify isolation
	session1, err := db.GetWhatsAppSession(user1.ID)
	require.NoError(t, err)
	assert.Equal(t, "+1111111111", session1.PhoneNumber)

	session2, err := db.GetWhatsAppSession(user2.ID)
	require.NoError(t, err)
	assert.Equal(t, "+2222222222", session2.PhoneNumber)

	// Delete user1's session shouldn't affect user2
	err = db.DeleteWhatsAppSession(user1.ID)
	require.NoError(t, err)

	session1, err = db.GetWhatsAppSession(user1.ID)
	require.NoError(t, err)
	assert.Nil(t, session1)

	session2, err = db.GetWhatsAppSession(user2.ID)
	require.NoError(t, err)
	assert.NotNil(t, session2)
}
