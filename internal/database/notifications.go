package database

import (
	"fmt"
	"time"
)

// UserNotificationPrefs represents all user notification preferences
type UserNotificationPrefs struct {
	// Email notifications
	EmailEnabled bool   `json:"email_enabled"`
	EmailAddress string `json:"email_address"`

	// Push notifications (future)
	PushEnabled bool   `json:"push_enabled"`
	PushToken   string `json:"push_token,omitempty"`

	// SMS notifications (future)
	SMSEnabled bool   `json:"sms_enabled"`
	SMSPhone   string `json:"sms_phone,omitempty"`

	// Webhook notifications (future)
	WebhookEnabled bool   `json:"webhook_enabled"`
	WebhookURL     string `json:"webhook_url,omitempty"`

	UpdatedAt time.Time `json:"updated_at"`
}

// GetUserNotificationPrefs retrieves all notification preferences
func (d *DB) GetUserNotificationPrefs() (*UserNotificationPrefs, error) {
	var prefs UserNotificationPrefs
	err := d.QueryRow(`
		SELECT
			email_enabled, COALESCE(email_address, ''),
			push_enabled, COALESCE(push_token, ''),
			sms_enabled, COALESCE(sms_phone, ''),
			webhook_enabled, COALESCE(webhook_url, ''),
			updated_at
		FROM user_notification_preferences
		WHERE id = 1
	`).Scan(
		&prefs.EmailEnabled, &prefs.EmailAddress,
		&prefs.PushEnabled, &prefs.PushToken,
		&prefs.SMSEnabled, &prefs.SMSPhone,
		&prefs.WebhookEnabled, &prefs.WebhookURL,
		&prefs.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification prefs: %w", err)
	}
	return &prefs, nil
}

// UpdateEmailPrefs updates only email notification settings
func (d *DB) UpdateEmailPrefs(enabled bool, address string) error {
	_, err := d.Exec(`
		UPDATE user_notification_preferences
		SET email_enabled = ?, email_address = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, enabled, address)
	if err != nil {
		return fmt.Errorf("failed to update email prefs: %w", err)
	}
	return nil
}
