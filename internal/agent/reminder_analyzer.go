package agent

import (
	"context"

	"github.com/omriShneor/project_alfred/internal/database"
)

// ReminderAnalyzer is the interface for reminder detection
// It runs independently from the event Analyzer
type ReminderAnalyzer interface {
	// AnalyzeMessages analyzes chat messages for reminders
	AnalyzeMessages(
		ctx context.Context,
		history []database.MessageRecord,
		newMessage database.MessageRecord,
		existingReminders []database.Reminder,
	) (*ReminderAnalysis, error)

	// AnalyzeEmail analyzes an email for reminders
	AnalyzeEmail(ctx context.Context, email EmailContent) (*ReminderAnalysis, error)

	// IsConfigured returns true if the analyzer is properly configured
	IsConfigured() bool
}

// ReminderAnalysis represents the result of reminder analysis
type ReminderAnalysis struct {
	HasReminder bool          `json:"has_reminder"`
	Action      string        `json:"action"` // "create", "update", "delete", "none"
	Reminder    *ReminderData `json:"reminder,omitempty"`
	Reasoning   string        `json:"reasoning"`
	Confidence  float64       `json:"confidence"`
}

// ReminderData contains the extracted reminder details
type ReminderData struct {
	Title             string `json:"title"`
	Description       string `json:"description,omitempty"`
	DueDate           string `json:"due_date"`            // ISO 8601 format
	ReminderTime      string `json:"reminder_time,omitempty"` // When to notify (optional)
	Priority          string `json:"priority,omitempty"`  // low, normal, high
	AlfredReminderRef int64  `json:"alfred_reminder_ref,omitempty"` // Internal DB ID for pending reminders
}
