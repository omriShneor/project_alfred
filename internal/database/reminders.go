package database

import (
	"database/sql"
	"fmt"
	"time"
)

// ReminderStatus represents the status of a reminder
type ReminderStatus string

const (
	ReminderStatusPending   ReminderStatus = "pending"
	ReminderStatusConfirmed ReminderStatus = "confirmed"
	ReminderStatusSynced    ReminderStatus = "synced"
	ReminderStatusRejected  ReminderStatus = "rejected"
	ReminderStatusCompleted ReminderStatus = "completed"
	ReminderStatusDismissed ReminderStatus = "dismissed"
)

// ReminderPriority represents the priority level of a reminder
type ReminderPriority string

const (
	ReminderPriorityLow    ReminderPriority = "low"
	ReminderPriorityNormal ReminderPriority = "normal"
	ReminderPriorityHigh   ReminderPriority = "high"
)

// ReminderActionType represents the type of action for a reminder
type ReminderActionType string

const (
	ReminderActionCreate ReminderActionType = "create"
	ReminderActionUpdate ReminderActionType = "update"
	ReminderActionDelete ReminderActionType = "delete"
)

// Reminder represents a detected reminder
type Reminder struct {
	ID            int64              `json:"id"`
	UserID        int64              `json:"user_id"`
	ChannelID     int64              `json:"channel_id"`
	GoogleEventID *string            `json:"google_event_id,omitempty"`
	CalendarID    string             `json:"calendar_id"`
	Title         string             `json:"title"`
	Description   string             `json:"description,omitempty"`
	Location      string             `json:"location,omitempty"`
	DueDate       *time.Time         `json:"due_date,omitempty"`
	ReminderTime  *time.Time         `json:"reminder_time,omitempty"`
	Priority      ReminderPriority   `json:"priority"`
	Status        ReminderStatus     `json:"status"`
	ActionType    ReminderActionType `json:"action_type"`
	OriginalMsgID *int64             `json:"original_message_id,omitempty"`
	LLMReasoning  string             `json:"llm_reasoning,omitempty"`
	LLMConfidence float64            `json:"llm_confidence"`
	QualityFlags  []string           `json:"quality_flags,omitempty"`
	Source        string             `json:"source,omitempty"`
	EmailSourceID *int64             `json:"email_source_id,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	ChannelName   string             `json:"channel_name,omitempty"` // Joined from channels table
}

// CreatePendingReminder creates a new pending reminder in the database
// The reminder must have UserID set
func (d *DB) CreatePendingReminder(reminder *Reminder) (*Reminder, error) {
	result, err := d.Exec(`
		INSERT INTO reminders (
			user_id, channel_id, google_event_id, calendar_id, title, description,
			location, due_date, reminder_time, priority, status, action_type,
			original_message_id, llm_reasoning, llm_confidence, quality_flags, source, email_source_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		reminder.UserID, reminder.ChannelID, reminder.GoogleEventID, reminder.CalendarID, reminder.Title, reminder.Description,
		reminder.Location, reminder.DueDate, reminder.ReminderTime, reminder.Priority, ReminderStatusPending, reminder.ActionType,
		reminder.OriginalMsgID, reminder.LLMReasoning, reminder.LLMConfidence, encodeQualityFlags(reminder.QualityFlags), reminder.Source, reminder.EmailSourceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder id: %w", err)
	}

	reminder.ID = id
	reminder.Status = ReminderStatusPending
	reminder.CreatedAt = time.Now()
	reminder.UpdatedAt = time.Now()

	return reminder, nil
}

type reminderScanner interface {
	Scan(dest ...any) error
}

func scanReminder(scanner reminderScanner) (*Reminder, error) {
	var reminder Reminder
	var googleEventID sql.NullString
	var descriptionNull sql.NullString
	var locationNull sql.NullString
	var dueDateNull sql.NullTime
	var reminderTimeNull sql.NullTime
	var origMsgIDNull sql.NullInt64
	var emailSourceIDNull sql.NullInt64
	var sourceNull sql.NullString
	var qualityFlagsNull sql.NullString

	err := scanner.Scan(
		&reminder.ID, &reminder.UserID, &reminder.ChannelID, &googleEventID, &reminder.CalendarID, &reminder.Title,
		&descriptionNull, &locationNull, &dueDateNull, &reminderTimeNull, &reminder.Priority, &reminder.Status,
		&reminder.ActionType, &origMsgIDNull, &reminder.LLMReasoning, &reminder.LLMConfidence, &qualityFlagsNull, &sourceNull, &emailSourceIDNull,
		&reminder.CreatedAt, &reminder.UpdatedAt, &reminder.ChannelName,
	)
	if err != nil {
		return nil, err
	}

	if googleEventID.Valid {
		reminder.GoogleEventID = &googleEventID.String
	}
	if descriptionNull.Valid {
		reminder.Description = descriptionNull.String
	}
	if locationNull.Valid {
		reminder.Location = locationNull.String
	}
	if dueDateNull.Valid {
		dueDate := dueDateNull.Time
		reminder.DueDate = &dueDate
	}
	if reminderTimeNull.Valid {
		reminder.ReminderTime = &reminderTimeNull.Time
	}
	if origMsgIDNull.Valid {
		reminder.OriginalMsgID = &origMsgIDNull.Int64
	}
	if emailSourceIDNull.Valid {
		reminder.EmailSourceID = &emailSourceIDNull.Int64
	}
	if sourceNull.Valid {
		reminder.Source = sourceNull.String
	}
	reminder.QualityFlags = decodeQualityFlags(qualityFlagsNull)

	return &reminder, nil
}

