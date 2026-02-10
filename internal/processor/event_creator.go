package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/omriShneor/project_alfred/internal/agent"
	"github.com/omriShneor/project_alfred/internal/database"
	"github.com/omriShneor/project_alfred/internal/notify"
	"github.com/omriShneor/project_alfred/internal/source"
	"github.com/omriShneor/project_alfred/internal/timeutil"
)

// EventCreationParams contains parameters for creating a calendar event from analysis
type EventCreationParams struct {
	// User info
	UserID int64

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

	actionType, err := mapActionType(params.Analysis.Action)
	if err != nil {
		return nil, err
	}

	userTimezone, _ := ec.db.GetUserTimezone(params.UserID)
	if userTimezone == "" {
		userTimezone = "UTC"
	}

	// Handle update/delete of existing pending event
	if params.ExistingEvent != nil && params.ExistingEvent.Status == database.EventStatusPending {
		return ec.handleExistingPendingEvent(params.ExistingEvent, params.Analysis, userTimezone)
	}

	existingRefEvent, err := ec.resolveEventReference(params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve event reference: %w", err)
	}

	startTime, endTime, timezoneFallback, err := ec.resolveEventTimes(params.Analysis, actionType, existingRefEvent, userTimezone)
	if err != nil {
		return nil, err
	}

	// For updates/deletes of synced events, store the Google event ID reference.
	var googleEventID *string
	if params.Analysis.Event.UpdateRef != "" {
		googleEventID = &params.Analysis.Event.UpdateRef
	} else if existingRefEvent != nil && existingRefEvent.GoogleEventID != nil {
		googleEventID = existingRefEvent.GoogleEventID
	}

	// Get calendar ID if not provided
	calendarID := params.CalendarID
	if calendarID == "" {
		calendarID, _ = ec.db.GetSelectedCalendarID(params.UserID)
	}
	if calendarID == "" && existingRefEvent != nil {
		calendarID = existingRefEvent.CalendarID
	}
	if calendarID == "" {
		calendarID = "primary"
	}

	title := strings.TrimSpace(params.Analysis.Event.Title)
	description := strings.TrimSpace(params.Analysis.Event.Description)
	location := strings.TrimSpace(params.Analysis.Event.Location)
	if existingRefEvent != nil {
		if title == "" {
			title = existingRefEvent.Title
		}
		if description == "" {
			description = existingRefEvent.Description
		}
		if location == "" {
			location = existingRefEvent.Location
		}
	}
	if title == "" {
		title = "Untitled event"
	}

	qualityFlags := buildQualityFlags(params.Analysis.Confidence, timezoneFallback)

	event := &database.CalendarEvent{
		UserID:        params.UserID,
		ChannelID:     params.ChannelID,
		GoogleEventID: googleEventID,
		CalendarID:    calendarID,
		Title:         title,
		Description:   description,
		StartTime:     startTime,
		EndTime:       endTime,
		Location:      location,
		ActionType:    actionType,
		OriginalMsgID: params.MessageID,
		LLMReasoning:  params.Analysis.Reasoning,
		LLMConfidence: params.Analysis.Confidence,
		QualityFlags:  qualityFlags,
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

	if err := ec.persistEventAttendees(created.ID, params.Analysis.Event); err != nil {
		return nil, fmt.Errorf("failed to persist event attendees: %w", err)
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
	userTimezone string,
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

	startTime, endTime, timezoneFallback, err := ec.resolveEventTimes(
		analysis,
		database.EventActionUpdate,
		existing,
		userTimezone,
	)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(analysis.Event.Title)
	description := strings.TrimSpace(analysis.Event.Description)
	location := strings.TrimSpace(analysis.Event.Location)
	if title == "" {
		title = existing.Title
	}
	if description == "" {
		description = existing.Description
	}
	if location == "" {
		location = existing.Location
	}

	// Update the existing pending event
	if err := ec.db.UpdatePendingEvent(
		existing.ID,
		title,
		description,
		startTime,
		endTime,
		location,
	); err != nil {
		return nil, fmt.Errorf("failed to update pending event: %w", err)
	}

	_, _ = ec.db.Exec(`
		UPDATE calendar_events
		SET llm_reasoning = ?, llm_confidence = ?, quality_flags = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, analysis.Reasoning, analysis.Confidence, qualityFlagsJSON(buildQualityFlags(analysis.Confidence, timezoneFallback)), existing.ID)

	if err := ec.persistEventAttendees(existing.ID, analysis.Event); err != nil {
		return nil, fmt.Errorf("failed to update event attendees: %w", err)
	}

	fmt.Printf("Updated pending event: %s (ID: %d)\n",
		title, existing.ID)

	// Return the updated event
	updated, _ := ec.db.GetEventByID(existing.ID)
	if updated != nil {
		return updated, nil
	}
	return existing, nil
}

func (ec *EventCreator) resolveEventReference(params EventCreationParams) (*database.CalendarEvent, error) {
	if params.Analysis == nil || params.Analysis.Event == nil {
		return nil, nil
	}

	event := params.Analysis.Event
	if event.AlfredEventRef != 0 {
		ref, err := ec.db.GetEventByID(event.AlfredEventRef)
		if err != nil {
			return nil, err
		}
		return ref, nil
	}
	if event.UpdateRef != "" {
		ref, err := ec.db.GetEventByGoogleIDForUser(params.UserID, event.UpdateRef)
		if err != nil {
			return nil, err
		}
		return ref, nil
	}
	return nil, nil
}

func (ec *EventCreator) resolveEventTimes(
	analysis *agent.EventAnalysis,
	actionType database.EventActionType,
	base *database.CalendarEvent,
	userTimezone string,
) (time.Time, *time.Time, bool, error) {
	if analysis == nil || analysis.Event == nil {
		return time.Time{}, nil, false, fmt.Errorf("analysis has no event data")
	}

	event := analysis.Event
	timezoneFallback := false
	parseWithTZ := func(raw string) (time.Time, error) {
		t, usedFallback, err := timeutil.ParseDateTime(raw, userTimezone)
		if usedFallback {
			timezoneFallback = true
		}
		return t, err
	}

	switch actionType {
	case database.EventActionCreate:
		if strings.TrimSpace(event.StartTime) == "" {
			return time.Time{}, nil, false, fmt.Errorf("create action requires start_time")
		}

		startTime, err := parseWithTZ(event.StartTime)
		if err != nil {
			return time.Time{}, nil, timezoneFallback, fmt.Errorf("failed to parse start time: %w", err)
		}

		var endTime *time.Time
		if strings.TrimSpace(event.EndTime) != "" {
			et, err := parseWithTZ(event.EndTime)
			if err == nil {
				endTime = &et
			}
		}
		if endTime == nil {
			et := startTime.Add(time.Hour)
			endTime = &et
		}
		return startTime, endTime, timezoneFallback, nil

	case database.EventActionUpdate:
		startProvided := strings.TrimSpace(event.StartTime) != ""
		endProvided := strings.TrimSpace(event.EndTime) != ""

		var startTime time.Time
		if startProvided {
			st, err := parseWithTZ(event.StartTime)
			if err != nil {
				return time.Time{}, nil, timezoneFallback, fmt.Errorf("failed to parse start time: %w", err)
			}
			startTime = st
		} else if base != nil {
			startTime = base.StartTime
		} else {
			return time.Time{}, nil, timezoneFallback, fmt.Errorf("update action requires start_time or existing event reference")
		}

		var endTime *time.Time
		if endProvided {
			et, err := parseWithTZ(event.EndTime)
			if err == nil {
				endTime = &et
			}
		}

		if endTime == nil {
			if startProvided && base != nil && base.EndTime != nil {
				duration := base.EndTime.Sub(base.StartTime)
				if duration > 0 {
					et := startTime.Add(duration)
					endTime = &et
				}
			}
			if endTime == nil && base != nil && !startProvided {
				endTime = base.EndTime
			}
			if endTime == nil {
				et := startTime.Add(time.Hour)
				endTime = &et
			}
		}
		return startTime, endTime, timezoneFallback, nil

	case database.EventActionDelete:
		if base != nil {
			var endCopy *time.Time
			if base.EndTime != nil {
				t := *base.EndTime
				endCopy = &t
			}
			return base.StartTime, endCopy, timezoneFallback, nil
		}
		// Delete needs no time parse semantically; use now for non-null DB constraint.
		now := time.Now().UTC()
		end := now.Add(time.Hour)
		return now, &end, timezoneFallback, nil
	}

	return time.Time{}, nil, timezoneFallback, fmt.Errorf("unknown action type: %s", actionType)
}

func (ec *EventCreator) persistEventAttendees(eventID int64, event *agent.EventData) error {
	if event == nil {
		return nil
	}

	if len(event.Attendees) == 0 {
		return nil
	}

	attendees := make([]database.Attendee, 0, len(event.Attendees))
	for _, attendee := range event.Attendees {
		email := strings.TrimSpace(attendee.Email)
		if email == "" {
			continue
		}
		displayName := strings.TrimSpace(attendee.Name)
		attendees = append(attendees, database.Attendee{
			Email:       email,
			DisplayName: displayName,
			Optional:    strings.EqualFold(strings.TrimSpace(attendee.Role), "optional"),
		})
	}

	if len(attendees) == 0 {
		return nil
	}
	return ec.db.SetEventAttendees(eventID, attendees)
}

func buildQualityFlags(confidence float64, timezoneFallback bool) []string {
	flags := make([]string, 0, 2)
	if confidence < 0.6 {
		flags = append(flags, "low_confidence")
	}
	if timezoneFallback {
		flags = append(flags, "timezone_fallback")
	}
	return flags
}

func qualityFlagsJSON(flags []string) string {
	if len(flags) == 0 {
		return "[]"
	}
	b, err := json.Marshal(flags)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// parseEventTimes is kept for backward compatibility in tests.
func parseEventTimes(event *agent.EventData) (startTime time.Time, endTime *time.Time, err error) {
	analysis := &agent.EventAnalysis{Event: event}
	creator := &EventCreator{}
	startTime, endTime, _, err = creator.resolveEventTimes(analysis, database.EventActionCreate, nil, "UTC")
	return startTime, endTime, err
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
