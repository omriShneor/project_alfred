package database

import (
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestChannel creates a channel for testing events (events require a channel and user)
func createTestChannel(t *testing.T, db *DB, userID int64) *SourceChannel {
	t.Helper()
	channel, err := db.CreateSourceChannel(
		userID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"1234567890@s.whatsapp.net",
		"Test Contact",
	)
	require.NoError(t, err)
	require.NotNil(t, channel)

	return channel
}

func TestCreatePendingEvent(t *testing.T) {
	tests := []struct {
		name      string
		event     func(userID, channelID int64) *CalendarEvent
		wantErr   bool
		checkFunc func(t *testing.T, created *CalendarEvent)
	}{
		{
			name: "create event with all fields",
			event: func(userID, channelID int64) *CalendarEvent {
				endTime := time.Now().Add(time.Hour)
				googleEventID := "google-event-123"
				return &CalendarEvent{
					UserID:        userID,
					ChannelID:     channelID,
					GoogleEventID: &googleEventID,
					CalendarID:    "primary",
					Title:         "Team Meeting",
					Description:   "Weekly sync meeting",
					StartTime:     time.Now(),
					EndTime:       &endTime,
					Location:      "Conference Room A",
					ActionType:    EventActionCreate,
					LLMReasoning:  "User mentioned a meeting",
				}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *CalendarEvent) {
				assert.NotZero(t, created.ID)
				assert.NotZero(t, created.UserID)
				assert.Equal(t, "Team Meeting", created.Title)
				assert.Equal(t, EventStatusPending, created.Status)
				assert.Equal(t, EventActionCreate, created.ActionType)
				assert.NotNil(t, created.GoogleEventID)
				assert.Equal(t, "google-event-123", *created.GoogleEventID)
				assert.NotNil(t, created.EndTime)
			},
		},
		{
			name: "create event with minimal fields",
			event: func(userID, channelID int64) *CalendarEvent {
				return &CalendarEvent{
					UserID:     userID,
					ChannelID:  channelID,
					CalendarID: "primary",
					Title:      "Quick Event",
					StartTime:  time.Now(),
					ActionType: EventActionCreate,
				}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *CalendarEvent) {
				assert.NotZero(t, created.ID)
				assert.NotZero(t, created.UserID)
				assert.Equal(t, "Quick Event", created.Title)
				assert.Equal(t, EventStatusPending, created.Status)
				assert.Nil(t, created.GoogleEventID)
				assert.Nil(t, created.EndTime)
				assert.Nil(t, created.OriginalMsgID)
			},
		},
		{
			name: "create update action event",
			event: func(userID, channelID int64) *CalendarEvent {
				googleEventID := "existing-google-id"
				return &CalendarEvent{
					UserID:        userID,
					ChannelID:     channelID,
					GoogleEventID: &googleEventID,
					CalendarID:    "primary",
					Title:         "Updated Meeting",
					StartTime:     time.Now(),
					ActionType:    EventActionUpdate,
				}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *CalendarEvent) {
				assert.Equal(t, EventActionUpdate, created.ActionType)
			},
		},
		{
			name: "create delete action event",
			event: func(userID, channelID int64) *CalendarEvent {
				googleEventID := "event-to-delete"
				return &CalendarEvent{
					UserID:        userID,
					ChannelID:     channelID,
					GoogleEventID: &googleEventID,
					CalendarID:    "primary",
					Title:         "Cancelled Meeting",
					StartTime:     time.Now(),
					ActionType:    EventActionDelete,
				}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, created *CalendarEvent) {
				assert.Equal(t, EventActionDelete, created.ActionType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewTestDB(t)
			user := CreateTestUser(t, db)
			channel := createTestChannel(t, db, user.ID)

			event := tt.event(user.ID, channel.ID)
			created, err := db.CreatePendingEvent(event)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, created)
			tt.checkFunc(t, created)
		})
	}
}

func TestCreatePendingEvent_InvalidChannelID(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)

	event := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  999999, // Non-existent channel
		CalendarID: "primary",
		Title:      "Test Event",
		StartTime:  time.Now(),
		ActionType: EventActionCreate,
	}

	_, err := db.CreatePendingEvent(event)
	assert.Error(t, err, "should fail with non-existent channel due to foreign key constraint")
}

