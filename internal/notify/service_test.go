package notify

import (
	"context"
	"testing"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockNotifier for testing
type MockNotifier struct {
	mock.Mock
}

func (m *MockNotifier) Send(ctx context.Context, event *database.CalendarEvent, recipient string) error {
	args := m.Called(ctx, event, recipient)
	return args.Error(0)
}

func (m *MockNotifier) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockNotifier) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestNewService(t *testing.T) {
	db := database.NewTestDB(t)
	emailNotifier := &MockNotifier{}
	pushNotifier := &MockNotifier{}

	service := NewService(db, emailNotifier, pushNotifier)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, emailNotifier, service.emailNotifier)
	assert.Equal(t, pushNotifier, service.pushNotifier)
}

func TestNewService_NilNotifiers(t *testing.T) {
	db := database.NewTestDB(t)

	service := NewService(db, nil, nil)

	assert.NotNil(t, service)
	assert.Nil(t, service.emailNotifier)
	assert.Nil(t, service.pushNotifier)
}

func TestIsEmailAvailable(t *testing.T) {
	db := database.NewTestDB(t)

	t.Run("available when notifier configured", func(t *testing.T) {
		emailNotifier := &MockNotifier{}
		emailNotifier.On("IsConfigured").Return(true)

		service := NewService(db, emailNotifier, nil)
		assert.True(t, service.IsEmailAvailable())

		emailNotifier.AssertExpectations(t)
	})

	t.Run("not available when notifier not configured", func(t *testing.T) {
		emailNotifier := &MockNotifier{}
		emailNotifier.On("IsConfigured").Return(false)

		service := NewService(db, emailNotifier, nil)
		assert.False(t, service.IsEmailAvailable())

		emailNotifier.AssertExpectations(t)
	})

	t.Run("not available when notifier is nil", func(t *testing.T) {
		service := NewService(db, nil, nil)
		assert.False(t, service.IsEmailAvailable())
	})
}

func TestIsPushAvailable(t *testing.T) {
	db := database.NewTestDB(t)

	t.Run("available when notifier configured", func(t *testing.T) {
		pushNotifier := &MockNotifier{}
		pushNotifier.On("IsConfigured").Return(true)

		service := NewService(db, nil, pushNotifier)
		assert.True(t, service.IsPushAvailable())

		pushNotifier.AssertExpectations(t)
	})

	t.Run("not available when notifier not configured", func(t *testing.T) {
		pushNotifier := &MockNotifier{}
		pushNotifier.On("IsConfigured").Return(false)

		service := NewService(db, nil, pushNotifier)
		assert.False(t, service.IsPushAvailable())

		pushNotifier.AssertExpectations(t)
	})

	t.Run("not available when notifier is nil", func(t *testing.T) {
		service := NewService(db, nil, nil)
		assert.False(t, service.IsPushAvailable())
	})
}

func TestNotifyPendingEvent_EmailEnabled(t *testing.T) {
	db := database.NewTestDB(t)

	// Set up notification preferences
	err := db.UpdateEmailPrefs(true, "test@example.com")
	require.NoError(t, err)

	// Create mock email notifier
	emailNotifier := &MockNotifier{}
	emailNotifier.On("IsConfigured").Return(true)
	emailNotifier.On("Send", mock.Anything, mock.Anything, "test@example.com").Return(nil)

	service := NewService(db, emailNotifier, nil)

	event := &database.CalendarEvent{
		ID:    1,
		Title: "Test Event",
	}

	service.NotifyPendingEvent(context.Background(), event)

	emailNotifier.AssertExpectations(t)
	emailNotifier.AssertCalled(t, "Send", mock.Anything, event, "test@example.com")
}

