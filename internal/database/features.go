package database

import (
	"fmt"
	"time"
)

// FeatureSettings represents the feature settings for the app
type FeatureSettings struct {
	// Smart Calendar feature
	SmartCalendarEnabled       bool `json:"smart_calendar_enabled"`
	SmartCalendarSetupComplete bool `json:"smart_calendar_setup_complete"`

	// Inputs (where to scan for events)
	WhatsAppInputEnabled bool `json:"whatsapp_input_enabled"`
	EmailInputEnabled    bool `json:"email_input_enabled"`
	SMSInputEnabled      bool `json:"sms_input_enabled"`

	// Calendars (where to sync events)
	AlfredCalendarEnabled  bool `json:"alfred_calendar_enabled"`  // Local Alfred calendar (always available)
	GoogleCalendarEnabled  bool `json:"google_calendar_enabled"`
	OutlookCalendarEnabled bool `json:"outlook_calendar_enabled"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetFeatureSettings retrieves the feature settings
func (d *DB) GetFeatureSettings() (*FeatureSettings, error) {
	var settings FeatureSettings
	err := d.QueryRow(`
		SELECT
			smart_calendar_enabled,
			smart_calendar_setup_complete,
			whatsapp_input_enabled,
			email_input_enabled,
			sms_input_enabled,
			COALESCE(alfred_calendar_enabled, 1) as alfred_calendar_enabled,
			google_calendar_enabled,
			outlook_calendar_enabled,
			created_at,
			updated_at
		FROM feature_settings WHERE id = 1
	`).Scan(
		&settings.SmartCalendarEnabled,
		&settings.SmartCalendarSetupComplete,
		&settings.WhatsAppInputEnabled,
		&settings.EmailInputEnabled,
		&settings.SMSInputEnabled,
		&settings.AlfredCalendarEnabled,
		&settings.GoogleCalendarEnabled,
		&settings.OutlookCalendarEnabled,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature settings: %w", err)
	}
	return &settings, nil
}

// UpdateFeatureSettings updates all feature settings
func (d *DB) UpdateFeatureSettings(settings *FeatureSettings) error {
	_, err := d.Exec(`
		UPDATE feature_settings SET
			smart_calendar_enabled = ?,
			smart_calendar_setup_complete = ?,
			whatsapp_input_enabled = ?,
			email_input_enabled = ?,
			sms_input_enabled = ?,
			alfred_calendar_enabled = ?,
			google_calendar_enabled = ?,
			outlook_calendar_enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`,
		settings.SmartCalendarEnabled,
		settings.SmartCalendarSetupComplete,
		settings.WhatsAppInputEnabled,
		settings.EmailInputEnabled,
		settings.SMSInputEnabled,
		settings.AlfredCalendarEnabled,
		settings.GoogleCalendarEnabled,
		settings.OutlookCalendarEnabled,
	)
	if err != nil {
		return fmt.Errorf("failed to update feature settings: %w", err)
	}
	return nil
}

// SetSmartCalendarEnabled enables or disables the Smart Calendar feature
func (d *DB) SetSmartCalendarEnabled(enabled bool) error {
	_, err := d.Exec(`
		UPDATE feature_settings SET
			smart_calendar_enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, enabled)
	if err != nil {
		return fmt.Errorf("failed to set smart calendar enabled: %w", err)
	}
	return nil
}

// SetSmartCalendarSetupComplete marks the Smart Calendar setup as complete
func (d *DB) SetSmartCalendarSetupComplete(complete bool) error {
	_, err := d.Exec(`
		UPDATE feature_settings SET
			smart_calendar_setup_complete = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, complete)
	if err != nil {
		return fmt.Errorf("failed to set smart calendar setup complete: %w", err)
	}
	return nil
}

// UpdateSmartCalendarInputs updates which inputs are enabled for Smart Calendar
func (d *DB) UpdateSmartCalendarInputs(whatsapp, email, sms bool) error {
	_, err := d.Exec(`
		UPDATE feature_settings SET
			whatsapp_input_enabled = ?,
			email_input_enabled = ?,
			sms_input_enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, whatsapp, email, sms)
	if err != nil {
		return fmt.Errorf("failed to update smart calendar inputs: %w", err)
	}
	return nil
}

// UpdateSmartCalendarCalendars updates which calendars are enabled for Smart Calendar
func (d *DB) UpdateSmartCalendarCalendars(alfred, googleCalendar, outlook bool) error {
	_, err := d.Exec(`
		UPDATE feature_settings SET
			alfred_calendar_enabled = ?,
			google_calendar_enabled = ?,
			outlook_calendar_enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, alfred, googleCalendar, outlook)
	if err != nil {
		return fmt.Errorf("failed to update smart calendar calendars: %w", err)
	}
	return nil
}