func TestGetEventByID(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	// Create an event first
	endTime := time.Now().Add(time.Hour)
	event := &CalendarEvent{
		UserID:      user.ID,
		ChannelID:   channel.ID,
		CalendarID:  "primary",
		Title:       "Retrievable Event",
		Description: "Test description",
		StartTime:   time.Now(),
		EndTime:     &endTime,
		Location:    "Test Location",
		ActionType:  EventActionCreate,
	}

	created, err := db.CreatePendingEvent(event)
	require.NoError(t, err)

	t.Run("get existing event", func(t *testing.T) {
		retrieved, err := db.GetEventByID(created.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, user.ID, retrieved.UserID)
		assert.Equal(t, "Retrievable Event", retrieved.Title)
		assert.Equal(t, "Test description", retrieved.Description)
		assert.Equal(t, "Test Location", retrieved.Location)
		assert.Equal(t, EventStatusPending, retrieved.Status)
		assert.Equal(t, "Test Contact", retrieved.ChannelName) // From joined channel
		assert.NotNil(t, retrieved.EndTime)
	})

	t.Run("get non-existent event", func(t *testing.T) {
		_, err := db.GetEventByID(999999)
		assert.Error(t, err)
	})
}

func TestListEvents(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	// Create multiple events with different statuses
	pendingEvent := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Pending Event",
		StartTime:  time.Now(),
		ActionType: EventActionCreate,
	}
	created1, err := db.CreatePendingEvent(pendingEvent)
	require.NoError(t, err)

	// Create another event and change its status
	syncedEvent := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Synced Event",
		StartTime:  time.Now().Add(time.Hour),
		ActionType: EventActionCreate,
	}
	created2, err := db.CreatePendingEvent(syncedEvent)
	require.NoError(t, err)
	err = db.UpdateEventStatus(created2.ID, EventStatusSynced)
	require.NoError(t, err)

	// Create another channel and event for filtering
	channel2, err := db.CreateSourceChannel(
		user.ID,
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"9876543210@s.whatsapp.net",
		"Another Contact",
	)
	require.NoError(t, err)

	otherChannelEvent := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel2.ID,
		CalendarID: "primary",
		Title:      "Other Channel Event",
		StartTime:  time.Now(),
		ActionType: EventActionCreate,
	}
	_, err = db.CreatePendingEvent(otherChannelEvent)
	require.NoError(t, err)

	t.Run("list all events (no filter)", func(t *testing.T) {
		events, err := db.ListEvents(user.ID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, events, 3)
	})

	t.Run("filter by pending status", func(t *testing.T) {
		status := EventStatusPending
		events, err := db.ListEvents(user.ID, &status, nil)
		require.NoError(t, err)
		assert.Len(t, events, 2) // Pending Event + Other Channel Event

		for _, e := range events {
			assert.Equal(t, EventStatusPending, e.Status)
		}
	})

	t.Run("filter by synced status", func(t *testing.T) {
		status := EventStatusSynced
		events, err := db.ListEvents(user.ID, &status, nil)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "Synced Event", events[0].Title)
	})

	t.Run("filter by channel ID", func(t *testing.T) {
		events, err := db.ListEvents(user.ID, nil, &channel.ID)
		require.NoError(t, err)
		assert.Len(t, events, 2) // Pending Event + Synced Event

		for _, e := range events {
			assert.Equal(t, channel.ID, e.ChannelID)
		}
	})

	t.Run("filter by status and channel ID", func(t *testing.T) {
		status := EventStatusPending
		events, err := db.ListEvents(user.ID, &status, &channel.ID)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, created1.ID, events[0].ID)
	})

	t.Run("empty result for non-matching filter", func(t *testing.T) {
		status := EventStatusRejected
		events, err := db.ListEvents(user.ID, &status, nil)
		require.NoError(t, err)
		assert.Len(t, events, 0)
	})

	t.Run("empty result for different user", func(t *testing.T) {
		user2 := CreateTestUser(t, db)
		events, err := db.ListEvents(user2.ID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, events, 0)
	})
}

func TestUpdateEventStatus(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	event := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Status Test Event",
		StartTime:  time.Now(),
		ActionType: EventActionCreate,
	}

	created, err := db.CreatePendingEvent(event)
	require.NoError(t, err)
	assert.Equal(t, EventStatusPending, created.Status)

	tests := []struct {
		name      string
		newStatus EventStatus
	}{
		{"pending to confirmed", EventStatusConfirmed},
		{"confirmed to synced", EventStatusSynced},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpdateEventStatus(created.ID, tt.newStatus)
			require.NoError(t, err)

			updated, err := db.GetEventByID(created.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.newStatus, updated.Status)
		})
	}

	t.Run("update to rejected", func(t *testing.T) {
		err := db.UpdateEventStatus(created.ID, EventStatusRejected)
		require.NoError(t, err)

		updated, err := db.GetEventByID(created.ID)
		require.NoError(t, err)
		assert.Equal(t, EventStatusRejected, updated.Status)
	})
}

