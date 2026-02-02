package processor

import (
	"context"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEventTimes(t *testing.T) {
	tests := []struct {
		name          string
		event         *agent.EventData
		wantStartStr  string
		wantEndStr    string
		wantErr       bool
		checkDefault  bool // if true, check that end time defaults to start + 1hr
	}{
		{
			name: "RFC3339 format with timezone",
			event: &agent.EventData{
				StartTime: "2024-01-15T14:00:00Z",
				EndTime:   "2024-01-15T15:00:00Z",
			},
			wantStartStr: "2024-01-15T14:00:00Z",
			wantEndStr:   "2024-01-15T15:00:00Z",
			wantErr:      false,
		},
		{
			name: "ISO8601 format without timezone",
			event: &agent.EventData{
				StartTime: "2024-01-15T14:00:00",
				EndTime:   "2024-01-15T16:30:00",
			},
			wantStartStr: "2024-01-15T14:00:00",
			wantEndStr:   "2024-01-15T16:30:00",
			wantErr:      false,
		},
		{
			name: "missing end time defaults to 1 hour",
			event: &agent.EventData{
				StartTime: "2024-01-15T14:00:00Z",
				EndTime:   "",
			},
			wantStartStr: "2024-01-15T14:00:00Z",
			checkDefault: true,
			wantErr:      false,
		},
		{
			name: "invalid start time format",
			event: &agent.EventData{
				StartTime: "not-a-date",
				EndTime:   "2024-01-15T15:00:00Z",
			},
			wantErr: true,
		},
		{
			name: "invalid end time is ignored (defaults to 1hr)",
			event: &agent.EventData{
				StartTime: "2024-01-15T14:00:00Z",
				EndTime:   "invalid-end-time",
			},
			wantStartStr: "2024-01-15T14:00:00Z",
			checkDefault: true,
			wantErr:      false,
		},
		{
			name: "RFC3339 with positive timezone offset",
			event: &agent.EventData{
				StartTime: "2024-01-15T14:00:00+02:00",
				EndTime:   "2024-01-15T15:00:00+02:00",
			},
			wantStartStr: "2024-01-15T14:00:00+02:00",
			wantEndStr:   "2024-01-15T15:00:00+02:00",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime, endTime, err := parseEventTimes(tt.event)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Parse expected start time
			expectedStart, _ := time.Parse(time.RFC3339, tt.wantStartStr)
			if expectedStart.IsZero() {
				expectedStart, _ = time.Parse("2006-01-02T15:04:05", tt.wantStartStr)
			}
			assert.Equal(t, expectedStart, startTime)

			// Check end time
			require.NotNil(t, endTime, "end time should not be nil")

			if tt.checkDefault {
				// End time should be 1 hour after start
				expectedEnd := startTime.Add(time.Hour)
				assert.Equal(t, expectedEnd, *endTime)
			} else if tt.wantEndStr != "" {
				expectedEnd, _ := time.Parse(time.RFC3339, tt.wantEndStr)
				if expectedEnd.IsZero() {
					expectedEnd, _ = time.Parse("2006-01-02T15:04:05", tt.wantEndStr)
				}
				assert.Equal(t, expectedEnd, *endTime)
			}
		})
	}
}

