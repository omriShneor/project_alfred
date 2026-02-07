package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/omriShneor/project_alfred/internal/source"
)

// SourceChannel represents a tracked channel with source type information
type SourceChannel struct {
	ID                int64              `json:"id"`
	UserID            int64              `json:"user_id"`
	SourceType        source.SourceType  `json:"source_type"`
	Type              source.ChannelType `json:"type"`
	Identifier        string             `json:"identifier"`
	Name              string             `json:"name"`
	Enabled           bool               `json:"enabled"`
	TotalMessageCount int                `json:"total_message_count"` // Actual message count from HistorySync
	LastMessageAt     *time.Time         `json:"last_message_at"`     // Timestamp of most recent message
	CreatedAt         time.Time          `json:"created_at"`
}

const (
	manualReminderSourceType  source.SourceType = "manual"
	manualReminderChannelID                     = "manual:todo"
	manualReminderChannelName                   = "My Tasks"
)

// ToSourceChannel converts a SourceChannel to source.Channel
func (sc *SourceChannel) ToSourceChannel() source.Channel {
	return source.Channel{
		ID:         sc.ID,
		UserID:     sc.UserID,
		SourceType: sc.SourceType,
		Type:       sc.Type,
		Identifier: sc.Identifier,
		Name:       sc.Name,
		Enabled:    sc.Enabled,
		CreatedAt:  sc.CreatedAt,
	}
}

