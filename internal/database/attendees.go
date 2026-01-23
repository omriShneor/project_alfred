package database

import (
	"fmt"
)

// Attendee represents a participant for a calendar event
type Attendee struct {
	ID          int64  `json:"id"`
	EventID     int64  `json:"event_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	Optional    bool   `json:"optional"`
}

// GetEventAttendees retrieves all attendees for a specific event
func (d *DB) GetEventAttendees(eventID int64) ([]Attendee, error) {
	rows, err := d.Query(`
		SELECT id, event_id, email, display_name, optional
		FROM event_attendees
		WHERE event_id = ?
		ORDER BY id
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query attendees: %w", err)
	}
	defer rows.Close()

	var attendees []Attendee
	for rows.Next() {
		var a Attendee
		var displayName *string
		if err := rows.Scan(&a.ID, &a.EventID, &a.Email, &displayName, &a.Optional); err != nil {
			return nil, fmt.Errorf("failed to scan attendee: %w", err)
		}
		if displayName != nil {
			a.DisplayName = *displayName
		}
		attendees = append(attendees, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attendees: %w", err)
	}

	return attendees, nil
}

// AddEventAttendee adds a single attendee to an event
func (d *DB) AddEventAttendee(eventID int64, email, displayName string, optional bool) (*Attendee, error) {
	result, err := d.Exec(`
		INSERT INTO event_attendees (event_id, email, display_name, optional)
		VALUES (?, ?, ?, ?)
	`, eventID, email, displayName, optional)
	if err != nil {
		return nil, fmt.Errorf("failed to add attendee: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get attendee id: %w", err)
	}

	return &Attendee{
		ID:          id,
		EventID:     eventID,
		Email:       email,
		DisplayName: displayName,
		Optional:    optional,
	}, nil
}

// DeleteEventAttendee removes an attendee by ID
func (d *DB) DeleteEventAttendee(id int64) error {
	_, err := d.Exec(`DELETE FROM event_attendees WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete attendee: %w", err)
	}
	return nil
}

// DeleteEventAttendees removes all attendees for an event
func (d *DB) DeleteEventAttendees(eventID int64) error {
	_, err := d.Exec(`DELETE FROM event_attendees WHERE event_id = ?`, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete attendees: %w", err)
	}
	return nil
}

// SetEventAttendees replaces all attendees for an event with the provided list
func (d *DB) SetEventAttendees(eventID int64, attendees []Attendee) error {
	// Delete existing attendees
	if err := d.DeleteEventAttendees(eventID); err != nil {
		return err
	}

	// Add new attendees
	for _, a := range attendees {
		_, err := d.Exec(`
			INSERT INTO event_attendees (event_id, email, display_name, optional)
			VALUES (?, ?, ?, ?)
		`, eventID, a.Email, a.DisplayName, a.Optional)
		if err != nil {
			return fmt.Errorf("failed to add attendee: %w", err)
		}
	}

	return nil
}