func TestMapActionType(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		want     database.EventActionType
		wantErr  bool
	}{
		{
			name:    "create action",
			action:  "create",
			want:    database.EventActionCreate,
			wantErr: false,
		},
		{
			name:    "update action",
			action:  "update",
			want:    database.EventActionUpdate,
			wantErr: false,
		},
		{
			name:    "delete action",
			action:  "delete",
			want:    database.EventActionDelete,
			wantErr: false,
		},
		{
			name:    "unknown action",
			action:  "unknown",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty action",
			action:  "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "none action (invalid - should be filtered before)",
			action:  "none",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapActionType(tt.action)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown action type")
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateEventFromAnalysis_NilAnalysis(t *testing.T) {
	db := database.NewTestDB(t)
	creator := NewEventCreator(db, nil)

	params := EventCreationParams{
		ChannelID:  1,
		SourceType: source.SourceTypeWhatsApp,
		Analysis:   nil,
	}

	_, err := creator.CreateEventFromAnalysis(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "analysis has no event data")
}

func TestCreateEventFromAnalysis_NilEventData(t *testing.T) {
	db := database.NewTestDB(t)
	creator := NewEventCreator(db, nil)

	params := EventCreationParams{
		ChannelID:  1,
		SourceType: source.SourceTypeWhatsApp,
		Analysis: &agent.EventAnalysis{
			HasEvent: true,
			Action:   "create",
			Event:    nil, // No event data
		},
	}

	_, err := creator.CreateEventFromAnalysis(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "analysis has no event data")
}

func TestCreateEventFromAnalysis_InvalidStartTime(t *testing.T) {
	db := database.NewTestDB(t)
	creator := NewEventCreator(db, nil)

	params := EventCreationParams{
		ChannelID:  1,
		SourceType: source.SourceTypeWhatsApp,
		Analysis: &agent.EventAnalysis{
			HasEvent: true,
			Action:   "create",
			Event: &agent.EventData{
				Title:     "Test Event",
				StartTime: "invalid-time",
			},
		},
	}

	_, err := creator.CreateEventFromAnalysis(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse start time")
}

func TestCreateEventFromAnalysis_UnknownAction(t *testing.T) {
	db := database.NewTestDB(t)
	creator := NewEventCreator(db, nil)

	params := EventCreationParams{
		ChannelID:  1,
		SourceType: source.SourceTypeWhatsApp,
		Analysis: &agent.EventAnalysis{
			HasEvent: true,
			Action:   "unknown_action",
			Event: &agent.EventData{
				Title:     "Test Event",
				StartTime: "2024-01-15T14:00:00Z",
			},
		},
	}

	_, err := creator.CreateEventFromAnalysis(context.Background(), params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action type")
}

func TestCreateEventFromAnalysis_Success(t *testing.T) {
	db := database.NewTestDB(t)

	// Create a channel first (required for foreign key)
	channel, err := db.CreateSourceChannel(
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"test@s.whatsapp.net",
		"Test Contact",
	)
	require.NoError(t, err)

	creator := NewEventCreator(db, nil)

	params := EventCreationParams{
		ChannelID:  channel.ID,
		SourceType: source.SourceTypeWhatsApp,
		Analysis: &agent.EventAnalysis{
			HasEvent:   true,
			Action:     "create",
			Reasoning:  "User mentioned a meeting",
			Confidence: 0.95,
			Event: &agent.EventData{
				Title:       "Team Meeting",
				Description: "Weekly sync meeting",
				StartTime:   "2024-01-15T14:00:00Z",
				EndTime:     "2024-01-15T15:00:00Z",
				Location:    "Conference Room A",
			},
		},
	}

	created, err := creator.CreateEventFromAnalysis(context.Background(), params)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotZero(t, created.ID)
	assert.Equal(t, "Team Meeting", created.Title)
	assert.Equal(t, "Weekly sync meeting", created.Description)
	assert.Equal(t, "Conference Room A", created.Location)
	assert.Equal(t, database.EventStatusPending, created.Status)
	assert.Equal(t, database.EventActionCreate, created.ActionType)
	assert.Equal(t, "User mentioned a meeting", created.LLMReasoning)
}

func TestCreateEventFromAnalysis_WithGoogleEventRef(t *testing.T) {
	db := database.NewTestDB(t)

	channel, err := db.CreateSourceChannel(
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"test@s.whatsapp.net",
		"Test Contact",
	)
	require.NoError(t, err)

	creator := NewEventCreator(db, nil)

	params := EventCreationParams{
		ChannelID:  channel.ID,
		SourceType: source.SourceTypeWhatsApp,
		Analysis: &agent.EventAnalysis{
			HasEvent: true,
			Action:   "update",
			Event: &agent.EventData{
				Title:     "Updated Meeting",
				StartTime: "2024-01-15T14:00:00Z",
				UpdateRef: "google-event-id-123", // Reference to existing Google event
			},
		},
	}

	created, err := creator.CreateEventFromAnalysis(context.Background(), params)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotNil(t, created.GoogleEventID)
	assert.Equal(t, "google-event-id-123", *created.GoogleEventID)
	assert.Equal(t, database.EventActionUpdate, created.ActionType)
}

func TestHandleExistingPendingEvent_Update(t *testing.T) {
	db := database.NewTestDB(t)

	channel, err := db.CreateSourceChannel(
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"test@s.whatsapp.net",
		"Test Contact",
	)
	require.NoError(t, err)

	// Create an existing pending event
	existingEvent := &database.CalendarEvent{
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Original Title",
		StartTime:  time.Now().Add(24 * time.Hour),
		ActionType: database.EventActionCreate,
	}
	created, err := db.CreatePendingEvent(existingEvent)
	require.NoError(t, err)

	creator := NewEventCreator(db, nil)

	// Update the existing pending event
	params := EventCreationParams{
		ChannelID:     channel.ID,
		SourceType:    source.SourceTypeWhatsApp,
		ExistingEvent: created,
		Analysis: &agent.EventAnalysis{
			HasEvent: true,
			Action:   "update",
			Event: &agent.EventData{
				Title:       "Updated Title",
				Description: "New description",
				StartTime:   "2024-01-20T10:00:00Z",
				EndTime:     "2024-01-20T11:00:00Z",
				Location:    "New Location",
			},
		},
	}

	updated, err := creator.CreateEventFromAnalysis(context.Background(), params)

	require.NoError(t, err)
	require.NotNil(t, updated)

	// Fetch the event from DB to verify
	fetched, err := db.GetEventByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", fetched.Title)
	assert.Equal(t, "New description", fetched.Description)
	assert.Equal(t, "New Location", fetched.Location)
}

func TestHandleExistingPendingEvent_Delete(t *testing.T) {
	db := database.NewTestDB(t)

	channel, err := db.CreateSourceChannel(
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"test@s.whatsapp.net",
		"Test Contact",
	)
	require.NoError(t, err)

	// Create an existing pending event
	existingEvent := &database.CalendarEvent{
		ChannelID:  channel.ID,
		CalendarID: "primary",
		Title:      "Event to Cancel",
		StartTime:  time.Now().Add(24 * time.Hour),
		ActionType: database.EventActionCreate,
	}
	created, err := db.CreatePendingEvent(existingEvent)
	require.NoError(t, err)

	creator := NewEventCreator(db, nil)

	// Delete (cancel) the existing pending event
	params := EventCreationParams{
		ChannelID:     channel.ID,
		SourceType:    source.SourceTypeWhatsApp,
		ExistingEvent: created,
		Analysis: &agent.EventAnalysis{
			HasEvent: true,
			Action:   "delete",
			Event: &agent.EventData{
				Title:     "Event to Cancel",
				StartTime: "2024-01-20T10:00:00Z",
			},
		},
	}

	result, err := creator.CreateEventFromAnalysis(context.Background(), params)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Fetch the event from DB to verify it was rejected
	fetched, err := db.GetEventByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, database.EventStatusRejected, fetched.Status)
}

func TestCreateEventFromAnalysis_WithMessageID(t *testing.T) {
	db := database.NewTestDB(t)

	channel, err := db.CreateSourceChannel(
		source.SourceTypeWhatsApp,
		source.ChannelTypeSender,
		"test@s.whatsapp.net",
		"Test Contact",
	)
	require.NoError(t, err)

	// Store a message first
	msg, err := db.StoreSourceMessage(
		source.SourceTypeWhatsApp,
		channel.ID,
		"sender@s.whatsapp.net",
		"Sender",
		"Let's have a meeting tomorrow",
		"",
		time.Now(),
	)
	require.NoError(t, err)

	creator := NewEventCreator(db, nil)

	params := EventCreationParams{
		ChannelID:  channel.ID,
		SourceType: source.SourceTypeWhatsApp,
		MessageID:  &msg.ID,
		Analysis: &agent.EventAnalysis{
			HasEvent: true,
			Action:   "create",
			Event: &agent.EventData{
				Title:     "Meeting",
				StartTime: "2024-01-15T14:00:00Z",
			},
		},
	}

	created, err := creator.CreateEventFromAnalysis(context.Background(), params)

	require.NoError(t, err)
	require.NotNil(t, created)
	assert.NotNil(t, created.OriginalMsgID)
	assert.Equal(t, msg.ID, *created.OriginalMsgID)
}

func TestEventCreationParams(t *testing.T) {
	// Test that EventCreationParams struct fields are properly set
	msgID := int64(42)
	emailSourceID := int64(100)

	params := EventCreationParams{
		ChannelID:     1,
		CalendarID:    "custom-calendar",
		SourceType:    source.SourceTypeGmail,
		EmailSourceID: &emailSourceID,
		MessageID:     &msgID,
		Analysis: &agent.EventAnalysis{
			HasEvent: true,
		},
		ExistingEvent: &database.CalendarEvent{
			ID:    999,
			Title: "Existing",
		},
	}

	assert.Equal(t, int64(1), params.ChannelID)
	assert.Equal(t, "custom-calendar", params.CalendarID)
	assert.Equal(t, source.SourceTypeGmail, params.SourceType)
	assert.NotNil(t, params.EmailSourceID)
	assert.Equal(t, int64(100), *params.EmailSourceID)
	assert.NotNil(t, params.MessageID)
	assert.Equal(t, int64(42), *params.MessageID)
	assert.NotNil(t, params.Analysis)
	assert.NotNil(t, params.ExistingEvent)
}