// CreateSourceChannel creates a channel for any source type for a user
func (d *DB) CreateSourceChannel(userID int64, sourceType source.SourceType, channelType source.ChannelType, identifier, name string) (*SourceChannel, error) {
	result, err := d.Exec(
		`INSERT INTO channels (user_id, source_type, type, identifier, name) VALUES (?, ?, ?, ?, ?)`,
		userID, sourceType, channelType, identifier, name,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create source channel: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return d.GetSourceChannelByID(userID, id)
}

// EnsureManualReminderChannel returns a stable per-user channel for manually created reminders.
func (d *DB) EnsureManualReminderChannel(userID int64) (*SourceChannel, error) {
	channel, err := d.GetSourceChannelByIdentifier(userID, manualReminderSourceType, manualReminderChannelID)
	if err != nil {
		return nil, err
	}
	if channel != nil {
		return channel, nil
	}

	created, err := d.CreateSourceChannel(
		userID,
		manualReminderSourceType,
		source.ChannelTypeSender,
		manualReminderChannelID,
		manualReminderChannelName,
	)
	if err == nil {
		return created, nil
	}

	// Handle races where another request created the channel first.
	channel, lookupErr := d.GetSourceChannelByIdentifier(userID, manualReminderSourceType, manualReminderChannelID)
	if lookupErr == nil && channel != nil {
		return channel, nil
	}

	return nil, fmt.Errorf("failed to ensure manual reminder channel: %w", err)
}

// GetSourceChannelByID retrieves a channel by ID for a specific user
func (d *DB) GetSourceChannelByID(userID int64, id int64) (*SourceChannel, error) {
	row := d.QueryRow(
		`SELECT id, user_id, COALESCE(source_type, 'whatsapp'), type, identifier, name, enabled, created_at
		 FROM channels WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	return scanSourceChannel(row)
}

// GetSourceChannelByIdentifier retrieves a channel by source type and identifier for a specific user
func (d *DB) GetSourceChannelByIdentifier(userID int64, sourceType source.SourceType, identifier string) (*SourceChannel, error) {
	row := d.QueryRow(
		`SELECT id, user_id, COALESCE(source_type, 'whatsapp'), type, identifier, name, enabled, created_at
		 FROM channels WHERE user_id = ? AND source_type = ? AND identifier = ?`,
		userID, sourceType, identifier,
	)
	return scanSourceChannel(row)
}

// ListSourceChannels lists all channels for a given source type for a specific user
func (d *DB) ListSourceChannels(userID int64, sourceType source.SourceType) ([]*SourceChannel, error) {
	rows, err := d.Query(
		`SELECT id, user_id, COALESCE(source_type, 'whatsapp'), type, identifier, name, enabled, created_at
		 FROM channels WHERE user_id = ? AND source_type = ? ORDER BY created_at DESC`,
		userID, sourceType,
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

// UpdateSourceChannel updates a channel's properties for a specific user
func (d *DB) UpdateSourceChannel(userID int64, id int64, name string, enabled bool) error {
	result, err := d.Exec(
		`UPDATE channels SET name = ?, enabled = ? WHERE id = ? AND user_id = ?`,
		name, enabled, id, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update source channel: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update source channel: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("channel not found")
	}
	return nil
}

// DeleteSourceChannel deletes a channel by ID for a specific user
func (d *DB) DeleteSourceChannel(userID int64, id int64) error {
	result, err := d.Exec(`DELETE FROM channels WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete source channel: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to delete source channel: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("channel not found")
	}
	return nil
}

// DeleteSourceChannelByIdentifier deletes a channel by source type + identifier for a specific user
func (d *DB) DeleteSourceChannelByIdentifier(userID int64, sourceType source.SourceType, identifier string) error {
	result, err := d.Exec(
		`DELETE FROM channels WHERE user_id = ? AND source_type = ? AND identifier = ?`,
		userID, sourceType, identifier,
	)
	if err != nil {
		return fmt.Errorf("failed to delete source channel: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to delete source channel: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("channel not found")
	}
	return nil
}

// IsSourceChannelTracked checks if a channel is tracked and enabled for a specific source type and user
// Returns: isTracked, channelID, channelType, error
func (d *DB) IsSourceChannelTracked(userID int64, sourceType source.SourceType, identifier string) (bool, int64, source.ChannelType, error) {
	var id int64
	var channelType source.ChannelType
	err := d.QueryRow(
		`SELECT id, type FROM channels WHERE user_id = ? AND source_type = ? AND identifier = ? AND enabled = 1`,
		userID, sourceType, identifier,
	).Scan(&id, &channelType)

	if err == sql.ErrNoRows {
		return false, 0, "", nil
	}
	if err != nil {
		return false, 0, "", fmt.Errorf("failed to check source channel: %w", err)
	}
	return true, id, channelType, nil
}

// UserHasAnySources returns true if the user has any enabled sources (channels or email sources)
func (d *DB) UserHasAnySources(userID int64) (bool, error) {
	var exists int
	err := d.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM channels WHERE user_id = ? AND enabled = 1
			UNION ALL
			SELECT 1 FROM email_sources WHERE user_id = ? AND enabled = 1
		)
	`, userID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user sources: %w", err)
	}
	return exists == 1, nil
}

// UpdateChannelStats updates the message count and last message time for a channel
func (d *DB) UpdateChannelStats(id int64, totalMessageCount int, lastMessageAt *time.Time) error {
	_, err := d.Exec(`
		UPDATE channels
		SET total_message_count = ?, last_message_at = ?
		WHERE id = ?
	`, totalMessageCount, lastMessageAt, id)
	if err != nil {
		return fmt.Errorf("failed to update channel stats: %w", err)
	}
	return nil
}

// GetTopChannelsByMessageCount returns top channels by actual message count for a specific user
// This uses total_message_count which is populated during HistorySync with accurate counts
func (d *DB) GetTopChannelsByMessageCount(userID int64, sourceType source.SourceType, limit int) ([]*SourceChannel, error) {
	rows, err := d.Query(`
		SELECT id, user_id, COALESCE(source_type, 'whatsapp'), type, identifier, name, enabled,
		       total_message_count, last_message_at, created_at
		FROM channels
		WHERE user_id = ? AND source_type = ? AND total_message_count > 0
		ORDER BY total_message_count DESC
		LIMIT ?
	`, userID, sourceType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top channels: %w", err)
	}
	defer rows.Close()

	var channels []*SourceChannel
	for rows.Next() {
		var c SourceChannel
		var lastMsgAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.UserID, &c.SourceType, &c.Type, &c.Identifier, &c.Name,
			&c.Enabled, &c.TotalMessageCount, &lastMsgAt, &c.CreatedAt); err != nil {
			continue
		}
		if lastMsgAt.Valid {
			c.LastMessageAt = &lastMsgAt.Time
		}
		channels = append(channels, &c)
	}

	return channels, rows.Err()
}

func scanSourceChannel(row *sql.Row) (*SourceChannel, error) {
	var c SourceChannel
	err := row.Scan(&c.ID, &c.UserID, &c.SourceType, &c.Type, &c.Identifier, &c.Name, &c.Enabled, &c.CreatedAt)
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
	err := rows.Scan(&c.ID, &c.UserID, &c.SourceType, &c.Type, &c.Identifier, &c.Name, &c.Enabled, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan source channel: %w", err)
	}
	return &c, nil
}
