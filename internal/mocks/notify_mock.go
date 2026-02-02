package mocks

import (
	"context"

	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/stretchr/testify/mock"
)

// MockNotifyService is a mock implementation of the notification service
type MockNotifyService struct {
	mock.Mock
}

func (m *MockNotifyService) NotifyPendingEvent(ctx context.Context, event *database.CalendarEvent) {
	m.Called(ctx, event)
}

func (m *MockNotifyService) IsEmailAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockNotifyService) IsPushAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockNotifier is a mock implementation of the Notifier interface
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