// GetReminderByID retrieves a reminder by its ID
func (d *DB) GetReminderByID(id int64) (*Reminder, error) {
	reminder, err := scanReminder(d.QueryRow(`
		SELECT r.id, r.user_id, r.channel_id, r.google_event_id, r.calendar_id, r.title,
			r.description, r.location, r.due_date, r.reminder_time, r.priority, r.status,
			r.action_type, r.original_message_id, r.llm_reasoning, r.llm_confidence, r.quality_flags, r.source, r.email_source_id,
			r.created_at, r.updated_at,
			c.name as channel_name
		FROM reminders r
		JOIN channels c ON r.channel_id = c.id
		WHERE r.id = ?
	`, id))
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}

	return reminder, nil
}

// ListReminders retrieves reminders with optional filtering by status and channel
func (d *DB) ListReminders(userID int64, status *ReminderStatus, channelID *int64) ([]Reminder, error) {
	query := `
		SELECT r.id, r.user_id, r.channel_id, r.google_event_id, r.calendar_id, r.title,
			r.description, r.location, r.due_date, r.reminder_time, r.priority, r.status,
			r.action_type, r.original_message_id, r.llm_reasoning, r.llm_confidence, r.quality_flags, r.source, r.email_source_id,
			r.created_at, r.updated_at,
			c.name as channel_name
		FROM reminders r
		JOIN channels c ON r.channel_id = c.id
		WHERE r.user_id = ?
	`
	args := []any{userID}

	if status != nil {
		query += " AND r.status = ?"
		args = append(args, *status)
	}

	if channelID != nil {
		query += " AND r.channel_id = ?"
		args = append(args, *channelID)
	}

	query += " ORDER BY (r.due_date IS NULL) ASC, r.due_date ASC, r.created_at DESC"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		reminder, err := scanReminder(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, *reminder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reminders: %w", err)
	}

	return reminders, nil
}

// GetPendingReminders retrieves all pending reminders, optionally filtered by channel
func (d *DB) GetPendingReminders(userID int64, channelID *int64) ([]Reminder, error) {
	status := ReminderStatusPending
	return d.ListReminders(userID, &status, channelID)
}

