package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
)

// EventCreationParams contains parameters for creating a calendar event from analysis
type EventCreationParams struct {
	// Channel info
	ChannelID  int64
	CalendarID string // If empty, will be looked up from settings

	// Source tracking
	SourceType    source.SourceType
	EmailSourceID *int64 // Only for gmail sources
	MessageID     *int64 // Reference to triggering message

	// From event analysis
	Analysis *agent.EventAnalysis

	// For update/delete of pending events (chat-specific)
	ExistingEvent *database.CalendarEvent
}

// EventCreator handles shared event creation logic
type EventCreator struct {
	db            *database.DB
	notifyService *notify.Service
}

// NewEventCreator creates a new EventCreator
func NewEventCreator(db *database.DB, notifyService *notify.Service) *EventCreator {
	return &EventCreator{
		db:            db,
		notifyService: notifyService,
	}
}

// CreateEventFromAnalysis creates or updates a pending event from Claude's analysis
func (ec *EventCreator) CreateEventFromAnalysis(ctx context.Context, params EventCreationParams) (*database.CalendarEvent, error) {
	if params.Analysis == nil || params.Analysis.Event == nil {
		return nil, fmt.Errorf("analysis has no event data")
	}

	// Parse times
	startTime, endTime, err := parseEventTimes(params.Analysis.Event)
	if err != nil {
		return nil, err
	}

	// Handle update/delete of existing pending event
	if params.ExistingEvent != nil && params.ExistingEvent.Status == database.EventStatusPending {
		return ec.handleExistingPendingEvent(params.ExistingEvent, params.Analysis, startTime, endTime)
	}

	// Determine action type
	actionType, err := mapActionType(params.Analysis.Action)
	if err != nil {
		return nil, err
	}

	// For updates/deletes of synced events, store the Google event ID reference
	var googleEventID *string
	if params.Analysis.Event.UpdateRef != "" {
		googleEventID = &params.Analysis.Event.UpdateRef
	}

	// Get calendar ID if not provided
	calendarID := params.CalendarID
	if calendarID == "" {
		calendarID, _ = ec.db.GetSelectedCalendarID()
	}

	event := &database.CalendarEvent{
		ChannelID:     params.ChannelID,
		GoogleEventID: googleEventID,
		CalendarID:    calendarID,
		Title:         params.Analysis.Event.Title,
		Description:   params.Analysis.Event.Description,
		StartTime:     startTime,
		EndTime:       endTime,
		Location:      params.Analysis.Event.Location,
		ActionType:    actionType,
		OriginalMsgID: params.MessageID,
		LLMReasoning:  params.Analysis.Reasoning,
	}

	created, err := ec.db.CreatePendingEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to save event: %w", err)
	}

	// Update source type on the event
	if params.EmailSourceID != nil {
		_, _ = ec.db.Exec(`UPDATE calendar_events SET source = ?, email_source_id = ? WHERE id = ?`,
			params.SourceType, *params.EmailSourceID, created.ID)
	} else {
		_, _ = ec.db.Exec(`UPDATE calendar_events SET source = ? WHERE id = ?`,
			params.SourceType, created.ID)
	}

	fmt.Printf("Created pending event: %s (ID: %d, Action: %s, Source: %s)\n",
		created.Title, created.ID, created.ActionType, params.SourceType)

	// Send notification (non-blocking, don't fail event creation)
	if ec.notifyService != nil {
		go ec.notifyService.NotifyPendingEvent(context.Background(), created)
	}

	return created, nil
}

// handleExistingPendingEvent handles update/delete of an existing pending event
func (ec *EventCreator) handleExistingPendingEvent(
	existing *database.CalendarEvent,
	analysis *agent.EventAnalysis,
	startTime time.Time,
	endTime *time.Time,
) (*database.CalendarEvent, error) {
	// Handle delete action on pending event
	if analysis.Action == "delete" {
		if err := ec.db.UpdateEventStatus(existing.ID, database.EventStatusRejected); err != nil {
			return nil, fmt.Errorf("failed to reject pending event: %w", err)
		}
		fmt.Printf("Rejected pending event: %s (ID: %d) - user cancelled\n",
			existing.Title, existing.ID)
		return existing, nil
	}

	// Update the existing pending event
	if err := ec.db.UpdatePendingEvent(
		existing.ID,
		analysis.Event.Title,
		analysis.Event.Description,
		startTime,
		endTime,
		analysis.Event.Location,
	); err != nil {
		return nil, fmt.Errorf("failed to update pending event: %w", err)
	}
	fmt.Printf("Updated pending event: %s (ID: %d)\n",
		analysis.Event.Title, existing.ID)

	// Return the updated event
	updated, _ := ec.db.GetEventByID(existing.ID)
	if updated != nil {
		return updated, nil
	}
	return existing, nil
}

// parseEventTimes parses start and end times from the analysis
func parseEventTimes(event *agent.EventData) (startTime time.Time, endTime *time.Time, err error) {
	// Parse start time with RFC3339 + fallback
	startTime, err = time.Parse(time.RFC3339, event.StartTime)
	if err != nil {
		startTime, err = time.Parse("2006-01-02T15:04:05", event.StartTime)
		if err != nil {
			return time.Time{}, nil, fmt.Errorf("failed to parse start time: %w", err)
		}
	}

	// Parse end time if provided
	if event.EndTime != "" {
		et, parseErr := time.Parse(time.RFC3339, event.EndTime)
		if parseErr != nil {
			et, parseErr = time.Parse("2006-01-02T15:04:05", event.EndTime)
			if parseErr == nil {
				endTime = &et
			}
		} else {
			endTime = &et
		}
	}

	// Default to 1 hour if no end time
	if endTime == nil {
		et := startTime.Add(time.Hour)
		endTime = &et
	}

	return startTime, endTime, nil
}

// mapActionType converts action string to EventActionType
func mapActionType(action string) (database.EventActionType, error) {
	switch action {
	case "create":
		return database.EventActionCreate, nil
	case "update":
		return database.EventActionUpdate, nil
	case "delete":
		return database.EventActionDelete, nil
	default:
		return "", fmt.Errorf("unknown action type: %s", action)
	}
}
