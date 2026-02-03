package database

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestGoogleTokenStorage(t *testing.T) {
	// Set encryption key for tests
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-key-for-google-tokens")
	defer os.Unsetenv("ALFRED_ENCRYPTION_KEY")

	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	t.Run("get non-existent token returns nil", func(t *testing.T) {
		token, err := db.GetGoogleToken(user.ID)
		require.NoError(t, err)
		assert.Nil(t, token)
	})

	t.Run("save and retrieve token", func(t *testing.T) {
		testToken := &oauth2.Token{
			AccessToken:  "access-token-12345",
			RefreshToken: "refresh-token-67890",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour),
		}

		err := db.SaveGoogleToken(user.ID, testToken, "test@example.com")
		require.NoError(t, err)

		retrieved, err := db.GetGoogleToken(user.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, testToken.AccessToken, retrieved.AccessToken)
		assert.Equal(t, testToken.RefreshToken, retrieved.RefreshToken)
		assert.Equal(t, testToken.TokenType, retrieved.TokenType)
	})

	t.Run("update existing token", func(t *testing.T) {
		newToken := &oauth2.Token{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(2 * time.Hour),
		}

		err := db.UpdateGoogleToken(user.ID, newToken)
		require.NoError(t, err)

		retrieved, err := db.GetGoogleToken(user.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, newToken.AccessToken, retrieved.AccessToken)
		assert.Equal(t, newToken.RefreshToken, retrieved.RefreshToken)
	})

	t.Run("get token info", func(t *testing.T) {
		info, err := db.GetGoogleTokenInfo(user.ID)
		require.NoError(t, err)
		require.NotNil(t, info)

		assert.Equal(t, user.ID, info.UserID)
		assert.True(t, info.HasToken)
	})

	t.Run("list users with tokens", func(t *testing.T) {
		users, err := db.ListUsersWithGoogleToken()
		require.NoError(t, err)
		assert.Contains(t, users, user.ID)
	})

	t.Run("delete token", func(t *testing.T) {
		err := db.DeleteGoogleToken(user.ID)
		require.NoError(t, err)

		token, err := db.GetGoogleToken(user.ID)
		require.NoError(t, err)
		assert.Nil(t, token)
	})
}

func TestGoogleTokenInfoNoToken(t *testing.T) {
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-key")
	defer os.Unsetenv("ALFRED_ENCRYPTION_KEY")

	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	info, err := db.GetGoogleTokenInfo(user.ID)
	require.NoError(t, err)
	require.NotNil(t, info)

	assert.Equal(t, user.ID, info.UserID)
	assert.False(t, info.HasToken)
}
