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

func TestReminderLifecycle(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup: Create a channel and pending reminder
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("John Doe").
		WithIdentifier("john@s.whatsapp.net").
		MustBuild(ts.DB)

	reminder := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Call Mom").
		WithDescription("Don't forget to wish her happy birthday").
		HighPriority().
		Pending().
		MustBuild(ts.DB)

	t.Run("list pending reminders", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders?status=pending")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 1)
		assert.Equal(t, "Call Mom", reminders[0].Title)
		assert.Equal(t, database.ReminderStatusPending, reminders[0].Status)
		assert.Equal(t, database.ReminderPriorityHigh, reminders[0].Priority)
	})

	t.Run("get reminder by ID", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders/%d", reminder.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.NotNil(t, result["reminder"])
		reminderData := result["reminder"].(map[string]interface{})
		assert.Equal(t, "Call Mom", reminderData["title"])
		assert.Equal(t, "Don't forget to wish her happy birthday", reminderData["description"])
	})

	t.Run("update pending reminder", func(t *testing.T) {
		newDueDate := time.Now().Add(48 * time.Hour).Truncate(time.Second)
		updateData := map[string]interface{}{
			"title":    "Call Mom - Updated",
			"due_date": newDueDate.Format(time.RFC3339),
			"priority": "normal",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d", reminder.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update persisted
		updated, err := ts.DB.GetReminderByID(reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, "Call Mom - Updated", updated.Title)
		assert.Equal(t, database.ReminderPriorityNormal, updated.Priority)
	})
}