func TestUpdatePendingEvent(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	event := &CalendarEvent{
		UserID:      user.ID,
		ChannelID:   channel.ID,
		CalendarID:  "primary",
		Title:       "Original Title",
		Description: "Original Description",
		StartTime:   time.Now(),
		Location:    "Original Location",
		ActionType:  EventActionCreate,
	}

	created, err := db.CreatePendingEvent(event)
	require.NoError(t, err)

	t.Run("update pending event successfully", func(t *testing.T) {
		newStartTime := time.Now().Add(24 * time.Hour)
		newEndTime := newStartTime.Add(time.Hour)

		err := db.UpdatePendingEvent(
			created.ID,
			"Updated Title",
			"Updated Description",
			newStartTime,
			&newEndTime,
			"Updated Location",
		)
		require.NoError(t, err)

		updated, err := db.GetEventByID(created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", updated.Title)
		assert.Equal(t, "Updated Description", updated.Description)
		assert.Equal(t, "Updated Location", updated.Location)
		assert.NotNil(t, updated.EndTime)
	})

	t.Run("cannot update non-pending event", func(t *testing.T) {
		// Change status to synced
		err := db.UpdateEventStatus(created.ID, EventStatusSynced)
		require.NoError(t, err)

		// Try to update - this will execute but not match any rows
		err = db.UpdatePendingEvent(
			created.ID,
			"Should Not Update",
			"Should Not Update",
			time.Now(),
			nil,
			"Should Not Update",
		)
		require.NoError(t, err) // No error, but no rows affected

		// Verify title didn't change
		event, err := db.GetEventByID(created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Title", event.Title) // Still the previously updated title
	})
}

func TestGetActiveEventsForChannel(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	// Create events with different statuses
	pendingEvent := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Pending",
		StartTime:  time.Now().Add(time.Hour),
		ActionType: EventActionCreate,
	}
	_, err := db.CreatePendingEvent(pendingEvent)
	require.NoError(t, err)

	syncedEvent := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Synced",
		StartTime:  time.Now().Add(2 * time.Hour),
		ActionType: EventActionCreate,
	}
	created2, err := db.CreatePendingEvent(syncedEvent)
	require.NoError(t, err)
	err = db.UpdateEventStatus(created2.ID, EventStatusSynced)
	require.NoError(t, err)

	rejectedEvent := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Rejected",
		StartTime:  time.Now().Add(3 * time.Hour),
		ActionType: EventActionCreate,
	}
	created3, err := db.CreatePendingEvent(rejectedEvent)
	require.NoError(t, err)
	err = db.UpdateEventStatus(created3.ID, EventStatusRejected)
	require.NoError(t, err)

	t.Run("returns only pending and synced events", func(t *testing.T) {
		events, err := db.GetActiveEventsForChannel(user.ID, channel.ID)
		require.NoError(t, err)
		assert.Len(t, events, 2)

		statuses := make(map[EventStatus]bool)
		for _, e := range events {
			statuses[e.Status] = true
		}
		assert.True(t, statuses[EventStatusPending])
		assert.True(t, statuses[EventStatusSynced])
		assert.False(t, statuses[EventStatusRejected])
	})

	t.Run("orders by start time ascending", func(t *testing.T) {
		events, err := db.GetActiveEventsForChannel(user.ID, channel.ID)
		require.NoError(t, err)

		for i := 1; i < len(events); i++ {
			assert.True(t, events[i].StartTime.After(events[i-1].StartTime) ||
				events[i].StartTime.Equal(events[i-1].StartTime),
				"events should be ordered by start time ascending")
		}
	})

	t.Run("empty for channel with no events", func(t *testing.T) {
		channel2, err := db.CreateSourceChannel(
			user.ID,
			source.SourceTypeWhatsApp,
			source.ChannelTypeSender,
			"empty@s.whatsapp.net",
			"Empty Channel",
		)
		require.NoError(t, err)

		events, err := db.GetActiveEventsForChannel(user.ID, channel2.ID)
		require.NoError(t, err)
		assert.Len(t, events, 0)
	})

	t.Run("empty for different user", func(t *testing.T) {
		user2 := CreateTestUser(t, db)
		events, err := db.GetActiveEventsForChannel(user2.ID, channel.ID)
		require.NoError(t, err)
		assert.Len(t, events, 0)
	})
}

