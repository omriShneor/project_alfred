package intents

import (
	"context"
	"fmt"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
)

// MessageInput contains message-based context used by intent modules.
type MessageInput struct {
	History           []database.MessageRecord
	NewMessage        database.MessageRecord
	ExistingEvents    []database.CalendarEvent
	ExistingReminders []database.Reminder
}

// EmailInput contains email-based context used by intent modules.
type EmailInput struct {
	Email agent.EmailContent
}

// ModuleOutput is a normalized decision payload produced by an intent module.
type ModuleOutput struct {
	Intent           string
	Action           string
	Confidence       float64
	Reasoning        string
	EventAnalysis    *agent.EventAnalysis
	ReminderAnalysis *agent.ReminderAnalysis
}

// Persister is implemented by orchestrators to persist module outputs.
type Persister interface {
	PersistEvent(ctx context.Context, analysis *agent.EventAnalysis) error
	PersistReminder(ctx context.Context, analysis *agent.ReminderAnalysis) error
}

// IntentModule defines a pluggable intent analyzer.
type IntentModule interface {
	IntentName() string
	AnalyzeMessages(ctx context.Context, in MessageInput) (*ModuleOutput, error)
	AnalyzeEmail(ctx context.Context, in EmailInput) (*ModuleOutput, error)
	Validate(ctx context.Context, out *ModuleOutput) error
	Persist(ctx context.Context, out *ModuleOutput, persister Persister) error
}

// EventModule adapts an EventAnalyzer into an IntentModule.
type EventModule struct {
	Analyzer agent.EventAnalyzer
}

func (m *EventModule) IntentName() string { return "event" }

func (m *EventModule) AnalyzeMessages(ctx context.Context, in MessageInput) (*ModuleOutput, error) {
	if m.Analyzer == nil {
		return nil, fmt.Errorf("event analyzer is not configured")
	}

	analysis, err := m.Analyzer.AnalyzeMessages(ctx, in.History, in.NewMessage, in.ExistingEvents)
	if err != nil {
		return nil, err
	}

	return &ModuleOutput{
		Intent:        "event",
		Action:        analysis.Action,
		Confidence:    analysis.Confidence,
		Reasoning:     analysis.Reasoning,
		EventAnalysis: analysis,
	}, nil
}

func (m *EventModule) AnalyzeEmail(ctx context.Context, in EmailInput) (*ModuleOutput, error) {
	if m.Analyzer == nil {
		return nil, fmt.Errorf("event analyzer is not configured")
	}

	analysis, err := m.Analyzer.AnalyzeEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}

	return &ModuleOutput{
		Intent:        "event",
		Action:        analysis.Action,
		Confidence:    analysis.Confidence,
		Reasoning:     analysis.Reasoning,
		EventAnalysis: analysis,
	}, nil
}

func (m *EventModule) Validate(_ context.Context, out *ModuleOutput) error {
	if out == nil || out.EventAnalysis == nil {
		return fmt.Errorf("event output is nil")
	}
	if out.EventAnalysis.Action == "none" || !out.EventAnalysis.HasEvent {
		return nil
	}
	switch out.EventAnalysis.Action {
	case "create":
		if out.EventAnalysis.Event == nil || out.EventAnalysis.Event.StartTime == "" || out.EventAnalysis.Event.Title == "" {
			return fmt.Errorf("event create action requires title and start_time")
		}
	case "update":
		if out.EventAnalysis.Event == nil {
			return fmt.Errorf("event update action requires event payload")
		}
		if out.EventAnalysis.Event.AlfredEventRef == 0 &&
			out.EventAnalysis.Event.UpdateRef == "" &&
			out.EventAnalysis.Event.StartTime == "" &&
			out.EventAnalysis.Event.Title == "" &&
			out.EventAnalysis.Event.Description == "" &&
			out.EventAnalysis.Event.Location == "" &&
			out.EventAnalysis.Event.EndTime == "" {
			return fmt.Errorf("event update action has no target and no patch fields")
		}
	case "delete":
		if out.EventAnalysis.Event == nil {
			return fmt.Errorf("event delete action requires event payload")
		}
		if out.EventAnalysis.Event.AlfredEventRef == 0 && out.EventAnalysis.Event.UpdateRef == "" {
			return fmt.Errorf("event delete action requires event reference")
		}
	default:
		return fmt.Errorf("unknown event action: %s", out.EventAnalysis.Action)
	}
	return nil
}

func (m *EventModule) Persist(ctx context.Context, out *ModuleOutput, persister Persister) error {
	if out == nil || out.EventAnalysis == nil {
		return fmt.Errorf("event output is nil")
	}
	if out.EventAnalysis.Action == "none" || !out.EventAnalysis.HasEvent {
		return nil
	}
	return persister.PersistEvent(ctx, out.EventAnalysis)
}

// ReminderModule adapts a ReminderAnalyzer into an IntentModule.
type ReminderModule struct {
	Analyzer agent.ReminderAnalyzer
}

func (m *ReminderModule) IntentName() string { return "reminder" }

func (m *ReminderModule) AnalyzeMessages(ctx context.Context, in MessageInput) (*ModuleOutput, error) {
	if m.Analyzer == nil {
		return nil, fmt.Errorf("reminder analyzer is not configured")
	}

	analysis, err := m.Analyzer.AnalyzeMessages(ctx, in.History, in.NewMessage, in.ExistingReminders)
	if err != nil {
		return nil, err
	}

	return &ModuleOutput{
		Intent:           "reminder",
		Action:           analysis.Action,
		Confidence:       analysis.Confidence,
		Reasoning:        analysis.Reasoning,
		ReminderAnalysis: analysis,
	}, nil
}

func (m *ReminderModule) AnalyzeEmail(ctx context.Context, in EmailInput) (*ModuleOutput, error) {
	if m.Analyzer == nil {
		return nil, fmt.Errorf("reminder analyzer is not configured")
	}

	analysis, err := m.Analyzer.AnalyzeEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}

	return &ModuleOutput{
		Intent:           "reminder",
		Action:           analysis.Action,
		Confidence:       analysis.Confidence,
		Reasoning:        analysis.Reasoning,
		ReminderAnalysis: analysis,
	}, nil
}

func (m *ReminderModule) Validate(_ context.Context, out *ModuleOutput) error {
	if out == nil || out.ReminderAnalysis == nil {
		return fmt.Errorf("reminder output is nil")
	}
	if out.ReminderAnalysis.Action == "none" || !out.ReminderAnalysis.HasReminder {
		return nil
	}
	switch out.ReminderAnalysis.Action {
	case "create":
		if out.ReminderAnalysis.Reminder == nil || out.ReminderAnalysis.Reminder.Title == "" || out.ReminderAnalysis.Reminder.DueDate == "" {
			return fmt.Errorf("reminder create action requires title and due_date")
		}
	case "update", "delete":
		if out.ReminderAnalysis.Reminder == nil || out.ReminderAnalysis.Reminder.AlfredReminderRef == 0 {
			return fmt.Errorf("reminder %s action requires alfred_reminder_id", out.ReminderAnalysis.Action)
		}
	default:
		return fmt.Errorf("unknown reminder action: %s", out.ReminderAnalysis.Action)
	}
	return nil
}

func (m *ReminderModule) Persist(ctx context.Context, out *ModuleOutput, persister Persister) error {
	if out == nil || out.ReminderAnalysis == nil {
		return fmt.Errorf("reminder output is nil")
	}
	if out.ReminderAnalysis.Action == "none" || !out.ReminderAnalysis.HasReminder {
		return nil
	}
	return persister.PersistReminder(ctx, out.ReminderAnalysis)
}