// GetActiveRemindersForChannel retrieves both pending and synced reminders for a channel
// This is used for Claude context so it can reference and update pending reminders
func (d *DB) GetActiveRemindersForChannel(channelID int64) ([]Reminder, error) {
	query := `
		SELECT r.id, r.user_id, r.channel_id, r.google_event_id, r.calendar_id, r.title,
			r.description, r.location, r.due_date, r.reminder_time, r.priority, r.status,
			r.action_type, r.original_message_id, r.llm_reasoning, r.llm_confidence, r.quality_flags, r.source, r.email_source_id,
			r.created_at, r.updated_at,
			c.name as channel_name
		FROM reminders r
		JOIN channels c ON r.channel_id = c.id
		WHERE r.channel_id = ? AND r.status IN (?, ?, ?)
		ORDER BY (r.due_date IS NULL) ASC, r.due_date ASC, r.created_at DESC
	`

	rows, err := d.Query(query, channelID, ReminderStatusPending, ReminderStatusConfirmed, ReminderStatusSynced)
	if err != nil {
		return nil, fmt.Errorf("failed to list active reminders: %w", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		reminder, err := scanReminder(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, *reminder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reminders: %w", err)
	}

	return reminders, nil
}

// UpdatePendingReminder updates a pending reminder's details
func (d *DB) UpdatePendingReminder(
	id int64,
	title, description, location string,
	dueDate *time.Time,
	reminderTime *time.Time,
	priority ReminderPriority,
) error {
	_, err := d.Exec(`
		UPDATE reminders
		SET title = ?, description = ?, location = ?, due_date = ?, reminder_time = ?, priority = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = ?
	`, title, description, location, dueDate, reminderTime, priority, id, ReminderStatusPending)
	if err != nil {
		return fmt.Errorf("failed to update pending reminder: %w", err)
	}
	return nil
}

// UpdateReminderStatus updates the status of a reminder
func (d *DB) UpdateReminderStatus(id int64, status ReminderStatus) error {
	_, err := d.Exec(`
		UPDATE reminders
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, id)
	if err != nil {
		return fmt.Errorf("failed to update reminder status: %w", err)
	}
	return nil
}

// UpdateReminderGoogleID sets the Google Calendar event ID after syncing
func (d *DB) UpdateReminderGoogleID(id int64, googleEventID string) error {
	_, err := d.Exec(`
		UPDATE reminders
		SET google_event_id = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, googleEventID, ReminderStatusSynced, id)
	if err != nil {
		return fmt.Errorf("failed to update reminder google id: %w", err)
	}
	return nil
}

// DeleteReminder removes a reminder from the database
func (d *DB) DeleteReminder(id int64) error {
	_, err := d.Exec(`DELETE FROM reminders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}
	return nil
}

// CountPendingReminders returns the number of pending reminders
func (d *DB) CountPendingReminders() (int, error) {
	var count int
	err := d.QueryRow(`SELECT COUNT(*) FROM reminders WHERE status = ?`, ReminderStatusPending).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending reminders: %w", err)
	}
	return count, nil
}

// GetUpcomingReminders retrieves confirmed/synced reminders due within a time window
func (d *DB) GetUpcomingReminders(window time.Duration) ([]Reminder, error) {
	now := time.Now()
	endTime := now.Add(window)

	query := `
		SELECT r.id, r.user_id, r.channel_id, r.google_event_id, r.calendar_id, r.title,
			r.description, r.location, r.due_date, r.reminder_time, r.priority, r.status,
			r.action_type, r.original_message_id, r.llm_reasoning, r.llm_confidence, r.quality_flags, r.source, r.email_source_id,
			r.created_at, r.updated_at,
			COALESCE(c.name, 'Alfred') as channel_name
		FROM reminders r
		LEFT JOIN channels c ON r.channel_id = c.id
		WHERE r.status IN (?, ?)
		  AND r.due_date IS NOT NULL
		  AND r.due_date >= ?
		  AND r.due_date <= ?
		ORDER BY r.due_date ASC
	`

	rows, err := d.Query(query, ReminderStatusConfirmed, ReminderStatusSynced, now, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming reminders: %w", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		reminder, err := scanReminder(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, *reminder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reminders: %w", err)
	}

	return reminders, nil
}

// GetDueRemindersForNotification retrieves active reminders that reached their scheduled notification time.
// Uses reminder_time when present, otherwise falls back to due_date.
func (d *DB) GetDueRemindersForNotification(now time.Time, limit int) ([]Reminder, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT r.id, r.user_id, r.channel_id, r.google_event_id, r.calendar_id, r.title,
			r.description, r.location, r.due_date, r.reminder_time, r.priority, r.status,
			r.action_type, r.original_message_id, r.llm_reasoning, r.llm_confidence, r.quality_flags, r.source, r.email_source_id,
			r.created_at, r.updated_at,
			COALESCE(c.name, 'Alfred') as channel_name
		FROM reminders r
		LEFT JOIN channels c ON r.channel_id = c.id
		WHERE r.status IN (?, ?)
		  AND COALESCE(r.reminder_time, r.due_date) IS NOT NULL
		  AND COALESCE(r.reminder_time, r.due_date) <= ?
		  AND r.due_notification_sent_at IS NULL
		ORDER BY COALESCE(r.reminder_time, r.due_date) ASC
		LIMIT ?
	`

	rows, err := d.Query(query, ReminderStatusConfirmed, ReminderStatusSynced, now, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get due reminders for notification: %w", err)
	}
	defer rows.Close()

	var reminders []Reminder
	for rows.Next() {
		reminder, err := scanReminder(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, *reminder)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating due reminders: %w", err)
	}

	return reminders, nil
}

// MarkReminderDueNotificationSent marks a reminder as already notified.
// Returns true only when this call changed the row.
func (d *DB) MarkReminderDueNotificationSent(id int64, sentAt time.Time) (bool, error) {
	result, err := d.Exec(`
		UPDATE reminders
		SET due_notification_sent_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND due_notification_sent_at IS NULL
	`, sentAt, id)
	if err != nil {
		return false, fmt.Errorf("failed to mark reminder notification sent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to read rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}
