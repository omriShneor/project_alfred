package database

import (
	"database/sql"
	"fmt"
	"time"
)

// GmailSettings represents Gmail integration settings
type GmailSettings struct {
	ID                  int64      `json:"id"`
	UserID              int64      `json:"user_id"`
	Enabled             bool       `json:"enabled"`
	PollIntervalMinutes int        `json:"poll_interval_minutes"`
	LastPollAt          *time.Time `json:"last_poll_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// GetGmailSettings retrieves the Gmail settings for a user
func (d *DB) GetGmailSettings(userID int64) (*GmailSettings, error) {
	var settings GmailSettings
	var lastPollAt sql.NullTime

	err := d.QueryRow(`
		SELECT id, user_id, enabled, poll_interval_minutes, last_poll_at, created_at, updated_at
		FROM gmail_settings WHERE user_id = ?
	`, userID).Scan(&settings.ID, &settings.UserID, &settings.Enabled, &settings.PollIntervalMinutes,
		&lastPollAt, &settings.CreatedAt, &settings.UpdatedAt)

	if err == sql.ErrNoRows {
		// Create default settings for this user
		_, err = d.Exec(`
			INSERT INTO gmail_settings (user_id, enabled, poll_interval_minutes)
			VALUES (?, 0, 5)
		`, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to create default gmail settings: %w", err)
		}
		// Return fresh settings (avoid recursion)
		return &GmailSettings{
			UserID:              userID,
			Enabled:             false,
			PollIntervalMinutes: 5,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get gmail settings: %w", err)
	}

	if lastPollAt.Valid {
		settings.LastPollAt = &lastPollAt.Time
	}

	return &settings, nil
}

// SetGmailEnabled enables or disables Gmail processing for a user.
func (d *DB) SetGmailEnabled(userID int64, enabled bool) error {
	// Ensure settings row exists
	if _, err := d.Exec(`
		INSERT OR IGNORE INTO gmail_settings (user_id, enabled)
		VALUES (?, ?)
	`, userID, enabled); err != nil {
		return fmt.Errorf("failed to ensure gmail settings: %w", err)
	}

	_, err := d.Exec(`
		UPDATE gmail_settings
		SET enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, enabled, userID)
	if err != nil {
		return fmt.Errorf("failed to set gmail enabled: %w", err)
	}
	return nil
}

// UpdateGmailLastPoll updates the last poll timestamp for a user
func (d *DB) UpdateGmailLastPoll(userID int64) error {
	_, err := d.Exec(`
		UPDATE gmail_settings
		SET last_poll_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to update gmail last poll: %w", err)
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

// TopContact represents a cached top email contact for quick discovery
type TopContact struct {
	ID          int64     `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	EmailCount  int       `json:"email_count"`
	LastUpdated time.Time `json:"last_updated"`
}

// GetTopContacts retrieves the cached top contacts for a user up to the specified limit
func (d *DB) GetTopContacts(userID int64, limit int) ([]TopContact, error) {
	rows, err := d.Query(`
		SELECT id, email, name, email_count, last_updated
		FROM gmail_top_contacts
		WHERE user_id = ?
		ORDER BY email_count DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top contacts: %w", err)
	}
	defer rows.Close()

	var contacts []TopContact
	for rows.Next() {
		var c TopContact
		var name sql.NullString
		if err := rows.Scan(&c.ID, &c.Email, &name, &c.EmailCount, &c.LastUpdated); err != nil {
			return nil, fmt.Errorf("failed to scan top contact: %w", err)
		}
		if name.Valid {
			c.Name = name.String
		}
		contacts = append(contacts, c)
	}

	return contacts, nil
}

// ReplaceTopContacts replaces all cached top contacts for a user with the new list
func (d *DB) ReplaceTopContacts(userID int64, contacts []TopContact) error {
	tx, err := d.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing contacts for this user
	if _, err := tx.Exec(`DELETE FROM gmail_top_contacts WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("failed to clear top contacts: %w", err)
	}

	// Insert new contacts
	stmt, err := tx.Prepare(`
		INSERT INTO gmail_top_contacts (user_id, email, name, email_count, last_updated)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, c := range contacts {
		var name interface{}
		if c.Name != "" {
			name = c.Name
		}
		if _, err := stmt.Exec(userID, c.Email, name, c.EmailCount); err != nil {
			return fmt.Errorf("failed to insert top contact: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetTopContactsComputedAt returns when top contacts were last computed for a user
func (d *DB) GetTopContactsComputedAt(userID int64) (*time.Time, error) {
	var computedAt sql.NullTime
	err := d.QueryRow(`
		SELECT top_contacts_computed_at FROM gmail_settings WHERE user_id = ?
	`, userID).Scan(&computedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get top contacts computed at: %w", err)
	}
	if computedAt.Valid {
		return &computedAt.Time, nil
	}
	return nil, nil
}

// SetTopContactsComputedAt updates when top contacts were last computed for a user
func (d *DB) SetTopContactsComputedAt(userID int64, t time.Time) error {
	_, err := d.Exec(`
		UPDATE gmail_settings
		SET top_contacts_computed_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, t, userID)
	if err != nil {
		return fmt.Errorf("failed to set top contacts computed at: %w", err)
	}
	return nil
}
