package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiUserDataIsolation verifies that different users cannot see each other's data
func TestMultiUserDataIsolation(t *testing.T) {
	// Create test servers for two different users
	ts1 := testutil.NewTestServer(t)
	ts2 := testutil.NewTestServerWithUser(t, "user2@example.com")

	// Create channels for each user
	channel1 := testutil.NewChannelBuilder().
		WithUserID(ts1.TestUser.ID).
		WhatsApp().
		WithName("User1 Contact").
		WithIdentifier("user1contact@s.whatsapp.net").
		MustBuild(ts1.DB)

	channel2 := testutil.NewChannelBuilder().
		WithUserID(ts2.TestUser.ID).
		WhatsApp().
		WithName("User2 Contact").
		WithIdentifier("user2contact@s.whatsapp.net").
		MustBuild(ts2.DB) // Use ts2's DB which is the same but authenticated as user2

	// Create events for each user
	testutil.NewEventBuilder(channel1.ID).
		WithUserID(ts1.TestUser.ID).
		WithTitle("User1 Event").
		Pending().
		MustBuild(ts1.DB)

	testutil.NewEventBuilder(channel2.ID).
		WithUserID(ts2.TestUser.ID).
		WithTitle("User2 Event").
		Pending().
		MustBuild(ts2.DB)

	t.Run("user1 only sees user1 channels", func(t *testing.T) {
		resp, err := http.Get(ts1.BaseURL() + "/api/channel")
		require.NoError(t, err)
		defer resp.Body.Close()

		var channels []database.SourceChannel
		err = json.NewDecoder(resp.Body).Decode(&channels)
		require.NoError(t, err)

		// Should only see User1 Contact
		assert.Len(t, channels, 1)
		assert.Equal(t, "User1 Contact", channels[0].Name)
	})

	t.Run("user2 only sees user2 channels", func(t *testing.T) {
		resp, err := http.Get(ts2.BaseURL() + "/api/channel")
		require.NoError(t, err)
		defer resp.Body.Close()

		var channels []database.SourceChannel
		err = json.NewDecoder(resp.Body).Decode(&channels)
		require.NoError(t, err)

		// Should only see User2 Contact
		assert.Len(t, channels, 1)
		assert.Equal(t, "User2 Contact", channels[0].Name)
	})

	t.Run("user1 only sees user1 events", func(t *testing.T) {
		resp, err := http.Get(ts1.BaseURL() + "/api/events")
		require.NoError(t, err)
		defer resp.Body.Close()

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		// Should only see User1 Event
		assert.Len(t, events, 1)
		assert.Equal(t, "User1 Event", events[0].Title)
	})

	t.Run("user2 only sees user2 events", func(t *testing.T) {
		resp, err := http.Get(ts2.BaseURL() + "/api/events")
		require.NoError(t, err)
		defer resp.Body.Close()

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		// Should only see User2 Event
		assert.Len(t, events, 1)
		assert.Equal(t, "User2 Event", events[0].Title)
	})
}

// TestMultiUserFeatureSettings verifies per-user feature settings
func TestMultiUserFeatureSettings(t *testing.T) {
	ts1 := testutil.NewTestServer(t)
	ts2 := testutil.NewTestServerWithUser(t, "user2@example.com")

	t.Run("users have independent app status", func(t *testing.T) {
		// Complete onboarding for user1
		body := map[string]interface{}{
			"whatsapp_enabled": true,
			"telegram_enabled": false,
			"gmail_enabled":    false,
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequest("POST", ts1.BaseURL()+"/api/onboarding/complete", bytes.NewReader(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts1.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Check user1 status - should be complete
		resp1, err := http.Get(ts1.BaseURL() + "/api/app/status")
		require.NoError(t, err)
		defer resp1.Body.Close()

		var status1 map[string]interface{}
		err = json.NewDecoder(resp1.Body).Decode(&status1)
		require.NoError(t, err)
		assert.True(t, status1["onboarding_complete"].(bool))

		// Check user2 status - should NOT be complete
		resp2, err := http.Get(ts2.BaseURL() + "/api/app/status")
		require.NoError(t, err)
		defer resp2.Body.Close()

		var status2 map[string]interface{}
		err = json.NewDecoder(resp2.Body).Decode(&status2)
		require.NoError(t, err)
		assert.False(t, status2["onboarding_complete"].(bool))
	})
}

// TestMultiUserReminders verifies per-user reminder isolation
func TestMultiUserReminders(t *testing.T) {
	ts1 := testutil.NewTestServer(t)
	ts2 := testutil.NewTestServerWithUser(t, "user2@example.com")

	// Create channels for reminders
	channel1 := testutil.NewChannelBuilder().
		WithUserID(ts1.TestUser.ID).
		WhatsApp().
		WithName("User1 Reminder Contact").
		MustBuild(ts1.DB)

	channel2 := testutil.NewChannelBuilder().
		WithUserID(ts2.TestUser.ID).
		WhatsApp().
		WithName("User2 Reminder Contact").
		MustBuild(ts2.DB)

	// Create reminders for each user
	testutil.NewReminderBuilder(channel1.ID).
		WithUserID(ts1.TestUser.ID).
		WithTitle("User1 Reminder").
		WithDueDate(time.Now().Add(time.Hour)).
		Pending().
		MustBuild(ts1.DB)

	testutil.NewReminderBuilder(channel2.ID).
		WithUserID(ts2.TestUser.ID).
		WithTitle("User2 Reminder").
		WithDueDate(time.Now().Add(time.Hour)).
		Pending().
		MustBuild(ts2.DB)

	t.Run("user1 only sees user1 reminders", func(t *testing.T) {
		resp, err := http.Get(ts1.BaseURL() + "/api/reminders")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 1)
		assert.Equal(t, "User1 Reminder", reminders[0].Title)
	})

	t.Run("user2 only sees user2 reminders", func(t *testing.T) {
		resp, err := http.Get(ts2.BaseURL() + "/api/reminders")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 1)
		assert.Equal(t, "User2 Reminder", reminders[0].Title)
	})
}

// TestCrossUserAccessDenied verifies users cannot access each other's resources by ID
func TestCrossUserAccessDenied(t *testing.T) {
	ts1 := testutil.NewTestServer(t)
	ts2 := testutil.NewTestServerWithUser(t, "user2@example.com")

	// Create event for user2
	channel2 := testutil.NewChannelBuilder().
		WithUserID(ts2.TestUser.ID).
		WhatsApp().
		WithName("User2 Channel").
		MustBuild(ts2.DB)

	event2 := testutil.NewEventBuilder(channel2.ID).
		WithUserID(ts2.TestUser.ID).
		WithTitle("User2 Secret Event").
		Pending().
		MustBuild(ts2.DB)

	t.Run("user1 cannot access user2 event by ID", func(t *testing.T) {
		// User1 tries to get User2's event by ID
		resp, err := http.Get(ts1.BaseURL() + fmt.Sprintf("/api/events/%d", event2.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 404 (not found) since user1 doesn't own this event
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("user1 cannot delete user2 channel", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", ts1.BaseURL()+fmt.Sprintf("/api/channel/%d", channel2.ID), nil)
		require.NoError(t, err)

		resp, err := ts1.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 404 (not found) since user1 doesn't own this channel
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// Verify channel still exists for user2
		respGet, err := http.Get(ts2.BaseURL() + "/api/channel")
		require.NoError(t, err)
		defer respGet.Body.Close()

		var channels []database.SourceChannel
		err = json.NewDecoder(respGet.Body).Decode(&channels)
		require.NoError(t, err)
		assert.Len(t, channels, 1)
	})
}
