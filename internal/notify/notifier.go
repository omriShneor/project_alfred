package notify

import (
	"context"

	"github.com/omriShneor/project_alfred/internal/database"
)

// Notifier sends notifications for events to a specific recipient
type Notifier interface {
	// Send sends a notification for an event to the specified recipient
	Send(ctx context.Context, event *database.CalendarEvent, recipient string) error
	// Name returns the notifier type name (for logging)
	Name() string
	// IsConfigured returns true if the notifier has server-side config
	IsConfigured() bool
}
