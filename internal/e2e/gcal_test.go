package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/omriShneor/project_alfred/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGCalStatus(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get GCal status when not authenticated", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/gcal/status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var status map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&status)
		require.NoError(t, err)

		// Should show not connected since no GCal client is configured
		assert.Equal(t, false, status["connected"])
	})
}

func TestGCalSettings(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get GCal settings", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/gcal/settings")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var settings map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&settings)
		require.NoError(t, err)

		// Should have default settings
		assert.Contains(t, settings, "sync_enabled")
		assert.Contains(t, settings, "selected_calendar_id")
	})

	t.Run("update GCal settings", func(t *testing.T) {
		updateData := map[string]interface{}{
			"sync_enabled":          true,
			"selected_calendar_id":  "work_calendar",
			"selected_calendar_name": "Work Calendar",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+"/api/gcal/settings", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update
		getResp, err := http.Get(ts.BaseURL() + "/api/gcal/settings")
		require.NoError(t, err)
		defer getResp.Body.Close()

		var settings map[string]interface{}
		err = json.NewDecoder(getResp.Body).Decode(&settings)
		require.NoError(t, err)

		assert.Equal(t, true, settings["sync_enabled"])
		assert.Equal(t, "work_calendar", settings["selected_calendar_id"])
	})
}

func TestGCalListCalendars(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("list calendars when not authenticated returns empty list", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/gcal/calendars")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Returns 200 with empty list when not authenticated (graceful UI handling)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var calendars []interface{}
		err = json.NewDecoder(resp.Body).Decode(&calendars)
		require.NoError(t, err)
		assert.Empty(t, calendars)
	})
}

// TestGCalConnect is obsolete - the /api/gcal/connect endpoint no longer exists
// OAuth flow now uses /api/auth/google/add-scopes for incremental authorization
func TestGCalConnect(t *testing.T) {
	t.Skip("Endpoint /api/gcal/connect no longer exists - OAuth handled by /api/auth/google")
}

func TestGCalDisconnect(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("disconnect when not connected returns error", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/gcal/disconnect", nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Returns 503 when GCal client is not configured
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})
}

func TestGCalTodayEvents(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get today's GCal events when not authenticated", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/gcal/events/today")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error since GCal is not authenticated
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	})
}

func TestGCalEventSync(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create channel and event
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Sync Test Channel").
		MustBuild(ts.DB)

	event := testutil.NewEventBuilder(channel.ID).
		WithUserID(ts.TestUser.ID).
		WithTitle("Event to Sync").
		Pending().
		MustBuild(ts.DB)

	t.Run("confirm event without GCal sets status to confirmed not synced", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/events/%d/confirm", event.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Without GCal client, event should be confirmed but not synced
		confirmed, err := ts.DB.GetEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, "confirmed", string(confirmed.Status))
		assert.Nil(t, confirmed.GoogleEventID)
	})
}
