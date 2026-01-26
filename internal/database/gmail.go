package database

import (
	"database/sql"
	"fmt"
	"time"
)

// GmailSettings represents Gmail integration settings
type GmailSettings struct {
	ID                  int64      `json:"id"`
	Enabled             bool       `json:"enabled"`
	PollIntervalMinutes int        `json:"poll_interval_minutes"`
	LastPollAt          *time.Time `json:"last_poll_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// GetGmailSettings retrieves the Gmail settings (single row)
func (d *DB) GetGmailSettings() (*GmailSettings, error) {
	var settings GmailSettings
	var lastPollAt sql.NullTime

	err := d.QueryRow(`
		SELECT id, enabled, poll_interval_minutes, last_poll_at, created_at, updated_at
		FROM gmail_settings WHERE id = 1
	`).Scan(&settings.ID, &settings.Enabled, &settings.PollIntervalMinutes,
		&lastPollAt, &settings.CreatedAt, &settings.UpdatedAt)

	if err == sql.ErrNoRows {
		// Create default settings
		_, err = d.Exec(`INSERT OR IGNORE INTO gmail_settings (id) VALUES (1)`)
		if err != nil {
			return nil, fmt.Errorf("failed to create default gmail settings: %w", err)
		}
		return d.GetGmailSettings()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get gmail settings: %w", err)
	}

	if lastPollAt.Valid {
		settings.LastPollAt = &lastPollAt.Time
	}

	return &settings, nil
}

// UpdateGmailSettings updates the Gmail settings
func (d *DB) UpdateGmailSettings(enabled bool, pollIntervalMinutes int) error {
	_, err := d.Exec(`
		UPDATE gmail_settings
		SET enabled = ?, poll_interval_minutes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, enabled, pollIntervalMinutes)
	if err != nil {
		return fmt.Errorf("failed to update gmail settings: %w", err)
	}
	return nil
}

// UpdateGmailLastPoll updates the last poll timestamp
func (d *DB) UpdateGmailLastPoll() error {
	_, err := d.Exec(`
		UPDATE gmail_settings
		SET last_poll_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`)
	if err != nil {
		return fmt.Errorf("failed to update gmail last poll: %w", err)
	}
	return nil
}

// EnableGmail enables or disables Gmail integration
func (d *DB) EnableGmail(enabled bool) error {
	_, err := d.Exec(`
		UPDATE gmail_settings
		SET enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, enabled)
	if err != nil {
		return fmt.Errorf("failed to update gmail enabled: %w", err)
	}
	return nil
}

// IsEmailProcessed checks if an email has already been processed
func (d *DB) IsEmailProcessed(emailID string) (bool, error) {
	var count int
	err := d.QueryRow(`
		SELECT COUNT(*) FROM processed_emails WHERE email_id = ?
	`, emailID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check processed email: %w", err)
	}
	return count > 0, nil
}

// MarkEmailProcessed marks an email as processed
func (d *DB) MarkEmailProcessed(emailID string) error {
	_, err := d.Exec(`
		INSERT OR IGNORE INTO processed_emails (email_id) VALUES (?)
	`, emailID)
	if err != nil {
		return fmt.Errorf("failed to mark email processed: %w", err)
	}
	return nil
}

// CleanupOldProcessedEmails removes processed email records older than the specified duration
func (d *DB) CleanupOldProcessedEmails(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := d.Exec(`
		DELETE FROM processed_emails WHERE processed_at < ?
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup processed emails: %w", err)
	}
	return result.RowsAffected()
}
