package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestOnboardingFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("initial app status shows onboarding incomplete", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/app/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		assert.Equal(t, false, status["onboarding_complete"])
	})

	t.Run("complete onboarding", func(t *testing.T) {
		// Complete onboarding requires at least one input enabled
		body := map[string]bool{
			"whatsapp_enabled": true,
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/onboarding/complete", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("app status shows onboarding complete", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/app/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		assert.Equal(t, true, status["onboarding_complete"])
	})

	t.Run("reset onboarding", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/onboarding/reset", nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify reset
		resp2, err := http.Get(ts.BaseURL() + "/api/app/status")
		require.NoError(t, err)
		defer resp2.Body.Close()

		var status map[string]interface{}
		err = json.NewDecoder(resp2.Body).Decode(&status)
		require.NoError(t, err)

		assert.Equal(t, false, status["onboarding_complete"])
	})
}

func TestOnboardingResetCleansAllTokens(t *testing.T) {
	// Set encryption key for testing Google token storage
	t.Setenv("ALFRED_ENCRYPTION_KEY", "test-encryption-key-for-e2e-tests")

	ts := testutil.NewTestServer(t)
	userID := ts.TestUser.ID

	t.Run("setup - create tokens and sessions", func(t *testing.T) {
		// Create Google OAuth token
		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			TokenType:    "Bearer",
		}
		testScopes := []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}
		err := ts.DB.SaveGoogleToken(userID, token, "test@example.com", testScopes)
		require.NoError(t, err)

		// Create WhatsApp session
		err = ts.DB.SaveWhatsAppSession(userID, "+1234567890", "test-device-jid", true)
		require.NoError(t, err)

		// Create Telegram session
		err = ts.DB.SaveTelegramSession(userID, "+1234567890", true)
		require.NoError(t, err)

		// Verify tokens exist
		googleToken, err := ts.DB.GetGoogleToken(userID)
		require.NoError(t, err)
		assert.NotNil(t, googleToken, "Google token should exist before reset")

		waSession, err := ts.DB.GetWhatsAppSession(userID)
		require.NoError(t, err)
		assert.NotNil(t, waSession, "WhatsApp session should exist before reset")

		tgSession, err := ts.DB.GetTelegramSession(userID)
		require.NoError(t, err)
		assert.NotNil(t, tgSession, "Telegram session should exist before reset")
	})

	t.Run("reset onboarding", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/onboarding/reset", nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("verify all tokens are deleted", func(t *testing.T) {
		// Verify Google token is deleted
		googleToken, err := ts.DB.GetGoogleToken(userID)
		require.NoError(t, err)
		assert.Nil(t, googleToken, "Google token should be deleted after reset")

		// Verify WhatsApp session is deleted
		waSession, err := ts.DB.GetWhatsAppSession(userID)
		require.NoError(t, err)
		assert.Nil(t, waSession, "WhatsApp session should be deleted after reset")

		// Verify Telegram session is deleted
		tgSession, err := ts.DB.GetTelegramSession(userID)
		require.NoError(t, err)
		assert.Nil(t, tgSession, "Telegram session should be deleted after reset")
	})

	t.Run("verify onboarding flags are reset", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/app/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		assert.Equal(t, false, status["onboarding_complete"])
	})
}

func TestOnboardingResetWithoutTokens(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("reset onboarding when no tokens exist", func(t *testing.T) {
		// Reset without creating any tokens first
		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/onboarding/reset", nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should succeed even if no tokens exist
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("verify onboarding is reset", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/app/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		assert.Equal(t, false, status["onboarding_complete"])
	})
}

func TestOnboardingResetDeletesAllUserScopedData(t *testing.T) {
	// Set encryption key for testing Google token storage
	t.Setenv("ALFRED_ENCRYPTION_KEY", "test-encryption-key-for-e2e-tests")

	ts := testutil.NewTestServer(t)
	user1ID := ts.TestUser.ID
	user2 := database.CreateTestUser(t, ts.DB)
	user2ID := user2.ID

	seedUserScopedOnboardingData(t, ts.DB, user1ID, "user1")
	seedUserScopedOnboardingData(t, ts.DB, user2ID, "user2")

	clearedTables := []string{
		"channels",
		"message_history",
		"calendar_events",
		"reminders",
		"email_sources",
		"processed_emails",
		"google_contacts",
		"google_tokens",
		"whatsapp_sessions",
		"telegram_sessions",
		"user_notification_preferences",
		"gmail_settings",
		"gcal_settings",
		"user_sessions",
	}

	for _, table := range clearedTables {
		assert.Greater(t, countRowsByUser(t, ts.DB, table, user1ID), 0, "user1 should have data in %s before reset", table)
		assert.Greater(t, countRowsByUser(t, ts.DB, table, user2ID), 0, "user2 should have data in %s before reset", table)
	}
	assert.Greater(t, countEventAttendeesByUser(t, ts.DB, user1ID), 0, "user1 should have event attendees before reset")
	assert.Greater(t, countEventAttendeesByUser(t, ts.DB, user2ID), 0, "user2 should have event attendees before reset")

	req, err := http.NewRequest("POST", ts.BaseURL()+"/api/onboarding/reset", nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	for _, table := range clearedTables {
		assert.Equal(t, 0, countRowsByUser(t, ts.DB, table, user1ID), "user1 data in %s should be deleted", table)
		assert.Greater(t, countRowsByUser(t, ts.DB, table, user2ID), 0, "user2 data in %s should remain", table)
	}
	assert.Equal(t, 0, countEventAttendeesByUser(t, ts.DB, user1ID), "user1 event attendees should be deleted")
	assert.Greater(t, countEventAttendeesByUser(t, ts.DB, user2ID), 0, "user2 event attendees should remain")

	user1Status, err := ts.DB.GetAppStatus(user1ID)
	require.NoError(t, err)
	assert.False(t, user1Status.OnboardingComplete)
	assert.False(t, user1Status.WhatsAppEnabled)
	assert.False(t, user1Status.TelegramEnabled)
	assert.False(t, user1Status.GmailEnabled)
	assert.False(t, user1Status.GoogleCalEnabled)

	user2Status, err := ts.DB.GetAppStatus(user2ID)
	require.NoError(t, err)
	assert.True(t, user2Status.OnboardingComplete)
	assert.True(t, user2Status.WhatsAppEnabled)
	assert.True(t, user2Status.TelegramEnabled)
	assert.True(t, user2Status.GmailEnabled)
}

func TestOnboardingStatus(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get onboarding status", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/onboarding/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		// Check structure
		assert.Contains(t, status, "whatsapp")
		assert.Contains(t, status, "telegram")
		assert.Contains(t, status, "gcal")
		assert.Contains(t, status, "complete")
	})
}

