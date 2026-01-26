package database

import (
	"database/sql"
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

// EmailSource represents a tracked email source
type EmailSource struct {
	ID         int64           `json:"id"`
	Type       EmailSourceType `json:"type"`
	Identifier string          `json:"identifier"`
	Name       string          `json:"name"`
	Enabled    bool            `json:"enabled"`
	CalendarID string          `json:"calendar_id"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// CreateEmailSource creates a new email source
func (d *DB) CreateEmailSource(sourceType EmailSourceType, identifier, name string) (*EmailSource, error) {
	result, err := d.Exec(`
		INSERT INTO email_sources (type, identifier, name)
		VALUES (?, ?, ?)
	`, sourceType, identifier, name)
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
		SELECT id, type, identifier, name, enabled, calendar_id, created_at, updated_at
		FROM email_sources WHERE id = ?
	`, id).Scan(&source.ID, &source.Type, &source.Identifier, &source.Name,
		&source.Enabled, &source.CalendarID, &source.CreatedAt, &source.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email source: %w", err)
	}

	return &source, nil
}

// GetEmailSourceByIdentifier retrieves an email source by type and identifier
func (d *DB) GetEmailSourceByIdentifier(sourceType EmailSourceType, identifier string) (*EmailSource, error) {
	var source EmailSource
	err := d.QueryRow(`
		SELECT id, type, identifier, name, enabled, calendar_id, created_at, updated_at
		FROM email_sources WHERE type = ? AND identifier = ?
	`, sourceType, identifier).Scan(&source.ID, &source.Type, &source.Identifier, &source.Name,
		&source.Enabled, &source.CalendarID, &source.CreatedAt, &source.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email source by identifier: %w", err)
	}

	return &source, nil
}

// ListEmailSources retrieves all email sources
func (d *DB) ListEmailSources() ([]*EmailSource, error) {
	rows, err := d.Query(`
		SELECT id, type, identifier, name, enabled, calendar_id, created_at, updated_at
		FROM email_sources ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list email sources: %w", err)
	}
	defer rows.Close()

	var sources []*EmailSource
	for rows.Next() {
		var source EmailSource
		if err := rows.Scan(&source.ID, &source.Type, &source.Identifier, &source.Name,
			&source.Enabled, &source.CalendarID, &source.CreatedAt, &source.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan email source: %w", err)
		}
		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// ListEmailSourcesByType retrieves email sources filtered by type
func (d *DB) ListEmailSourcesByType(sourceType EmailSourceType) ([]*EmailSource, error) {
	rows, err := d.Query(`
		SELECT id, type, identifier, name, enabled, calendar_id, created_at, updated_at
		FROM email_sources WHERE type = ? ORDER BY created_at DESC
	`, sourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to list email sources by type: %w", err)
	}
	defer rows.Close()

	var sources []*EmailSource
	for rows.Next() {
		var source EmailSource
		if err := rows.Scan(&source.ID, &source.Type, &source.Identifier, &source.Name,
			&source.Enabled, &source.CalendarID, &source.CreatedAt, &source.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan email source: %w", err)
		}
		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// ListEnabledEmailSources retrieves all enabled email sources
func (d *DB) ListEnabledEmailSources() ([]*EmailSource, error) {
	rows, err := d.Query(`
		SELECT id, type, identifier, name, enabled, calendar_id, created_at, updated_at
		FROM email_sources WHERE enabled = 1 ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled email sources: %w", err)
	}
	defer rows.Close()

	var sources []*EmailSource
	for rows.Next() {
		var source EmailSource
		if err := rows.Scan(&source.ID, &source.Type, &source.Identifier, &source.Name,
			&source.Enabled, &source.CalendarID, &source.CreatedAt, &source.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan email source: %w", err)
		}
		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// UpdateEmailSource updates an email source
func (d *DB) UpdateEmailSource(id int64, name, calendarID string, enabled bool) error {
	_, err := d.Exec(`
		UPDATE email_sources
		SET name = ?, calendar_id = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, calendarID, enabled, id)
	if err != nil {
		return fmt.Errorf("failed to update email source: %w", err)
	}
	return nil
}

// DeleteEmailSource deletes an email source
func (d *DB) DeleteEmailSource(id int64) error {
	_, err := d.Exec(`DELETE FROM email_sources WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete email source: %w", err)
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
