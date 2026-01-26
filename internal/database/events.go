package database

import (
	"database/sql"
	"fmt"
	"time"
)

// EventStatus represents the status of a calendar event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusConfirmed EventStatus = "confirmed"
	EventStatusSynced    EventStatus = "synced"
	EventStatusRejected  EventStatus = "rejected"
	EventStatusDeleted   EventStatus = "deleted"
)

// EventActionType represents the type of action for an event
type EventActionType string

const (
	EventActionCreate EventActionType = "create"
	EventActionUpdate EventActionType = "update"
	EventActionDelete EventActionType = "delete"
)

// CalendarEvent represents a detected calendar event
type CalendarEvent struct {
	ID              int64           `json:"id"`
	ChannelID       int64           `json:"channel_id"`
	GoogleEventID   *string         `json:"google_event_id,omitempty"`
	CalendarID      string          `json:"calendar_id"`
	Title           string          `json:"title"`
	Description     string          `json:"description,omitempty"`
	StartTime       time.Time       `json:"start_time"`
	EndTime         *time.Time      `json:"end_time,omitempty"`
	Location        string          `json:"location,omitempty"`
	Status          EventStatus     `json:"status"`
	ActionType      EventActionType `json:"action_type"`
	OriginalMsgID   *int64          `json:"original_message_id,omitempty"`
	LLMReasoning    string          `json:"llm_reasoning,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	ChannelName     string          `json:"channel_name,omitempty"` // Joined from channels table
	Attendees       []Attendee      `json:"attendees,omitempty"`    // Participants for this event
}

// CreatePendingEvent creates a new pending event in the database
func (d *DB) CreatePendingEvent(event *CalendarEvent) (*CalendarEvent, error) {
	result, err := d.Exec(`
		INSERT INTO calendar_events (
			channel_id, google_event_id, calendar_id, title, description,
			start_time, end_time, location, status, action_type,
			original_message_id, llm_reasoning
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.ChannelID, event.GoogleEventID, event.CalendarID, event.Title, event.Description,
		event.StartTime, event.EndTime, event.Location, EventStatusPending, event.ActionType,
		event.OriginalMsgID, event.LLMReasoning,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get event id: %w", err)
	}

	event.ID = id
	event.Status = EventStatusPending
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	return event, nil
}

