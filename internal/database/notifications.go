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

// EnsureNotificationPrefs creates default notification preferences for a user if they don't exist
func (d *DB) EnsureNotificationPrefs(userID int64) error {
	_, err := d.Exec(`
		INSERT INTO user_notification_preferences (user_id)
		VALUES (?)
		ON CONFLICT(user_id) DO NOTHING
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to ensure notification prefs: %w", err)
	}
	return nil
}

// GetUserNotificationPrefs retrieves all notification preferences for a user
func (d *DB) GetUserNotificationPrefs(userID int64) (*UserNotificationPrefs, error) {
	// Ensure row exists first
	if err := d.EnsureNotificationPrefs(userID); err != nil {
		return nil, err
	}

	var prefs UserNotificationPrefs
	err := d.QueryRow(`
		SELECT
			email_enabled, COALESCE(email_address, ''),
			push_enabled, COALESCE(push_token, ''),
			sms_enabled, COALESCE(sms_phone, ''),
			webhook_enabled, COALESCE(webhook_url, ''),
			updated_at
		FROM user_notification_preferences
		WHERE user_id = ?
	`, userID).Scan(
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

// UpdateEmailPrefs updates only email notification settings for a user
func (d *DB) UpdateEmailPrefs(userID int64, enabled bool, address string) error {
	// First ensure the row exists
	if err := d.EnsureNotificationPrefs(userID); err != nil {
		return err
	}

	_, err := d.Exec(`
		UPDATE user_notification_preferences
		SET email_enabled = ?, email_address = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, enabled, address, userID)
	if err != nil {
		return fmt.Errorf("failed to update email prefs: %w", err)
	}
	return nil
}

// UpdatePushPrefs enables/disables push notifications for a user
func (d *DB) UpdatePushPrefs(userID int64, enabled bool) error {
	// First ensure the row exists
	if err := d.EnsureNotificationPrefs(userID); err != nil {
		return err
	}

	_, err := d.Exec(`
		UPDATE user_notification_preferences
		SET push_enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, enabled, userID)
	if err != nil {
		return fmt.Errorf("failed to update push prefs: %w", err)
	}
	return nil
}

// UpdatePushToken stores the Expo push token for a user
func (d *DB) UpdatePushToken(userID int64, token string) error {
	// First ensure the row exists
	if err := d.EnsureNotificationPrefs(userID); err != nil {
		return err
	}

	_, err := d.Exec(`
		UPDATE user_notification_preferences
		SET push_token = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, token, userID)
	if err != nil {
		return fmt.Errorf("failed to update push token: %w", err)
	}
	return nil
}
