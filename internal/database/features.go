package database

import (
	"database/sql"
	"fmt"
	"time"
)

// FeatureSettings represents the feature settings for the app
type FeatureSettings struct {
	UserID int64 `json:"user_id"`

	// Smart Calendar feature
	SmartCalendarEnabled       bool `json:"smart_calendar_enabled"`
	SmartCalendarSetupComplete bool `json:"smart_calendar_setup_complete"`

	// Inputs (where to scan for events)
	WhatsAppInputEnabled bool `json:"whatsapp_input_enabled"`
	TelegramInputEnabled bool `json:"telegram_input_enabled"`
	EmailInputEnabled    bool `json:"email_input_enabled"`
	SMSInputEnabled      bool `json:"sms_input_enabled"`

	// Calendars (where to sync events)
	AlfredCalendarEnabled  bool `json:"alfred_calendar_enabled"` // Local Alfred calendar (always available)
	GoogleCalendarEnabled  bool `json:"google_calendar_enabled"`
	OutlookCalendarEnabled bool `json:"outlook_calendar_enabled"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetFeatureSettings retrieves the feature settings for a user
func (d *DB) GetFeatureSettings(userID int64) (*FeatureSettings, error) {
	var settings FeatureSettings
	err := d.QueryRow(`
		SELECT
			user_id,
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
		FROM feature_settings WHERE user_id = ?
	`, userID).Scan(
		&settings.UserID,
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
		// Create default settings for this user if not found
		_, insertErr := d.Exec(`
			INSERT INTO feature_settings (user_id, smart_calendar_enabled, smart_calendar_setup_complete,
				whatsapp_input_enabled, telegram_input_enabled, email_input_enabled, sms_input_enabled,
				alfred_calendar_enabled, google_calendar_enabled, outlook_calendar_enabled)
			VALUES (?, 0, 0, 0, 0, 0, 0, 1, 0, 0)
		`, userID)
		if insertErr != nil {
			return nil, fmt.Errorf("failed to create feature settings: %w", insertErr)
		}
		// Return default settings
		return &FeatureSettings{
			UserID:                userID,
			AlfredCalendarEnabled: true,
		}, nil
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

// GetAppStatus retrieves the simplified app status for a user
func (d *DB) GetAppStatus(userID int64) (*AppStatus, error) {
	// Ensure feature settings exist for this user
	_, err := d.GetFeatureSettings(userID)
	if err != nil {
		return nil, err
	}

	var status AppStatus
	err = d.QueryRow(`
		SELECT
			COALESCE(onboarding_complete, 0),
			whatsapp_input_enabled,
			COALESCE(telegram_input_enabled, 0),
			email_input_enabled,
			google_calendar_enabled
		FROM feature_settings WHERE user_id = ?
	`, userID).Scan(
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

// CompleteOnboarding marks onboarding as complete and enables the configured inputs for a user
func (d *DB) CompleteOnboarding(userID int64, whatsappEnabled, telegramEnabled, gmailEnabled bool) error {
	// Ensure feature settings exist for this user
	_, err := d.GetFeatureSettings(userID)
	if err != nil {
		return err
	}

	_, err = d.Exec(`
		UPDATE feature_settings SET
			onboarding_complete = 1,
			whatsapp_input_enabled = ?,
			telegram_input_enabled = ?,
			email_input_enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, whatsappEnabled, telegramEnabled, gmailEnabled, userID)
	if err != nil {
		return fmt.Errorf("failed to complete onboarding: %w", err)
	}
	return nil
}

// ResetOnboarding resets the onboarding status for a user (for testing)
func (d *DB) ResetOnboarding(userID int64) error {
	// Ensure feature settings exist for this user
	_, err := d.GetFeatureSettings(userID)
	if err != nil {
		return err
	}

	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin reset onboarding transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete user-scoped data in dependency-safe order.
	deleteSteps := []struct {
		name  string
		query string
	}{
		{
			name:  "event attendees",
			query: `DELETE FROM event_attendees WHERE event_id IN (SELECT id FROM calendar_events WHERE user_id = ?)`,
		},
		{name: "reminders", query: `DELETE FROM reminders WHERE user_id = ?`},
		{name: "calendar events", query: `DELETE FROM calendar_events WHERE user_id = ?`},
		{name: "message history", query: `DELETE FROM message_history WHERE user_id = ?`},
		{
			name:  "message history by channel ownership",
			query: `DELETE FROM message_history WHERE channel_id IN (SELECT id FROM channels WHERE user_id = ?)`,
		},
		{name: "channels", query: `DELETE FROM channels WHERE user_id = ?`},
		{name: "email sources", query: `DELETE FROM email_sources WHERE user_id = ?`},
		{name: "processed emails", query: `DELETE FROM processed_emails WHERE user_id = ?`},
		{name: "google tokens", query: `DELETE FROM google_tokens WHERE user_id = ?`},
		{name: "whatsapp sessions", query: `DELETE FROM whatsapp_sessions WHERE user_id = ?`},
		{name: "telegram sessions", query: `DELETE FROM telegram_sessions WHERE user_id = ?`},
		{name: "gmail settings", query: `DELETE FROM gmail_settings WHERE user_id = ?`},
		{name: "gcal settings", query: `DELETE FROM gcal_settings WHERE user_id = ?`},
		{name: "notification preferences", query: `DELETE FROM user_notification_preferences WHERE user_id = ?`},
		{name: "auth sessions", query: `DELETE FROM user_sessions WHERE user_id = ?`},
	}

	for _, step := range deleteSteps {
		if _, err := tx.Exec(step.query, userID); err != nil {
			return fmt.Errorf("failed to delete %s during onboarding reset: %w", step.name, err)
		}
	}

	// Handle both possible contact cache table names for migration compatibility.
	if err := deleteFromUserScopedTableIfExists(tx, "google_contacts", userID); err != nil {
		return err
	}
	if err := deleteFromUserScopedTableIfExists(tx, "gmail_top_contacts", userID); err != nil {
		return err
	}

	// Reset feature flags back to defaults.
	_, err = tx.Exec(`
		UPDATE feature_settings SET
			smart_calendar_enabled = 0,
			smart_calendar_setup_complete = 0,
			onboarding_complete = 0,
			whatsapp_input_enabled = 0,
			telegram_input_enabled = 0,
			email_input_enabled = 0,
			sms_input_enabled = 0,
			alfred_calendar_enabled = 1,
			google_calendar_enabled = 0,
			outlook_calendar_enabled = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to reset onboarding: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit onboarding reset: %w", err)
	}
	return nil
}

func deleteFromUserScopedTableIfExists(tx *sql.Tx, table string, userID int64) error {
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check table %s during onboarding reset: %w", table, err)
	}
	if exists == 0 {
		return nil
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE user_id = ?", table)
	if _, err := tx.Exec(query, userID); err != nil {
		return fmt.Errorf("failed to delete %s during onboarding reset: %w", table, err)
	}
	return nil
}
