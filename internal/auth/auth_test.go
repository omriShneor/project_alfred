package auth

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	calendar "google.golang.org/api/calendar/v3"
	gmail "google.golang.org/api/gmail/v1"
)

func TestEncryptor(t *testing.T) {
	// Set a test encryption key
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-encryption-key-for-tests")
	defer os.Unsetenv("ALFRED_ENCRYPTION_KEY")

	encryptor, err := NewEncryptor(nil)
	require.NoError(t, err)
	require.NotNil(t, encryptor)

	t.Run("encrypt and decrypt bytes", func(t *testing.T) {
		plaintext := []byte("Hello, World! This is a test message.")

		ciphertext, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, ciphertext)

		decrypted, err := encryptor.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("encrypt and decrypt string", func(t *testing.T) {
		plaintext := "This is a secret token"

		encrypted, err := encryptor.EncryptString(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, encrypted)

		decrypted, err := encryptor.DecryptString(encrypted)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("decrypt invalid ciphertext fails", func(t *testing.T) {
		_, err := encryptor.Decrypt([]byte("invalid"))
		assert.Error(t, err)
	})

	t.Run("decrypt invalid base64 fails", func(t *testing.T) {
		_, err := encryptor.DecryptString("not-valid-base64!!!")
		assert.Error(t, err)
	})

	t.Run("different encryptions produce different ciphertexts", func(t *testing.T) {
		plaintext := []byte("same message")

		ct1, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)

		ct2, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)

		// Due to random nonce, ciphertexts should be different
		assert.NotEqual(t, ct1, ct2)

		// But both should decrypt to the same plaintext
		dec1, _ := encryptor.Decrypt(ct1)
		dec2, _ := encryptor.Decrypt(ct2)
		assert.Equal(t, dec1, dec2)
	})
}

func TestEncryptorWithCustomKey(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := "test message"
	encrypted, err := encryptor.EncryptString(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key1, 32)

	key2, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key2, 32)

	// Keys should be random, so different
	assert.NotEqual(t, key1, key2)
}

