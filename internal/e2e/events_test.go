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

func TestEventLifecycle(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup: Create a channel and pending event
	channel := testutil.NewChannelBuilder().
		WhatsApp().
		WithName("John Doe").
		WithIdentifier("john@s.whatsapp.net").
		MustBuild(ts.DB)

	event := testutil.NewEventBuilder(channel.ID).
		WithTitle("Lunch Meeting").
		WithDescription("Discuss project updates").
		WithLocation("Italian Restaurant").
		Pending().
		MustBuild(ts.DB)

	t.Run("list pending events", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/events?status=pending")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		assert.Len(t, events, 1)
		assert.Equal(t, "Lunch Meeting", events[0].Title)
		assert.Equal(t, database.EventStatusPending, events[0].Status)
	})

	t.Run("get event by ID", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/events/%d", event.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.NotNil(t, result["event"])
		eventData := result["event"].(map[string]interface{})
		assert.Equal(t, "Lunch Meeting", eventData["title"])
	})

	t.Run("update pending event", func(t *testing.T) {
		updateData := map[string]interface{}{
			"title":      "Updated Lunch Meeting",
			"start_time": event.StartTime.Format(time.RFC3339),
			"location":   "Conference Room A",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/events/%d", event.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update persisted
		updated, err := ts.DB.GetEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Lunch Meeting", updated.Title)
		assert.Equal(t, "Conference Room A", updated.Location)
	})
}

func TestEventConfirmation(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel and event
	channel := testutil.NewChannelBuilder().
		WhatsApp().
		WithName("Alice").
		MustBuild(ts.DB)

	event := testutil.NewEventBuilder(channel.ID).
		WithTitle("Team Standup").
		Pending().
		MustBuild(ts.DB)

	t.Run("confirm pending event", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/events/%d/confirm", event.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status changed (should be confirmed since no GCal client)
		confirmed, err := ts.DB.GetEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, database.EventStatusConfirmed, confirmed.Status)
	})

	t.Run("cannot confirm already confirmed event", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/events/%d/confirm", event.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error since event is already confirmed
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestEventRejection(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel and event
	channel := testutil.NewChannelBuilder().
		WhatsApp().
		WithName("Bob").
		MustBuild(ts.DB)

	event := testutil.NewEventBuilder(channel.ID).
		WithTitle("Optional Meeting").
		Pending().
		MustBuild(ts.DB)

	t.Run("reject pending event", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/events/%d/reject", event.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status changed
		rejected, err := ts.DB.GetEventByID(event.ID)
		require.NoError(t, err)
		assert.Equal(t, database.EventStatusRejected, rejected.Status)
	})

	t.Run("rejected event not in pending list", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/events?status=pending")
		require.NoError(t, err)
		defer resp.Body.Close()

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		// Should not contain the rejected event
		for _, e := range events {
			assert.NotEqual(t, event.ID, e.ID)
		}
	})
}

func TestEventFiltering(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WhatsApp().
		WithName("Test Channel").
		MustBuild(ts.DB)

	// Create events with different statuses
	pendingEvent := testutil.NewEventBuilder(channel.ID).
		WithTitle("Pending Event").
		Pending().
		MustBuild(ts.DB)

	// Create and confirm an event
	confirmedEvent := testutil.NewEventBuilder(channel.ID).
		WithTitle("Confirmed Event").
		Pending().
		MustBuild(ts.DB)
	_ = ts.DB.UpdateEventStatus(confirmedEvent.ID, database.EventStatusConfirmed)

	// Create and reject an event
	rejectedEvent := testutil.NewEventBuilder(channel.ID).
		WithTitle("Rejected Event").
		Pending().
		MustBuild(ts.DB)
	_ = ts.DB.UpdateEventStatus(rejectedEvent.ID, database.EventStatusRejected)

	t.Run("filter by pending status", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/events?status=pending")
		require.NoError(t, err)
		defer resp.Body.Close()

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		assert.Len(t, events, 1)
		assert.Equal(t, pendingEvent.ID, events[0].ID)
	})

	t.Run("filter by confirmed status", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/events?status=confirmed")
		require.NoError(t, err)
		defer resp.Body.Close()

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		assert.Len(t, events, 1)
		assert.Equal(t, confirmedEvent.ID, events[0].ID)
	})

	t.Run("filter by rejected status", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/events?status=rejected")
		require.NoError(t, err)
		defer resp.Body.Close()

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		assert.Len(t, events, 1)
		assert.Equal(t, rejectedEvent.ID, events[0].ID)
	})

	t.Run("list all events without filter", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/events")
		require.NoError(t, err)
		defer resp.Body.Close()

		var events []database.CalendarEvent
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		assert.Len(t, events, 3)
	})
}

func TestTodayEvents(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WhatsApp().
		WithName("Today Channel").
		MustBuild(ts.DB)

	// Create event for today
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	endOfEvent := startOfDay.Add(time.Hour)

	todayEvent := testutil.NewEventBuilder(channel.ID).
		WithTitle("Today's Meeting").
		WithStartTime(startOfDay).
		WithEndTime(endOfEvent).
		Pending().
		MustBuild(ts.DB)

	// Confirm the event so it shows in today's schedule
	_ = ts.DB.UpdateEventStatus(todayEvent.ID, database.EventStatusConfirmed)

	t.Run("get today's events", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/events/today")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Response is an array of events
		var events []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&events)
		require.NoError(t, err)

		// Should contain our confirmed event
		assert.NotEmpty(t, events)
	})
}

func TestEventWithAttendees(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WhatsApp().
		WithName("Meeting Channel").
		MustBuild(ts.DB)

	// Create event
	event := testutil.NewEventBuilder(channel.ID).
		WithTitle("Team Meeting with Attendees").
		Pending().
		MustBuild(ts.DB)

	// Add attendees
	attendees := []database.Attendee{
		{Email: "alice@example.com", DisplayName: "Alice"},
		{Email: "bob@example.com", DisplayName: "Bob"},
	}
	for _, a := range attendees {
		_, err := ts.DB.AddEventAttendee(event.ID, a.Email, a.DisplayName, false)
		require.NoError(t, err)
	}

	t.Run("get event with attendees", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/events/%d", event.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		eventData := result["event"].(map[string]interface{})
		eventAttendees := eventData["attendees"].([]interface{})
		assert.Len(t, eventAttendees, 2)
	})

	t.Run("update event with new attendees", func(t *testing.T) {
		// Get current event to preserve required fields
		currentEvent, err := ts.DB.GetEventByID(event.ID)
		require.NoError(t, err)

		updateData := map[string]interface{}{
			"title":      currentEvent.Title,
			"start_time": currentEvent.StartTime.Format(time.RFC3339),
			"attendees": []map[string]string{
				{"email": "charlie@example.com", "display_name": "Charlie"},
				{"email": "diana@example.com", "display_name": "Diana"},
				{"email": "eve@example.com", "display_name": "Eve"},
			},
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/events/%d", event.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify attendees updated
		updatedAttendees, err := ts.DB.GetEventAttendees(event.ID)
		require.NoError(t, err)
		assert.Len(t, updatedAttendees, 3)
	})
}
