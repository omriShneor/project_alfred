package database

import (
	"database/sql"
	"fmt"
	"time"
)

// GCalSettings represents Google Calendar sync settings
type GCalSettings struct {
	ID                   int64     `json:"id"`
	UserID               int64     `json:"user_id"`
	SyncEnabled          bool      `json:"sync_enabled"`
	SelectedCalendarID   string    `json:"selected_calendar_id"`
	SelectedCalendarName string    `json:"selected_calendar_name"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// GetGCalSettings retrieves the Google Calendar settings for a user
func (d *DB) GetGCalSettings(userID int64) (*GCalSettings, error) {
	var settings GCalSettings
	err := d.QueryRow(`
		SELECT id, user_id, sync_enabled, selected_calendar_id, selected_calendar_name, created_at, updated_at
		FROM gcal_settings WHERE user_id = ?
	`, userID).Scan(
		&settings.ID,
		&settings.UserID,
		&settings.SyncEnabled,
		&settings.SelectedCalendarID,
		&settings.SelectedCalendarName,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Insert default row for this user
		_, err = d.Exec(`
			INSERT INTO gcal_settings (user_id, sync_enabled, selected_calendar_id, selected_calendar_name)
			VALUES (?, 0, 'primary', 'Primary')
		`, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to create default gcal settings: %w", err)
		}
		// Return fresh settings (avoid recursion)
		return &GCalSettings{
			UserID:               userID,
			SyncEnabled:          false,
			SelectedCalendarID:   "primary",
			SelectedCalendarName: "Primary",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get gcal settings: %w", err)
	}
	return &settings, nil
}

// UpdateGCalSettings updates the Google Calendar sync settings for a user
func (d *DB) UpdateGCalSettings(userID int64, syncEnabled bool, calendarID, calendarName string) error {
	// Ensure settings exist first
	_, err := d.GetGCalSettings(userID)
	if err != nil {
		return fmt.Errorf("failed to ensure gcal settings exist: %w", err)
	}

	_, err = d.Exec(`
		UPDATE gcal_settings
		SET sync_enabled = ?, selected_calendar_id = ?, selected_calendar_name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, syncEnabled, calendarID, calendarName, userID)
	if err != nil {
		return fmt.Errorf("failed to update gcal settings: %w", err)
	}
	return nil
}

// GetSelectedCalendarID returns the selected calendar ID for a user or "primary" as default
func (d *DB) GetSelectedCalendarID(userID int64) (string, error) {
	settings, err := d.GetGCalSettings(userID)
	if err != nil {
		return "primary", err
	}
	if settings.SelectedCalendarID == "" {
		return "primary", nil
	}
	return settings.SelectedCalendarID, nil
}
