package database

import (
	"database/sql"
	"fmt"
	"time"
)

// GCalSettings represents Google Calendar sync settings
type GCalSettings struct {
	ID                   int64     `json:"id"`
	SyncEnabled          bool      `json:"sync_enabled"`
	SelectedCalendarID   string    `json:"selected_calendar_id"`
	SelectedCalendarName string    `json:"selected_calendar_name"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// GetGCalSettings retrieves the Google Calendar settings (single row)
func (d *DB) GetGCalSettings() (*GCalSettings, error) {
	var settings GCalSettings
	err := d.QueryRow(`
		SELECT id, sync_enabled, selected_calendar_id, selected_calendar_name, created_at, updated_at
		FROM gcal_settings WHERE id = 1
	`).Scan(
		&settings.ID,
		&settings.SyncEnabled,
		&settings.SelectedCalendarID,
		&settings.SelectedCalendarName,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Insert default row and try again
		_, err = d.Exec(`INSERT OR IGNORE INTO gcal_settings (id) VALUES (1)`)
		if err != nil {
			return nil, fmt.Errorf("failed to create default gcal settings: %w", err)
		}
		return d.GetGCalSettings()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get gcal settings: %w", err)
	}
	return &settings, nil
}

// UpdateGCalSettings updates the Google Calendar sync settings
func (d *DB) UpdateGCalSettings(syncEnabled bool, calendarID, calendarName string) error {
	_, err := d.Exec(`
		UPDATE gcal_settings
		SET sync_enabled = ?, selected_calendar_id = ?, selected_calendar_name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, syncEnabled, calendarID, calendarName)
	if err != nil {
		return fmt.Errorf("failed to update gcal settings: %w", err)
	}
	return nil
}

// GetSelectedCalendarID returns the selected calendar ID or "primary" as default
func (d *DB) GetSelectedCalendarID() (string, error) {
	settings, err := d.GetGCalSettings()
	if err != nil {
		return "primary", err
	}
	if settings.SelectedCalendarID == "" {
		return "primary", nil
	}
	return settings.SelectedCalendarID, nil
}
