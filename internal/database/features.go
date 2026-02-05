package database

import (
	"fmt"
	"log"
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

	if err := d.DeleteGoogleToken(userID); err != nil {
		log.Printf("Warning: failed to delete Google token during reset: %v", err)
	}
	if err := d.DeleteWhatsAppSession(userID); err != nil {
		log.Printf("Warning: failed to delete WhatsApp session during reset: %v", err)
	}
	if err := d.DeleteTelegramSession(userID); err != nil {
		log.Printf("Warning: failed to delete Telegram session during reset: %v", err)
	}

	// Reset onboarding flags (critical operation)
	_, err = d.Exec(`
		UPDATE feature_settings SET
			onboarding_complete = 0,
			whatsapp_input_enabled = 0,
			telegram_input_enabled = 0,
			email_input_enabled = 0,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to reset onboarding: %w", err)
	}
	return nil
}