func TestUpdateEventGoogleID(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	event := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "To Be Synced",
		StartTime:  time.Now(),
		ActionType: EventActionCreate,
	}

	created, err := db.CreatePendingEvent(event)
	require.NoError(t, err)
	assert.Nil(t, created.GoogleEventID)
	assert.Equal(t, EventStatusPending, created.Status)

	t.Run("sets google event id and updates status to synced", func(t *testing.T) {
		err := db.UpdateEventGoogleID(created.ID, "new-google-id-123")
		require.NoError(t, err)

		updated, err := db.GetEventByID(created.ID)
		require.NoError(t, err)
		require.NotNil(t, updated.GoogleEventID)
		assert.Equal(t, "new-google-id-123", *updated.GoogleEventID)
		assert.Equal(t, EventStatusSynced, updated.Status)
	})
}

func TestDeleteEvent(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	event := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "To Delete",
		StartTime:  time.Now(),
		ActionType: EventActionCreate,
	}

	created, err := db.CreatePendingEvent(event)
	require.NoError(t, err)

	t.Run("delete existing event", func(t *testing.T) {
		err := db.DeleteEvent(created.ID)
		require.NoError(t, err)

		_, err = db.GetEventByID(created.ID)
		assert.Error(t, err, "should not find deleted event")
	})

	t.Run("delete non-existent event (no error)", func(t *testing.T) {
		err := db.DeleteEvent(999999)
		require.NoError(t, err) // SQLite DELETE doesn't error on missing rows
	})
}

