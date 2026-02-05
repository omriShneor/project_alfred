package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/omriShneor/project_alfred/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationPreferences(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get notification preferences", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/notifications/preferences")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Should have preferences and available sections
		assert.Contains(t, result, "preferences")
		assert.Contains(t, result, "available")

		prefs := result["preferences"].(map[string]interface{})
		assert.Contains(t, prefs, "email_enabled")
		assert.Contains(t, prefs, "push_enabled")

		available := result["available"].(map[string]interface{})
		assert.Contains(t, available, "email")
		assert.Contains(t, available, "push")
	})

	t.Run("push availability is true when notify service is configured", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/notifications/preferences")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		available := result["available"].(map[string]interface{})
		assert.Equal(t, true, available["push"])
	})
}

func TestEmailNotificationPreferences(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("update email preferences - enable", func(t *testing.T) {
		updateData := map[string]interface{}{
			"enabled": true,
			"address": "user@example.com",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+"/api/notifications/email", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update
		getResp, err := http.Get(ts.BaseURL() + "/api/notifications/preferences")
		require.NoError(t, err)
		defer getResp.Body.Close()

		var result map[string]interface{}
		err = json.NewDecoder(getResp.Body).Decode(&result)
		require.NoError(t, err)

		prefs := result["preferences"].(map[string]interface{})
		assert.Equal(t, true, prefs["email_enabled"])
		assert.Equal(t, "user@example.com", prefs["email_address"])
	})

	t.Run("update email preferences - disable", func(t *testing.T) {
		updateData := map[string]interface{}{
			"enabled": false,
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+"/api/notifications/email", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestPushNotificationPreferences(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("register push token", func(t *testing.T) {
		tokenData := map[string]string{
			"token": "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]",
		}
		body, _ := json.Marshal(tokenData)

		resp, err := http.Post(ts.BaseURL()+"/api/notifications/push/register", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify token stored
		getResp, err := http.Get(ts.BaseURL() + "/api/notifications/preferences")
		require.NoError(t, err)
		defer getResp.Body.Close()

		var result map[string]interface{}
		err = json.NewDecoder(getResp.Body).Decode(&result)
		require.NoError(t, err)

		prefs := result["preferences"].(map[string]interface{})
		assert.Equal(t, "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]", prefs["push_token"])
	})

	t.Run("update push preferences - enable", func(t *testing.T) {
		updateData := map[string]interface{}{
			"enabled": true,
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+"/api/notifications/push", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update
		getResp, err := http.Get(ts.BaseURL() + "/api/notifications/preferences")
		require.NoError(t, err)
		defer getResp.Body.Close()

		var result map[string]interface{}
		err = json.NewDecoder(getResp.Body).Decode(&result)
		require.NoError(t, err)

		prefs := result["preferences"].(map[string]interface{})
		assert.Equal(t, true, prefs["push_enabled"])
	})

	t.Run("update push preferences - disable", func(t *testing.T) {
		updateData := map[string]interface{}{
			"enabled": false,
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+"/api/notifications/push", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