// GetEventByID retrieves an event by its ID
func (d *DB) GetEventByID(id int64) (*CalendarEvent, error) {
	var event CalendarEvent
	var googleEventID sql.NullString
	var endTimeNull sql.NullTime
	var origMsgIDNull sql.NullInt64

	err := d.QueryRow(`
		SELECT e.id, e.channel_id, e.google_event_id, e.calendar_id, e.title,
			e.description, e.start_time, e.end_time, e.location, e.status,
			e.action_type, e.original_message_id, e.llm_reasoning, e.created_at, e.updated_at,
			c.name as channel_name
		FROM calendar_events e
		JOIN channels c ON e.channel_id = c.id
		WHERE e.id = ?
	`, id).Scan(
		&event.ID, &event.ChannelID, &googleEventID, &event.CalendarID, &event.Title,
		&event.Description, &event.StartTime, &endTimeNull, &event.Location, &event.Status,
		&event.ActionType, &origMsgIDNull, &event.LLMReasoning, &event.CreatedAt, &event.UpdatedAt,
		&event.ChannelName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if googleEventID.Valid {
		event.GoogleEventID = &googleEventID.String
	}
	if endTimeNull.Valid {
		event.EndTime = &endTimeNull.Time
	}
	if origMsgIDNull.Valid {
		event.OriginalMsgID = &origMsgIDNull.Int64
	}

	// Fetch attendees for this event
	attendees, err := d.GetEventAttendees(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get event attendees: %w", err)
	}
	event.Attendees = attendees

	return &event, nil
}

// GetEventByGoogleID retrieves an event by its Google Calendar event ID
func (d *DB) GetEventByGoogleID(googleEventID string) (*CalendarEvent, error) {
	var event CalendarEvent
	var gEventID sql.NullString
	var endTimeNull sql.NullTime
	var origMsgIDNull sql.NullInt64

	err := d.QueryRow(`
		SELECT e.id, e.channel_id, e.google_event_id, e.calendar_id, e.title,
			e.description, e.start_time, e.end_time, e.location, e.status,
			e.action_type, e.original_message_id, e.llm_reasoning, e.created_at, e.updated_at,
			c.name as channel_name
		FROM calendar_events e
		JOIN channels c ON e.channel_id = c.id
		WHERE e.google_event_id = ?
	`, googleEventID).Scan(
		&event.ID, &event.ChannelID, &gEventID, &event.CalendarID, &event.Title,
		&event.Description, &event.StartTime, &endTimeNull, &event.Location, &event.Status,
		&event.ActionType, &origMsgIDNull, &event.LLMReasoning, &event.CreatedAt, &event.UpdatedAt,
		&event.ChannelName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get event by google id: %w", err)
	}

	if gEventID.Valid {
		event.GoogleEventID = &gEventID.String
	}
	if endTimeNull.Valid {
		event.EndTime = &endTimeNull.Time
	}
	if origMsgIDNull.Valid {
		event.OriginalMsgID = &origMsgIDNull.Int64
	}

	return &event, nil
}

// ListEvents retrieves events with optional filtering by status and channel
func (d *DB) ListEvents(status *EventStatus, channelID *int64) ([]CalendarEvent, error) {
	query := `
		SELECT e.id, e.channel_id, e.google_event_id, e.calendar_id, e.title,
			e.description, e.start_time, e.end_time, e.location, e.status,
			e.action_type, e.original_message_id, e.llm_reasoning, e.created_at, e.updated_at,
			c.name as channel_name
		FROM calendar_events e
		JOIN channels c ON e.channel_id = c.id
		WHERE 1=1
	`
	args := []interface{}{}

	if status != nil {
		query += " AND e.status = ?"
		args = append(args, *status)
	}

	if channelID != nil {
		query += " AND e.channel_id = ?"
		args = append(args, *channelID)
	}

	query += " ORDER BY e.created_at DESC"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var event CalendarEvent
		var googleEventID sql.NullString
		var endTimeNull sql.NullTime
		var origMsgIDNull sql.NullInt64

		if err := rows.Scan(
			&event.ID, &event.ChannelID, &googleEventID, &event.CalendarID, &event.Title,
			&event.Description, &event.StartTime, &endTimeNull, &event.Location, &event.Status,
			&event.ActionType, &origMsgIDNull, &event.LLMReasoning, &event.CreatedAt, &event.UpdatedAt,
			&event.ChannelName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if googleEventID.Valid {
			event.GoogleEventID = &googleEventID.String
		}
		if endTimeNull.Valid {
			event.EndTime = &endTimeNull.Time
		}
		if origMsgIDNull.Valid {
			event.OriginalMsgID = &origMsgIDNull.Int64
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	// Fetch attendees for each event
	for i := range events {
		attendees, err := d.GetEventAttendees(events[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get attendees for event %d: %w", events[i].ID, err)
		}
		events[i].Attendees = attendees
	}

	return events, nil
}

// GetPendingEvents retrieves all pending events, optionally filtered by channel
func (d *DB) GetPendingEvents(channelID *int64) ([]CalendarEvent, error) {
	status := EventStatusPending
	return d.ListEvents(&status, channelID)
}

// ListEventsByChannel retrieves all events for a specific channel
func (d *DB) ListEventsByChannel(channelID int64) ([]CalendarEvent, error) {
	return d.ListEvents(nil, &channelID)
}

// UpdatePendingEvent updates a pending event's details (title, description, start_time, end_time, location)
func (d *DB) UpdatePendingEvent(id int64, title, description string, startTime time.Time, endTime *time.Time, location string) error {
	_, err := d.Exec(`
		UPDATE calendar_events
		SET title = ?, description = ?, start_time = ?, end_time = ?, location = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = ?
	`, title, description, startTime, endTime, location, id, EventStatusPending)
	if err != nil {
		return fmt.Errorf("failed to update pending event: %w", err)
	}
	return nil
}

// UpdateEventStatus updates the status of an event
func (d *DB) UpdateEventStatus(id int64, status EventStatus) error {
	_, err := d.Exec(`
		UPDATE calendar_events
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, id)
	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}
	return nil
}

// UpdateEventGoogleID sets the Google Calendar event ID after syncing
func (d *DB) UpdateEventGoogleID(id int64, googleEventID string) error {
	_, err := d.Exec(`
		UPDATE calendar_events
		SET google_event_id = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, googleEventID, EventStatusSynced, id)
	if err != nil {
		return fmt.Errorf("failed to update google event id: %w", err)
	}
	return nil
}

// DeleteEvent removes an event from the database
func (d *DB) DeleteEvent(id int64) error {
	_, err := d.Exec(`DELETE FROM calendar_events WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// GetExistingEventsForChannel retrieves synced events for a channel (used for Claude context)
// Deprecated: Use GetActiveEventsForChannel instead which includes pending events
func (d *DB) GetExistingEventsForChannel(channelID int64) ([]CalendarEvent, error) {
	status := EventStatusSynced
	return d.ListEvents(&status, &channelID)
}

// GetActiveEventsForChannel retrieves both pending and synced events for a channel
// This is used for Claude context so it can reference and update pending events
func (d *DB) GetActiveEventsForChannel(channelID int64) ([]CalendarEvent, error) {
	query := `
		SELECT e.id, e.channel_id, e.google_event_id, e.calendar_id, e.title,
			e.description, e.start_time, e.end_time, e.location, e.status,
			e.action_type, e.original_message_id, e.llm_reasoning, e.created_at, e.updated_at,
			c.name as channel_name
		FROM calendar_events e
		JOIN channels c ON e.channel_id = c.id
		WHERE e.channel_id = ? AND e.status IN (?, ?)
		ORDER BY e.start_time ASC
	`

	rows, err := d.Query(query, channelID, EventStatusPending, EventStatusSynced)
	if err != nil {
		return nil, fmt.Errorf("failed to list active events: %w", err)
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var event CalendarEvent
		var googleEventID sql.NullString
		var endTimeNull sql.NullTime
		var origMsgIDNull sql.NullInt64

		if err := rows.Scan(
			&event.ID, &event.ChannelID, &googleEventID, &event.CalendarID, &event.Title,
			&event.Description, &event.StartTime, &endTimeNull, &event.Location, &event.Status,
			&event.ActionType, &origMsgIDNull, &event.LLMReasoning, &event.CreatedAt, &event.UpdatedAt,
			&event.ChannelName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if googleEventID.Valid {
			event.GoogleEventID = &googleEventID.String
		}
		if endTimeNull.Valid {
			event.EndTime = &endTimeNull.Time
		}
		if origMsgIDNull.Valid {
			event.OriginalMsgID = &origMsgIDNull.Int64
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	// Fetch attendees for each event
	for i := range events {
		attendees, err := d.GetEventAttendees(events[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get attendees for event %d: %w", events[i].ID, err)
		}
		events[i].Attendees = attendees
	}

	return events, nil
}

// CountPendingEvents returns the number of pending events
func (d *DB) CountPendingEvents() (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM calendar_events WHERE status = ?`, EventStatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending events: %w", err)
	}
	return count, nil
}
