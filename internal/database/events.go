package database

import (
	"database/sql"
	"fmt"
	"time"
)

type PendingEvent struct {
	ID         int64
	SenderJID  string
	SourceType string // "sender" or "group"
	SourceID   int64
	EventJSON  string
	Status     string // pending, confirmed, rejected, timeout
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

type EventSourceMessage struct {
	ID             int64
	SenderJID      string
	SourceType     string
	SourceID       int64
	MessageText    string
	PendingEventID int64
	CreatedAt      time.Time
}

type CalendarEvent struct {
	ID             int64
	PendingEventID int64
	GoogleEventID  string
	Title          string
	EventDate      time.Time
	CreatedAt      time.Time
}

// PendingEvent CRUD

func (d *DB) CreatePendingEvent(senderJID, sourceType string, sourceID int64, eventJSON string, expiresAt time.Time) (*PendingEvent, error) {
	result, err := d.Exec(
		`INSERT INTO pending_events (sender_jid, source_type, source_id, event_json, expires_at)
		 VALUES (?, ?, ?, ?, ?)`,
		senderJID, sourceType, sourceID, eventJSON, expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return d.GetPendingEventByID(id)
}

func (d *DB) GetPendingEventByID(id int64) (*PendingEvent, error) {
	row := d.QueryRow(
		`SELECT id, sender_jid, source_type, source_id, event_json, status, created_at, expires_at
		 FROM pending_events WHERE id = ?`,
		id,
	)
	return scanPendingEvent(row)
}

func (d *DB) GetPendingEventBySender(senderJID string) (*PendingEvent, error) {
	row := d.QueryRow(
		`SELECT id, sender_jid, source_type, source_id, event_json, status, created_at, expires_at
		 FROM pending_events
		 WHERE sender_jid = ? AND status = 'pending' AND expires_at > datetime('now')
		 ORDER BY created_at DESC LIMIT 1`,
		senderJID,
	)
	return scanPendingEvent(row)
}

func (d *DB) ListPendingEvents() ([]*PendingEvent, error) {
	rows, err := d.Query(
		`SELECT id, sender_jid, source_type, source_id, event_json, status, created_at, expires_at
		 FROM pending_events WHERE status = 'pending' AND expires_at > datetime('now')
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending events: %w", err)
	}
	defer rows.Close()

	var events []*PendingEvent
	for rows.Next() {
		event, err := scanPendingEventRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (d *DB) ListAllEvents(limit int) ([]*PendingEvent, error) {
	rows, err := d.Query(
		`SELECT id, sender_jid, source_type, source_id, event_json, status, created_at, expires_at
		 FROM pending_events ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []*PendingEvent
	for rows.Next() {
		event, err := scanPendingEventRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (d *DB) UpdatePendingEventStatus(id int64, status string) error {
	_, err := d.Exec(
		`UPDATE pending_events SET status = ? WHERE id = ?`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update pending event status: %w", err)
	}
	return nil
}

func (d *DB) ExpireOldPendingEvents() (int64, error) {
	result, err := d.Exec(
		`UPDATE pending_events SET status = 'timeout'
		 WHERE status = 'pending' AND expires_at <= datetime('now')`,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to expire pending events: %w", err)
	}
	return result.RowsAffected()
}

// EventSourceMessage CRUD

func (d *DB) CreateEventSourceMessage(senderJID, sourceType string, sourceID int64, messageText string, pendingEventID int64) (*EventSourceMessage, error) {
	result, err := d.Exec(
		`INSERT INTO event_source_messages (sender_jid, source_type, source_id, message_text, pending_event_id)
		 VALUES (?, ?, ?, ?, ?)`,
		senderJID, sourceType, sourceID, messageText, pendingEventID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create event source message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return d.GetEventSourceMessageByID(id)
}

func (d *DB) GetEventSourceMessageByID(id int64) (*EventSourceMessage, error) {
	row := d.QueryRow(
		`SELECT id, sender_jid, source_type, source_id, message_text, pending_event_id, created_at
		 FROM event_source_messages WHERE id = ?`,
		id,
	)
	return scanEventSourceMessage(row)
}

func (d *DB) GetMessagesByPendingEvent(pendingEventID int64) ([]*EventSourceMessage, error) {
	rows, err := d.Query(
		`SELECT id, sender_jid, source_type, source_id, message_text, pending_event_id, created_at
		 FROM event_source_messages WHERE pending_event_id = ? ORDER BY created_at`,
		pendingEventID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*EventSourceMessage
	for rows.Next() {
		msg, err := scanEventSourceMessageRows(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// CalendarEvent CRUD

func (d *DB) CreateCalendarEvent(pendingEventID int64, googleEventID, title string, eventDate time.Time) (*CalendarEvent, error) {
	result, err := d.Exec(
		`INSERT INTO calendar_events (pending_event_id, google_event_id, title, event_date)
		 VALUES (?, ?, ?, ?)`,
		pendingEventID, googleEventID, title, eventDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return d.GetCalendarEventByID(id)
}

func (d *DB) GetCalendarEventByID(id int64) (*CalendarEvent, error) {
	row := d.QueryRow(
		`SELECT id, pending_event_id, google_event_id, title, event_date, created_at
		 FROM calendar_events WHERE id = ?`,
		id,
	)
	return scanCalendarEvent(row)
}

func (d *DB) GetCalendarEventByPendingEvent(pendingEventID int64) (*CalendarEvent, error) {
	row := d.QueryRow(
		`SELECT id, pending_event_id, google_event_id, title, event_date, created_at
		 FROM calendar_events WHERE pending_event_id = ?`,
		pendingEventID,
	)
	return scanCalendarEvent(row)
}

func (d *DB) ListRecentCalendarEvents(limit int) ([]*CalendarEvent, error) {
	rows, err := d.Query(
		`SELECT id, pending_event_id, google_event_id, title, event_date, created_at
		 FROM calendar_events ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar events: %w", err)
	}
	defer rows.Close()

	var events []*CalendarEvent
	for rows.Next() {
		event, err := scanCalendarEventRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// Scan helpers

func scanPendingEvent(row *sql.Row) (*PendingEvent, error) {
	var e PendingEvent
	err := row.Scan(&e.ID, &e.SenderJID, &e.SourceType, &e.SourceID, &e.EventJSON, &e.Status, &e.CreatedAt, &e.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan pending event: %w", err)
	}
	return &e, nil
}

func scanPendingEventRows(rows *sql.Rows) (*PendingEvent, error) {
	var e PendingEvent
	err := rows.Scan(&e.ID, &e.SenderJID, &e.SourceType, &e.SourceID, &e.EventJSON, &e.Status, &e.CreatedAt, &e.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan pending event: %w", err)
	}
	return &e, nil
}

func scanEventSourceMessage(row *sql.Row) (*EventSourceMessage, error) {
	var m EventSourceMessage
	err := row.Scan(&m.ID, &m.SenderJID, &m.SourceType, &m.SourceID, &m.MessageText, &m.PendingEventID, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan event source message: %w", err)
	}
	return &m, nil
}

func scanEventSourceMessageRows(rows *sql.Rows) (*EventSourceMessage, error) {
	var m EventSourceMessage
	err := rows.Scan(&m.ID, &m.SenderJID, &m.SourceType, &m.SourceID, &m.MessageText, &m.PendingEventID, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan event source message: %w", err)
	}
	return &m, nil
}

func scanCalendarEvent(row *sql.Row) (*CalendarEvent, error) {
	var e CalendarEvent
	err := row.Scan(&e.ID, &e.PendingEventID, &e.GoogleEventID, &e.Title, &e.EventDate, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan calendar event: %w", err)
	}
	return &e, nil
}

func scanCalendarEventRows(rows *sql.Rows) (*CalendarEvent, error) {
	var e CalendarEvent
	err := rows.Scan(&e.ID, &e.PendingEventID, &e.GoogleEventID, &e.Title, &e.EventDate, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan calendar event: %w", err)
	}
	return &e, nil
}
