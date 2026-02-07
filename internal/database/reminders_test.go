package database

import (
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDueRemindersForNotification(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	channel, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"due-reminder-test@s.whatsapp.net",
		"Due Reminder Test",
	)
	require.NoError(t, err)

	past := time.Now().Add(-30 * time.Minute)
	future := time.Now().Add(2 * time.Hour)
	reminderAt := time.Now().Add(-10 * time.Minute)

	dueReminder, err := db.CreatePendingReminder(&Reminder{
		UserID:       user.ID,
		ChannelID:    channel.ID,
		CalendarID:   "primary",
		Title:        "Past due reminder",
		DueDate:      &past,
		ActionType:   ReminderActionCreate,
		Priority:     ReminderPriorityNormal,
		LLMReasoning: "test",
	})
	require.NoError(t, err)
	require.NoError(t, db.UpdateReminderStatus(dueReminder.ID, ReminderStatusConfirmed))

	futureReminder, err := db.CreatePendingReminder(&Reminder{
		UserID:       user.ID,
		ChannelID:    channel.ID,
		CalendarID:   "primary",
		Title:        "Future reminder",
		DueDate:      &future,
		ActionType:   ReminderActionCreate,
		Priority:     ReminderPriorityNormal,
		LLMReasoning: "test",
	})
	require.NoError(t, err)
	require.NoError(t, db.UpdateReminderStatus(futureReminder.ID, ReminderStatusConfirmed))

	pendingReminder, err := db.CreatePendingReminder(&Reminder{
		UserID:       user.ID,
		ChannelID:    channel.ID,
		CalendarID:   "primary",
		Title:        "Pending reminder",
		DueDate:      &past,
		ActionType:   ReminderActionCreate,
		Priority:     ReminderPriorityNormal,
		LLMReasoning: "test",
	})
	require.NoError(t, err)
	// Keep as pending to verify only active statuses are included.

	reminderTimeReminder, err := db.CreatePendingReminder(&Reminder{
		UserID:       user.ID,
		ChannelID:    channel.ID,
		CalendarID:   "primary",
		Title:        "Reminder-time trigger",
		DueDate:      &future,
		ReminderTime: &reminderAt,
		ActionType:   ReminderActionCreate,
		Priority:     ReminderPriorityNormal,
		LLMReasoning: "test",
	})
	require.NoError(t, err)
	require.NoError(t, db.UpdateReminderStatus(reminderTimeReminder.ID, ReminderStatusSynced))

	dueReminders, err := db.GetDueRemindersForNotification(time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, dueReminders, 2)
	assert.Equal(t, dueReminder.ID, dueReminders[0].ID)
	assert.Equal(t, reminderTimeReminder.ID, dueReminders[1].ID)

	notified, err := db.MarkReminderDueNotificationSent(dueReminder.ID, time.Now())
	require.NoError(t, err)
	assert.True(t, notified)

	dueReminders, err = db.GetDueRemindersForNotification(time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, dueReminders, 1)
	assert.Equal(t, reminderTimeReminder.ID, dueReminders[0].ID)
	assert.NotEqual(t, futureReminder.ID, dueReminders[0].ID)
	assert.NotEqual(t, pendingReminder.ID, dueReminders[0].ID)
}

func TestMarkReminderDueNotificationSent_Idempotent(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	channel, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"mark-reminder-test@s.whatsapp.net",
		"Mark Reminder Test",
	)
	require.NoError(t, err)

	past := time.Now().Add(-time.Hour)
	reminder, err := db.CreatePendingReminder(&Reminder{
		UserID:       user.ID,
		ChannelID:    channel.ID,
		CalendarID:   "primary",
		Title:        "Mark once",
		DueDate:      &past,
		ActionType:   ReminderActionCreate,
		Priority:     ReminderPriorityNormal,
		LLMReasoning: "test",
	})
	require.NoError(t, err)
	require.NoError(t, db.UpdateReminderStatus(reminder.ID, ReminderStatusConfirmed))

	first, err := db.MarkReminderDueNotificationSent(reminder.ID, time.Now())
	require.NoError(t, err)
	assert.True(t, first)

	second, err := db.MarkReminderDueNotificationSent(reminder.ID, time.Now())
	require.NoError(t, err)
	assert.False(t, second)
}