func TestManualReminderCreation(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("create manual reminder without due date", func(t *testing.T) {
		payload := map[string]interface{}{
			"title":       "Buy new running shoes",
			"description": "Check weekend sales",
			"location":    "City Mall",
			"priority":    "normal",
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/reminders", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)

		assert.Equal(t, "Buy new running shoes", created.Title)
		assert.Equal(t, "Check weekend sales", created.Description)
		assert.Equal(t, "City Mall", created.Location)
		assert.Equal(t, database.ReminderStatusConfirmed, created.Status)
		assert.Equal(t, "manual", created.Source)
		assert.Nil(t, created.DueDate)
	})

	t.Run("create manual reminder with due date", func(t *testing.T) {
		due := time.Now().Add(36 * time.Hour).Truncate(time.Second)
		payload := map[string]interface{}{
			"title":       "Renew car registration",
			"description": "Submit online",
			"due_date":    due.Format(time.RFC3339),
			"priority":    "high",
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/reminders", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&created)
		require.NoError(t, err)

		require.NotNil(t, created.DueDate)
		assert.WithinDuration(t, due, *created.DueDate, time.Second)
		assert.Equal(t, database.ReminderPriorityHigh, created.Priority)
		assert.Equal(t, database.ReminderStatusConfirmed, created.Status)
	})
}

func TestReminderConfirmation(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel and reminder
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Alice").
		MustBuild(ts.DB)

	reminder := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Submit Report").
		Pending().
		MustBuild(ts.DB)

	t.Run("confirm pending reminder", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/confirm", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status changed (should be confirmed since no GCal client)
		confirmed, err := ts.DB.GetReminderByID(reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, database.ReminderStatusConfirmed, confirmed.Status)
	})

	t.Run("cannot confirm already confirmed reminder", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/confirm", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return error since reminder is already confirmed
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestReminderRejection(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel and reminder
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Bob").
		MustBuild(ts.DB)

	reminder := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Optional Task").
		Pending().
		MustBuild(ts.DB)

	t.Run("reject pending reminder", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/reject", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status changed
		rejected, err := ts.DB.GetReminderByID(reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, database.ReminderStatusRejected, rejected.Status)
	})

	t.Run("rejected reminder not in pending list", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders?status=pending")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		// Should not contain the rejected reminder
		for _, r := range reminders {
			assert.NotEqual(t, reminder.ID, r.ID)
		}
	})

	t.Run("cannot reject already rejected reminder", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/reject", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestReminderCompletion(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Charlie").
		MustBuild(ts.DB)

	t.Run("complete confirmed reminder", func(t *testing.T) {
		// Create and confirm a reminder
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Complete This Task").
			Pending().
			MustBuild(ts.DB)

		// First confirm it
		_ = ts.DB.UpdateReminderStatus(reminder.ID, database.ReminderStatusConfirmed)

		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/complete", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status changed
		completed, err := ts.DB.GetReminderByID(reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, database.ReminderStatusCompleted, completed.Status)
	})

	t.Run("complete synced reminder", func(t *testing.T) {
		// Create and sync a reminder
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Synced Task").
			Pending().
			MustBuild(ts.DB)

		_ = ts.DB.UpdateReminderStatus(reminder.ID, database.ReminderStatusSynced)

		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/complete", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status changed
		completed, err := ts.DB.GetReminderByID(reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, database.ReminderStatusCompleted, completed.Status)
	})

	t.Run("cannot complete pending reminder", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Still Pending").
			Pending().
			MustBuild(ts.DB)

		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/complete", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestReminderDismissal(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Diana").
		MustBuild(ts.DB)

	t.Run("dismiss pending reminder", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Dismiss This").
			Pending().
			MustBuild(ts.DB)

		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/dismiss", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status changed
		dismissed, err := ts.DB.GetReminderByID(reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, database.ReminderStatusDismissed, dismissed.Status)
	})

	t.Run("dismiss confirmed reminder", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Confirmed But Dismiss").
			Pending().
			MustBuild(ts.DB)

		_ = ts.DB.UpdateReminderStatus(reminder.ID, database.ReminderStatusConfirmed)

		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/dismiss", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status changed
		dismissed, err := ts.DB.GetReminderByID(reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, database.ReminderStatusDismissed, dismissed.Status)
	})

	t.Run("cannot dismiss already dismissed reminder", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Already Dismissed").
			Pending().
			MustBuild(ts.DB)

		_ = ts.DB.UpdateReminderStatus(reminder.ID, database.ReminderStatusDismissed)

		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/dismiss", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("cannot dismiss completed reminder", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Already Completed").
			Pending().
			MustBuild(ts.DB)

		_ = ts.DB.UpdateReminderStatus(reminder.ID, database.ReminderStatusCompleted)

		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/dismiss", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestReminderFiltering(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Test Channel").
		MustBuild(ts.DB)

	// Create reminders with different statuses
	pendingReminder := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Pending Reminder").
		Pending().
		MustBuild(ts.DB)

	// Create and confirm a reminder
	confirmedReminder := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Confirmed Reminder").
		Pending().
		MustBuild(ts.DB)
	_ = ts.DB.UpdateReminderStatus(confirmedReminder.ID, database.ReminderStatusConfirmed)

	// Create and reject a reminder
	rejectedReminder := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Rejected Reminder").
		Pending().
		MustBuild(ts.DB)
	_ = ts.DB.UpdateReminderStatus(rejectedReminder.ID, database.ReminderStatusRejected)

	// Create and complete a reminder
	completedReminder := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Completed Reminder").
		Pending().
		MustBuild(ts.DB)
	_ = ts.DB.UpdateReminderStatus(completedReminder.ID, database.ReminderStatusCompleted)

	t.Run("filter by pending status", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders?status=pending")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 1)
		assert.Equal(t, pendingReminder.ID, reminders[0].ID)
	})

	t.Run("filter by confirmed status", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders?status=confirmed")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 1)
		assert.Equal(t, confirmedReminder.ID, reminders[0].ID)
	})

	t.Run("filter by rejected status", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders?status=rejected")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 1)
		assert.Equal(t, rejectedReminder.ID, reminders[0].ID)
	})

	t.Run("filter by completed status", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders?status=completed")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 1)
		assert.Equal(t, completedReminder.ID, reminders[0].ID)
	})

	t.Run("list all reminders without filter", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 4)
	})
}

