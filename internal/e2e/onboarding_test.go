package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

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
	ts := testutil.NewTestServer(t)
	userID := ts.TestUser.ID

	t.Run("setup - create tokens and sessions", func(t *testing.T) {
		// Create Google OAuth token
		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			TokenType:    "Bearer",
		}
		err := ts.DB.SaveGoogleToken(userID, token, "test@example.com")
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
