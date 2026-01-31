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
	TelegramInputEnabled bool `json:"telegram_input_enabled"`
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
			COALESCE(telegram_input_enabled, 0) as telegram_input_enabled,
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
		&settings.TelegramInputEnabled,
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

// ---- Simplified App Status API ----

// AppStatus represents the simplified app status
type AppStatus struct {
	OnboardingComplete bool `json:"onboarding_complete"`
	WhatsAppEnabled    bool `json:"whatsapp_enabled"`
	TelegramEnabled    bool `json:"telegram_enabled"`
	GmailEnabled       bool `json:"gmail_enabled"`
	GoogleCalEnabled   bool `json:"google_calendar_enabled"`
}

// GetAppStatus retrieves the simplified app status
func (d *DB) GetAppStatus() (*AppStatus, error) {
	var status AppStatus
	err := d.QueryRow(`
		SELECT
			COALESCE(onboarding_complete, 0),
			whatsapp_input_enabled,
			COALESCE(telegram_input_enabled, 0),
			email_input_enabled,
			google_calendar_enabled
		FROM feature_settings WHERE id = 1
	`).Scan(
		&status.OnboardingComplete,
		&status.WhatsAppEnabled,
		&status.TelegramEnabled,
		&status.GmailEnabled,
		&status.GoogleCalEnabled,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get app status: %w", err)
	}
	return &status, nil
}

// CompleteOnboarding marks onboarding as complete and enables the configured inputs
func (d *DB) CompleteOnboarding(whatsappEnabled, telegramEnabled, gmailEnabled bool) error {
	_, err := d.Exec(`
		UPDATE feature_settings SET
			onboarding_complete = 1,
			whatsapp_input_enabled = ?,
			telegram_input_enabled = ?,
			email_input_enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, whatsappEnabled, telegramEnabled, gmailEnabled)
	if err != nil {
		return fmt.Errorf("failed to complete onboarding: %w", err)
	}
	return nil
}

// ResetOnboarding resets the onboarding status (for testing)
func (d *DB) ResetOnboarding() error {
	_, err := d.Exec(`
		UPDATE feature_settings SET
			onboarding_complete = 0,
			whatsapp_input_enabled = 0,
			telegram_input_enabled = 0,
			email_input_enabled = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`)
	if err != nil {
		return fmt.Errorf("failed to reset onboarding: %w", err)
	}
	return nil
}