func TestHealthCheck(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("health check returns healthy", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "healthy", health["status"])
		assert.Equal(t, "disconnected", health["whatsapp"])
		assert.Equal(t, "disconnected", health["telegram"])
		assert.Equal(t, "disconnected", health["gcal"])
	})
}

func seedUserScopedOnboardingData(t *testing.T, db *database.DB, userID int64, prefix string) {
	t.Helper()

	require.NoError(t, db.CompleteOnboarding(userID, true, true, true))
	require.NoError(t, db.SetGmailEnabled(userID, true))
	require.NoError(t, db.UpdateGCalSettings(userID, true, "primary", "Primary"))
	require.NoError(t, db.UpdateEmailPrefs(userID, true, prefix+"@example.com"))

	token := &oauth2.Token{
		AccessToken:  prefix + "-access-token",
		RefreshToken: prefix + "-refresh-token",
		TokenType:    "Bearer",
	}
	scopes := []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"}
	require.NoError(t, db.SaveGoogleToken(userID, token, prefix+"@example.com", scopes))

	require.NoError(t, db.SaveWhatsAppSession(userID, "+1234567890", prefix+"-device-jid", true))
	require.NoError(t, db.SaveTelegramSession(userID, "+1234567890", true))

	_, err := db.Exec(`
		INSERT INTO user_sessions (user_id, token_hash, expires_at, device_info)
		VALUES (?, ?, DATETIME('now', '+1 day'), ?)
	`, userID, prefix+"-session-hash", prefix+"-device")
	require.NoError(t, err)

	channel := testutil.NewChannelBuilder().
		WithUserID(userID).
		WhatsApp().
		WithIdentifier(prefix + "@s.whatsapp.net").
		WithName(prefix + " channel").
		MustBuild(db)

	testutil.NewMessageBuilder(channel.ID).
		WhatsApp().
		WithSenderID(prefix + "-sender@s.whatsapp.net").
		WithSenderName(prefix + " sender").
		WithText("message for " + prefix).
		MustBuild(db)

	event := testutil.NewEventBuilder(channel.ID).
		WithUserID(userID).
		WithTitle(prefix + " event").
		Pending().
		MustBuild(db)

	_, err = db.AddEventAttendee(event.ID, prefix+"-attendee@example.com", prefix+" attendee", false)
	require.NoError(t, err)

	testutil.NewReminderBuilder(channel.ID).
		WithUserID(userID).
		WithTitle(prefix + " reminder").
		Pending().
		MustBuild(db)

	_, err = db.CreateEmailSource(userID, database.EmailSourceTypeSender, prefix+"-sender@example.com", prefix+" sender")
	require.NoError(t, err)

	require.NoError(t, db.MarkEmailProcessed(userID, prefix+"-processed-email"))
	require.NoError(t, db.ReplaceTopContacts(userID, []database.TopContact{
		{
			Email:      prefix + "-contact@example.com",
			Name:       prefix + " contact",
			EmailCount: 5,
		},
	}))
}

func countRowsByUser(t *testing.T, db *database.DB, table string, userID int64) int {
	t.Helper()

	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE user_id = ?", table), userID).Scan(&count)
	require.NoError(t, err)
	return count
}

func countEventAttendeesByUser(t *testing.T, db *database.DB, userID int64) int {
	t.Helper()

	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM event_attendees ea
		JOIN calendar_events ce ON ce.id = ea.event_id
		WHERE ce.user_id = ?
	`, userID).Scan(&count)
	require.NoError(t, err)
	return count
}