func TestNotifyPendingEvent_EmailDisabled(t *testing.T) {
	db := database.NewTestDB(t)

	// Email disabled by default (or explicitly disable)
	err := db.UpdateEmailPrefs(false, "")
	require.NoError(t, err)

	emailNotifier := &MockNotifier{}
	// Should not be called

	service := NewService(db, emailNotifier, nil)

	event := &database.CalendarEvent{
		ID:    1,
		Title: "Test Event",
	}

	service.NotifyPendingEvent(context.Background(), event)

	// Email notifier should not be called
	emailNotifier.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotifyPendingEvent_PushEnabled(t *testing.T) {
	db := database.NewTestDB(t)

	// Set up push preferences
	err := db.UpdatePushPrefs(true)
	require.NoError(t, err)
	err = db.UpdatePushToken("ExponentPushToken[test-token-12345678]")
	require.NoError(t, err)

	// Create mock push notifier
	pushNotifier := &MockNotifier{}
	pushNotifier.On("IsConfigured").Return(true)
	pushNotifier.On("Send", mock.Anything, mock.Anything, "ExponentPushToken[test-token-12345678]").Return(nil)

	service := NewService(db, nil, pushNotifier)

	event := &database.CalendarEvent{
		ID:    1,
		Title: "Test Event",
	}

	service.NotifyPendingEvent(context.Background(), event)

	pushNotifier.AssertExpectations(t)
}

func TestNotifyPendingEvent_PushDisabled(t *testing.T) {
	db := database.NewTestDB(t)

	// Push disabled by default
	pushNotifier := &MockNotifier{}
	// Should not be called

	service := NewService(db, nil, pushNotifier)

	event := &database.CalendarEvent{
		ID:    1,
		Title: "Test Event",
	}

	service.NotifyPendingEvent(context.Background(), event)

	// Push notifier should not be called
	pushNotifier.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotifyPendingEvent_BothEnabled(t *testing.T) {
	db := database.NewTestDB(t)

	// Enable both
	err := db.UpdateEmailPrefs(true, "email@test.com")
	require.NoError(t, err)
	err = db.UpdatePushPrefs(true)
	require.NoError(t, err)
	err = db.UpdatePushToken("ExponentPushToken[both-enabled-token]")
	require.NoError(t, err)

	emailNotifier := &MockNotifier{}
	emailNotifier.On("IsConfigured").Return(true)
	emailNotifier.On("Send", mock.Anything, mock.Anything, "email@test.com").Return(nil)

	pushNotifier := &MockNotifier{}
	pushNotifier.On("IsConfigured").Return(true)
	pushNotifier.On("Send", mock.Anything, mock.Anything, "ExponentPushToken[both-enabled-token]").Return(nil)

	service := NewService(db, emailNotifier, pushNotifier)

	event := &database.CalendarEvent{
		ID:    1,
		Title: "Test Event",
	}

	service.NotifyPendingEvent(context.Background(), event)

	emailNotifier.AssertExpectations(t)
	pushNotifier.AssertExpectations(t)
}

func TestNotifyPendingEvent_NotifierNotConfigured(t *testing.T) {
	db := database.NewTestDB(t)

	// Email enabled but notifier not configured
	err := db.UpdateEmailPrefs(true, "test@example.com")
	require.NoError(t, err)

	emailNotifier := &MockNotifier{}
	emailNotifier.On("IsConfigured").Return(false) // Not configured

	service := NewService(db, emailNotifier, nil)

	event := &database.CalendarEvent{
		ID:    1,
		Title: "Test Event",
	}

	service.NotifyPendingEvent(context.Background(), event)

	// Send should not be called because notifier is not configured
	emailNotifier.AssertNotCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything)
}

func TestNotifyPendingEvent_NilNotifiers(t *testing.T) {
	db := database.NewTestDB(t)

	// Enable preferences but notifiers are nil
	err := db.UpdateEmailPrefs(true, "test@example.com")
	require.NoError(t, err)
	err = db.UpdatePushPrefs(true)
	require.NoError(t, err)
	err = db.UpdatePushToken("ExponentPushToken[nil-notifier-test]")
	require.NoError(t, err)

	// Both notifiers are nil
	service := NewService(db, nil, nil)

	event := &database.CalendarEvent{
		ID:    1,
		Title: "Test Event",
	}

	// Should not panic
	service.NotifyPendingEvent(context.Background(), event)
}

func TestNotifyPendingEvent_WithRealEvent(t *testing.T) {
	db := database.NewTestDB(t)

	// Create a real channel first
	channel, err := db.CreateChannel(database.ChannelTypeSender, "notify-test@s.whatsapp.net", "Notify Test")
	require.NoError(t, err)

	// Create a real event
	endTime := time.Now().Add(time.Hour)
	event := &database.CalendarEvent{
		ChannelID:   channel.ID,
		CalendarID:  "primary",
		Title:       "Real Test Event",
		Description: "Event for testing notifications",
		StartTime:   time.Now(),
		EndTime:     &endTime,
		Location:    "Test Location",
		ActionType:  database.EventActionCreate,
	}
	created, err := db.CreatePendingEvent(event)
	require.NoError(t, err)

	// Set up email preferences
	err = db.UpdateEmailPrefs(true, "real@test.com")
	require.NoError(t, err)

	emailNotifier := &MockNotifier{}
	emailNotifier.On("IsConfigured").Return(true)
	emailNotifier.On("Send", mock.Anything, mock.Anything, "real@test.com").Return(nil)

	service := NewService(db, emailNotifier, nil)

	service.NotifyPendingEvent(context.Background(), created)

	emailNotifier.AssertExpectations(t)
	// Verify the event passed to Send has the correct title
	emailNotifier.AssertCalled(t, "Send", mock.Anything, mock.MatchedBy(func(e *database.CalendarEvent) bool {
		return e.Title == "Real Test Event"
	}), "real@test.com")
}
