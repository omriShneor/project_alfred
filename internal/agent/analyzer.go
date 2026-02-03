package agent

import (
	"context"

	"github.com/omriShneor/project_alfred/internal/database"
)

// EventAnalyzer is the interface for event analysis, providing backward compatibility
// between the old claude.Client and the new tool-calling agent
type EventAnalyzer interface {
	// AnalyzeMessages analyzes chat messages (WhatsApp, Telegram) for calendar events
	AnalyzeMessages(
		ctx context.Context,
		history []database.MessageRecord,
		newMessage database.MessageRecord,
		existingEvents []database.CalendarEvent,
	) (*EventAnalysis, error)

	// AnalyzeEmail analyzes an email for calendar events
	AnalyzeEmail(ctx context.Context, email EmailContent) (*EventAnalysis, error)

	// IsConfigured returns true if the analyzer is properly configured
	IsConfigured() bool
}

// EmailContent represents an email for analysis
// This duplicates the type to avoid circular imports
type EmailContent struct {
	Subject       string
	From          string
	To            string
	Date          string
	Body          string
	ThreadHistory []EmailThreadMessage
}

// EmailThreadMessage represents a message in thread history
type EmailThreadMessage struct {
	From    string
	Date    string
	Subject string
	Body    string
}