func TestReminderFilterByChannel(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup two channels
	channel1 := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Channel 1").
		WithIdentifier("channel1@s.whatsapp.net").
		MustBuild(ts.DB)

	channel2 := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Channel 2").
		WithIdentifier("channel2@s.whatsapp.net").
		MustBuild(ts.DB)

	// Create reminders for each channel
	for i := 1; i <= 3; i++ {
		testutil.NewReminderBuilder(channel1.ID).
			WithTitle(fmt.Sprintf("Channel 1 Reminder %d", i)).
			Pending().
			MustBuild(ts.DB)
	}

	for i := 1; i <= 2; i++ {
		testutil.NewReminderBuilder(channel2.ID).
			WithTitle(fmt.Sprintf("Channel 2 Reminder %d", i)).
			Pending().
			MustBuild(ts.DB)
	}

	t.Run("filter reminders by channel 1", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders?channel_id=%d", channel1.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 3)
		for _, r := range reminders {
			assert.Equal(t, channel1.ID, r.ChannelID)
		}
	})

	t.Run("filter reminders by channel 2", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders?channel_id=%d", channel2.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 2)
		for _, r := range reminders {
			assert.Equal(t, channel2.ID, r.ChannelID)
		}
	})

	t.Run("filter by status and channel combined", func(t *testing.T) {
		// Confirm one reminder from channel 1
		reminders, _ := ts.DB.ListReminders(ts.TestUser.ID, nil, &channel1.ID)
		_ = ts.DB.UpdateReminderStatus(reminders[0].ID, database.ReminderStatusConfirmed)

		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders?status=confirmed&channel_id=%d", channel1.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var filteredReminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&filteredReminders)
		require.NoError(t, err)

		assert.Len(t, filteredReminders, 1)
		assert.Equal(t, database.ReminderStatusConfirmed, filteredReminders[0].Status)
		assert.Equal(t, channel1.ID, filteredReminders[0].ChannelID)
	})
}

