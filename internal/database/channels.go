package database

import (
	"database/sql"
	"fmt"
	"time"
)

// ChannelType represents the type of communication channel
type ChannelType string

const (
	ChannelTypeSender ChannelType = "sender"
)

// Channel represents a tracked communication channel (contacts only)
type Channel struct {
	ID         int64       `json:"id"`
	Type       ChannelType `json:"type"`       // "sender" (contacts only)
	Identifier string      `json:"identifier"` // phone number for WhatsApp, user ID for Telegram
	Name       string      `json:"name"`       // display name
	Enabled    bool        `json:"enabled"`    // whether to track this channel
	CreatedAt  time.Time   `json:"created_at"`
}

func (d *DB) CreateChannel(channelType ChannelType, identifier, name string) (*Channel, error) {
	result, err := d.Exec(
		`INSERT INTO channels (type, identifier, name) VALUES (?, ?, ?)`,
		channelType, identifier, name,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return d.GetChannelByID(id)
}

func (d *DB) GetChannelByID(id int64) (*Channel, error) {
	row := d.QueryRow(
		`SELECT id, type, identifier, name, enabled, created_at
		 FROM channels WHERE id = ?`,
		id,
	)
	return scanChannel(row)
}

func (d *DB) GetChannelByIdentifier(identifier string) (*Channel, error) {
	row := d.QueryRow(
		`SELECT id, type, identifier, name, enabled, created_at
		 FROM channels WHERE identifier = ?`,
		identifier,
	)
	return scanChannel(row)
}

func (d *DB) ListChannels() ([]*Channel, error) {
	rows, err := d.Query(
		`SELECT id, type, identifier, name, enabled, created_at
		 FROM channels ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		channel, err := scanChannelRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

func (d *DB) ListEnabledChannels() ([]*Channel, error) {
	rows, err := d.Query(
		`SELECT id, type, identifier, name, enabled, created_at
		 FROM channels WHERE enabled = 1 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		channel, err := scanChannelRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

func (d *DB) UpdateChannel(id int64, name string, enabled bool) error {
	_, err := d.Exec(
		`UPDATE channels SET name = ?, enabled = ? WHERE id = ?`,
		name, enabled, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update channel: %w", err)
	}
	return nil
}

func (d *DB) DeleteChannel(userID, id int64) error {
	result, err := d.Exec(`DELETE FROM channels WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("channel not found")
	}
	return nil
}

// IsChannelTracked checks if a channel with the given identifier is tracked and enabled
// Returns: isTracked, channelID, channelType, error
func (d *DB) IsChannelTracked(identifier string) (bool, int64, ChannelType, error) {
	var id int64
	var channelType ChannelType
	err := d.QueryRow(
		`SELECT id, type FROM channels WHERE identifier = ? AND enabled = 1`,
		identifier,
	).Scan(&id, &channelType)

	if err == sql.ErrNoRows {
		return false, 0, "", nil
	}
	if err != nil {
		return false, 0, "", fmt.Errorf("failed to check channel: %w", err)
	}
	return true, id, channelType, nil
}

func scanChannel(row *sql.Row) (*Channel, error) {
	var c Channel
	err := row.Scan(&c.ID, &c.Type, &c.Identifier, &c.Name, &c.Enabled, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan channel: %w", err)
	}
	return &c, nil
}

func scanChannelRows(rows *sql.Rows) (*Channel, error) {
	var c Channel
	err := rows.Scan(&c.ID, &c.Type, &c.Identifier, &c.Name, &c.Enabled, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan channel: %w", err)
	}
	return &c, nil
}
