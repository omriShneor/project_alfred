package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// EmailSourceType represents the type of email source
type EmailSourceType string

const (
	EmailSourceTypeCategory EmailSourceType = "category"
	EmailSourceTypeSender   EmailSourceType = "sender"
	EmailSourceTypeDomain   EmailSourceType = "domain"
)

var ErrEmailSourceNotFound = errors.New("email source not found")

// EmailSource represents a tracked email source
type EmailSource struct {
	ID         int64           `json:"id"`
	UserID     int64           `json:"user_id"`
	Type       EmailSourceType `json:"type"`
	Identifier string          `json:"identifier"`
	Name       string          `json:"name"`
	Enabled    bool            `json:"enabled"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// CreateEmailSource creates a new email source for a user
func (d *DB) CreateEmailSource(userID int64, sourceType EmailSourceType, identifier, name string) (*EmailSource, error) {
	result, err := d.Exec(`
		INSERT INTO email_sources (user_id, type, identifier, name)
		VALUES (?, ?, ?, ?)
	`, userID, sourceType, identifier, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create email source: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get email source id: %w", err)
	}

	return d.GetEmailSourceByID(id)
}

// GetEmailSourceByID retrieves an email source by ID
func (d *DB) GetEmailSourceByID(id int64) (*EmailSource, error) {
	var source EmailSource
	err := d.QueryRow(`
		SELECT id, user_id, type, identifier, name, enabled, created_at, updated_at
		FROM email_sources WHERE id = ?
	`, id).Scan(&source.ID, &source.UserID, &source.Type, &source.Identifier, &source.Name,
		&source.Enabled, &source.CreatedAt, &source.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email source: %w", err)
	}

	return &source, nil
}

// GetEmailSourceByIDForUser retrieves an email source by ID for a specific user.
func (d *DB) GetEmailSourceByIDForUser(userID, id int64) (*EmailSource, error) {
	var source EmailSource
	err := d.QueryRow(`
		SELECT id, user_id, type, identifier, name, enabled, created_at, updated_at
		FROM email_sources WHERE user_id = ? AND id = ?
	`, userID, id).Scan(&source.ID, &source.UserID, &source.Type, &source.Identifier, &source.Name,
		&source.Enabled, &source.CreatedAt, &source.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email source for user: %w", err)
	}

	return &source, nil
}

// GetEmailSourceByIdentifier retrieves an email source by user, type and identifier
func (d *DB) GetEmailSourceByIdentifier(userID int64, sourceType EmailSourceType, identifier string) (*EmailSource, error) {
	var source EmailSource
	err := d.QueryRow(`
		SELECT id, user_id, type, identifier, name, enabled, created_at, updated_at
		FROM email_sources WHERE user_id = ? AND type = ? AND identifier = ?
	`, userID, sourceType, identifier).Scan(&source.ID, &source.UserID, &source.Type, &source.Identifier, &source.Name,
		&source.Enabled, &source.CreatedAt, &source.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email source by identifier: %w", err)
	}

	return &source, nil
}

// ListEmailSources retrieves all email sources for a user
func (d *DB) ListEmailSources(userID int64) ([]*EmailSource, error) {
	rows, err := d.Query(`
		SELECT id, user_id, type, identifier, name, enabled, created_at, updated_at
		FROM email_sources WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list email sources: %w", err)
	}
	defer rows.Close()

	var sources []*EmailSource
	for rows.Next() {
		var source EmailSource
		if err := rows.Scan(&source.ID, &source.UserID, &source.Type, &source.Identifier, &source.Name,
			&source.Enabled, &source.CreatedAt, &source.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan email source: %w", err)
		}
		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// ListEmailSourcesByType retrieves email sources for a user filtered by type
func (d *DB) ListEmailSourcesByType(userID int64, sourceType EmailSourceType) ([]*EmailSource, error) {
	rows, err := d.Query(`
		SELECT id, user_id, type, identifier, name, enabled, created_at, updated_at
		FROM email_sources WHERE user_id = ? AND type = ? ORDER BY created_at DESC
	`, userID, sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to list email sources by type: %w", err)
	}
	defer rows.Close()

	var sources []*EmailSource
	for rows.Next() {
		var source EmailSource
		if err := rows.Scan(&source.ID, &source.UserID, &source.Type, &source.Identifier, &source.Name,
			&source.Enabled, &source.CreatedAt, &source.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan email source: %w", err)
		}
		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// ListEnabledEmailSources retrieves all enabled email sources for a user
func (d *DB) ListEnabledEmailSources(userID int64) ([]*EmailSource, error) {
	rows, err := d.Query(`
		SELECT id, user_id, type, identifier, name, enabled, created_at, updated_at
		FROM email_sources WHERE user_id = ? AND enabled = 1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled email sources: %w", err)
	}
	defer rows.Close()

	var sources []*EmailSource
	for rows.Next() {
		var source EmailSource
		if err := rows.Scan(&source.ID, &source.UserID, &source.Type, &source.Identifier, &source.Name,
			&source.Enabled, &source.CreatedAt, &source.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan email source: %w", err)
		}
		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// UpdateEmailSource updates an email source
func (d *DB) UpdateEmailSource(id int64, name string, enabled bool) error {
	result, err := d.Exec(`
		UPDATE email_sources
		SET name = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, enabled, id)
	if err != nil {
		return fmt.Errorf("failed to update email source: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update email source: %w", err)
	}
	if rowsAffected == 0 {
		return ErrEmailSourceNotFound
	}
	return nil
}

// UpdateEmailSourceForUser updates an email source by ID scoped to a specific user.
func (d *DB) UpdateEmailSourceForUser(userID, id int64, name string, enabled bool) error {
	result, err := d.Exec(`
		UPDATE email_sources
		SET name = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND id = ?
	`, name, enabled, userID, id)
	if err != nil {
		return fmt.Errorf("failed to update email source: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update email source: %w", err)
	}
	if rowsAffected == 0 {
		return ErrEmailSourceNotFound
	}
	return nil
}

// DeleteEmailSource deletes an email source
func (d *DB) DeleteEmailSource(id int64) error {
	result, err := d.Exec(`DELETE FROM email_sources WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete email source: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to delete email source: %w", err)
	}
	if rowsAffected == 0 {
		return ErrEmailSourceNotFound
	}
	return nil
}

// DeleteEmailSourceForUser deletes an email source by ID scoped to a specific user.
func (d *DB) DeleteEmailSourceForUser(userID, id int64) error {
	result, err := d.Exec(`DELETE FROM email_sources WHERE user_id = ? AND id = ?`, userID, id)
	if err != nil {
		return fmt.Errorf("failed to delete email source: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to delete email source: %w", err)
	}
	if rowsAffected == 0 {
		return ErrEmailSourceNotFound
	}
	return nil
}

// IsEmailSourceTracked checks if an email source exists and is enabled
func (d *DB) IsEmailSourceTracked(sourceType EmailSourceType, identifier string) (bool, int64, error) {
	var id int64
	err := d.QueryRow(`
		SELECT id FROM email_sources WHERE type = ? AND identifier = ? AND enabled = 1
	`, sourceType, identifier).Scan(&id)

	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, fmt.Errorf("failed to check email source: %w", err)
	}
	return true, id, nil
}
