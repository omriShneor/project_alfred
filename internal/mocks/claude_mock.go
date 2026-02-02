package mocks

import (
	"context"

	"github.com/omriShneor/project_alfred/internal/claude"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/stretchr/testify/mock"
)

// MockClaudeClient is a mock implementation of the Claude client
type MockClaudeClient struct {
	mock.Mock
}

func (m *MockClaudeClient) AnalyzeMessages(
	ctx context.Context,
	history []database.MessageRecord,
	newMessage database.MessageRecord,
	existingEvents []database.CalendarEvent,
) (*claude.EventAnalysis, error) {
	args := m.Called(ctx, history, newMessage, existingEvents)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*claude.EventAnalysis), args.Error(1)
}

func (m *MockClaudeClient) AnalyzeEmail(ctx context.Context, email claude.EmailContent) (*claude.EventAnalysis, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*claude.EventAnalysis), args.Error(1)
}

func (m *MockClaudeClient) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}
