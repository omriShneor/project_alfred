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

// TestSplitScopes tests the critical scope parsing function
// This function was the source of the OAuth bug where scopes were stored
// as JSON but parsed as space-separated strings
func TestSplitScopes(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		result := splitScopes("")
		assert.Nil(t, result)
	})

	t.Run("JSON array format parses correctly", func(t *testing.T) {
		input := `["scope1","scope2","scope3"]`
		result := splitScopes(input)

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "scope1", result[0])
		assert.Equal(t, "scope2", result[1])
		assert.Equal(t, "scope3", result[2])
	})

	t.Run("JSON array with URL scopes", func(t *testing.T) {
		input := `["https://www.googleapis.com/auth/userinfo.email","https://www.googleapis.com/auth/userinfo.profile","https://www.googleapis.com/auth/calendar"]`
		result := splitScopes(input)

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "https://www.googleapis.com/auth/userinfo.email", result[0])
		assert.Equal(t, "https://www.googleapis.com/auth/userinfo.profile", result[1])
		assert.Equal(t, "https://www.googleapis.com/auth/calendar", result[2])
	})

	t.Run("single scope in JSON array", func(t *testing.T) {
		input := `["single-scope"]`
		result := splitScopes(input)

		assert.Equal(t, 1, len(result))
		assert.Equal(t, "single-scope", result[0])
	})

	t.Run("empty JSON array returns empty slice", func(t *testing.T) {
		input := `[]`
		result := splitScopes(input)

		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})

	t.Run("space-separated fallback for backward compatibility", func(t *testing.T) {
		input := "scope1 scope2 scope3"
		result := splitScopes(input)

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "scope1", result[0])
		assert.Equal(t, "scope2", result[1])
		assert.Equal(t, "scope3", result[2])
	})

	t.Run("single scope space-separated", func(t *testing.T) {
		input := "single-scope"
		result := splitScopes(input)

		assert.Equal(t, 1, len(result))
		assert.Equal(t, "single-scope", result[0])
	})

	t.Run("invalid JSON falls back to space-separated", func(t *testing.T) {
		input := `["invalid json`  // Malformed JSON
		result := splitScopes(input)

		// Should parse as space-separated (splits on space)
		assert.Equal(t, 2, len(result))
		assert.Equal(t, `["invalid`, result[0])
		assert.Equal(t, `json`, result[1])
	})

	t.Run("JSON with extra whitespace", func(t *testing.T) {
		input := `["scope1", "scope2", "scope3"]`  // Spaces after commas
		result := splitScopes(input)

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "scope1", result[0])
		assert.Equal(t, "scope2", result[1])
		assert.Equal(t, "scope3", result[2])
	})

	t.Run("space-separated with multiple spaces", func(t *testing.T) {
		input := "scope1    scope2  scope3"
		result := splitScopes(input)

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "scope1", result[0])
		assert.Equal(t, "scope2", result[1])
		assert.Equal(t, "scope3", result[2])
	})
}