func TestEncryptorWithDerivedKey(t *testing.T) {
	// Test that encryptor can derive key from ANTHROPIC_API_KEY
	os.Unsetenv("ALFRED_ENCRYPTION_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "sk-test-key-12345")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	encryptor, err := NewEncryptor(nil)
	require.NoError(t, err)

	plaintext := "test message"
	encrypted, err := encryptor.EncryptString(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptorNoKeyAvailable(t *testing.T) {
	// Save and clear environment variables
	savedEncKey := os.Getenv("ALFRED_ENCRYPTION_KEY")
	savedAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ALFRED_ENCRYPTION_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")

	defer func() {
		// Restore environment
		if savedEncKey != "" {
			os.Setenv("ALFRED_ENCRYPTION_KEY", savedEncKey)
		}
		if savedAPIKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", savedAPIKey)
		}
	}()

	_, err := NewEncryptor(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no encryption key available")
}

// TestScopeDefinitions verifies that scope constants are properly defined
func TestScopeDefinitions(t *testing.T) {
	t.Run("ProfileScopes contains expected scopes", func(t *testing.T) {
		assert.Len(t, ProfileScopes, 2)
		assert.Contains(t, ProfileScopes, "https://www.googleapis.com/auth/userinfo.email")
		assert.Contains(t, ProfileScopes, "https://www.googleapis.com/auth/userinfo.profile")
	})

	t.Run("GmailScopes contains Gmail readonly scope", func(t *testing.T) {
		assert.Len(t, GmailScopes, 1)
		assert.Contains(t, GmailScopes, gmail.GmailReadonlyScope)
	})

	t.Run("CalendarScopes contains Calendar scope", func(t *testing.T) {
		assert.Len(t, CalendarScopes, 1)
		assert.Contains(t, CalendarScopes, calendar.CalendarScope)
	})

	t.Run("OAuthScopes contains all scopes for backward compatibility", func(t *testing.T) {
		assert.Len(t, OAuthScopes, 4)
		assert.Contains(t, OAuthScopes, gmail.GmailReadonlyScope)
		assert.Contains(t, OAuthScopes, calendar.CalendarScope)
		assert.Contains(t, OAuthScopes, "https://www.googleapis.com/auth/userinfo.email")
		assert.Contains(t, OAuthScopes, "https://www.googleapis.com/auth/userinfo.profile")
	})
}

// TestMergeScopes tests the scope merging helper function
func TestMergeScopes(t *testing.T) {
	t.Run("merges two disjoint scope lists", func(t *testing.T) {
		existing := []string{"scope1", "scope2"}
		newScopes := []string{"scope3", "scope4"}

		merged := mergeScopes(existing, newScopes)

		assert.Len(t, merged, 4)
		assert.Contains(t, merged, "scope1")
		assert.Contains(t, merged, "scope2")
		assert.Contains(t, merged, "scope3")
		assert.Contains(t, merged, "scope4")
	})

	t.Run("deduplicates overlapping scopes", func(t *testing.T) {
		existing := []string{"scope1", "scope2"}
		newScopes := []string{"scope2", "scope3"}

		merged := mergeScopes(existing, newScopes)

		assert.Len(t, merged, 3)
		assert.Contains(t, merged, "scope1")
		assert.Contains(t, merged, "scope2")
		assert.Contains(t, merged, "scope3")
	})

	t.Run("handles empty existing scopes", func(t *testing.T) {
		existing := []string{}
		newScopes := []string{"scope1", "scope2"}

		merged := mergeScopes(existing, newScopes)

		assert.Len(t, merged, 2)
		assert.Contains(t, merged, "scope1")
		assert.Contains(t, merged, "scope2")
	})

	t.Run("handles empty new scopes", func(t *testing.T) {
		existing := []string{"scope1", "scope2"}
		newScopes := []string{}

		merged := mergeScopes(existing, newScopes)

		assert.Len(t, merged, 2)
		assert.Contains(t, merged, "scope1")
		assert.Contains(t, merged, "scope2")
	})

	t.Run("handles both empty", func(t *testing.T) {
		existing := []string{}
		newScopes := []string{}

		merged := mergeScopes(existing, newScopes)

		assert.Len(t, merged, 0)
	})

	t.Run("merges profile and gmail scopes", func(t *testing.T) {
		merged := mergeScopes(ProfileScopes, GmailScopes)

		assert.Len(t, merged, 3)
		assert.Contains(t, merged, "https://www.googleapis.com/auth/userinfo.email")
		assert.Contains(t, merged, "https://www.googleapis.com/auth/userinfo.profile")
		assert.Contains(t, merged, gmail.GmailReadonlyScope)
	})
}

// TestServiceScopeMethods tests scope-related Service methods with a real database
func TestServiceScopeMethods(t *testing.T) {
	// Setup test environment
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-encryption-key-for-scope-tests")
	defer os.Unsetenv("ALFRED_ENCRYPTION_KEY")

	db := database.NewTestDB(t)
	user := database.CreateTestUser(t, db)

	// Create a minimal OAuth config for testing
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       ProfileScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	service, err := NewService(db.DB, config)
	require.NoError(t, err)

	t.Run("GetUserScopes returns nil for user without token", func(t *testing.T) {
		scopes, err := service.GetUserScopes(user.ID)
		require.NoError(t, err)
		assert.Nil(t, scopes)
	})

	t.Run("HasScope returns false for user without token", func(t *testing.T) {
		hasScope, err := service.HasScope(user.ID, gmail.GmailReadonlyScope)
		require.NoError(t, err)
		assert.False(t, hasScope)
	})

	t.Run("HasGmailScope returns false for user without token", func(t *testing.T) {
		hasGmail, err := service.HasGmailScope(user.ID)
		require.NoError(t, err)
		assert.False(t, hasGmail)
	})

	t.Run("HasCalendarScope returns false for user without token", func(t *testing.T) {
		hasCalendar, err := service.HasCalendarScope(user.ID)
		require.NoError(t, err)
		assert.False(t, hasCalendar)
	})

	// Store a token with profile scopes only
	t.Run("storeGoogleTokenWithScopes stores scopes correctly", func(t *testing.T) {
		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			TokenType:    "Bearer",
		}

		err := service.storeGoogleTokenWithScopes(user.ID, token, ProfileScopes)
		require.NoError(t, err)

		// Verify scopes were stored
		scopes, err := service.GetUserScopes(user.ID)
		require.NoError(t, err)
		assert.Len(t, scopes, 2)
		assert.Contains(t, scopes, "https://www.googleapis.com/auth/userinfo.email")
		assert.Contains(t, scopes, "https://www.googleapis.com/auth/userinfo.profile")
	})

	t.Run("HasGmailScope returns false for profile-only scopes", func(t *testing.T) {
		hasGmail, err := service.HasGmailScope(user.ID)
		require.NoError(t, err)
		assert.False(t, hasGmail)
	})

	t.Run("HasCalendarScope returns false for profile-only scopes", func(t *testing.T) {
		hasCalendar, err := service.HasCalendarScope(user.ID)
		require.NoError(t, err)
		assert.False(t, hasCalendar)
	})

	// Update token to include Gmail scope
	t.Run("updating token with Gmail scope works", func(t *testing.T) {
		token := &oauth2.Token{
			AccessToken:  "test-access-token-2",
			RefreshToken: "test-refresh-token-2",
			TokenType:    "Bearer",
		}

		mergedScopes := mergeScopes(ProfileScopes, GmailScopes)
		err := service.storeGoogleTokenWithScopes(user.ID, token, mergedScopes)
		require.NoError(t, err)

		// Verify Gmail scope is now present
		hasGmail, err := service.HasGmailScope(user.ID)
		require.NoError(t, err)
		assert.True(t, hasGmail)

		// Calendar should still be false
		hasCalendar, err := service.HasCalendarScope(user.ID)
		require.NoError(t, err)
		assert.False(t, hasCalendar)
	})

	// Add Calendar scope
	t.Run("updating token with Calendar scope works", func(t *testing.T) {
		// Get current scopes and add calendar
		currentScopes, err := service.GetUserScopes(user.ID)
		require.NoError(t, err)

		token := &oauth2.Token{
			AccessToken:  "test-access-token-3",
			RefreshToken: "test-refresh-token-3",
			TokenType:    "Bearer",
		}

		mergedScopes := mergeScopes(currentScopes, CalendarScopes)
		err = service.storeGoogleTokenWithScopes(user.ID, token, mergedScopes)
		require.NoError(t, err)

		// Now both should be true
		hasGmail, err := service.HasGmailScope(user.ID)
		require.NoError(t, err)
		assert.True(t, hasGmail)

		hasCalendar, err := service.HasCalendarScope(user.ID)
		require.NoError(t, err)
		assert.True(t, hasCalendar)
	})
}

// TestGetUserScopes_BackwardCompatibility tests legacy user handling
func TestGetUserScopes_BackwardCompatibility(t *testing.T) {
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-encryption-key-for-compat-tests")
	defer os.Unsetenv("ALFRED_ENCRYPTION_KEY")

	db := database.NewTestDB(t)
	user := database.CreateTestUser(t, db)

	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       ProfileScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	service, err := NewService(db.DB, config)
	require.NoError(t, err)

	// Create encryptor to manually insert token
	encryptor, err := NewEncryptor(nil)
	require.NoError(t, err)

	accessEncrypted, _ := encryptor.Encrypt([]byte("test-token"))
	refreshEncrypted, _ := encryptor.Encrypt([]byte("test-refresh"))

	t.Run("empty scopes column returns all scopes (backward compat)", func(t *testing.T) {
		// Insert token with empty scopes (like legacy users would have)
		_, err := db.Exec(`
			INSERT INTO google_tokens (user_id, access_token_encrypted, refresh_token_encrypted, token_type, scopes)
			VALUES (?, ?, ?, 'Bearer', '')
		`, user.ID, accessEncrypted, refreshEncrypted)
		require.NoError(t, err)

		scopes, err := service.GetUserScopes(user.ID)
		require.NoError(t, err)
		assert.Equal(t, OAuthScopes, scopes)

		// Legacy user should have all scopes
		hasGmail, _ := service.HasGmailScope(user.ID)
		hasCalendar, _ := service.HasCalendarScope(user.ID)
		assert.True(t, hasGmail)
		assert.True(t, hasCalendar)
	})

	t.Run("null scopes column returns all scopes (backward compat)", func(t *testing.T) {
		user2 := database.CreateTestUser(t, db)

		// Insert token with NULL scopes
		_, err := db.Exec(`
			INSERT INTO google_tokens (user_id, access_token_encrypted, refresh_token_encrypted, token_type, scopes)
			VALUES (?, ?, ?, 'Bearer', NULL)
		`, user2.ID, accessEncrypted, refreshEncrypted)
		require.NoError(t, err)

		scopes, err := service.GetUserScopes(user2.ID)
		require.NoError(t, err)
		assert.Equal(t, OAuthScopes, scopes)
	})

	t.Run("scopes='null' string returns all scopes (backward compat)", func(t *testing.T) {
		user3 := database.CreateTestUser(t, db)

		// Insert token with "null" string
		_, err := db.Exec(`
			INSERT INTO google_tokens (user_id, access_token_encrypted, refresh_token_encrypted, token_type, scopes)
			VALUES (?, ?, ?, 'Bearer', 'null')
		`, user3.ID, accessEncrypted, refreshEncrypted)
		require.NoError(t, err)

		scopes, err := service.GetUserScopes(user3.ID)
		require.NoError(t, err)
		assert.Equal(t, OAuthScopes, scopes)
	})

	t.Run("valid JSON scopes are parsed correctly", func(t *testing.T) {
		user4 := database.CreateTestUser(t, db)

		scopesJSON, _ := json.Marshal(ProfileScopes)
		_, err := db.Exec(`
			INSERT INTO google_tokens (user_id, access_token_encrypted, refresh_token_encrypted, token_type, scopes)
			VALUES (?, ?, ?, 'Bearer', ?)
		`, user4.ID, accessEncrypted, refreshEncrypted, string(scopesJSON))
		require.NoError(t, err)

		scopes, err := service.GetUserScopes(user4.ID)
		require.NoError(t, err)
		assert.Equal(t, ProfileScopes, scopes)
	})
}

// TestGetAuthURLWithScopes tests the URL generation with scopes
func TestGetAuthURLWithScopes(t *testing.T) {
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-encryption-key-for-url-tests")
	defer os.Unsetenv("ALFRED_ENCRYPTION_KEY")

	db := database.NewTestDB(t)

	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       ProfileScopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	service, err := NewService(db.DB, config)
	require.NoError(t, err)

	t.Run("generates URL with profile scopes only", func(t *testing.T) {
		url := service.GetAuthURLWithScopes(ProfileScopes, "test-state", false)

		assert.Contains(t, url, "https://accounts.google.com/o/oauth2/auth")
		assert.Contains(t, url, "client_id=test-client-id")
		assert.Contains(t, url, "state=test-state")
		assert.Contains(t, url, "userinfo.email")
		assert.Contains(t, url, "userinfo.profile")
		assert.NotContains(t, url, "include_granted_scopes")
	})

	t.Run("generates URL with Gmail scopes", func(t *testing.T) {
		url := service.GetAuthURLWithScopes(GmailScopes, "test-state", false)

		// Scopes are URL-encoded, so check for "gmail.readonly" substring
		assert.Contains(t, url, "gmail.readonly")
		assert.NotContains(t, url, "userinfo.email")
	})

	t.Run("generates URL with Calendar scopes", func(t *testing.T) {
		url := service.GetAuthURLWithScopes(CalendarScopes, "test-state", false)

		// Scopes are URL-encoded, so check for "calendar" substring
		assert.Contains(t, url, "calendar")
		assert.NotContains(t, url, "userinfo.email")
	})

	t.Run("includes include_granted_scopes when requested", func(t *testing.T) {
		url := service.GetAuthURLWithScopes(GmailScopes, "test-state", true)

		assert.Contains(t, url, "include_granted_scopes=true")
	})

	t.Run("excludes include_granted_scopes when not requested", func(t *testing.T) {
		url := service.GetAuthURLWithScopes(GmailScopes, "test-state", false)

		assert.NotContains(t, url, "include_granted_scopes")
	})
}