func TestReminderPriorities(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Priority Test").
		MustBuild(ts.DB)

	// Create reminders with different priorities
	lowPriority := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Low Priority Task").
		LowPriority().
		Pending().
		MustBuild(ts.DB)

	normalPriority := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Normal Priority Task").
		NormalPriority().
		Pending().
		MustBuild(ts.DB)

	highPriority := testutil.NewReminderBuilder(channel.ID).
		WithTitle("High Priority Task").
		HighPriority().
		Pending().
		MustBuild(ts.DB)

	t.Run("verify reminder priorities", func(t *testing.T) {
		// Get low priority reminder
		lowResp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders/%d", lowPriority.ID))
		require.NoError(t, err)
		defer lowResp.Body.Close()

		var lowResult map[string]interface{}
		err = json.NewDecoder(lowResp.Body).Decode(&lowResult)
		require.NoError(t, err)
		assert.Equal(t, "low", lowResult["reminder"].(map[string]interface{})["priority"])

		// Get normal priority reminder
		normalResp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders/%d", normalPriority.ID))
		require.NoError(t, err)
		defer normalResp.Body.Close()

		var normalResult map[string]interface{}
		err = json.NewDecoder(normalResp.Body).Decode(&normalResult)
		require.NoError(t, err)
		assert.Equal(t, "normal", normalResult["reminder"].(map[string]interface{})["priority"])

		// Get high priority reminder
		highResp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders/%d", highPriority.ID))
		require.NoError(t, err)
		defer highResp.Body.Close()

		var highResult map[string]interface{}
		err = json.NewDecoder(highResp.Body).Decode(&highResult)
		require.NoError(t, err)
		assert.Equal(t, "high", highResult["reminder"].(map[string]interface{})["priority"])
	})

	t.Run("update reminder priority", func(t *testing.T) {
		updateData := map[string]interface{}{
			"priority": "high",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d", lowPriority.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify update
		updated, err := ts.DB.GetReminderByID(lowPriority.ID)
		require.NoError(t, err)
		assert.Equal(t, database.ReminderPriorityHigh, updated.Priority)
	})
}

func TestReminderNotFound(t *testing.T) {
	ts := testutil.NewTestServer(t)

	t.Run("get non-existent reminder", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders/99999")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("update non-existent reminder", func(t *testing.T) {
		updateData := map[string]interface{}{
			"title": "Updated",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+"/api/reminders/99999", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("confirm non-existent reminder", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/reminders/99999/confirm", nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("reject non-existent reminder", func(t *testing.T) {
		req, err := http.NewRequest("POST", ts.BaseURL()+"/api/reminders/99999/reject", nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestReminderUpdateValidation(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Validation Test").
		MustBuild(ts.DB)

	t.Run("cannot update confirmed reminder", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Already Confirmed").
			Pending().
			MustBuild(ts.DB)

		_ = ts.DB.UpdateReminderStatus(reminder.ID, database.ReminderStatusConfirmed)

		updateData := map[string]interface{}{
			"title": "Try to Update",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d", reminder.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("cannot update rejected reminder", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Already Rejected").
			Pending().
			MustBuild(ts.DB)

		_ = ts.DB.UpdateReminderStatus(reminder.ID, database.ReminderStatusRejected)

		updateData := map[string]interface{}{
			"title": "Try to Update",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d", reminder.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid due_date format returns error", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Date Format Test").
			Pending().
			MustBuild(ts.DB)

		updateData := map[string]interface{}{
			"due_date": "invalid-date",
		}
		body, _ := json.Marshal(updateData)

		req, err := http.NewRequest("PUT", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d", reminder.ID), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestReminderFromMultipleSources(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create WhatsApp channel
	waChannel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("WhatsApp Contact").
		WithIdentifier("wa@s.whatsapp.net").
		MustBuild(ts.DB)

	// Create Telegram channel
	tgChannel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		Telegram().
		WithName("Telegram Contact").
		WithIdentifier("tg_user_456").
		MustBuild(ts.DB)

	// Create reminders from different sources
	waReminder := testutil.NewReminderBuilder(waChannel.ID).
		WithTitle("WhatsApp Reminder").
		WithSource("whatsapp").
		Pending().
		MustBuild(ts.DB)

	tgReminder := testutil.NewReminderBuilder(tgChannel.ID).
		WithTitle("Telegram Reminder").
		WithSource("telegram").
		Pending().
		MustBuild(ts.DB)

	t.Run("reminders from different sources are listed together", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders?status=pending")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 2)

		// Find WhatsApp reminder
		var foundWa, foundTg bool
		for _, r := range reminders {
			if r.ID == waReminder.ID {
				foundWa = true
				assert.Equal(t, "WhatsApp Reminder", r.Title)
			}
			if r.ID == tgReminder.ID {
				foundTg = true
				assert.Equal(t, "Telegram Reminder", r.Title)
			}
		}
		assert.True(t, foundWa, "WhatsApp reminder should be in list")
		assert.True(t, foundTg, "Telegram reminder should be in list")
	})

	t.Run("filter by channel returns only that source", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders?channel_id=%d", waChannel.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		assert.Len(t, reminders, 1)
		assert.Equal(t, waReminder.ID, reminders[0].ID)
	})
}

func TestReminderDeleteAction(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Delete Test").
		MustBuild(ts.DB)

	t.Run("confirm delete action sets dismissed status", func(t *testing.T) {
		reminder := testutil.NewReminderBuilder(channel.ID).
			WithTitle("Cancel This Reminder").
			DeleteAction().
			Pending().
			MustBuild(ts.DB)

		req, err := http.NewRequest("POST", ts.BaseURL()+fmt.Sprintf("/api/reminders/%d/confirm", reminder.ID), nil)
		require.NoError(t, err)

		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify status is dismissed (not confirmed) for delete actions
		updated, err := ts.DB.GetReminderByID(reminder.ID)
		require.NoError(t, err)
		assert.Equal(t, database.ReminderStatusDismissed, updated.Status)
	})
}

func TestReminderChannelName(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Setup channel with a specific name
	channel := testutil.NewChannelBuilder().
		WithUserID(ts.TestUser.ID).
		WhatsApp().
		WithName("Important Contact").
		WithIdentifier("important@s.whatsapp.net").
		MustBuild(ts.DB)

	reminder := testutil.NewReminderBuilder(channel.ID).
		WithTitle("Check Channel Name").
		Pending().
		MustBuild(ts.DB)

	t.Run("reminder includes channel name", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + fmt.Sprintf("/api/reminders/%d", reminder.ID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		reminderData := result["reminder"].(map[string]interface{})
		assert.Equal(t, "Important Contact", reminderData["channel_name"])
	})

	t.Run("list reminders includes channel names", func(t *testing.T) {
		resp, err := http.Get(ts.BaseURL() + "/api/reminders")
		require.NoError(t, err)
		defer resp.Body.Close()

		var reminders []database.Reminder
		err = json.NewDecoder(resp.Body).Decode(&reminders)
		require.NoError(t, err)

		for _, r := range reminders {
			if r.ID == reminder.ID {
				assert.Equal(t, "Important Contact", r.ChannelName)
			}
		}
	})
}