func TestCountPendingEvents(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	t.Run("zero pending events initially", func(t *testing.T) {
		count, err := db.CountPendingEvents(user.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	// Create some pending events
	for i := 0; i < 3; i++ {
		event := &CalendarEvent{
			UserID:     user.ID,
			ChannelID:  channel.ID,
			CalendarID: "primary",
			Title:      "Pending Event",
			StartTime:  time.Now(),
			ActionType: EventActionCreate,
		}
		_, err := db.CreatePendingEvent(event)
		require.NoError(t, err)
	}

	t.Run("counts pending events correctly", func(t *testing.T) {
		count, err := db.CountPendingEvents(user.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	// Create and sync an event
	syncedEvent := &CalendarEvent{
		UserID:     user.ID,
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Synced Event",
		StartTime:  time.Now(),
		ActionType: EventActionCreate,
	}
	created, err := db.CreatePendingEvent(syncedEvent)
	require.NoError(t, err)
	err = db.UpdateEventStatus(created.ID, EventStatusSynced)
	require.NoError(t, err)

	t.Run("does not count synced events", func(t *testing.T) {
		count, err := db.CountPendingEvents(user.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, count) // Still 3, not 4
	})

	t.Run("zero pending events for different user", func(t *testing.T) {
		user2 := CreateTestUser(t, db)
		count, err := db.CountPendingEvents(user2.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestGetEventByGoogleID(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	googleEventID := "test-google-event-id"
	event := &CalendarEvent{
		UserID:        user.ID,
		ChannelID:     channel.ID,
		GoogleEventID: &googleEventID,
		CalendarID:    "primary",
		Title:         "Google Synced Event",
		StartTime:     time.Now(),
		ActionType:    EventActionCreate,
	}

	created, err := db.CreatePendingEvent(event)
	require.NoError(t, err)

	t.Run("find by google event id", func(t *testing.T) {
		found, err := db.GetEventByGoogleID(googleEventID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, "Google Synced Event", found.Title)
	})

	t.Run("not found for non-existent google id", func(t *testing.T) {
		_, err := db.GetEventByGoogleID("non-existent-google-id")
		assert.Error(t, err)
	})
}

func TestGetEventByGoogleIDForUser(t *testing.T) {
	db := NewTestDB(t)
	user1 := CreateTestUser(t, db)
	user2 := CreateTestUser(t, db)
	channel1 := createTestChannel(t, db, user1.ID)
	channel2 := createTestChannel(t, db, user2.ID)

	googleEventID := "shared-google-event-id"

	event1 := &CalendarEvent{
		UserID:        user1.ID,
		ChannelID:     channel1.ID,
		GoogleEventID: &googleEventID,
		CalendarID:    "primary",
		Title:         "User 1 Event",
		StartTime:     time.Now(),
		ActionType:    EventActionCreate,
	}
	created1, err := db.CreatePendingEvent(event1)
	require.NoError(t, err)

	event2 := &CalendarEvent{
		UserID:        user2.ID,
		ChannelID:     channel2.ID,
		GoogleEventID: &googleEventID,
		CalendarID:    "primary",
		Title:         "User 2 Event",
		StartTime:     time.Now(),
		ActionType:    EventActionCreate,
	}
	created2, err := db.CreatePendingEvent(event2)
	require.NoError(t, err)

	found1, err := db.GetEventByGoogleIDForUser(user1.ID, googleEventID)
	require.NoError(t, err)
	require.NotNil(t, found1)
	assert.Equal(t, created1.ID, found1.ID)
	assert.Equal(t, user1.ID, found1.UserID)

	found2, err := db.GetEventByGoogleIDForUser(user2.ID, googleEventID)
	require.NoError(t, err)
	require.NotNil(t, found2)
	assert.Equal(t, created2.ID, found2.ID)
	assert.Equal(t, user2.ID, found2.UserID)

	notFound, err := db.GetEventByGoogleIDForUser(user1.ID, "does-not-exist")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUpdateSyncedEventFromGoogle(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)

	start := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour)

	event := &CalendarEvent{
		UserID:      user.ID,
		ChannelID:   channel.ID,
		CalendarID:  "primary",
		Title:       "Original",
		Description: "Original Description",
		StartTime:   start,
		EndTime:     &end,
		Location:    "Original Location",
		ActionType:  EventActionCreate,
	}

	created, err := db.CreatePendingEvent(event)
	require.NoError(t, err)

	// Pending events should not be updated by Google sync helper.
	err = db.UpdateSyncedEventFromGoogle(
		created.ID,
		"Should Not Apply",
		"Should Not Apply",
		start.Add(24*time.Hour),
		nil,
		"Should Not Apply",
	)
	require.NoError(t, err)

	pending, err := db.GetEventByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Original", pending.Title)
	assert.Equal(t, "Original Description", pending.Description)
	assert.Equal(t, "Original Location", pending.Location)

	err = db.UpdateEventStatus(created.ID, EventStatusSynced)
	require.NoError(t, err)

	updatedStart := start.Add(24 * time.Hour)
	updatedEnd := updatedStart.Add(90 * time.Minute)
	err = db.UpdateSyncedEventFromGoogle(
		created.ID,
		"Synced Title",
		"Synced Description",
		updatedStart,
		&updatedEnd,
		"Synced Location",
	)
	require.NoError(t, err)

	synced, err := db.GetEventByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Synced Title", synced.Title)
	assert.Equal(t, "Synced Description", synced.Description)
	assert.Equal(t, "Synced Location", synced.Location)
	assert.True(t, synced.StartTime.Equal(updatedStart))
	require.NotNil(t, synced.EndTime)
	assert.True(t, synced.EndTime.Equal(updatedEnd))
}

func TestListSyncedEventsWithGoogleID(t *testing.T) {
	db := NewTestDB(t)
	user := CreateTestUser(t, db)
	otherUser := CreateTestUser(t, db)
	channel := createTestChannel(t, db, user.ID)
	otherChannel := createTestChannel(t, db, otherUser.ID)

	makeEvent := func(userID, channelID int64, title string, status EventStatus, googleEventID *string) *CalendarEvent {
		event := &CalendarEvent{
			UserID:        userID,
			ChannelID:     channelID,
			GoogleEventID: googleEventID,
			CalendarID:    "primary",
			Title:         title,
			StartTime:     time.Now().Add(time.Minute).Truncate(time.Second),
			ActionType:    EventActionCreate,
		}
		created, err := db.CreatePendingEvent(event)
		require.NoError(t, err)
		if status != EventStatusPending {
			err = db.UpdateEventStatus(created.ID, status)
			require.NoError(t, err)
		}
		return created
	}

	idA := "google-a"
	idB := "google-b"
	idC := "google-c"
	idD := "google-d"

	synced := makeEvent(user.ID, channel.ID, "Synced", EventStatusSynced, &idA)
	confirmed := makeEvent(user.ID, channel.ID, "Confirmed", EventStatusConfirmed, &idB)
	_ = makeEvent(user.ID, channel.ID, "Pending", EventStatusPending, &idC)
	_ = makeEvent(user.ID, channel.ID, "No Google", EventStatusSynced, nil)
	_ = makeEvent(user.ID, channel.ID, "Deleted", EventStatusDeleted, &idD)
	_ = makeEvent(otherUser.ID, otherChannel.ID, "Other User", EventStatusSynced, &idA)

	results, err := db.ListSyncedEventsWithGoogleID(user.ID)
	require.NoError(t, err)
	require.Len(t, results, 2)

	foundIDs := map[int64]bool{
		results[0].ID: true,
		results[1].ID: true,
	}
	assert.True(t, foundIDs[synced.ID])
	assert.True(t, foundIDs[confirmed.ID])
	for _, event := range results {
		require.NotNil(t, event.GoogleEventID)
		assert.NotEmpty(t, *event.GoogleEventID)
		assert.Contains(t, []EventStatus{EventStatusSynced, EventStatusConfirmed}, event.Status)
	}
}
