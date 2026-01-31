package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/source"
)

// SourceChannel represents a tracked channel with source type information
type SourceChannel struct {
	ID         int64             `json:"id"`
	SourceType source.SourceType `json:"source_type"`
	Type       source.ChannelType `json:"type"`
	Identifier string            `json:"identifier"`
	Name       string            `json:"name"`
	CalendarID string            `json:"calendar_id"`
	Enabled    bool              `json:"enabled"`
	CreatedAt  time.Time         `json:"created_at"`
}

// ToSourceChannel converts a SourceChannel to source.Channel
func (sc *SourceChannel) ToSourceChannel() source.Channel {
	return source.Channel{
		ID:         sc.ID,
		SourceType: sc.SourceType,
		Type:       sc.Type,
		Identifier: sc.Identifier,
		Name:       sc.Name,
		CalendarID: sc.CalendarID,
		Enabled:    sc.Enabled,
		CreatedAt:  sc.CreatedAt,
	}
}

// CreateSourceChannel creates a channel for any source type
func (d *DB) CreateSourceChannel(sourceType source.SourceType, channelType source.ChannelType, identifier, name, calendarID string) (*SourceChannel, error) {
	if calendarID == "" {
		calendarID = "primary"
	}

	result, err := d.Exec(
		`INSERT INTO channels (source_type, type, identifier, name, calendar_id) VALUES (?, ?, ?, ?, ?)`,
		sourceType, channelType, identifier, name, calendarID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create source channel: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return d.GetSourceChannelByID(id)
}

// GetSourceChannelByID retrieves a channel by ID
func (d *DB) GetSourceChannelByID(id int64) (*SourceChannel, error) {
	row := d.QueryRow(
		`SELECT id, COALESCE(source_type, 'whatsapp'), type, identifier, name, calendar_id, enabled, created_at
		 FROM channels WHERE id = ?`,
		id,
	)
	return scanSourceChannel(row)
}

// GetSourceChannelByIdentifier retrieves a channel by source type and identifier
func (d *DB) GetSourceChannelByIdentifier(sourceType source.SourceType, identifier string) (*SourceChannel, error) {
	row := d.QueryRow(
		`SELECT id, COALESCE(source_type, 'whatsapp'), type, identifier, name, calendar_id, enabled, created_at
		 FROM channels WHERE source_type = ? AND identifier = ?`,
		sourceType, identifier,
	)
	return scanSourceChannel(row)
}

// ListSourceChannels lists all channels for a given source type
func (d *DB) ListSourceChannels(sourceType source.SourceType) ([]*SourceChannel, error) {
	rows, err := d.Query(
		`SELECT id, COALESCE(source_type, 'whatsapp'), type, identifier, name, calendar_id, enabled, created_at
		 FROM channels WHERE source_type = ? ORDER BY created_at DESC`,
		sourceType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list source channels: %w", err)
	}
	defer rows.Close()

	var channels []*SourceChannel
	for rows.Next() {
		channel, err := scanSourceChannelRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

// ListEnabledSourceChannels lists all enabled channels for a source type
func (d *DB) ListEnabledSourceChannels(sourceType source.SourceType) ([]*SourceChannel, error) {
	rows, err := d.Query(
		`SELECT id, COALESCE(source_type, 'whatsapp'), type, identifier, name, calendar_id, enabled, created_at
		 FROM channels WHERE source_type = ? AND enabled = 1 ORDER BY created_at DESC`,
		sourceType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled source channels: %w", err)
	}
	defer rows.Close()

	var channels []*SourceChannel
	for rows.Next() {
		channel, err := scanSourceChannelRows(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, rows.Err()
}

// UpdateSourceChannel updates a channel's properties
func (d *DB) UpdateSourceChannel(id int64, name, calendarID string, enabled bool) error {
	_, err := d.Exec(
		`UPDATE channels SET name = ?, calendar_id = ?, enabled = ? WHERE id = ?`,
		name, calendarID, enabled, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update source channel: %w", err)
	}
	return nil
}

// DeleteSourceChannel deletes a channel by ID
func (d *DB) DeleteSourceChannel(id int64) error {
	_, err := d.Exec(`DELETE FROM channels WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete source channel: %w", err)
	}
	return nil
}

// IsSourceChannelTracked checks if a channel is tracked and enabled for a specific source type
// Returns: isTracked, channelID, channelType, error
func (d *DB) IsSourceChannelTracked(sourceType source.SourceType, identifier string) (bool, int64, source.ChannelType, error) {
	var id int64
	var channelType source.ChannelType
	err := d.QueryRow(
		`SELECT id, type FROM channels WHERE source_type = ? AND identifier = ? AND enabled = 1`,
		sourceType, identifier,
	).Scan(&id, &channelType)

	if err == sql.ErrNoRows {
		return false, 0, "", nil
	}
	if err != nil {
		return false, 0, "", fmt.Errorf("failed to check source channel: %w", err)
	}
	return true, id, channelType, nil
}

func scanSourceChannel(row *sql.Row) (*SourceChannel, error) {
	var c SourceChannel
	err := row.Scan(&c.ID, &c.SourceType, &c.Type, &c.Identifier, &c.Name, &c.CalendarID, &c.Enabled, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan source channel: %w", err)
	}
	return &c, nil
}

func scanSourceChannelRows(rows *sql.Rows) (*SourceChannel, error) {
	var c SourceChannel
	err := rows.Scan(&c.ID, &c.SourceType, &c.Type, &c.Identifier, &c.Name, &c.CalendarID, &c.Enabled, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan source channel: %w", err)
	}
	return &c, nil
}
