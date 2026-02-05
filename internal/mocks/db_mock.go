package mocks

import (
	"context"
	"time"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/stretchr/testify/mock"
)

// MockDB is a mock implementation of database operations
type MockDB struct {
	mock.Mock
}

// Source Channels

func (m *MockDB) GetSourceChannelByID(userID int64, id int64) (*database.SourceChannel, error) {
	args := m.Called(userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.SourceChannel), args.Error(1)
}

func (m *MockDB) CreateSourceChannel(sourceType source.SourceType, channelType source.ChannelType, identifier, name string) (*database.SourceChannel, error) {
	args := m.Called(sourceType, channelType, identifier, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.SourceChannel), args.Error(1)
}

func (m *MockDB) GetSourceChannelByIdentifier(sourceType source.SourceType, identifier string) (*database.SourceChannel, error) {
	args := m.Called(sourceType, identifier)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.SourceChannel), args.Error(1)
}

func (m *MockDB) ListSourceChannels(sourceType source.SourceType) ([]*database.SourceChannel, error) {
	args := m.Called(sourceType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*database.SourceChannel), args.Error(1)
}

func (m *MockDB) ListEnabledSourceChannels(sourceType source.SourceType) ([]*database.SourceChannel, error) {
	args := m.Called(sourceType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*database.SourceChannel), args.Error(1)
}

func (m *MockDB) UpdateSourceChannel(userID int64, id int64, name string, enabled bool) error {
	args := m.Called(userID, id, name, enabled)
	return args.Error(0)
}

func (m *MockDB) DeleteSourceChannel(userID int64, id int64) error {
	args := m.Called(userID, id)
	return args.Error(0)
}

func (m *MockDB) IsSourceChannelTracked(sourceType source.SourceType, identifier string) (bool, int64, source.ChannelType, error) {
	args := m.Called(sourceType, identifier)
	return args.Bool(0), args.Get(1).(int64), args.Get(2).(source.ChannelType), args.Error(3)
}

// Source Messages

func (m *MockDB) StoreSourceMessage(sourceType source.SourceType, channelID int64, senderID, senderName, text, subject string, timestamp time.Time) (*database.SourceMessage, error) {
	args := m.Called(sourceType, channelID, senderID, senderName, text, subject, timestamp)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.SourceMessage), args.Error(1)
}

func (m *MockDB) GetSourceMessageHistory(userID int64, sourceType source.SourceType, channelID int64, limit int) ([]database.SourceMessage, error) {
	args := m.Called(userID, sourceType, channelID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.SourceMessage), args.Error(1)
}

func (m *MockDB) PruneSourceMessages(userID int64, sourceType source.SourceType, channelID int64, keepCount int) error {
	args := m.Called(userID, sourceType, channelID, keepCount)
	return args.Error(0)
}

func (m *MockDB) GetSourceMessageByID(userID int64, id int64) (*database.SourceMessage, error) {
	args := m.Called(userID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.SourceMessage), args.Error(1)
}

func (m *MockDB) CountSourceMessages(userID int64, sourceType source.SourceType, channelID int64) (int, error) {
	args := m.Called(userID, sourceType, channelID)
	return args.Int(0), args.Error(1)
}

// Events

func (m *MockDB) CreatePendingEvent(event *database.CalendarEvent) (*database.CalendarEvent, error) {
	args := m.Called(event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.CalendarEvent), args.Error(1)
}

func (m *MockDB) GetEventByID(id int64) (*database.CalendarEvent, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.CalendarEvent), args.Error(1)
}

func (m *MockDB) ListEvents(status *database.EventStatus, channelID *int64) ([]database.CalendarEvent, error) {
	args := m.Called(status, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.CalendarEvent), args.Error(1)
}

func (m *MockDB) UpdatePendingEvent(id int64, title, description string, startTime time.Time, endTime *time.Time, location string) error {
	args := m.Called(id, title, description, startTime, endTime, location)
	return args.Error(0)
}

func (m *MockDB) UpdateEventStatus(id int64, status database.EventStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockDB) UpdateEventGoogleID(id int64, googleEventID string) error {
	args := m.Called(id, googleEventID)
	return args.Error(0)
}

func (m *MockDB) DeleteEvent(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDB) GetActiveEventsForChannel(channelID int64) ([]database.CalendarEvent, error) {
	args := m.Called(channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.CalendarEvent), args.Error(1)
}

func (m *MockDB) GetSelectedCalendarID(userID int64) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockDB) GetUserNotificationPrefs() (*database.UserNotificationPrefs, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.UserNotificationPrefs), args.Error(1)
}

// Exec for raw SQL (used in event_creator.go)
func (m *MockDB) Exec(query string, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(append([]interface{}{query}, args...)...)
	return callArgs.Get(0), callArgs.Error(1)
}

// Unused context parameter to match interfaces
var _ = context.Background
